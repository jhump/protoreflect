package remotereg

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/jhump/protoreflect/v2/protoresolve"
)

const defaultBaseURL = "type.googleapis.com"

// Registry is a registry of types that are registered by remote URL.
// A Registry can be configured with a TypeFetcher, which can be used
// to dynamically retrieve message definitions.
//
// It differs from a Registry in that it only exposes a subset of the Resolver
// interface, focused on messages and enums, which are types which may be
// resolved by downloading schemas from a remote source.
//
// This registry is intended to help resolve message type URLs in
// google.protobuf.Any messages.
type Registry struct {
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
	Fallback protoresolve.DescriptorResolver

	mu          sync.RWMutex
	typeCache   map[string]protoreflect.Descriptor
	typeURLs    map[protoreflect.FullName]string
	descProtos  map[protoreflect.Descriptor]proto.Message
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

var _ protoresolve.MessageResolver = (*Registry)(nil)

// URLForType computes the type URL for the given descriptor. If the
// given type has been explicitly registered or has been fetched by this
// registry (via configured TypeFetcher), this will return the URL that
// was associated with the type. Otherwise, this will compute a URL for
// the type. Computing the URL will first look at explicitly registered
// package base URLs, then the registry's PackageBaseURLMapper (if
// configured), and finally the registry's DefaultBaseURL (if configured).
func (r *Registry) URLForType(desc protoreflect.Descriptor) string {
	return r.urlForType(desc.FullName(), desc.ParentFile().Package())
}

func (r *Registry) urlForType(typeName, pkgName protoreflect.FullName) string {
	// Check known types and explicit package registrations first.
	if url := r.urlFromRegistrations(typeName, pkgName); url != "" {
		return url
	}
	// Then consult package mapper and default base URL.
	return r.baseURLWithoutRegistrations(pkgName) + "/" + string(typeName)
}

func (r *Registry) urlFromRegistrations(typeName, pkgName protoreflect.FullName) string {
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

func (r *Registry) baseURLFromRegistrationsLocked(pkgName protoreflect.FullName) string {
	var ancestor bool
	for pkgName != "" {
		if urlEntry, ok := r.pkgBaseURLs[pkgName]; ok && (!ancestor || urlEntry.applyToSubPackages) {
			return urlEntry.baseURL
		}
		pkgName = pkgName.Parent()
	}
	return ""
}

func (r *Registry) baseURLWithoutRegistrations(pkgName protoreflect.FullName) string {
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

// RegisterPackageBaseURL registers the given base URL to be used with elements
// in the given package. If includeSubPackages is true, this base URL will also
// be applied to all sub-packages (unless overridden via separate call to
// RegisterPackageBaseURL for a particular sub-package).
func (r *Registry) RegisterPackageBaseURL(pkgName protoreflect.FullName, baseURL string, includeSubPackages bool) (string, bool) {
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

func (r *Registry) baseURL(pkgName protoreflect.FullName) string {
	if baseURL := r.baseURLFromRegistrations(pkgName); baseURL != "" {
		return baseURL
	}
	return r.baseURLWithoutRegistrations(pkgName)
}

func (r *Registry) baseURLFromRegistrations(pkgName protoreflect.FullName) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.baseURLFromRegistrationsLocked(pkgName)
}

// RegisterMessage registers the given message type. The URL that corresponds to the given
// type will be computed via URLForType. Also see RegisterMessageWithURL.
func (r *Registry) RegisterMessage(md protoreflect.MessageDescriptor) error {
	return r.RegisterMessageWithURL(md, r.URLForType(md))
}

// RegisterEnum registers the given enum type. The URL that corresponds to the given
// type will be computed via URLForType. Also see RegisterEnumWithURL.
func (r *Registry) RegisterEnum(ed protoreflect.EnumDescriptor) error {
	return r.RegisterEnumWithURL(ed, r.URLForType(ed))
}

// RegisterMessageWithURL registers the given message type with the given URL.
func (r *Registry) RegisterMessageWithURL(md protoreflect.MessageDescriptor, url string) error {
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

// RegisterEnumWithURL registers the given enum type with the given URL.
func (r *Registry) RegisterEnumWithURL(ed protoreflect.EnumDescriptor, url string) error {
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

func (r *Registry) checkTypeLocked(desc protoreflect.Descriptor, descKind string, url string) error {
	if _, alreadyRegistered := r.typeURLs[desc.FullName()]; alreadyRegistered {
		return fmt.Errorf("%s type %s already registered", descKind, desc.FullName())
	}
	if _, alreadyRegistered := r.typeCache[url]; alreadyRegistered {
		return fmt.Errorf("type for %s already registered", url)
	}
	return nil
}

// RegisterTypesInFile registers all message and enum types present in the given file.
// The base URL used for all types will be computed based on explicit base URL
// registrations, then the registry's PackageBaseURLMapper (if present), and finally
// the registry's DefaultBaseURL.
func (r *Registry) RegisterTypesInFile(fd protoreflect.FileDescriptor) error {
	return r.RegisterTypesInFileWithBaseURL(fd, r.baseURL(fd.Package()))
}

// RegisterTypesInFileWithBaseURL registers all message and enum types present in the
// given file. The given base URL is used to construct the URLs for all types.
func (r *Registry) RegisterTypesInFileWithBaseURL(fd protoreflect.FileDescriptor, baseURL string) error {
	baseURL = ensureScheme(baseURL)
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.checkTypesInContainerLocked(fd, baseURL); err != nil {
		return err
	}
	r.registerTypesInContainerLocked(fd, baseURL)
	return nil
}

func (r *Registry) checkTypesInContainerLocked(container protoresolve.TypeContainer, baseURL string) error {
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

func (r *Registry) registerTypesInContainerLocked(container protoresolve.TypeContainer, baseURL string) {
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

// FindMessageByName has the same signature as the method of the same name
// in the Resolver interface.
//
// But since finding a type definition may involve retrieving data via a
// TypeFetcher, it is recommended to use FindMessageByNameContext instead.
// Calling this version will implicitly use [context.Background]().
func (r *Registry) FindMessageByName(name protoreflect.FullName) (protoreflect.MessageDescriptor, error) {
	return r.FindMessageByNameContext(context.Background(), name)
}

// FindEnumByName has the same signature as the method of the same name
// in the Resolver interface.
//
// But since finding a type definition may involve retrieving data via a
// TypeFetcher, it is recommended to use FindEnumByNameContext instead.
// Calling this version will implicitly use [context.Background]().
func (r *Registry) FindEnumByName(name protoreflect.FullName) (protoreflect.EnumDescriptor, error) {
	return r.FindEnumByNameContext(context.Background(), name)
}

// FindMessageByNameContext finds a type definition for a message with
// the given name. This function computes a URL for the given type using
// the logic described in URLForType and then delegates to
// FindMessageByURLContext.
func (r *Registry) FindMessageByNameContext(ctx context.Context, name protoreflect.FullName) (protoreflect.MessageDescriptor, error) {
	// We don't actually know the package for the given name, so we assume it is name.Parent().
	return r.FindMessageByURLContext(ctx, r.urlForType(name, name.Parent()))
}

// FindEnumByNameContext finds a type definition for an enum with the
// given name. This function computes a URL for the given type using
// the logic described in URLForType and then delegates to
// FindEnumByURLContext.
func (r *Registry) FindEnumByNameContext(ctx context.Context, name protoreflect.FullName) (protoreflect.EnumDescriptor, error) {
	// We don't actually know the package for the given name, so we assume it is name.Parent().
	return r.FindEnumByURLContext(ctx, r.urlForType(name, name.Parent()))
}

// FindMessageByURL has the same signature as the method of the same name
// in the protoresolve.MessageResolver interface.
//
// But since finding a type definition may involve retrieving data via a
// TypeFetcher, it is recommended to use FindMessageByURLContext instead.
// Calling this version will implicitly use [context.Background]().
func (r *Registry) FindMessageByURL(url string) (protoreflect.MessageDescriptor, error) {
	return r.FindMessageByURLContext(context.Background(), url)
}

// FindMessageByURLContext finds a type definition for a message with
// the given type URL. This method first examines explicitly registered
// message types (via RegisterMessage and RegisterMessageWithURL) and
// types already downloaded via the TypeFetcher. It will then use the
// TypeFetcher, if present, to try to download a type definition. And
// if fails to produce a result, the registry's Fallback is queried.
func (r *Registry) FindMessageByURLContext(ctx context.Context, url string) (protoreflect.MessageDescriptor, error) {
	desc, err := r.findTypeByURL(ctx, url, false)
	if err != nil {
		return nil, err
	}
	md, ok := desc.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, protoresolve.NewUnexpectedTypeError(protoresolve.DescriptorKindMessage, desc, url)
	}
	return md, nil
}

// FindEnumByURL has a signature that is consistent with that of
// FindMessageByURL and is present for symmetry.
//
// But since finding a type definition may involve retrieving data via a
// TypeFetcher, it is recommended to use FindEnumByURLContext instead.
// Calling this version will implicitly use [context.Background]().
func (r *Registry) FindEnumByURL(url string) (protoreflect.EnumDescriptor, error) {
	return r.FindEnumByURLContext(context.Background(), url)
}

// FindEnumByURLContext finds a type definition for an enum with the
// given type URL. This method first examines explicitly registered
// enum types (via RegisterEnum and RegisterEnumWithURL) and types
// already downloaded via the TypeFetcher. It will then use the
// TypeFetcher, if present, to try to download a type definition. And
// if fails to produce a result, the registry's Fallback is queried.
func (r *Registry) FindEnumByURLContext(ctx context.Context, url string) (protoreflect.EnumDescriptor, error) {
	desc, err := r.findTypeByURL(ctx, url, true)
	if err != nil {
		return nil, err
	}
	ed, ok := desc.(protoreflect.EnumDescriptor)
	if !ok {
		return nil, protoresolve.NewUnexpectedTypeError(protoresolve.DescriptorKindEnum, desc, url)
	}
	return ed, nil
}

func (r *Registry) findTypeByURL(ctx context.Context, url string, isEnum bool) (protoreflect.Descriptor, error) {
	url = ensureScheme(url)
	r.mu.RLock()
	d := r.typeCache[url]
	r.mu.RUnlock()
	if d != nil {
		return d, nil
	}
	if r.TypeFetcher != nil {
		en, err := r.fetchTypeForURL(ctx, url, isEnum)
		if err == nil || !errors.Is(err, protoregistry.NotFound) {
			return en, err
		}
	}
	fb := r.Fallback
	if fb == nil {
		fb = protoregistry.GlobalFiles
	}
	return fb.FindDescriptorByName(protoresolve.TypeNameFromURL(url))
}

func (r *Registry) fetchTypeForURL(ctx context.Context, url string, isEnum bool) (protoreflect.Descriptor, error) {
	cc := newConvertContext(r, r.TypeFetcher)
	if err := cc.addType(ctx, url, isEnum); err != nil {
		return nil, err
	}
	return r.resolveURLFromConvertContext(cc, url)
}

func (r *Registry) findMessageTypesByURL(ctx context.Context, urls []string) (map[string]protoreflect.MessageDescriptor, error) {
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
	protoOracle := protoresolve.NewProtoOracle(files)
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
			d, err := files.FindDescriptorByName(protoresolve.TypeNameFromURL(typeUrl))
			if err != nil {
				// should not be possible
				return nil, err
			}
			r.typeURLs[d.FullName()] = typeUrl
			r.typeCache[typeUrl] = d
			if dProto, err := protoOracle.ProtoFromDescriptor(d); err == nil {
				r.descProtos[d] = dProto
			}
			if _, ok := ret[typeUrl]; ok {
				ret[typeUrl] = d.(protoreflect.MessageDescriptor)
			}
		}
	}
	return ret, nil
}

func (r *Registry) resolveURLFromConvertContext(cc *convertContext, url string) (protoreflect.Descriptor, error) {
	files, err := cc.toFileDescriptors()
	protoOracle := protoresolve.NewProtoOracle(files)
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
		if r.descProtos == nil {
			r.descProtos = map[protoreflect.Descriptor]proto.Message{}
		}
		for typeUrl := range cc.typeLocations {
			d, err := files.FindDescriptorByName(protoresolve.TypeNameFromURL(typeUrl))
			if err != nil {
				// should not be possible
				return nil, err
			}
			r.typeURLs[d.FullName()] = typeUrl
			r.typeCache[typeUrl] = d
			if dProto, err := protoOracle.ProtoFromDescriptor(d); err == nil {
				r.descProtos[d] = dProto
			}
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

// AsTypeResolver returns a view of this registry that returns types instead
// of descriptors. The returned resolver implements TypeResolver
func (r *Registry) AsTypeResolver() *RemoteTypeResolver {
	return (*RemoteTypeResolver)(r)
}

// AsDescriptorConverter returns a view of this registry as a DescriptorConverter.
// The returned value may be used to convert type and service definitions from the
// descriptor representations to the google.protobuf.Type, google.protobuf.Enum, and
// google.protobuf.Service representations, and vice versa.
func (r *Registry) AsDescriptorConverter() *DescriptorConverter {
	return (*DescriptorConverter)(r)
}

// RemoteTypeResolver is an implementation of TypeResolver that uses
// a Registry to resolve symbols.
//
// All message and enum types returned will be dynamic types, created
// using the [dynamicpb] package, built on the descriptors resolved by
// the backing Registry.
type RemoteTypeResolver Registry

var _ protoresolve.TypeResolver = (*RemoteTypeResolver)(nil)

// FindExtensionByName implements the SerializationResolver interface.
//
// This method relies on the underlying Registry's fallback resolver.
// If the registry's fallback resolver is unconfigured or nil, then
// protoregistry.GlobalTypes will be used to find the extension. Otherwise,
// if the fallback has a method named AsTypeResolver that returns a
// TypeResolver, it will be invoked and the resulting type resolver will be
// used to find the extension type. Failing all of the above, the fallback's
// FindDescriptorByName method will be used to find the extension. In this
// case the returned extension type may be a dynamic extension type, created
// using the [dynamicpb] package.
func (r *RemoteTypeResolver) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	return (*remoteSubResolver)(r).FindExtensionByName(field)
}

// FindExtensionByNumber implements the SerializationResolver interface.
//
// This method relies on the underlying Registry's fallback resolver.
// If the registry's fallback resolver is unconfigured or nil, then
// protoregistry.GlobalTypes will be used to find the extension. Otherwise,
// if the fallback has a method named AsTypeResolver that returns a
// TypeResolver, it will be invoked and the resulting type resolver will be
// used to find the extension type. Failing all of the above, the extension
// can only be found if the fallback resolver implements DescriptorPool or
// ExtensionResolver and will otherwise return ErrNotFound.
func (r *RemoteTypeResolver) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	return (*remoteSubResolver)(r).FindExtensionByNumber(message, field)
}

// FindMessageByName implements the SerializationResolver interface. Since
// finding a message may incur fetching the definition via a TypeFetcher,
// it is recommended, where possible, to instead use FindMessageByNameContext.
// Calling this version will implicitly use [context.Background]().
func (r *RemoteTypeResolver) FindMessageByName(name protoreflect.FullName) (protoreflect.MessageType, error) {
	return r.FindMessageByNameContext(context.Background(), name)
}

// FindMessageByNameContext finds a message by name, using the given context
// if necessary to fetch the message type via a TypeFetcher. This uses the
// underlying Registry's method of the same name.
func (r *RemoteTypeResolver) FindMessageByNameContext(ctx context.Context, name protoreflect.FullName) (protoreflect.MessageType, error) {
	md, err := (*Registry)(r).FindMessageByNameContext(ctx, name)
	if err != nil {
		return nil, err
	}
	return dynamicpb.NewMessageType(md), nil
}

// FindMessageByURL implements the SerializationResolver interface. Since
// finding a message may incur fetching the definition via a TypeFetcher,
// it is recommended, where possible, to instead use FindMessageByURLContext.
// Calling this version will implicitly use [context.Background]().
func (r *RemoteTypeResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	return r.FindMessageByURLContext(context.Background(), url)
}

// FindMessageByURLContext finds a message by type URL, using the given
// context if necessary to fetch the message type via a TypeFetcher. This
// uses the underlying Registry's method of the same name.
func (r *RemoteTypeResolver) FindMessageByURLContext(ctx context.Context, url string) (protoreflect.MessageType, error) {
	md, err := (*Registry)(r).FindMessageByURLContext(ctx, url)
	if err != nil {
		return nil, err
	}
	return dynamicpb.NewMessageType(md), nil
}

// FindEnumByName implements the method of the same name on the TypeResolver
// interface. Since finding an enum may incur fetching the definition via a
// TypeFetcher, it is recommended, where possible, to instead use
// FindEnumByNameContext. Calling this version will implicitly use
// [context.Background]().
func (r *RemoteTypeResolver) FindEnumByName(name protoreflect.FullName) (protoreflect.EnumType, error) {
	return r.FindEnumByNameContext(context.Background(), name)
}

// FindEnumByNameContext finds an enum by name, using the given context
// if necessary to fetch the message type via a TypeFetcher. This uses the
// underlying Registry's method of the same name.
func (r *RemoteTypeResolver) FindEnumByNameContext(ctx context.Context, name protoreflect.FullName) (protoreflect.EnumType, error) {
	ed, err := (*Registry)(r).FindEnumByNameContext(ctx, name)
	if err != nil {
		return nil, err
	}
	return dynamicpb.NewEnumType(ed), nil
}

// FindEnumByURL has a signature that is consistent with that of
// FindMessageByURL and is present for symmetry. Since finding an
// enum may incur fetching the definition via a TypeFetcher, it is
// recommended, where possible, to instead use FindEnumByURLContext.
// Calling this version will implicitly use [context.Background]().
func (r *RemoteTypeResolver) FindEnumByURL(url string) (protoreflect.EnumType, error) {
	return r.FindEnumByURLContext(context.Background(), url)
}

// FindEnumByURLContext finds an enum by type URL, using the given context
// if necessary to fetch the message type via a TypeFetcher. This uses the
// underlying Registry's method of the same name.
func (r *RemoteTypeResolver) FindEnumByURLContext(ctx context.Context, url string) (protoreflect.EnumType, error) {
	ed, err := (*Registry)(r).FindEnumByURLContext(ctx, url)
	if err != nil {
		return nil, err
	}
	return dynamicpb.NewEnumType(ed), nil
}

// remoteSubResolver is an implementation of SerializationResolver that
// uses a Registry to resolve symbols. This is used when resolving
// fetched types, when the Registry's TypeFetcher cannot fetch a
// named type. It is also used to resolve extensions when processing the
// custom options of fetched types. Unlike Registry.AsTypeResolver,
// this resolver will *not* recursively resolve types via fetching remote
// types -- it only uses locally cached types and the registry's fallback
// resolver.
type remoteSubResolver Registry

var _ protoresolve.SerializationResolver = (*remoteSubResolver)(nil)

func (r *remoteSubResolver) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	fb := r.Fallback
	if fb == nil {
		return protoregistry.GlobalTypes.FindExtensionByName(field)
	}
	type typeRes interface {
		AsTypeResolver() protoresolve.TypeResolver
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
		return nil, protoresolve.NewUnexpectedTypeError(protoresolve.DescriptorKindExtension, d, "")
	}
	if !fld.IsExtension() {
		return nil, protoresolve.NewUnexpectedTypeError(protoresolve.DescriptorKindExtension, fld, "")
	}
	return protoresolve.ExtensionType(fld), nil
}

type typeResolvable interface {
	AsTypeResolver() protoresolve.TypeResolver
}

func (r *remoteSubResolver) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	fb := r.Fallback
	if fb == nil {
		return protoregistry.GlobalTypes.FindExtensionByNumber(message, field)
	}
	if tr, ok := fb.(typeResolvable); ok {
		return tr.AsTypeResolver().FindExtensionByNumber(message, field)
	}
	if pool, ok := fb.(protoresolve.DescriptorPool); ok {
		ext := protoresolve.FindExtensionByNumber(pool, message, field)
		if ext != nil {
			return protoresolve.ExtensionType(ext), nil
		}
	}
	return nil, protoresolve.ErrNotFound
}

func (r *remoteSubResolver) FindMessageByName(message protoreflect.FullName) (protoreflect.MessageType, error) {
	reg := (*Registry)(r)
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
		d, err = fb.FindDescriptorByName(protoresolve.TypeNameFromURL(url))
		if err != nil {
			return nil, err
		}
	}
	md, ok := d.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, protoresolve.NewUnexpectedTypeError(protoresolve.DescriptorKindMessage, d, url)
	}
	return dynamicpb.NewMessageType(md), nil
}

func ensureScheme(url string) string {
	pos := strings.Index(url, "://")
	if pos < 0 {
		return "https://" + url
	}
	return url
}
