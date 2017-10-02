package builder

import (
	"fmt"
	"sort"
	"strings"

	"github.com/golang/protobuf/proto"

	"github.com/jhump/protoreflect/desc"
)

// dependencyResolver is the work-horse for converting a tree of builders into a
// tree of descriptors. It scans a root (usually a file builder) and recursively
// resolves all dependencies (references to builders in other trees as well as
// references to other already-built descriptors). The result of resolution is a
// file descriptor (or an error).
type dependencyResolver struct {
	resolvedRoots map[Builder]*desc.FileDescriptor
	seen          map[Builder]struct{}
}

func newResolver() *dependencyResolver {
	return &dependencyResolver{
		resolvedRoots: map[Builder]*desc.FileDescriptor{},
		seen:          map[Builder]struct{}{},
	}
}

func (r *dependencyResolver) resolveElement(b Builder, seen []Builder) (*desc.FileDescriptor, error) {
	b = getRoot(b)

	if fd, ok := r.resolvedRoots[b]; ok {
		return fd, nil
	}

	for _, s := range seen {
		if s == b {
			names := make([]string, len(seen)+1)
			for i, s := range seen {
				names[i] = s.GetName()
			}
			names[len(seen)] = b.GetName()
			return nil, fmt.Errorf("descriptors have cyclic dependency: %s", strings.Join(names, " ->  "))
		}
	}
	seen = append(seen, b)

	var fd *desc.FileDescriptor
	var err error
	switch b := b.(type) {
	case *FileBuilder:
		fd, err = r.resolveFile(b, b, seen)
	default:
		fd, err = r.resolveSyntheticFile(b, seen)
	}
	if err != nil {
		return nil, err
	}
	r.resolvedRoots[b] = fd
	return fd, nil
}

func (r *dependencyResolver) resolveFile(fb *FileBuilder, root Builder, seen []Builder) (*desc.FileDescriptor, error) {
	deps := map[*desc.FileDescriptor]struct{}{}
	for _, mb := range fb.messages {
		if err := r.resolveTypesInMessage(root, seen, deps, mb); err != nil {
			return nil, err
		}
	}
	for _, exb := range fb.extensions {
		if err := r.resolveTypesInExtension(root, seen, deps, exb); err != nil {
			return nil, err
		}
	}
	for _, sb := range fb.services {
		if err := r.resolveTypesInService(root, seen, deps, sb); err != nil {
			return nil, err
		}
	}

	depSlice := make([]*desc.FileDescriptor, 0, len(deps))
	for dep := range deps {
		depSlice = append(depSlice, dep)
	}

	fp, err := fb.buildProto()
	if err != nil {
		return nil, err
	}
	for _, dep := range depSlice {
		fp.Dependency = append(fp.Dependency, dep.GetName())
	}
	sort.Strings(fp.Dependency)

	// make sure this file name doesn't collide with any of its dependencies
	fileNames := map[string]struct{}{}
	for _, d := range depSlice {
		addFileNames(d, fileNames)
		fileNames[d.GetName()] = struct{}{}
	}
	unique := makeUnique(fp.GetName(), fileNames)
	if unique != fp.GetName() {
		fp.Name = proto.String(unique)
	}

	return desc.CreateFileDescriptor(fp, depSlice...)
}

func addFileNames(fd *desc.FileDescriptor, files map[string]struct{}) {
	if _, ok := files[fd.GetName()]; ok {
		// already added
		return
	}
	files[fd.GetName()] = struct{}{}
	for _, d := range fd.GetDependencies() {
		addFileNames(d, files)
	}
}

func (r *dependencyResolver) resolveSyntheticFile(b Builder, seen []Builder) (*desc.FileDescriptor, error) {
	// find ancestor to temporarily attach to new file
	curr := b
	for curr.GetParent() != nil {
		curr = curr.GetParent()
	}
	f := NewFile("")
	switch curr := curr.(type) {
	case *MessageBuilder:
		f.messages = append(f.messages, curr)
	case *EnumBuilder:
		f.enums = append(f.enums, curr)
	case *ServiceBuilder:
		f.services = append(f.services, curr)
	case *FieldBuilder:
		if curr.IsExtension() {
			f.extensions = append(f.extensions, curr)
		} else {
			panic("field must be added to message before calling Build()")
		}
	case *OneOfBuilder:
		if _, ok := b.(*OneOfBuilder); ok {
			panic("one-of must be added to message before calling Build()")
		} else {
			// b was a child of one-of which means it must have been a field
			panic("field must be added to message before calling Build()")
		}
	case *MethodBuilder:
		panic("method must be added to service before calling Build()")
	case *EnumValueBuilder:
		panic("enum value must be added to enum before calling Build()")
	default:
		panic(fmt.Sprintf("Unrecognized kind of builder: %T", b))
	}
	curr.setParent(f)

	// don't forget to reset when done
	defer func() {
		curr.setParent(nil)
	}()

	return r.resolveFile(f, b, seen)
}

func (r *dependencyResolver) resolveTypesInMessage(root Builder, seen []Builder, deps map[*desc.FileDescriptor]struct{}, mb *MessageBuilder) error {
	for _, b := range mb.fieldsAndOneOfs {
		if flb, ok := b.(*FieldBuilder); ok {
			if err := r.resolveTypesInField(root, seen, flb, deps); err != nil {
				return err
			}
		} else {
			oob := b.(*OneOfBuilder)
			for _, flb := range oob.choices {
				if err := r.resolveTypesInField(root, seen, flb, deps); err != nil {
					return err
				}
			}
		}
	}
	for _, nmb := range mb.nestedMessages {
		if err := r.resolveTypesInMessage(root, seen, deps, nmb); err != nil {
			return err
		}
	}
	for _, exb := range mb.nestedExtensions {
		if err := r.resolveTypesInExtension(root, seen, deps, exb); err != nil {
			return err
		}
	}
	return nil
}

func (r *dependencyResolver) resolveTypesInExtension(root Builder, seen []Builder, deps map[*desc.FileDescriptor]struct{}, exb *FieldBuilder) error {
	if err := r.resolveTypesInField(root, seen, exb, deps); err != nil {
		return err
	}
	if exb.foreignExtendee != nil {
		deps[exb.foreignExtendee.GetFile()] = struct{}{}
	} else if err := r.resolveType(root, seen, exb.localExtendee, deps); err != nil {
		return err
	}
	return nil
}

func (r *dependencyResolver) resolveTypesInService(root Builder, seen []Builder, deps map[*desc.FileDescriptor]struct{}, sb *ServiceBuilder) error {
	for _, mtb := range sb.methods {
		if err := r.resolveRpcType(root, seen, mtb.ReqType, deps); err != nil {
			return err
		}
		if err := r.resolveRpcType(root, seen, mtb.RespType, deps); err != nil {
			return err
		}
	}
	return nil
}

func (r *dependencyResolver) resolveRpcType(root Builder, seen []Builder, t *RpcType, deps map[*desc.FileDescriptor]struct{}) error {
	if t.foreignType != nil {
		deps[t.foreignType.GetFile()] = struct{}{}
	} else {
		return r.resolveType(root, seen, t.localType, deps)
	}
	return nil
}

func (r *dependencyResolver) resolveTypesInField(root Builder, seen []Builder, flb *FieldBuilder, deps map[*desc.FileDescriptor]struct{}) error {
	if flb.fieldType.foreignMsgType != nil {
		deps[flb.fieldType.foreignMsgType.GetFile()] = struct{}{}
	} else if flb.fieldType.foreignEnumType != nil {
		deps[flb.fieldType.foreignEnumType.GetFile()] = struct{}{}
	} else if flb.fieldType.localMsgType != nil {
		if flb.fieldType.localMsgType == flb.msgType {
			return r.resolveTypesInMessage(root, seen, deps, flb.msgType)
		} else {
			return r.resolveType(root, seen, flb.fieldType.localMsgType, deps)
		}
	} else if flb.fieldType.localEnumType != nil {
		return r.resolveType(root, seen, flb.fieldType.localEnumType, deps)
	}
	return nil
}

func (r *dependencyResolver) resolveType(root Builder, seen []Builder, typeBuilder Builder, deps map[*desc.FileDescriptor]struct{}) error {
	otherRoot := getRoot(typeBuilder)
	if root == otherRoot {
		// local reference, so it will get resolved when we finish resolving this root
		return nil
	}
	fd, err := r.resolveElement(otherRoot, seen)
	if err != nil {
		return err
	}
	deps[fd] = struct{}{}
	return nil
}
