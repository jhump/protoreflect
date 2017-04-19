package dynamic

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"

	"github.com/jhump/protoreflect/desc"
)

const googleApisDomain = "type.googleapis.com"

// MessageRegistry is a registry that maps URLs to message types. It allows for marsalling
// and unmarshalling Any types to and from dynamic messages.
type MessageRegistry struct {
	includeDefault bool
	resolver       typeResolver
	mf             *MessageFactory
	er             *ExtensionRegistry
	mu             sync.RWMutex
	messages       map[string]desc.Descriptor
	domains        map[string]string
	defaultDomain  string
}

func NewMessageRegistryWithDefaults() *MessageRegistry {
	mf := NewMessageFactoryWithDefaults()
	return &MessageRegistry{
		includeDefault: true,
		mf:             mf,
		er:             mf.er,
	}
}

// WithFetcher sets the TypeFetcher that this registry uses to resolve unknown
// URLs. This method is not thread-safe and is intended to be used for one-time
// initialization of the registry, before it published for use by other threads.
func (r *MessageRegistry) WithFetcher(fetcher TypeFetcher) *MessageRegistry {
	r.resolver = typeResolver{fetcher: fetcher, mr: r}
	return r
}

// WithMessageFactory sets the MessageFactory used to instantiate any messages.
// This method is not thread-safe and is intended to be used for one-time
// initialization of the registry, before it published for use by other threads.
func (r *MessageRegistry) WithMessageFactory(mf *MessageFactory) *MessageRegistry {
	r.mf = mf
	if mf == nil {
		r.er = nil
	} else {
		r.er = mf.er
	}
	return r
}

// WithDefaultDomain sets the default domain used when constructing type URLs for
// marshalling messages as Any types. If unspecified, the default domain is
// "type.googleapis.com". This method is not thread-safe and is intended to be used
// for one-time initialization of the registry, before it published for use by other
// threads.
func (r *MessageRegistry) WithDefaultDomain(domain string) *MessageRegistry {
	domain = canonicalizeDomain(domain)
	r.defaultDomain = domain
	return r
}

func canonicalizeDomain(domain string) string {
	domain = ensureScheme(domain)
	if domain[len(domain)-1] == '/' {
		return domain[:len(domain)-1]
	}
	return domain
}

func (r *MessageRegistry) AddMessage(url string, md *desc.MessageDescriptor) error {
	if !strings.HasSuffix(url, "/"+md.GetFullyQualifiedName()) {
		return fmt.Errorf("URL %s is invalid: it should end with path element %s", url, md.GetFullyQualifiedName())
	}
	url = ensureScheme(url)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messages[url] = md
	return nil
}

func (r *MessageRegistry) AddEnum(url string, ed *desc.EnumDescriptor) error {
	if !strings.HasSuffix(url, "/"+ed.GetFullyQualifiedName()) {
		return fmt.Errorf("URL %s is invalid: it should end with path element %s", url, ed.GetFullyQualifiedName())
	}
	url = ensureScheme(url)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messages[url] = ed
	return nil
}

func (r *MessageRegistry) AddFile(domain string, fd *desc.FileDescriptor) {
	domain = canonicalizeDomain(domain)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.addEnumTypesLocked(domain, fd.GetEnumTypes())
	r.addMessageTypesLocked(domain, fd.GetMessageTypes())
}

func (r *MessageRegistry) addEnumTypesLocked(domain string, enums []*desc.EnumDescriptor) {
	for _, ed := range enums {
		r.messages[fmt.Sprintf("%s/%s", domain, ed.GetFullyQualifiedName())] = ed
	}
}

func (r *MessageRegistry) addMessageTypesLocked(domain string, msgs []*desc.MessageDescriptor) {
	for _, md := range msgs {
		r.messages[fmt.Sprintf("%s/%s", domain, md.GetFullyQualifiedName())] = md
		r.addEnumTypesLocked(domain, md.GetNestedEnumTypes())
		r.addMessageTypesLocked(domain, md.GetNestedMessageTypes())
	}
}

func (r *MessageRegistry) AddDomainForElement(domain, packageOrTypeName string) {
	if domain[len(domain)-1] == '/' {
		domain = domain[:len(domain)-1]
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.domains[packageOrTypeName] = domain
}

func (r *MessageRegistry) FindMessageTypeByUrl(url string) (*desc.MessageDescriptor, error) {
	if r == nil {
		return nil, nil
	}
	url = ensureScheme(url)
	r.mu.RLock()
	m := r.messages[url]
	r.mu.RUnlock()
	if md, ok := m.(*desc.MessageDescriptor); ok {
		return md, nil
	}
	if r.includeDefault {
		pos := strings.LastIndex(url, "/")
		var msgName string
		if pos >= 0 {
			msgName = url[pos+1:]
		} else {
			msgName = url
		}
		if md, err := desc.LoadMessageDescriptor(msgName); err != nil {
			return nil, err
		} else if md != nil {
			return md, nil
		}
	}
	if r.resolver.fetcher == nil {
		return nil, nil
	}
	if md, err := r.resolver.resolveUrlToMessageDescriptor(url); err != nil {
		return nil, err
	} else {
		return md, nil
	}
}

func (r *MessageRegistry) FindEnumTypeByUrl(url string) (*desc.EnumDescriptor, error) {
	if r == nil {
		return nil, nil
	}
	url = ensureScheme(url)
	r.mu.RLock()
	m := r.messages[url]
	r.mu.RUnlock()
	if ed, ok := m.(*desc.EnumDescriptor); ok {
		return ed, nil
	}
	if r.resolver.fetcher == nil {
		return nil, nil
	}
	if ed, err := r.resolver.resolveUrlToEnumDescriptor(url); err != nil {
		return nil, err
	} else {
		return ed, nil
	}
}

func (r *MessageRegistry) UnmarshalAny(any *any.Any) (proto.Message, error) {
	return r.unmarshalAny(any, r.FindMessageTypeByUrl)
}

func (r *MessageRegistry) unmarshalAny(any *any.Any, fetch func(string) (*desc.MessageDescriptor, error)) (proto.Message, error) {
	name, err := ptypes.AnyMessageName(any)
	if err != nil {
		return nil, err
	}

	var msg proto.Message
	if r == nil {
		// a nil registry only knows about well-known types
		if msg = (*KnownTypeRegistry)(nil).CreateIfKnown(name); msg == nil {
			return nil, fmt.Errorf("Unknown message type: %s", any.TypeUrl)
		}
	} else {
		var ktr *KnownTypeRegistry
		if r.mf != nil {
			ktr = r.mf.ktr
		}
		if msg = ktr.CreateIfKnown(name); msg == nil {
			if md, err := fetch(any.TypeUrl); err != nil {
				return nil, err
			} else if md == nil {
				return nil, fmt.Errorf("Unknown message type: %s", any.TypeUrl)
			} else {
				msg = newMessageWithMessageFactory(md, r.mf)
			}
		}
	}

	err = proto.Unmarshal(any.Value, msg)
	if err != nil {
		return nil, err
	} else {
		return msg, nil
	}
}

func (r *MessageRegistry) MarshalAny(m proto.Message) (*any.Any, error) {
	name := MessageName(m)
	if name == "" {
		return nil, fmt.Errorf("could not determine message name for %v", reflect.TypeOf(m))
	}
	r.mu.RLock()
	domain := r.domains[name]
	if domain == "" {
		// lookup domain for the package
		domain = r.domains[name[strings.LastIndex(name, ".")+1:]]
	}
	r.mu.RUnlock()

	if domain == "" {
		domain = r.defaultDomain
		if domain == "" {
			domain = googleApisDomain
		}
	}

	if b, err := proto.Marshal(m); err != nil {
		return nil, err
	} else {
		return &any.Any{TypeUrl: fmt.Sprintf("%s/%s", domain, name), Value: b}, nil
	}
}
