package protoresolve

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

const defaultBaseURL = "type.googleapis.com"

// RemoteRegistry is a registry of types that are registered by remote URL.
// A RemoteRegistry can be configured with a TypeFetcher, which can be used
// to dynamically retrieve message definitions.
//
// It differs from a Registry in that it only exposes a subset of the Resolver
// interface, focused on messages and enums, which are types which may be
// resolved by downloading schemas from a remote source.
//
// This registry is intended to help resolve message type URLs in
// google.protobuf.Any messages.
type RemoteRegistry struct {
	// The default base URL to apply when registering types without a URL and
	// when looking up types by name. The message name is prefixed with the
	// base URL to form a full type URL.
	//
	// If not specified or empty, a default of "type.googleapis.com" will be
	// used.
	//
	// If present, the PackageBaseURLMapper will be consulted first before
	// applying this default.
	DefaultBaseURL string
	// A function that provides the base URL for a given package. When types
	// are registered without a URL or looked up by name, this base URL is
	// used to construct a full type URL.
	//
	// If not specified or nil, or if it returns the empty string, the
	// DefaultBaseURL will be applied.
	PackageBaseURLMapper func(packageName protoreflect.FullName) string
	// A value that can retrieve type definitions at runtime. If non-nil,
	// this will be used to resolve types for URLs that have not been
	// explicitly registered.
	TypeFetcher TypeFetcher
	// The final fallback for resolving types. If a type URL has not been
	// explicitly registered and cannot be resolved by TypeFetcher (or
	// TypeFetcher is unset/nil), then Fallback will be used to resolve
	// the type by name.
	//
	// Fallback is also used to resolve custom options found in type
	// definitions returned from TypeFetcher.
	//
	// If not specified or nil, protoregistry.GlobalFiles will be used as
	// the fallback. To prevent any fallback from being used, set this to
	// an empty resolver, such as a new, empty Registry or
	// protoregistry.Files.
	Fallback DescriptorResolver

	mu          sync.RWMutex
	typeCache   map[string]protoreflect.Descriptor
	typeURLs    map[protoreflect.FullName]string
	pkgBaseURLs map[protoreflect.FullName]pkgBaseURL
	// Used to synthesize file names when source context information is insufficient
	// when converting google.protobuf.Type, google.protobuf.Enum, and google.protobuf.Api
	// to descriptors.
	fileCounter atomic.Int32
}

type pkgBaseURL struct {
	baseURL            string
	applyToSubPackages bool
}

var _ MessageResolver = (*RemoteRegistry)(nil)

func (r *RemoteRegistry) URLForType(desc protoreflect.Descriptor) string {
	return r.urlForType(desc.FullName(), desc.ParentFile().Package())
}

func (r *RemoteRegistry) urlForType(typeName, pkgName protoreflect.FullName) string {
	// Check known types and explicit package registrations first.
	if url := r.urlFromRegistrations(typeName, pkgName); url != "" {
		return url
	}
	// Then consult package mapper and default base URL.
	return r.baseURLWithoutRegistrations(pkgName) + "/" + string(typeName)
}

func (r *RemoteRegistry) urlFromRegistrations(typeName, pkgName protoreflect.FullName) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// See if we know this type.
	if url, ok := r.typeURLs[typeName]; ok {
		return url
	}
	// Next, look at explicit package registrations.
	if baseURL := r.baseURLFromRegistrationsLocked(pkgName); baseURL != "" {
		return baseURL + "/" + string(typeName)
	}
	return ""
}

func (r *RemoteRegistry) baseURLFromRegistrationsLocked(pkgName protoreflect.FullName) string {
	var ancestor bool
	for pkgName != "" {
		if urlEntry, ok := r.pkgBaseURLs[pkgName]; ok && (!ancestor || urlEntry.applyToSubPackages) {
			return urlEntry.baseURL
		}
		pkgName = pkgName.Parent()
	}
	return ""
}

func (r *RemoteRegistry) baseURLWithoutRegistrations(pkgName protoreflect.FullName) string {
	if r.PackageBaseURLMapper != nil {
		for {
			if url := r.PackageBaseURLMapper(pkgName); url != "" {
				return url
			}
			if pkgName == "" {
				break
			}
			pkgName = pkgName.Parent()
		}
	}
	if r.DefaultBaseURL != "" {
		return r.DefaultBaseURL
	}
	return defaultBaseURL
}

func (r *RemoteRegistry) RegisterPackageBaseURL(pkgName protoreflect.FullName, baseURL string, includeSubPackages bool) (string, bool) {
	baseURL = ensureScheme(baseURL)
	r.mu.Lock()
	defer r.mu.Unlock()
	previousEntry, previouslyRegistered := r.pkgBaseURLs[pkgName]
	if !previouslyRegistered {
		previousEntry.baseURL = r.baseURL(pkgName)
	}
	if r.pkgBaseURLs == nil {
		r.pkgBaseURLs = map[protoreflect.FullName]pkgBaseURL{}
	}
	r.pkgBaseURLs[pkgName] = pkgBaseURL{
		baseURL:            baseURL,
		applyToSubPackages: includeSubPackages,
	}
	return previousEntry.baseURL, previouslyRegistered
}

func (r *RemoteRegistry) baseURL(pkgName protoreflect.FullName) string {
	if baseURL := r.baseURLFromRegistrations(pkgName); baseURL != "" {
		return baseURL
	}
	return r.baseURLWithoutRegistrations(pkgName)
}

func (r *RemoteRegistry) baseURLFromRegistrations(pkgName protoreflect.FullName) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.baseURLFromRegistrationsLocked(pkgName)
}

func (r *RemoteRegistry) RegisterMessage(md protoreflect.MessageDescriptor) error {
	return r.RegisterMessageWithURL(md, r.URLForType(md))
}

func (r *RemoteRegistry) RegisterEnum(ed protoreflect.EnumDescriptor) error {
	return r.RegisterEnumWithURL(ed, r.URLForType(ed))
}

func (r *RemoteRegistry) RegisterMessageWithURL(md protoreflect.MessageDescriptor, url string) error {
	url = ensureScheme(url)
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.checkTypeLocked(md, "message", url); err != nil {
		return err
	}
	if r.typeURLs == nil {
		r.typeURLs = map[protoreflect.FullName]string{}
	}
	if r.typeCache == nil {
		r.typeCache = map[string]protoreflect.Descriptor{}
	}
	r.typeURLs[md.FullName()] = url
	r.typeCache[url] = md
	return nil
}

func (r *RemoteRegistry) RegisterEnumWithURL(ed protoreflect.EnumDescriptor, url string) error {
	url = ensureScheme(url)
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.checkTypeLocked(ed, "enum", url); err != nil {
		return err
	}
	if r.typeURLs == nil {
		r.typeURLs = map[protoreflect.FullName]string{}
	}
	if r.typeCache == nil {
		r.typeCache = map[string]protoreflect.Descriptor{}
	}
	r.typeURLs[ed.FullName()] = url
	r.typeCache[url] = ed
	return nil
}

func (r *RemoteRegistry) checkTypeLocked(desc protoreflect.Descriptor, descKind string, url string) error {
	if _, alreadyRegistered := r.typeURLs[desc.FullName()]; alreadyRegistered {
		return fmt.Errorf("%s type %s already registered", descKind, desc.FullName())
	}
	if _, alreadyRegistered := r.typeCache[url]; alreadyRegistered {
		return fmt.Errorf("type for %s already registered", url)
	}
	return nil
}

func (r *RemoteRegistry) RegisterTypesInFile(fd protoreflect.FileDescriptor) error {
	return r.RegisterTypesInFileWithBaseURL(fd, r.baseURL(fd.Package()))
}

func (r *RemoteRegistry) RegisterTypesInFileWithBaseURL(fd protoreflect.FileDescriptor, baseURL string) error {
	baseURL = ensureScheme(baseURL)
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.checkTypesInContainerLocked(fd, baseURL); err != nil {
		return err
	}
	r.registerTypesInContainerLocked(fd, baseURL)
	return nil
}

func (r *RemoteRegistry) checkTypesInContainerLocked(container TypeContainer, baseURL string) error {
	msgs := container.Messages()
	for i, length := 0, msgs.Len(); i < length; i++ {
		md := msgs.Get(i)
		url := baseURL + "/" + string(md.FullName())
		if err := r.checkTypeLocked(md, "message", url); err != nil {
			return err
		}
		if err := r.checkTypesInContainerLocked(md, baseURL); err != nil {
			return err
		}
	}
	enums := container.Enums()
	for i, length := 0, enums.Len(); i < length; i++ {
		ed := enums.Get(i)
		url := baseURL + "/" + string(ed.FullName())
		if err := r.checkTypeLocked(ed, "enum", url); err != nil {
			return err
		}
	}
	return nil
}

func (r *RemoteRegistry) registerTypesInContainerLocked(container TypeContainer, baseURL string) {
	if r.typeURLs == nil {
		r.typeURLs = map[protoreflect.FullName]string{}
	}
	if r.typeCache == nil {
		r.typeCache = map[string]protoreflect.Descriptor{}
	}
	msgs := container.Messages()
	for i, length := 0, msgs.Len(); i < length; i++ {
		md := msgs.Get(i)
		url := baseURL + "/" + string(md.FullName())
		r.typeURLs[md.FullName()] = url
		r.typeCache[url] = md
		r.registerTypesInContainerLocked(md, baseURL)
	}
	enums := container.Enums()
	for i, length := 0, enums.Len(); i < length; i++ {
		ed := enums.Get(i)
		url := baseURL + "/" + string(ed.FullName())
		r.typeURLs[ed.FullName()] = url
		r.typeCache[url] = ed
	}
}

func (r *RemoteRegistry) FindMessageByName(name protoreflect.FullName) (protoreflect.MessageDescriptor, error) {
	return r.FindMessageByNameContext(context.Background(), name)
}

func (r *RemoteRegistry) FindEnumByName(name protoreflect.FullName) (protoreflect.EnumDescriptor, error) {
	return r.FindEnumByNameContext(context.Background(), name)
}

func (r *RemoteRegistry) FindMessageByNameContext(ctx context.Context, name protoreflect.FullName) (protoreflect.MessageDescriptor, error) {
	// We don't actually know the package for the given name, so we assume it is name.Parent().
	return r.FindMessageByURLContext(ctx, r.urlForType(name, name.Parent()))
}

func (r *RemoteRegistry) FindEnumByNameContext(ctx context.Context, name protoreflect.FullName) (protoreflect.EnumDescriptor, error) {
	// We don't actually know the package for the given name, so we assume it is name.Parent().
	return r.FindEnumByURLContext(ctx, r.urlForType(name, name.Parent()))
}

func (r *RemoteRegistry) FindMessageByURL(url string) (protoreflect.MessageDescriptor, error) {
	return r.FindMessageByURLContext(context.Background(), url)
}

func (r *RemoteRegistry) FindMessageByURLContext(ctx context.Context, url string) (protoreflect.MessageDescriptor, error) {
	desc, err := r.findTypeByURL(ctx, url, false)
	if err != nil {
		return nil, err
	}
	md, ok := desc.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, NewUnexpectedTypeError(DescriptorKindMessage, desc, url)
	}
	return md, nil
}

func (r *RemoteRegistry) FindEnumByURL(url string) (protoreflect.EnumDescriptor, error) {
	return r.FindEnumByURLContext(context.Background(), url)
}

func (r *RemoteRegistry) FindEnumByURLContext(ctx context.Context, url string) (protoreflect.EnumDescriptor, error) {
	desc, err := r.findTypeByURL(ctx, url, true)
	if err != nil {
		return nil, err
	}
	ed, ok := desc.(protoreflect.EnumDescriptor)
	if !ok {
		return nil, NewUnexpectedTypeError(DescriptorKindEnum, desc, url)
	}
	return ed, nil
}

func (r *RemoteRegistry) findTypeByURL(ctx context.Context, url string, isEnum bool) (protoreflect.Descriptor, error) {
	url = ensureScheme(url)
	r.mu.RLock()
	d := r.typeCache[url]
	r.mu.RUnlock()
	if d != nil {
		return d, nil
	}
	if r.TypeFetcher != nil {
		en, err := r.fetchTypesForURL(ctx, url, isEnum)
		if err == nil || !errors.Is(err, protoregistry.NotFound) {
			return en, err
		}
	}
	fb := r.Fallback
	if fb == nil {
		fb = protoregistry.GlobalFiles
	}
	return fb.FindDescriptorByName(TypeNameFromURL(url))
}

func (r *RemoteRegistry) fetchTypesForURL(ctx context.Context, url string, isEnum bool) (protoreflect.Descriptor, error) {
	cc := newConvertContext(r, r.TypeFetcher)
	if err := cc.addType(ctx, url, isEnum); err != nil {
		return nil, err
	}
	return r.resolveURLFromConvertContext(cc, url)
}

func (r *RemoteRegistry) findMessageTypesByURL(ctx context.Context, urls []string) (map[string]protoreflect.MessageDescriptor, error) {
	ret := make(map[string]protoreflect.MessageDescriptor, len(urls))
	var unresolved []string
	err := func() error {
		r.mu.RLock()
		defer r.mu.RUnlock()
		for _, u := range urls {
			u = ensureScheme(u)
			cached := r.typeCache[u]
			if cached != nil {
				if md, ok := cached.(protoreflect.MessageDescriptor); ok {
					ret[u] = md
				} else {
					return fmt.Errorf("type for URL %v is the wrong type: wanted message, got enum", u)
				}
			} else {
				ret[u] = nil
				unresolved = append(unresolved, u)
			}
		}
		return nil
	}()
	if err != nil {
		return nil, err
	}

	if len(unresolved) == 0 {
		return ret, nil
	}

	cc := newConvertContext(r, r.TypeFetcher)
	for _, u := range unresolved {
		if err := cc.addType(ctx, u, false); err != nil {
			return nil, err
		}
	}

	files, err := cc.toFileDescriptors()
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(cc.typeLocations) > 0 {
		if r.typeURLs == nil {
			r.typeURLs = map[protoreflect.FullName]string{}
		}
		if r.typeCache == nil {
			r.typeCache = map[string]protoreflect.Descriptor{}
		}
		for typeUrl := range cc.typeLocations {
			d, err := files.FindDescriptorByName(TypeNameFromURL(typeUrl))
			if err != nil {
				// should not be possible
				return nil, err
			}
			r.typeURLs[d.FullName()] = typeUrl
			r.typeCache[typeUrl] = d
			if _, ok := ret[typeUrl]; ok {
				ret[typeUrl] = d.(protoreflect.MessageDescriptor)
			}
		}
	}
	return ret, nil
}

func (r *RemoteRegistry) resolveURLFromConvertContext(cc *convertContext, url string) (protoreflect.Descriptor, error) {
	files, err := cc.toFileDescriptors()
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var ret protoreflect.Descriptor
	if len(cc.typeLocations) > 0 {
		if r.typeURLs == nil {
			r.typeURLs = map[protoreflect.FullName]string{}
		}
		if r.typeCache == nil {
			r.typeCache = map[string]protoreflect.Descriptor{}
		}
		for typeUrl := range cc.typeLocations {
			d, err := files.FindDescriptorByName(TypeNameFromURL(typeUrl))
			if err != nil {
				// should not be possible
				return nil, err
			}
			r.typeURLs[d.FullName()] = typeUrl
			r.typeCache[typeUrl] = d
			if url == typeUrl {
				ret = d
			}
		}
	}
	if ret == nil {
		return nil, protoregistry.NotFound
	}
	return ret, nil
}

func (r *RemoteRegistry) AsTypeResolver() *RemoteTypeResolver {
	return (*RemoteTypeResolver)(r)
}

func (r *RemoteRegistry) AsDescriptorConverter() *DescriptorConverter {
	return (*DescriptorConverter)(r)
}

// RemoteTypeResolver is an implementation of TypeResolver that uses
// a RemoteRegistry to resolve symbols.
//
// It cannot resolve extensions, so calls to FindExtensionByName and
// FindExtensionByNumber always return ErrNotFound.
//
// All types returned will be dynamic types, created using the
// [google.golang.org/protobuf/types/dynamicpb] package, built on
// the descriptors resolved by the backing RemoteRegistry.
type RemoteTypeResolver RemoteRegistry

var _ TypeResolver = (*RemoteTypeResolver)(nil)

func (r *RemoteTypeResolver) FindExtensionByName(_ protoreflect.FullName) (protoreflect.ExtensionType, error) {
	return nil, ErrNotFound
}

func (r *RemoteTypeResolver) FindExtensionByNumber(_ protoreflect.FullName, _ protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	return nil, ErrNotFound
}

func (r *RemoteTypeResolver) FindMessageByName(name protoreflect.FullName) (protoreflect.MessageType, error) {
	md, err := (*RemoteRegistry)(r).FindMessageByName(name)
	if err != nil {
		return nil, err
	}
	return dynamicpb.NewMessageType(md), nil
}

func (r *RemoteTypeResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	md, err := (*RemoteRegistry)(r).FindMessageByURL(url)
	if err != nil {
		return nil, err
	}
	return dynamicpb.NewMessageType(md), nil
}

func (r *RemoteTypeResolver) FindEnumByName(name protoreflect.FullName) (protoreflect.EnumType, error) {
	ed, err := (*RemoteRegistry)(r).FindEnumByName(name)
	if err != nil {
		return nil, err
	}
	return dynamicpb.NewEnumType(ed), nil
}

func (r *RemoteTypeResolver) FindEnumByURL(url string) (protoreflect.EnumType, error) {
	ed, err := (*RemoteRegistry)(r).FindEnumByURL(url)
	if err != nil {
		return nil, err
	}
	return dynamicpb.NewEnumType(ed), nil
}

// remoteSubResolver is an implementation of SerializationResolver that
// uses a RemoteRegistry to resolve symbols. This is used when resolving
// fetched types, when the RemoteRegistry's TypeFetcher cannot fetch a
// named type. It is also used to resolve extensions when processing the
// custom options of fetched types. Unlike RemoteRegistry.AsTypeResolver,
// this resolver will *not* recursively resolve types via fetching remote
// types -- it only uses locally cached types and the registry's fallback
// resolver.
type remoteSubResolver RemoteRegistry

func (r *remoteSubResolver) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	fb := r.Fallback
	if fb == nil {
		return protoregistry.GlobalTypes.FindExtensionByName(field)
	}
	type typeRes interface {
		AsTypeResolver() TypeResolver
	}
	if tr, ok := fb.(typeRes); ok {
		return tr.AsTypeResolver().FindExtensionByName(field)
	}
	d, err := fb.FindDescriptorByName(field)
	if err != nil {
		return nil, err
	}
	fld, ok := d.(protoreflect.FieldDescriptor)
	if !ok {
		return nil, NewUnexpectedTypeError(DescriptorKindExtension, d, "")
	}
	if !fld.IsExtension() {
		return nil, NewUnexpectedTypeError(DescriptorKindExtension, fld, "")
	}
	return ExtensionType(fld), nil
}

type typeResolvable interface {
	AsTypeResolver() TypeResolver
}

func (r *remoteSubResolver) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	fb := r.Fallback
	if fb == nil {
		return protoregistry.GlobalTypes.FindExtensionByNumber(message, field)
	}
	if tr, ok := fb.(typeResolvable); ok {
		return tr.AsTypeResolver().FindExtensionByNumber(message, field)
	}
	if pool, ok := fb.(DescriptorPool); ok {
		ext := FindExtensionByNumber(pool, message, field)
		if ext != nil {
			return ExtensionType(ext), nil
		}
	}
	return nil, ErrNotFound
}

func (r *remoteSubResolver) FindMessageByName(message protoreflect.FullName) (protoreflect.MessageType, error) {
	reg := (*RemoteRegistry)(r)
	return r.FindMessageByURL(reg.urlForType(message, message.Parent()))
}

func (r *remoteSubResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	url = ensureScheme(url)
	r.mu.RLock()
	d := r.typeCache[url]
	r.mu.RUnlock()
	if d == nil {
		fb := r.Fallback
		if fb == nil {
			fb = protoregistry.GlobalFiles
		}
		if tr, ok := fb.(typeResolvable); ok {
			return tr.AsTypeResolver().FindMessageByURL(url)
		}
		var err error
		d, err = fb.FindDescriptorByName(TypeNameFromURL(url))
		if err != nil {
			return nil, err
		}
	}
	md, ok := d.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, NewUnexpectedTypeError(DescriptorKindMessage, d, url)
	}
	return dynamicpb.NewMessageType(md), nil
}

var _ SerializationResolver = (*remoteSubResolver)(nil)

func ensureScheme(url string) string {
	pos := strings.Index(url, "://")
	if pos < 0 {
		return "https://" + url
	}
	return url
}
