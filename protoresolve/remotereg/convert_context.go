package remotereg

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/sourcecontextpb"
	"google.golang.org/protobuf/types/known/typepb"

	"github.com/jhump/protoreflect/v2/protoresolve"
)

// defaultMaxReachableTypes is the default for Registry.MaxReachableTypes, which
// bounds the number of types that a single resolution operation may add to a
// convertContext.
//
// Besides preventing a malicious server from causing the client to fetch
// forever, this limit is also what bounds the depth of recursion when
// assembling the fetched types into files and descriptors, since there can be
// no more files than there are types.
//
// The largest schemas measured when choosing this default (Google Ads' API and
// Perfetto's trace format, both of which have a type that refers to nearly
// everything else) have about 1100-1200 types reachable from a single type.
const defaultMaxReachableTypes = 1500

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
		if err := checkFetchedName(url, et.Name); err != nil {
			return err
		}
		return cc.recordEnum(url, et)
	}

	mt, err := cc.fetcher.FetchMessageType(ctx, url)
	if errors.Is(err, protoresolve.ErrNotFound) {
		return cc.findWithFallback(url, enum)
	}
	if err != nil {
		return err
	}
	if err := checkFetchedName(url, mt.Name); err != nil {
		return err
	}
	return cc.recordTypeAndDependencies(ctx, url, mt)
}

// checkFetchedName verifies that the name of a type definition that was fetched
// for the given URL matches the type name in the URL. (Whether the name is
// valid is checked when the type is recorded.)
func checkFetchedName(url, name string) error {
	if want := protoresolve.TypeNameFromURL(url); protoreflect.FullName(name) != want {
		return fmt.Errorf("type fetched for URL %s has wrong name: wanted %q, got %q", url, want, name)
	}
	return nil
}

func (cc *convertContext) recordEnum(url string, e *typepb.Enum) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if _, ok := cc.typeLocations[url]; ok {
		// already resolved this one
		return nil
	}
	fe, err := cc.newTypeEntryLocked(url, e.Name, e.SourceContext, e)
	if err != nil {
		return err
	}
	if e.Syntax == typepb.Syntax_SYNTAX_PROTO3 {
		fe.proto3 = true
	}
	return nil
}

func (cc *convertContext) recordTypeAndDependencies(ctx context.Context, url string, mt *typepb.Type) error {
	fe, fileName, err := cc.recordType(url, mt)
	if err != nil {
		return err
	}
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
					return cc.recordDescriptor(typeURL, fileName, d)
				}

				// not in registry, so we have to recursively fetch
				err := cc.addType(grpCtx, typeURL, kind == typepb.Field_TYPE_ENUM)
				if errors.Is(err, protoresolve.ErrNotFound) {
					// The type we're resolving exists, but one that it refers
					// to does not. So we deliberately do not wrap the error with
					// %w: callers of the registry use ErrNotFound to mean that
					// the requested type is unknown, not that it is invalid.
					// (Since the returned error no longer wraps ErrNotFound,
					// callers further up the stack return it as is, instead of
					// adding more context for each level of recursion.)
					return fmt.Errorf("could not resolve type %s for field %s.%s: %v", typeURL, mt.Name, f.Name, err)
				}
				return err
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

// recordType records the given message type, returning the entry and name of
// the file to which it was added. If a type was already recorded for the given
// URL, it returns a nil entry.
func (cc *convertContext) recordType(url string, t *typepb.Type) (*fileEntry, string, error) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if _, ok := cc.typeLocations[url]; ok {
		return nil, "", nil
	}
	fe, err := cc.newTypeEntryLocked(url, t.Name, t.SourceContext, t)
	if err != nil {
		return nil, "", err
	}
	if t.Syntax == typepb.Syntax_SYNTAX_PROTO3 {
		fe.proto3 = true
	}
	return fe, cc.typeLocations[url], nil
}

// newTypeEntryLocked records the given message or enum type, which has the
// given name and source context, for the given URL. It returns the entry for
// the file to which it was added. Types with no source context are each put
// into their own synthesized file. The caller must hold cc.mu.
func (cc *convertContext) newTypeEntryLocked(
	url string,
	name string,
	sourceContext *sourcecontextpb.SourceContext,
	typ proto.Message,
) (*fileEntry, error) {
	if !protoreflect.FullName(name).IsValid() {
		return nil, fmt.Errorf("type for URL %s has invalid name %q", url, name)
	}
	if err := cc.checkLimitLocked(); err != nil {
		return nil, err
	}

	var fileName string
	synthetic := sourceContext.GetFileName() == ""
	if synthetic {
		fileName = fmt.Sprintf("--unknown--%d.proto", cc.reg.fileCounter.Add(1))
	} else {
		fileName = sourceContext.GetFileName()
	}
	cc.typeLocations[url] = fileName

	fe := cc.files[fileName]
	if fe == nil {
		fe = &fileEntry{synthetic: synthetic}
		cc.files[fileName] = fe
	}
	fe.types.addType(name, typ)
	return fe, nil
}

// checkLimitLocked returns an error if the context has already reached the
// limit on the number of types it may contain (see Registry.MaxReachableTypes).
// The caller must hold cc.mu.
func (cc *convertContext) checkLimitLocked() error {
	if limit := cc.reg.maxReachableTypes(); len(cc.typeLocations) >= limit {
		return fmt.Errorf("too many types reachable from requested type(s): limit is %d", limit)
	}
	return nil
}

func (cc *convertContext) recordDescriptor(url, ref string, d protoreflect.Descriptor) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	dc := (*DescriptorConverter)(cc.reg)
	var limitErr error
	location := d.ParentFile().Path()
	dc.addDescriptors(ref, cc.files, d, nil, func(dsc protoreflect.Descriptor) (string, bool) {
		u := ensureScheme(cc.reg.urlForType(dsc.FullName(), dsc.ParentFile().Package()))
		if fileName, ok := cc.typeLocations[u]; ok {
			// already seen this one
			if dsc == d {
				location = fileName
			}
			return fileName, false
		}
		if limitErr != nil {
			return dsc.ParentFile().Path(), false
		}
		if limitErr = cc.checkLimitLocked(); limitErr != nil {
			// stop crawling
			return dsc.ParentFile().Path(), false
		}
		cc.typeLocations[u] = dsc.ParentFile().Path()
		return dsc.ParentFile().Path(), true
	})
	if limitErr != nil {
		return limitErr
	}
	// Make sure we're also recording the actual URL reference used. This may
	// differ from the URL computed above, and must be recorded even if d was
	// already seen via another URL.
	if _, ok := cc.typeLocations[url]; !ok {
		cc.typeLocations[url] = location
	}
	return nil
}

func (cc *convertContext) findWithFallback(url string, enum bool) (err error) {
	var d protoreflect.Descriptor
	defer func() {
		// on success, record location
		if d != nil && err == nil {
			cc.mu.Lock()
			defer cc.mu.Unlock()
			if _, ok := cc.typeLocations[url]; ok {
				return
			}
			if err = cc.checkLimitLocked(); err != nil {
				return
			}
			cc.typeLocations[url] = d.ParentFile().Path()
			cc.fallbackTypes[url] = d
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
	renamed, err := mergeSyntheticFiles(cc.files)
	if err != nil {
		return nil, err
	}
	for url, fileName := range cc.typeLocations {
		if newName, ok := renamed[fileName]; ok {
			cc.typeLocations[url] = newName
		}
	}
	return toFileDescriptors(cc.files, fallback, func(tt *typeTrie, name string) (proto.Message, error) {
		return tt.typeToDescriptor(name, cc.res)
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
	// Visit files in a deterministic order so that results, including errors,
	// are reproducible.
	for _, name := range slices.Sorted(maps.Keys(fdps)) {
		if err := addToRegistry(fdps[name], &reg, fdps, fallback, nil); err != nil {
			return nil, err
		}
	}
	return &reg, nil
}

// addToRegistry registers the given file in reg, after first recursively
// registering its dependencies. The given depPath is the chain of files whose
// imports led to this one. It is used to detect import cycles, since the files
// may have been synthesized from type definitions provided by an untrusted
// server. (The depth of the recursion is bounded by the number of files, which
// is in turn bounded by Registry.MaxReachableTypes.)
func addToRegistry(
	fdp *descriptorpb.FileDescriptorProto,
	reg *protoresolve.Registry,
	fdps map[string]*descriptorpb.FileDescriptorProto,
	fallback protoresolve.FileResolver,
	depPath []string,
) error {
	if _, err := reg.FindFileByPath(fdp.GetName()); err == nil {
		return nil // already registered
	}
	if idx := slices.Index(depPath, fdp.GetName()); idx != -1 {
		return fmt.Errorf("file %s has cyclic import path: %s -> %s", fdp.GetName(), strings.Join(depPath[idx:], " -> "), fdp.GetName())
	}
	depPath = append(depPath, fdp.GetName()) // record ourselves in the path before we recurse
	for _, dep := range fdp.Dependency {
		depFd := fdps[dep]
		if depFd == nil {
			if fallback == nil {
				return fmt.Errorf("file %s has missing dependency %s", fdp.GetName(), dep)
			}
			file, err := fallback.FindFileByPath(dep)
			if err != nil {
				// Not wrapped with %w since, if this is ErrNotFound, it would
				// be mistaken for the requested type being unknown.
				return fmt.Errorf("file %s has missing dependency %s: %v", fdp.GetName(), dep, err)
			}
			if err := reg.RegisterFile(file); err != nil {
				return err
			}
			continue
		}
		if err := addToRegistry(depFd, reg, fdps, fallback, depPath); err != nil {
			return err
		}
	}
	_, err := reg.RegisterFileProto(fdp)
	return err
}
