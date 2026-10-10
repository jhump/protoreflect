package remotereg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/typepb"

	"github.com/jhump/protoreflect/v2/protoresolve"
)

// convertContext provides the state for a resolution operation, accumulating details about
// type descriptions and the files that contain them.
type convertContext struct {
	reg     *Registry
	res     *remoteSubResolver
	fetcher TypeFetcher

	mu sync.Mutex
	// map of file names to details regarding the files' contents
	files map[string]*fileEntry
	// map of type URLs to the file name that defines them
	typeLocations map[string]string
	// map of type URLs to descriptors found in the fallback resolver, for
	// types that the fetcher didn't have
	fallbackTypes map[string]protoreflect.Descriptor
}

func newConvertContext(reg *Registry, fetcher TypeFetcher) *convertContext {
	return &convertContext{
		reg:           reg,
		res:           (*remoteSubResolver)(reg),
		fetcher:       fetcher,
		typeLocations: map[string]string{},
		fallbackTypes: map[string]protoreflect.Descriptor{},
		files:         map[string]*fileEntry{},
	}
}

// addType adds the type at the given URL to the context, using the given fetcher to download the type's
// description. This function will recursively add dependencies (e.g. types referenced by the given type's
// fields if it is a message type), fetching their type descriptions concurrently.
func (cc *convertContext) addType(ctx context.Context, url string, enum bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if enum {
		var err error
		et, err := cc.fetcher.FetchEnumType(ctx, url)
		if errors.Is(err, protoresolve.ErrNotFound) {
			return cc.findWithFallback(url, enum)
		}
		if err != nil {
			return err
		}
		cc.recordEnum(url, et)
		return nil
	}

	mt, err := cc.fetcher.FetchMessageType(ctx, url)
	if errors.Is(err, protoresolve.ErrNotFound) {
		return cc.findWithFallback(url, enum)
	}
	if err != nil {
		return err
	}
	return cc.recordTypeAndDependencies(ctx, url, mt)
}

func (cc *convertContext) recordEnum(url string, e *typepb.Enum) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	var fileName string
	if e.SourceContext != nil && e.SourceContext.FileName != "" {
		fileName = e.SourceContext.FileName
	} else {
		fileName = fmt.Sprintf("--unknown--%d.proto", cc.reg.fileCounter.Add(1))
	}
	cc.typeLocations[url] = fileName

	fe := cc.files[fileName]
	if fe == nil {
		fe = &fileEntry{}
		cc.files[fileName] = fe
	}
	fe.types.addType(e.Name, e)
	if e.Syntax == typepb.Syntax_SYNTAX_PROTO3 {
		fe.proto3 = true
	}
}

func (cc *convertContext) recordTypeAndDependencies(ctx context.Context, url string, mt *typepb.Type) error {
	fe, fileName := cc.recordType(url, mt)
	if fe == nil {
		// already resolved this one
		return nil
	}

	// Resolve dependencies in parallel.
	grp, grpCtx := errgroup.WithContext(ctx)
	for _, f := range mt.Fields {
		if f.Kind == typepb.Field_TYPE_GROUP || f.Kind == typepb.Field_TYPE_MESSAGE || f.Kind == typepb.Field_TYPE_ENUM {
			typeURL := ensureScheme(f.TypeUrl)
			kind := f.Kind
			grp.Go(func() error {
				// first check the registry for descriptors
				cc.reg.mu.RLock()
				d := cc.reg.typeCache[typeURL]
				cc.reg.mu.RUnlock()

				if d != nil {
					// found it!
					cc.recordDescriptor(typeURL, fileName, d)
					return nil
				}

				// not in registry, so we have to recursively fetch
				if err := cc.addType(grpCtx, typeURL, kind == typepb.Field_TYPE_ENUM); err != nil {
					return err
				}
				return nil
			})
		}
	}
	if err := grp.Wait(); err != nil {
		return err
	}
	// double-check if parent context has been cancelled
	if ctx.Err() != nil {
		return ctx.Err()
	}

	cc.mu.Lock()
	defer cc.mu.Unlock()

	for _, f := range mt.Fields {
		if f.Kind == typepb.Field_TYPE_GROUP || f.Kind == typepb.Field_TYPE_MESSAGE || f.Kind == typepb.Field_TYPE_ENUM {
			typeURL := ensureScheme(f.TypeUrl)
			if fe.deps == nil {
				fe.deps = map[string]struct{}{}
			}
			dep := cc.typeLocations[typeURL]
			if dep != fileName {
				fe.deps[dep] = struct{}{}
			}
		}
	}
	return nil
}

func (cc *convertContext) recordType(url string, t *typepb.Type) (*fileEntry, string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if _, ok := cc.typeLocations[url]; ok {
		return nil, ""
	}

	var fileName string
	if t.SourceContext != nil && t.SourceContext.FileName != "" {
		fileName = t.SourceContext.FileName
	} else {
		fileName = fmt.Sprintf("--unknown--%d.proto", cc.reg.fileCounter.Add(1))
	}
	cc.typeLocations[url] = fileName

	fe := cc.files[fileName]
	if fe == nil {
		fe = &fileEntry{}
		cc.files[fileName] = fe
	}
	fe.types.addType(t.Name, t)
	if t.Syntax == typepb.Syntax_SYNTAX_PROTO3 {
		fe.proto3 = true
	}

	return fe, fileName
}

func (cc *convertContext) recordDescriptor(url, ref string, d protoreflect.Descriptor) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	dc := (*DescriptorConverter)(cc.reg)
	dc.addDescriptors(ref, cc.files, d, nil, func(dsc protoreflect.Descriptor) bool {
		u := ensureScheme(cc.reg.urlForType(dsc.FullName(), dsc.ParentFile().Package()))
		if _, ok := cc.typeLocations[u]; ok {
			// already seen this one
			return false
		}
		fileName := dsc.ParentFile().Path()
		cc.typeLocations[u] = fileName
		if dsc == d {
			// make sure we're also adding the actual URL reference used
			cc.typeLocations[url] = fileName
		}
		return true
	})
}

func (cc *convertContext) findWithFallback(url string, enum bool) (err error) {
	var d protoreflect.Descriptor
	defer func() {
		// on success, record location
		if d != nil && err == nil {
			cc.mu.Lock()
			cc.typeLocations[url] = d.ParentFile().Path()
			cc.fallbackTypes[url] = d
			cc.mu.Unlock()
		}
	}()
	fb := cc.res.Fallback
	if fb == nil {
		fb = protoregistry.GlobalFiles
	}
	d, err = fb.FindDescriptorByName(protoresolve.TypeNameFromURL(url))
	if err != nil {
		return err
	}
	switch d.(type) {
	case protoreflect.EnumDescriptor:
		if enum {
			return nil
		}
	case protoreflect.MessageDescriptor:
		if !enum {
			return nil
		}
	}
	var wanted protoresolve.DescriptorKind
	if enum {
		wanted = protoresolve.DescriptorKindEnum
	} else {
		wanted = protoresolve.DescriptorKindMessage
	}
	return protoresolve.NewUnexpectedTypeError(wanted, d, url)
}

// toFileDescriptors converts the information in the context into a map of file names to file descriptors.
func (cc *convertContext) toFileDescriptors() (*protoresolve.Registry, error) {
	var fallback protoresolve.FileResolver
	if cc.res.Fallback != nil {
		fallback, _ = cc.res.Fallback.(protoresolve.FileResolver)
	} else {
		fallback = protoregistry.GlobalFiles
	}
	return toFileDescriptors(cc.files, fallback, func(tt *typeTrie, name string) (proto.Message, error) {
		mdp, edp := tt.typeToDescriptor(name, cc.res)
		if mdp != nil {
			return mdp, nil
		}
		return edp, nil
	})
}

// converts a map of file entries into a map of file descriptors using the given function to convert
// each trie node into a descriptor proto.
func toFileDescriptors(files map[string]*fileEntry, fallback protoresolve.FileResolver, trieFn func(*typeTrie, string) (proto.Message, error)) (*protoresolve.Registry, error) {
	fdps := map[string]*descriptorpb.FileDescriptorProto{}
	for name, file := range files {
		fdp, err := file.toFileDescriptor(name, trieFn)
		if err != nil {
			return nil, err
		}
		fdps[name] = fdp
	}
	var reg protoresolve.Registry
	for _, fdp := range fdps {
		if err := addToRegistry(fdp, &reg, fdps, fallback); err != nil {
			return nil, err
		}
	}
	return &reg, nil
}

func addToRegistry(fdp *descriptorpb.FileDescriptorProto, reg *protoresolve.Registry, fdps map[string]*descriptorpb.FileDescriptorProto, fallback protoresolve.FileResolver) error {
	if _, err := reg.FindFileByPath(fdp.GetName()); err == nil {
		return nil // already registered
	}
	for _, dep := range fdp.Dependency {
		depFd := fdps[dep]
		if depFd == nil {
			if fallback == nil {
				return fmt.Errorf("missing dependency: %s", dep)
			}
			file, err := fallback.FindFileByPath(dep)
			if err != nil {
				return fmt.Errorf("missing dependency: %s: %w", dep, err)
			}
			if err := reg.RegisterFile(file); err != nil {
				return err
			}
			continue
		}
		if err := addToRegistry(depFd, reg, fdps, fallback); err != nil {
			return err
		}
	}
	_, err := reg.RegisterFileProto(fdp)
	return err
}

// fileEntry represents the contents of a single file.
type fileEntry struct {
	types  typeTrie
	deps   map[string]struct{}
	proto3 bool
}

// toFileDescriptor converts this file entry into a file descriptor proto. The given function
// is used to transform nodes in a typeTrie into message and/or enum descriptor protos.
func (fe *fileEntry) toFileDescriptor(name string, trieFn func(*typeTrie, string) (proto.Message, error)) (*descriptorpb.FileDescriptorProto, error) {
	var pkg bytes.Buffer
	tt := &fe.types
	first := true
	last := ""
	for tt.typ == nil {
		if last != "" {
			if first {
				first = false
			} else {
				pkg.WriteByte('.')
			}
			pkg.WriteString(last)
		}
		if len(tt.children) != 1 {
			break
		}
		for last, tt = range tt.children {
		}
	}
	fd := createFileDescriptor(name, pkg.String(), fe.proto3, fe.deps)
	if tt.typ != nil {
		pm, err := trieFn(tt, last)
		if err != nil {
			return nil, err
		}
		if mdp, ok := pm.(*descriptorpb.DescriptorProto); ok {
			fd.MessageType = append(fd.MessageType, mdp)
		} else if edp, ok := pm.(*descriptorpb.EnumDescriptorProto); ok {
			fd.EnumType = append(fd.EnumType, edp)
		} else {
			sdp := pm.(*descriptorpb.ServiceDescriptorProto)
			fd.Service = append(fd.Service, sdp)
		}
	} else {
		for name, nested := range tt.children {
			pm, err := trieFn(nested, name)
			if err != nil {
				return nil, err
			}
			if mdp, ok := pm.(*descriptorpb.DescriptorProto); ok {
				fd.MessageType = append(fd.MessageType, mdp)
			} else if edp, ok := pm.(*descriptorpb.EnumDescriptorProto); ok {
				fd.EnumType = append(fd.EnumType, edp)
			} else {
				sdp := pm.(*descriptorpb.ServiceDescriptorProto)
				fd.Service = append(fd.Service, sdp)
			}
		}
	}
	return fd, nil
}
