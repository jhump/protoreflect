package protobuilder

import (
	"fmt"
	"reflect"
	"strings"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/v2/internal/register"
	"github.com/jhump/protoreflect/v2/protoresolve"
)

type dependencies struct {
	descs map[protoreflect.FileDescriptor]struct{}
	res   protoregistry.Types
}

func newDependencies() *dependencies {
	return &dependencies{
		descs: map[protoreflect.FileDescriptor]struct{}{},
	}
}

func (d *dependencies) add(fd protoreflect.FileDescriptor) {
	if _, ok := d.descs[fd]; ok {
		// already added
		return
	}
	d.descs[fd] = struct{}{}
	register.RegisterTypesInImportedFile(fd, &d.res, false)
}

// dependencyResolver is the work-horse for converting a tree of builders into a
// tree of descriptors. It scans a root (usually a file builder) and recursively
// resolves all dependencies (references to builders in other trees as well as
// references to other already-built descriptors). The result of resolution is a
// file descriptor (or an error).
type dependencyResolver struct {
	registry      protoresolve.Registry
	resolvedRoots map[Builder]protoreflect.FileDescriptor
	seen          map[Builder]struct{}
	opts          BuilderOptions
}

func newResolver(opts BuilderOptions) *dependencyResolver {
	return &dependencyResolver{
		resolvedRoots: map[Builder]protoreflect.FileDescriptor{},
		seen:          map[Builder]struct{}{},
		opts:          opts,
	}
}

func (r *dependencyResolver) resolveElement(b Builder, seen []Builder) (protoreflect.FileDescriptor, error) {
	b = getRoot(b)

	if fd, ok := r.resolvedRoots[b]; ok {
		return fd, nil
	}

	for _, s := range seen {
		if s == b {
			names := make([]string, len(seen)+1)
			for i, s := range seen {
				names[i] = string(s.Name())
			}
			names[len(seen)] = string(b.Name())
			return nil, fmt.Errorf("descriptors have cyclic dependency: %s", strings.Join(names, " ->  "))
		}
	}
	seen = append(seen, b)

	var fd protoreflect.FileDescriptor
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

func (r *dependencyResolver) resolveFile(fb *FileBuilder, root Builder, seen []Builder) (protoreflect.FileDescriptor, error) {
	deps := newDependencies()
	// add explicit imports first
	for fd := range fb.explicitImports {
		deps.add(fd)
	}
	for dep := range fb.explicitDeps {
		if dep == fb {
			// ignore erroneous self references
			continue
		}
		fd, err := r.resolveElement(dep, seen)
		if err != nil {
			return nil, err
		}
		deps.add(fd)
	}
	// now accumulate implicit dependencies based on other types referenced
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

	// finally, resolve custom options (which may refer to deps already
	// computed above)
	if err := r.resolveTypesInFileOptions(root, deps, fb); err != nil {
		return nil, err
	}

	depSlice := make([]protoreflect.FileDescriptor, 0, len(deps.descs))
	depMap := make(filesByPath, len(deps.descs))
	for dep := range deps.descs {
		isDuplicate, err := isDuplicateDependency(dep, depMap)
		if err != nil {
			return nil, err
		}
		if !isDuplicate {
			depMap[dep.Path()] = dep
			depSlice = append(depSlice, dep)
		}
	}

	fp, err := fb.buildProto(depSlice)
	if err != nil {
		return nil, err
	}

	// make sure this file path doesn't collide with any of its dependencies
	fileNames := map[string]struct{}{}
	for _, d := range depSlice {
		addFileNames(d, fileNames)
	}
	unique := makeUnique(fp.GetName(), fileNames)
	if unique != fp.GetName() {
		fp.Name = proto.String(unique)
	}

	for _, dep := range depSlice {
		isDuplicate, err := isDuplicateDependency(dep, &r.registry)
		if err != nil {
			return nil, err
		}
		if isDuplicate {
			continue
		}
		if err := r.registry.RegisterFile(dep); err != nil {
			return nil, err
		}
	}
	return r.registry.RegisterFileProto(fp)
}

type filesByPath map[string]protoreflect.FileDescriptor

func (d filesByPath) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
	file, ok := d[path]
	if ok {
		return file, nil
	}
	return nil, protoregistry.NotFound
}

// isDuplicateDependency checks for duplicate descriptors
func isDuplicateDependency(dep protoreflect.FileDescriptor, files protoresolve.FileResolver) (bool, error) {
	existing, err := files.FindFileByPath(dep.Path())
	if err != nil {
		return false, nil
	}
	var prevFDP, depFDP *descriptorpb.FileDescriptorProto
	if oracle, ok := files.(protoresolve.ProtoFileOracle); ok {
		prevFDP, _ = oracle.ProtoFromFileDescriptor(existing)
		depFDP, _ = oracle.ProtoFromFileDescriptor(dep)
	}
	if prevFDP == nil {
		prevFDP = protodesc.ToFileDescriptorProto(existing)
	}
	if depFDP == nil {
		depFDP = protodesc.ToFileDescriptorProto(dep)
	}

	// temporarily reset source code info: builders do not have them
	defer setSourceCodeInfo(prevFDP, nil)()
	defer setSourceCodeInfo(depFDP, nil)()

	if !proto.Equal(prevFDP, depFDP) {
		return true, fmt.Errorf("multiple versions of descriptors found with same file path: %s", dep.Path())
	}
	return true, nil
}

func setSourceCodeInfo(fdp *descriptorpb.FileDescriptorProto, info *descriptorpb.SourceCodeInfo) (reset func()) {
	prevSourceCodeInfo := fdp.SourceCodeInfo
	fdp.SourceCodeInfo = info
	return func() { fdp.SourceCodeInfo = prevSourceCodeInfo }
}

func addFileNames(fd protoreflect.FileDescriptor, files map[string]struct{}) {
	if _, ok := files[fd.Path()]; ok {
		// already added
		return
	}
	files[fd.Path()] = struct{}{}
	imps := fd.Imports()
	for i, length := 0, imps.Len(); i < length; i++ {
		imp := imps.Get(i).FileDescriptor
		addFileNames(imp, files)
	}
}

func (r *dependencyResolver) resolveSyntheticFile(b Builder, seen []Builder) (protoreflect.FileDescriptor, error) {
	// find ancestor to temporarily attach to new file
	curr := b
	for curr.Parent() != nil {
		curr = curr.Parent()
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
	case *OneofBuilder:
		if _, ok := b.(*OneofBuilder); ok {
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

func (r *dependencyResolver) resolveTypesInMessage(root Builder, seen []Builder, deps *dependencies, mb *MessageBuilder) error {
	for _, b := range mb.fieldsAndOneofs {
		if flb, ok := b.(*FieldBuilder); ok {
			if err := r.resolveTypesInField(root, seen, deps, flb); err != nil {
				return err
			}
		} else {
			oob := b.(*OneofBuilder)
			for _, flb := range oob.choices {
				if err := r.resolveTypesInField(root, seen, deps, flb); err != nil {
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

func (r *dependencyResolver) resolveTypesInExtension(root Builder, seen []Builder, deps *dependencies, exb *FieldBuilder) error {
	if err := r.resolveTypesInField(root, seen, deps, exb); err != nil {
		return err
	}
	if exb.foreignExtendee != nil {
		deps.add(exb.foreignExtendee.ParentFile())
	} else if err := r.resolveType(root, seen, exb.localExtendee, deps); err != nil {
		return err
	}
	return nil
}

func (r *dependencyResolver) resolveTypesInService(root Builder, seen []Builder, deps *dependencies, sb *ServiceBuilder) error {
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

func (r *dependencyResolver) resolveRpcType(root Builder, seen []Builder, t *RpcType, deps *dependencies) error {
	if t.foreignType != nil {
		deps.add(t.foreignType.ParentFile())
	} else {
		return r.resolveType(root, seen, t.localType, deps)
	}
	return nil
}

func (r *dependencyResolver) resolveTypesInField(root Builder, seen []Builder, deps *dependencies, flb *FieldBuilder) error {
	switch {
	case flb.fieldType.foreignMsgType != nil:
		deps.add(flb.fieldType.foreignMsgType.ParentFile())
	case flb.fieldType.foreignEnumType != nil:
		deps.add(flb.fieldType.foreignEnumType.ParentFile())
	case flb.fieldType.localMsgType != nil:
		if flb.fieldType.localMsgType == flb.msgType {
			return r.resolveTypesInMessage(root, seen, deps, flb.msgType)
		}
		return r.resolveType(root, seen, flb.fieldType.localMsgType, deps)
	case flb.fieldType.localEnumType != nil:
		return r.resolveType(root, seen, flb.fieldType.localEnumType, deps)
	}
	return nil
}

func (r *dependencyResolver) resolveType(root Builder, seen []Builder, typeBuilder Builder, deps *dependencies) error {
	otherRoot := getRoot(typeBuilder)
	if root == otherRoot {
		// local reference, so it will get resolved when we finish resolving this root
		return nil
	}
	fd, err := r.resolveElement(otherRoot, seen)
	if err != nil {
		return err
	}
	deps.add(fd)
	return nil
}

func (r *dependencyResolver) resolveTypesInFileOptions(root Builder, deps *dependencies, fb *FileBuilder) error {
	for _, mb := range fb.messages {
		if err := r.resolveTypesInMessageOptions(root, &fb.origExts, deps, mb); err != nil {
			return err
		}
	}
	for _, eb := range fb.enums {
		if err := r.resolveTypesInEnumOptions(root, &fb.origExts, deps, eb); err != nil {
			return err
		}
	}
	for _, exb := range fb.extensions {
		if err := r.resolveTypesInOptions(root, &fb.origExts, deps, exb.Options); err != nil {
			return err
		}
	}
	for _, sb := range fb.services {
		for _, mtb := range sb.methods {
			if err := r.resolveTypesInOptions(root, &fb.origExts, deps, mtb.Options); err != nil {
				return err
			}
		}
		if err := r.resolveTypesInOptions(root, &fb.origExts, deps, sb.Options); err != nil {
			return err
		}
	}
	return r.resolveTypesInOptions(root, &fb.origExts, deps, fb.Options)
}

func (r *dependencyResolver) resolveTypesInMessageOptions(root Builder, fileExts protoresolve.ExtensionTypeResolver, deps *dependencies, mb *MessageBuilder) error {
	for _, b := range mb.fieldsAndOneofs {
		if flb, ok := b.(*FieldBuilder); ok {
			if err := r.resolveTypesInOptions(root, fileExts, deps, flb.Options); err != nil {
				return err
			}
		} else {
			oob := b.(*OneofBuilder)
			for _, flb := range oob.choices {
				if err := r.resolveTypesInOptions(root, fileExts, deps, flb.Options); err != nil {
					return err
				}
			}
			if err := r.resolveTypesInOptions(root, fileExts, deps, oob.Options); err != nil {
				return err
			}
		}
	}
	for _, extr := range mb.ExtensionRanges {
		if err := r.resolveTypesInOptions(root, fileExts, deps, extr.Options); err != nil {
			return err
		}
	}
	for _, eb := range mb.nestedEnums {
		if err := r.resolveTypesInEnumOptions(root, fileExts, deps, eb); err != nil {
			return err
		}
	}
	for _, nmb := range mb.nestedMessages {
		if err := r.resolveTypesInMessageOptions(root, fileExts, deps, nmb); err != nil {
			return err
		}
	}
	for _, exb := range mb.nestedExtensions {
		if err := r.resolveTypesInOptions(root, fileExts, deps, exb.Options); err != nil {
			return err
		}
	}
	if err := r.resolveTypesInOptions(root, fileExts, deps, mb.Options); err != nil {
		return err
	}
	return nil
}

func (r *dependencyResolver) resolveTypesInEnumOptions(root Builder, fileExts protoresolve.ExtensionTypeResolver, deps *dependencies, eb *EnumBuilder) error {
	for _, evb := range eb.values {
		if err := r.resolveTypesInOptions(root, fileExts, deps, evb.Options); err != nil {
			return err
		}
	}
	if err := r.resolveTypesInOptions(root, fileExts, deps, eb.Options); err != nil {
		return err
	}
	return nil
}

func (r *dependencyResolver) resolveTypesInOptions(root Builder, fileExts protoresolve.ExtensionTypeResolver, deps *dependencies, opts proto.Message) error {
	// nothing to see if opts is nil
	if opts == nil {
		return nil
	}
	if rv := reflect.ValueOf(opts); rv.Kind() == reflect.Ptr && rv.IsNil() {
		return nil
	}

	ref := opts.ProtoReflect()
	tags := map[protoreflect.FieldNumber]protoreflect.ExtensionType{}
	proto.RangeExtensions(opts, func(xt protoreflect.ExtensionType, _ interface{}) bool {
		num := xt.TypeDescriptor().Number()
		tags[num] = xt
		return true
	})

	unk := ref.GetUnknown()
	for len(unk) > 0 {
		v, n := protowire.ConsumeVarint(unk)
		if n < 0 {
			break
		}
		unk = unk[n:]

		num, t := protowire.DecodeTag(v)
		if _, ok := tags[num]; !ok {
			tags[num] = nil
		}

		switch t {
		case protowire.VarintType:
			_, n = protowire.ConsumeVarint(unk)
		case protowire.Fixed64Type:
			_, n = protowire.ConsumeFixed64(unk)
		case protowire.BytesType:
			_, n = protowire.ConsumeBytes(unk)
		case protowire.StartGroupType:
			_, n = protowire.ConsumeGroup(num, unk)
		case protowire.EndGroupType:
			// invalid encoding
		case protowire.Fixed32Type:
			_, n = protowire.ConsumeFixed32(unk)
		}
		if n < 0 {
			break
		}
		unk = unk[n:]
	}

	msgName := proto.MessageName(opts)
	for tag, xt := range tags {
		// see if known dependencies have this option
		if _, err := deps.res.FindExtensionByNumber(msgName, tag); err == nil {
			// yep! nothing else to do
			continue
		}
		// see if this extension is defined in *this* builder
		if findExtension(root, msgName, tag) {
			// yep!
			continue
		}
		// see if configured resolver knows about it
		if r.opts.Resolver != nil {
			if extd, err := r.opts.Resolver.FindExtensionByNumber(msgName, tag); err == nil {
				// extension registry recognized it!
				deps.add(extd.TypeDescriptor().ParentFile())
				continue
			}
		}
		// see if given file extensions knows about it
		if fileExts != nil {
			if extd, err := fileExts.FindExtensionByNumber(msgName, tag); err == nil {
				// file extensions recognized it!
				deps.add(extd.TypeDescriptor().ParentFile())
				continue
			}
		}

		if xt != nil {
			// known extension? add its file to builder's deps
			fd := xt.TypeDescriptor().ParentFile()
			deps.add(fd)
			continue
		}

		if r.opts.RequireInterpretedOptions {
			// we require options to be interpreted but are not able to!
			return fmt.Errorf("could not interpret custom option for %s, tag %d", msgName, tag)
		}
	}
	return nil
}

func findExtension(b Builder, messageName protoreflect.FullName, extTag protoreflect.FieldNumber) bool {
	if fb, ok := b.(*FileBuilder); ok && findExtensionInFile(fb, messageName, extTag) {
		return true
	}
	if mb, ok := b.(*MessageBuilder); ok && findExtensionInMessage(mb, messageName, extTag) {
		return true
	}
	return false
}

func findExtensionInFile(fb *FileBuilder, messageName protoreflect.FullName, extTag protoreflect.FieldNumber) bool {
	for _, extb := range fb.extensions {
		if extb.ExtendeeTypeName() == messageName && extb.number == extTag {
			return true
		}
	}
	for _, mb := range fb.messages {
		if findExtensionInMessage(mb, messageName, extTag) {
			return true
		}
	}
	return false
}

func findExtensionInMessage(mb *MessageBuilder, messageName protoreflect.FullName, extTag protoreflect.FieldNumber) bool {
	for _, extb := range mb.nestedExtensions {
		if extb.ExtendeeTypeName() == messageName && extb.number == extTag {
			return true
		}
	}
	for _, mb := range mb.nestedMessages {
		if findExtensionInMessage(mb, messageName, extTag) {
			return true
		}
	}
	return false
}
