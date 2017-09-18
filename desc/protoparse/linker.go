package protoparse

import (
	"bytes"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/golang/protobuf/proto"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
)

type linker struct {
	files          map[string]*dpb.FileDescriptorProto
	aggregates     map[string][]*aggregate
	descriptorPool map[*dpb.FileDescriptorProto]map[string]proto.Message
	extensions     map[string]map[int32]string
}

func newLinker(files map[string]*dpb.FileDescriptorProto, aggregates map[string][]*aggregate) *linker {
	return &linker{files: files, aggregates: aggregates}
}

func (l *linker) linkFiles() (map[string]*desc.FileDescriptor, error) {
	// First, we put all symbols into a single pool, which lets us ensure there
	// are no duplicate symbols and will also let us resolve and revise all type
	// references in next step.
	if err := l.createDescriptorPool(); err != nil {
		return nil, err
	}

	// After we've populated the pool, we can now try to resolve all type
	// references. All references must be checked for correct type, any field's
	// with enum types must be corrected (since we parse them as if they are
	// message references since we don't actually know message or enum until
	// link time), and references will be re-written to be fully-qualified
	// references (e.g. start with a dot ".").
	if err := l.resolveReferences(); err != nil {
		return nil, err
	}

	// Now we've validated the descriptors, so we can link them into rich
	// descriptors. This is a little redundant since that step does similar
	// checking of symbols. But, without breaking encapsulation (e.g. exporting
	// a lot of fields from desc package that are currently unexported) or
	// merging this into the same package, we can't really prevent it.
	linked, err := l.createdLinkedDescriptors()
	if err != nil {
		return nil, err
	}

	// Now that we have linked descriptors, we can interpret any uninterpreted
	// options that remain.
	for _, fd := range linked {
		if err := l.interpretFileOptions(fd); err != nil {
			return nil, err
		}
	}

	return linked, nil
}

func (l *linker) createDescriptorPool() error {
	l.descriptorPool = map[*dpb.FileDescriptorProto]map[string]proto.Message{}
	for _, fd := range l.files {
		pool := map[string]proto.Message{}
		l.descriptorPool[fd] = pool
		prefix := fd.GetPackage()
		if prefix != "" {
			prefix += "."
		}
		for _, md := range fd.MessageType {
			if err := addMessageToPool(fd, pool, prefix, md); err != nil {
				return err
			}
		}
		for _, fld := range fd.Extension {
			if err := addFieldToPool(fd, pool, prefix, fld); err != nil {
				return err
			}
		}
		for _, ed := range fd.EnumType {
			if err := addEnumToPool(fd, pool, prefix, ed); err != nil {
				return err
			}
		}
		for _, sd := range fd.Service {
			if err := addServiceToPool(fd, pool, prefix, sd); err != nil {
				return err
			}
		}
	}
	// try putting everything into a single pool, to ensure there are no duplicates
	// across files (e.g. same symbol, but declared in two different files)
	type entry struct {
		file string
		msg  proto.Message
	}
	pool := map[string]entry{}
	for f, p := range l.descriptorPool {
		for k, v := range p {
			if e, ok := pool[k]; ok {
				type1 := descriptorType(e.msg)
				file1 := e.file
				type2 := descriptorType(v)
				file2 := f.GetName()
				if file2 < file1 {
					file1, file2 = file2, file1
					type1, type2 = type2, type1
				}
				return fmt.Errorf("duplicate symbol %s: %s in %q and %s in %q", k, type1, file1, type2, file2)
			}
			pool[k] = entry{file: f.GetName(), msg: v}
		}
	}

	return nil
}

func addMessageToPool(fd *dpb.FileDescriptorProto, pool map[string]proto.Message, prefix string, md *dpb.DescriptorProto) error {
	fqn := prefix + md.GetName()
	if err := addToPool(fd, pool, fqn, md); err != nil {
		return err
	}
	prefix = fqn + "."
	for _, fld := range md.Field {
		if err := addFieldToPool(fd, pool, prefix, fld); err != nil {
			return err
		}
	}
	for _, fld := range md.Extension {
		if err := addFieldToPool(fd, pool, prefix, fld); err != nil {
			return err
		}
	}
	for _, nmd := range md.NestedType {
		if err := addMessageToPool(fd, pool, prefix, nmd); err != nil {
			return err
		}
	}
	for _, ed := range md.EnumType {
		if err := addEnumToPool(fd, pool, prefix, ed); err != nil {
			return err
		}
	}
	return nil
}

func addFieldToPool(fd *dpb.FileDescriptorProto, pool map[string]proto.Message, prefix string, fld *dpb.FieldDescriptorProto) error {
	fqn := prefix + fld.GetName()
	return addToPool(fd, pool, fqn, fld)
}

func addEnumToPool(fd *dpb.FileDescriptorProto, pool map[string]proto.Message, prefix string, ed *dpb.EnumDescriptorProto) error {
	fqn := prefix + ed.GetName()
	if err := addToPool(fd, pool, fqn, ed); err != nil {
		return err
	}
	for _, evd := range ed.Value {
		vfqn := fqn + "." + evd.GetName()
		if err := addToPool(fd, pool, vfqn, evd); err != nil {
			return err
		}
	}
	return nil
}

func addServiceToPool(fd *dpb.FileDescriptorProto, pool map[string]proto.Message, prefix string, sd *dpb.ServiceDescriptorProto) error {
	fqn := prefix + sd.GetName()
	if err := addToPool(fd, pool, fqn, sd); err != nil {
		return err
	}
	for _, mtd := range sd.Method {
		mfqn := fqn + "." + mtd.GetName()
		if err := addToPool(fd, pool, mfqn, mtd); err != nil {
			return err
		}
	}
	return nil
}

func addToPool(fd *dpb.FileDescriptorProto, pool map[string]proto.Message, fqn string, dsc proto.Message) error {
	if d, ok := pool[fqn]; ok {
		thisType := descriptorType(dsc)
		otherType := descriptorType(d)
		return fmt.Errorf("file %q: duplicate symbol %s: %s and %s", fd.GetName(), fqn, thisType, otherType)
	}
	pool[fqn] = dsc
	return nil
}

func descriptorType(m proto.Message) string {
	switch m := m.(type) {
	case *dpb.DescriptorProto:
		return "message"
	case *dpb.FieldDescriptorProto:
		if m.GetExtendee() == "" {
			return "field"
		} else {
			return "extension"
		}
	case *dpb.EnumDescriptorProto:
		return "enum"
	case *dpb.EnumValueDescriptorProto:
		return "enum value"
	case *dpb.ServiceDescriptorProto:
		return "service"
	case *dpb.MethodDescriptorProto:
		return "method"
	case *dpb.FileDescriptorProto:
		return "file"
	default:
		// shouldn't be possible
		return fmt.Sprintf("%T", m)
	}
}

func (l *linker) resolveReferences() error {
	l.extensions = map[string]map[int32]string{}
	for _, fd := range l.files {
		prefix := fd.GetPackage()
		scopes := []scope{fileScope(fd, l)}
		if prefix != "" {
			prefix += "."
		}
		if fd.Options != nil {
			if err := l.resolveOptions(fd, "file", fd.GetName(), proto.MessageName(fd.Options), fd.Options.UninterpretedOption, scopes); err != nil {
				return err
			}
		}
		for _, md := range fd.MessageType {
			if err := l.resolveMessageTypes(fd, prefix, md, scopes); err != nil {
				return err
			}
		}
		for _, fld := range fd.Extension {
			if err := l.resolveFieldTypes(fd, prefix, fld, scopes); err != nil {
				return err
			}
		}
		for _, ed := range fd.EnumType {
			enumFqn := prefix + ed.GetName()
			if ed.Options != nil {
				if err := l.resolveOptions(fd, "enum", enumFqn, proto.MessageName(ed.Options), ed.Options.UninterpretedOption, scopes); err != nil {
					return err
				}
			}
			for _, evd := range ed.Value {
				if evd.Options != nil {
					evFqn := enumFqn + "." + evd.GetName()
					if err := l.resolveOptions(fd, "enum value", evFqn, proto.MessageName(evd.Options), evd.Options.UninterpretedOption, scopes); err != nil {
						return err
					}
				}
			}
		}
		for _, sd := range fd.Service {
			if err := l.resolveServiceTypes(fd, prefix, sd, scopes); err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *linker) resolveMessageTypes(fd *dpb.FileDescriptorProto, prefix string, md *dpb.DescriptorProto, scopes []scope) error {
	fqn := prefix + md.GetName()
	scope := messageScope(fqn, l.descriptorPool[fd])
	scopes = append(scopes, scope)
	prefix = fqn + "."

	if md.Options != nil {
		if err := l.resolveOptions(fd, "message", fqn, proto.MessageName(md.Options), md.Options.UninterpretedOption, scopes); err != nil {
			return err
		}
	}

	for _, nmd := range md.NestedType {
		if err := l.resolveMessageTypes(fd, prefix, nmd, scopes); err != nil {
			return err
		}
	}
	for _, fld := range md.Field {
		if err := l.resolveFieldTypes(fd, prefix, fld, scopes); err != nil {
			return err
		}
	}
	for _, fld := range md.Extension {
		if err := l.resolveFieldTypes(fd, prefix, fld, scopes); err != nil {
			return err
		}
	}
	//for _, er := range md.ExtensionRange {
	//	if er.ExtensionRangeOptions != nil {
	//		if err := l.resolveOptions(fd, proto.MessageName(er.ExtensionRangeOptions), er.ExtensionRangeOptions.UninterpretedOption, scopes); err != nil {
	//			return err
	//		}
	//	}
	//}
	return nil
}

func (l *linker) resolveFieldTypes(fd *dpb.FileDescriptorProto, prefix string, fld *dpb.FieldDescriptorProto, scopes []scope) error {
	thisName := prefix + fld.GetName()
	elemType := "field"
	if fld.GetExtendee() != "" {
		fqn, dsc := l.resolve(fd, fld.GetExtendee(), scopes)
		if dsc == nil {
			return fmt.Errorf("file %q: field %s extends unknown type: %s", fd.GetName(), thisName, fld.GetExtendee())
		}
		extd, ok := dsc.(*dpb.DescriptorProto)
		if !ok {
			otherType := descriptorType(dsc)
			return fmt.Errorf("file %q: field %s extends invalid type: %s is a %s, not a message", fd.GetName(), thisName, fqn, otherType)
		}
		fld.Extendee = proto.String("." + fqn)
		// make sure the tag number is in range
		found := false
		tag := fld.GetNumber()
		for _, rng := range extd.ExtensionRange {
			if tag >= rng.GetStart() && tag < rng.GetEnd() {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("file %q: field %s tag is not in valid range for extended type %s: %d", fd.GetName(), thisName, fqn, tag)
		}
		// make sure tag is not a duplicate
		usedExtTags := l.extensions[fqn]
		if usedExtTags == nil {
			usedExtTags = map[int32]string{}
			l.extensions[fqn] = usedExtTags
		}
		if other := usedExtTags[fld.GetNumber()]; other != "" {
			return fmt.Errorf("file %q: duplicate extension for %s: %s and %s are both using tag %d", fd.GetName(), fqn, other, thisName, fld.GetNumber())
		}
		usedExtTags[fld.GetNumber()] = thisName
		elemType = "extension"
	}

	if fld.Options != nil {
		if err := l.resolveOptions(fd, elemType, thisName, proto.MessageName(fld.Options), fld.Options.UninterpretedOption, scopes); err != nil {
			return err
		}
	}

	if fld.GetTypeName() == "" {
		// scalar type; no further resolution required
		return nil
	}

	fqn, dsc := l.resolve(fd, fld.GetTypeName(), scopes)
	if dsc == nil {
		return fmt.Errorf("file %q: field %s references unknown type: %s", fd.GetName(), thisName, fld.GetTypeName())
	}
	switch dsc := dsc.(type) {
	case *dpb.DescriptorProto:
		fld.TypeName = proto.String("." + fqn)
	case *dpb.EnumDescriptorProto:
		fld.TypeName = proto.String("." + fqn)
		// we tentatively set type to message, but now we know it's actually an enum
		fld.Type = dpb.FieldDescriptorProto_TYPE_ENUM.Enum()
	default:
		otherType := descriptorType(dsc)
		return fmt.Errorf("file %q: field %s has invalid type: %s is a %s, not a message or enum", fd.GetName(), thisName, fqn, otherType)
	}
	return nil
}

func (l *linker) resolveServiceTypes(fd *dpb.FileDescriptorProto, prefix string, sd *dpb.ServiceDescriptorProto, scopes []scope) error {
	thisName := prefix + sd.GetName()
	if sd.Options != nil {
		if err := l.resolveOptions(fd, "service", thisName, proto.MessageName(sd.Options), sd.Options.UninterpretedOption, scopes); err != nil {
			return err
		}
	}

	for _, mtd := range sd.Method {
		if mtd.Options != nil {
			if err := l.resolveOptions(fd, "method", thisName+"."+mtd.GetName(), proto.MessageName(mtd.Options), mtd.Options.UninterpretedOption, scopes); err != nil {
				return err
			}
		}
		fqn, dsc := l.resolve(fd, mtd.GetInputType(), scopes)
		if dsc == nil {
			return fmt.Errorf("file %q: service %s method %s references unknown request type: %s", fd.GetName(), thisName, mtd.GetName(), mtd.GetInputType())
		}
		if _, ok := dsc.(*dpb.DescriptorProto); !ok {
			otherType := descriptorType(dsc)
			return fmt.Errorf("file %q: service %s method %s has invalid request type: %s is a %s, not a message", fd.GetName(), thisName, mtd.GetName(), fqn, otherType)
		}
		mtd.InputType = proto.String("." + fqn)

		fqn, dsc = l.resolve(fd, mtd.GetOutputType(), scopes)
		if dsc == nil {
			return fmt.Errorf("file %q: service %s method %s references unknown response type: %s", fd.GetName(), thisName, mtd.GetName(), mtd.GetOutputType())
		}
		if _, ok := dsc.(*dpb.DescriptorProto); !ok {
			otherType := descriptorType(dsc)
			return fmt.Errorf("file %q: service %s method %s has invalid response type: %s is a %s, not a message", fd.GetName(), thisName, mtd.GetName(), fqn, otherType)
		}
		mtd.OutputType = proto.String("." + fqn)
	}
	return nil
}

func (l *linker) resolveOptions(fd *dpb.FileDescriptorProto, elemType, elemName, optType string, opts []*dpb.UninterpretedOption, scopes []scope) error {
	mc := &messageContext{filename: fd.GetName(), elementType: elemType, elementName: elemName}
	for _, opt := range opts {
		for ni, nm := range opt.Name {
			if nm.GetIsExtension() {
				mc.optName = opt.Name[:ni+1]
				fqn, dsc := l.resolve(fd, nm.GetNamePart(), scopes)
				if dsc == nil {
					return fmt.Errorf("%v: unknown extension: %s", mc, nm.GetNamePart())
				}
				if ext, ok := dsc.(*dpb.FieldDescriptorProto); !ok {
					otherType := descriptorType(dsc)
					return fmt.Errorf("%v: invalid extension: %s is a %s, not an extension", mc, nm.GetNamePart(), otherType)
				} else if ext.GetExtendee() == "" {
					return fmt.Errorf("%v: invalid extension: %s is a field but not an extension", mc, nm.GetNamePart())
				}
				nm.NamePart = proto.String("." + fqn)
			}
		}
	}
	return nil
}

func (l *linker) resolve(fd *dpb.FileDescriptorProto, name string, scopes []scope) (string, proto.Message) {
	if strings.HasPrefix(name, ".") {
		// already fully-qualified
		d := l.findSymbol(fd, name[1:], false, map[*dpb.FileDescriptorProto]struct{}{})
		if d != nil {
			return name[1:], d
		}
	} else {
		// unqualified, so we look in the enclosing (last) scope first and move
		// towards outermost (first) scope, trying to resolve the symbol
		for i := len(scopes) - 1; i >= 0; i-- {
			fqn, d := scopes[i](name)
			if d != nil {
				return fqn, d
			}
		}
	}
	return "", nil
}

// scope represents a lexical scope in a proto file in which messages and enums
// can be declared.
type scope func(string) (string, proto.Message)

func fileScope(fd *dpb.FileDescriptorProto, l *linker) scope {
	// we search symbols in this file, but also symbols in other files
	// that have the same package as this file
	pkg := fd.GetPackage()
	return func(name string) (string, proto.Message) {
		var n string
		if pkg == "" {
			n = name
		} else {
			n = pkg + "." + name
		}
		d := l.findSymbol(fd, n, false, map[*dpb.FileDescriptorProto]struct{}{})
		if d != nil {
			return n, d
		}
		// maybe name is already fully-qualified, just without a leading dot
		d = l.findSymbol(fd, name, false, map[*dpb.FileDescriptorProto]struct{}{})
		if d != nil {
			return name, d
		}
		return "", nil
	}
}

func messageScope(messageName string, filePool map[string]proto.Message) scope {
	return func(name string) (string, proto.Message) {
		n := messageName + "." + name
		if d, ok := filePool[n]; ok {
			return n, d
		}
		return "", nil
	}
}

func (l *linker) findSymbol(fd *dpb.FileDescriptorProto, name string, public bool, checked map[*dpb.FileDescriptorProto]struct{}) proto.Message {
	if _, ok := checked[fd]; ok {
		// already checked this one
		return nil
	}
	checked[fd] = struct{}{}
	d := l.descriptorPool[fd][name]
	if d != nil {
		return d
	}

	// When public = false, we are searching only directly imported symbols. But we
	// also need to search transitive public imports due to semantics of public imports.
	if public {
		for _, depIndex := range fd.PublicDependency {
			dep := fd.Dependency[depIndex]
			depfd := l.files[dep]
			if depfd == nil {
				// we'll catch this error later
				continue
			}
			if d = l.findSymbol(depfd, name, true, checked); d != nil {
				return d
			}
		}
	} else {
		for _, dep := range fd.Dependency {
			depfd := l.files[dep]
			if depfd == nil {
				// we'll catch this error later
				continue
			}
			if d = l.findSymbol(depfd, name, true, checked); d != nil {
				return d
			}
		}
	}

	return nil
}

func (l *linker) createdLinkedDescriptors() (map[string]*desc.FileDescriptor, error) {
	names := make([]string, 0, len(l.files))
	for name := range l.files {
		names = append(names, name)
	}
	sort.Strings(names)
	linked := map[string]*desc.FileDescriptor{}
	for _, name := range names {
		if _, err := l.linkFile(name, nil, linked); err != nil {
			return nil, err
		}
	}
	return linked, nil
}

func (l *linker) linkFile(name string, seen []string, linked map[string]*desc.FileDescriptor) (*desc.FileDescriptor, error) {
	// check for import cycle
	for _, s := range seen {
		if name == s {
			var msg bytes.Buffer
			first := true
			for _, s := range seen {
				if first {
					first = false
				} else {
					msg.WriteString(" -> ")
				}
				fmt.Fprintf(&msg, "%q", s)
			}
			fmt.Fprintf(&msg, " -> %q", name)
			return nil, fmt.Errorf("cycle found in imports: %s", msg.String())
		}
	}
	seen = append(seen, name)

	if lfd, ok := linked[name]; ok {
		// already linked
		return lfd, nil
	}
	fd := l.files[name]
	if fd == nil {
		importer := seen[len(seen)-2] // len-1 is *this* file, before that is the one that imported it
		return nil, fmt.Errorf("no descriptor found for %q, imported by %q", name, importer)
	}
	var deps []*desc.FileDescriptor
	for _, dep := range fd.Dependency {
		ldep, err := l.linkFile(dep, seen, linked)
		if err != nil {
			return nil, err
		}
		deps = append(deps, ldep)
	}
	lfd, err := desc.CreateFileDescriptor(fd, deps...)
	if err != nil {
		return nil, fmt.Errorf("error linking %q: %s", name, err)
	}
	linked[name] = lfd
	return lfd, nil
}

func (l *linker) interpretFileOptions(fd *desc.FileDescriptor) error {
	opts := fd.GetFileOptions()
	if opts != nil {
		if len(opts.UninterpretedOption) > 0 {
			if err := l.interpretOptions(fd, opts, opts.UninterpretedOption); err != nil {
				return err
			}
		}
		opts.UninterpretedOption = nil
	}
	for _, md := range fd.GetMessageTypes() {
		if err := l.interpretMessageOptions(md); err != nil {
			return err
		}
	}
	for _, fld := range fd.GetExtensions() {
		if err := l.interpretFieldOptions(fld); err != nil {
			return err
		}
	}
	for _, ed := range fd.GetEnumTypes() {
		if err := l.interpretEnumOptions(ed); err != nil {
			return err
		}
	}
	for _, sd := range fd.GetServices() {
		opts := sd.GetServiceOptions()
		if opts != nil {
			if len(opts.UninterpretedOption) > 0 {
				if err := l.interpretOptions(sd, opts, opts.UninterpretedOption); err != nil {
					return err
				}
			}
			opts.UninterpretedOption = nil
		}
		for _, mtd := range sd.GetMethods() {
			opts := mtd.GetMethodOptions()
			if opts != nil {
				if len(opts.UninterpretedOption) > 0 {
					if err := l.interpretOptions(mtd, opts, opts.UninterpretedOption); err != nil {
						return err
					}
				}
				opts.UninterpretedOption = nil
			}
		}
	}
	return nil
}

func (l *linker) interpretMessageOptions(md *desc.MessageDescriptor) error {
	opts := md.GetMessageOptions()
	if opts != nil {
		if len(opts.UninterpretedOption) > 0 {
			if err := l.interpretOptions(md, opts, opts.UninterpretedOption); err != nil {
				return err
			}
		}
		opts.UninterpretedOption = nil
	}
	for _, fld := range md.GetFields() {
		if err := l.interpretFieldOptions(fld); err != nil {
			return err
		}
	}
	for _, fld := range md.GetNestedExtensions() {
		if err := l.interpretFieldOptions(fld); err != nil {
			return err
		}
	}
	//for _, er := range md.AsDescriptorProto().GetExtensionRange() {
	//	opts := er.GetExtensionRangeOptions()
	//	if opts != nil && len(opts.UninterpretedOption) > 0 {
	//		uo := opts.UninterpretedOption
	//		opts.UninterpretedOption = nil
	//		if err := l.interpretOptions(md.GetFile(), opts, uo); err != nil {
	//			return err
	//		}
	//	}
	//}
	for _, nmd := range md.GetNestedMessageTypes() {
		if err := l.interpretMessageOptions(nmd); err != nil {
			return err
		}
	}
	for _, ed := range md.GetNestedEnumTypes() {
		if err := l.interpretEnumOptions(ed); err != nil {
			return err
		}
	}
	return nil
}

var emptyFieldOptions dpb.FieldOptions

func (l *linker) interpretFieldOptions(fld *desc.FieldDescriptor) error {
	opts := fld.GetFieldOptions()
	if opts != nil {
		if len(opts.UninterpretedOption) > 0 {
			uo := opts.UninterpretedOption
			if i, err := processDefaultOption(fld, uo); err != nil {
				return err
			} else if i >= 0 {
				if len(uo) == 1 {
					// The only option was "default" and we just hoisted that out. So
					// clear out options if there is nothing there.
					opts.UninterpretedOption = nil
					if proto.Equal(opts, &emptyFieldOptions) {
						fld.AsFieldDescriptorProto().Options = nil
					}
					return nil
				}
				// if default was found, remove it before processing the other options
				uo = removeOption(uo, i)
			}

			if err := l.interpretOptions(fld, opts, uo); err != nil {
				return err
			}
		}
		opts.UninterpretedOption = nil
	}
	return nil
}

func processDefaultOption(fld *desc.FieldDescriptor, uos []*dpb.UninterpretedOption) (defaultIndex int, err error) {
	found, err := findOption(uos, "default")
	if err != nil {
		return -1, fmt.Errorf("file %q: field %s: %s", fld.GetFile().GetName(), fld.GetFullyQualifiedName(), err)
	} else if found == -1 {
		return -1, nil
	}
	opt := uos[found]
	if fld.IsRepeated() {
		return -1, fmt.Errorf("file %q: default value cannot be set for field %s because it is repeated",
			fld.GetFile().GetName(), fld.GetFullyQualifiedName())
	}
	if fld.GetType() == dpb.FieldDescriptorProto_TYPE_GROUP || fld.GetType() == dpb.FieldDescriptorProto_TYPE_MESSAGE {
		return -1, fmt.Errorf("file %q: default value cannot be set for field %s because it is a message",
			fld.GetFile().GetName(), fld.GetFullyQualifiedName())
	}
	var val interface{}
	if opt.AggregateValue != nil {
		return -1, fmt.Errorf("file %q: default value for field %s cannot be an aggregate",
			fld.GetFile().GetName(), fld.GetFullyQualifiedName())
	} else if opt.DoubleValue != nil {
		val = opt.GetDoubleValue()
	} else if opt.IdentifierValue != nil {
		id := opt.GetIdentifierValue()
		if id == "true" {
			val = true
		} else if id == "false" {
			val = false
		} else {
			val = identifier(id)
		}
	} else if opt.NegativeIntValue != nil {
		val = opt.GetNegativeIntValue()
	} else if opt.PositiveIntValue != nil {
		val = opt.GetPositiveIntValue()
	} else if opt.StringValue != nil {
		val = opt.GetStringValue()
	}

	mc := &messageContext{
		file:        fld.GetFile(),
		elementName: fld.GetFullyQualifiedName(),
		elementType: descriptorType(fld.AsProto()),
		optName:     opt.GetName(),
	}
	v, err := fieldValue(mc, fld, val, true)
	if err != nil {
		return -1, err
	}
	if str, ok := v.(string); ok {
		fld.AsFieldDescriptorProto().DefaultValue = proto.String(str)
	} else if b, ok := v.([]byte); ok {
		fld.AsFieldDescriptorProto().DefaultValue = proto.String(encodeDefaultBytes(b))
	} else {
		var flt float64
		var ok bool
		if flt, ok = v.(float64); !ok {
			var flt32 float32
			if flt32, ok = v.(float32); ok {
				flt = float64(flt32)
			}
		}
		if ok {
			if math.IsInf(flt, 1) {
				fld.AsFieldDescriptorProto().DefaultValue = proto.String("inf")
			} else if ok && math.IsInf(flt, -1) {
				fld.AsFieldDescriptorProto().DefaultValue = proto.String("-inf")
			} else if ok && math.IsNaN(flt) {
				fld.AsFieldDescriptorProto().DefaultValue = proto.String("nan")
			} else {
				fld.AsFieldDescriptorProto().DefaultValue = proto.String(fmt.Sprintf("%v", v))
			}
		} else {
			fld.AsFieldDescriptorProto().DefaultValue = proto.String(fmt.Sprintf("%v", v))
		}
	}
	return found, nil
}

func encodeDefaultBytes(b []byte) string {
	var buf bytes.Buffer
	writeEscapedBytes(&buf, b)
	return buf.String()
}

func (l *linker) interpretEnumOptions(ed *desc.EnumDescriptor) error {
	opts := ed.GetEnumOptions()
	if opts != nil {
		if len(opts.UninterpretedOption) > 0 {
			if err := l.interpretOptions(ed, opts, opts.UninterpretedOption); err != nil {
				return err
			}
		}
		opts.UninterpretedOption = nil
	}
	for _, evd := range ed.GetValues() {
		opts := evd.GetEnumValueOptions()
		if opts != nil {
			if len(opts.UninterpretedOption) > 0 {
				if err := l.interpretOptions(evd, opts, opts.UninterpretedOption); err != nil {
					return err
				}
			}
			opts.UninterpretedOption = nil
		}
	}
	return nil
}

func (l *linker) interpretOptions(element desc.Descriptor, opts proto.Message, uninterpreted []*dpb.UninterpretedOption) error {
	optsd, err := desc.LoadMessageDescriptorForMessage(opts)
	if err != nil {
		return err
	}
	dm := dynamic.NewMessage(optsd)
	err = dm.ConvertFrom(opts)
	if err != nil {
		return err
	}

	mc := &messageContext{file: element.GetFile(), elementName: element.GetName(), elementType: descriptorType(element.AsProto())}
	for _, uo := range uninterpreted {
		if !uo.Name[0].GetIsExtension() && uo.Name[0].GetNamePart() == "uninterpreted_option" {
			// uninterpreted_option might be found reflectively, but is not actually valid for use
			return fmt.Errorf("%v: invalid option 'uninterpreted_option'", mc)
		}
		mc.optName = uo.Name
		err = l.interpretField(mc, element, dm, uo, 0)
		if err != nil {
			return err
		}
	}

	return dm.ConvertTo(opts)
}

func (l *linker) interpretField(mc *messageContext, element desc.Descriptor, dm *dynamic.Message, opt *dpb.UninterpretedOption, nameIndex int) error {
	// TODO: better error messages: accumulate full path during recursion so message includes more comprehensible context
	var fld *desc.FieldDescriptor
	nm := opt.GetName()[nameIndex]
	if nm.GetIsExtension() {
		extName := nm.GetNamePart()[1:]
		fld = findExtension(element.GetFile(), extName /* skip leading dot */, false, map[*desc.FileDescriptor]struct{}{})
		if fld == nil {
			return fmt.Errorf("%v: unrecognized extension %s of %s",
				mc, extName, dm.GetMessageDescriptor().GetFullyQualifiedName())
		}
		if fld.GetOwner().GetFullyQualifiedName() != dm.GetMessageDescriptor().GetFullyQualifiedName() {
			return fmt.Errorf("%v: extension %s should extend %s but instead extends %s",
				mc, extName, dm.GetMessageDescriptor().GetFullyQualifiedName(), fld.GetOwner().GetFullyQualifiedName())
		}
	} else {
		fld = dm.GetMessageDescriptor().FindFieldByName(nm.GetNamePart())
		if fld == nil {
			return fmt.Errorf("%v: field %s of %s does not exist",
				mc, nm.GetNamePart(), dm.GetMessageDescriptor().GetFullyQualifiedName())
		}
	}

	if len(opt.GetName()) > nameIndex+1 {
		if fld.GetType() != dpb.FieldDescriptorProto_TYPE_MESSAGE {
			return fmt.Errorf("%v: cannot set field %s because %s is not a message",
				mc, opt.GetName()[nameIndex+1].GetNamePart(), nm.GetNamePart())
		}
		if fld.IsRepeated() {
			return fmt.Errorf("%v: cannot set field %s because %s is repeated (must use an aggregate)",
				mc, opt.GetName()[nameIndex+1].GetNamePart(), nm.GetNamePart())
		}
		var fdm *dynamic.Message
		var err error
		if dm.HasField(fld) {
			var v interface{}
			v, err = dm.TryGetField(fld)
			fdm = v.(*dynamic.Message)
		} else {
			fdm = dynamic.NewMessage(fld.GetMessageType())
			err = dm.TrySetField(fld, fdm)
		}
		if err != nil {
			return err
		}
		// recurse to set next part of name
		return l.interpretField(mc, element, fdm, opt, nameIndex+1)
	}

	var val interface{}
	if opt.AggregateValue != nil {
		val = l.aggregates[opt.GetAggregateValue()]
	} else if opt.DoubleValue != nil {
		val = opt.GetDoubleValue()
	} else if opt.IdentifierValue != nil {
		id := opt.GetIdentifierValue()
		if id == "true" {
			val = true
		} else if id == "false" {
			val = false
		} else {
			val = identifier(id)
		}
	} else if opt.NegativeIntValue != nil {
		val = opt.GetNegativeIntValue()
	} else if opt.PositiveIntValue != nil {
		val = opt.GetPositiveIntValue()
	} else if opt.StringValue != nil {
		val = opt.GetStringValue()
	}
	return setOptionField(mc, dm, fld, val)
}

func findExtension(fd *desc.FileDescriptor, name string, public bool, checked map[*desc.FileDescriptor]struct{}) *desc.FieldDescriptor {
	if _, ok := checked[fd]; ok {
		return nil
	}
	checked[fd] = struct{}{}
	d := fd.FindSymbol(name)
	if d != nil {
		if fld, ok := d.(*desc.FieldDescriptor); ok {
			return fld
		}
		return nil
	}

	// When public = false, we are searching only directly imported symbols. But we
	// also need to search transitive public imports due to semantics of public imports.
	if public {
		for _, dep := range fd.GetPublicDependencies() {
			d := findExtension(dep, name, true, checked)
			if d != nil {
				return d
			}
		}
	} else {
		for _, dep := range fd.GetDependencies() {
			d := findExtension(dep, name, true, checked)
			if d != nil {
				return d
			}
		}
	}
	return nil
}

func setOptionField(mc *messageContext, dm *dynamic.Message, fld *desc.FieldDescriptor, val interface{}) error {
	if sl, ok := val.([]interface{}); ok {
		// handle slices a little differently than the others
		if !fld.IsRepeated() {
			return fmt.Errorf("%v: value is an array but field is not repeated", mc)
		}
		origPath := mc.optAggPath
		defer func() {
			mc.optAggPath = origPath
		}()
		for index, item := range sl {
			mc.optAggPath = fmt.Sprintf("%s[%d]", origPath, index)
			if v, err := fieldValue(mc, fld, item, false); err != nil {
				return err
			} else if err = dm.TryAddRepeatedField(fld, v); err != nil {
				return fmt.Errorf("%v: error setting value: %s", mc, err)
			}
		}
		return nil
	}

	v, err := fieldValue(mc, fld, val, false)
	if err != nil {
		return err
	}
	if fld.IsRepeated() {
		err = dm.TryAddRepeatedField(fld, v)
	} else {
		if dm.HasField(fld) {
			return fmt.Errorf("%v: non-repeated option field %s already set", mc, fieldName(fld))
		}
		err = dm.TrySetField(fld, v)
	}
	if err != nil {
		return fmt.Errorf("%v: error setting value: %s", mc, err)
	}

	return nil
}

type messageContext struct {
	file        *desc.FileDescriptor
	filename    string
	elementType string
	elementName string
	optName     []*dpb.UninterpretedOption_NamePart
	optAggPath  string
}

func (c *messageContext) String() string {
	fn := c.filename
	if c.file != nil {
		fn = c.file.GetName()
	}
	var ctx bytes.Buffer
	fmt.Fprintf(&ctx, "file %q", fn)
	if c.elementType != "file" {
		fmt.Fprintf(&ctx, ": %s %s", c.elementType, c.elementName)
	}
	if c.optName != nil {
		ctx.WriteString(": option ")
		writeOptionName(&ctx, c.optName)
		if c.optAggPath != "" {
			fmt.Fprintf(&ctx, " at %s", c.optAggPath)
		}
	}
	return ctx.String()
}

func writeOptionName(buf *bytes.Buffer, parts []*dpb.UninterpretedOption_NamePart) {
	first := true
	for _, p := range parts {
		if first {
			first = false
		} else {
			buf.WriteByte('.')
		}
		nm := p.GetNamePart()
		if nm[0] == '.' {
			// skip leading dot
			nm = nm[1:]
		}
		if p.GetIsExtension() {
			buf.WriteByte('(')
			buf.WriteString(nm)
			buf.WriteByte(')')
		} else {
			buf.WriteString(nm)
		}
	}
}

func fieldName(fld *desc.FieldDescriptor) string {
	if fld.IsExtension() {
		return fld.GetFullyQualifiedName()
	} else {
		return fld.GetName()
	}
}

func valueKind(val interface{}) string {
	switch val := val.(type) {
	case identifier:
		return "identifier"
	case bool:
		return "bool"
	case int64:
		if val < 0 {
			return "negative integer"
		}
		return "integer"
	case uint64:
		return "integer"
	case float64:
		return "double"
	case string, []byte:
		return "string/bytes"
	case []*aggregate:
		return "message"
	default:
		return fmt.Sprintf("%T", val)
	}
}

func fieldValue(mc *messageContext, fld *desc.FieldDescriptor, val interface{}, enumAsString bool) (interface{}, error) {
	switch fld.GetType() {
	case dpb.FieldDescriptorProto_TYPE_ENUM:
		if id, ok := val.(identifier); ok {
			ev := fld.GetEnumType().FindValueByName(string(id))
			if ev == nil {
				return nil, fmt.Errorf("%v: enum %s has no value named %s", mc, fld.GetEnumType().GetFullyQualifiedName(), id)
			}
			if enumAsString {
				return ev.GetName(), nil
			} else {
				return ev.GetNumber(), nil
			}
		}
		return nil, fmt.Errorf("%v: expecting enum, got %s", mc, valueKind(val))
	case dpb.FieldDescriptorProto_TYPE_MESSAGE, dpb.FieldDescriptorProto_TYPE_GROUP:
		if aggs, ok := val.([]*aggregate); ok {
			fmd := fld.GetMessageType()
			fdm := dynamic.NewMessage(fmd)
			origPath := mc.optAggPath
			defer func() {
				mc.optAggPath = origPath
			}()
			for _, a := range aggs {
				if origPath == "" {
					mc.optAggPath = a.name
				} else {
					mc.optAggPath = origPath + "." + a.name
				}
				var ffld *desc.FieldDescriptor
				if a.name[0] == '[' {
					n := a.name[1 : len(a.name)-1]
					ffld = findExtension(mc.file, n, false, map[*desc.FileDescriptor]struct{}{})
					if ffld == nil {
						// may need to qualify with package name
						pkg := mc.file.GetPackage()
						if pkg != "" {
							ffld = findExtension(mc.file, pkg+"."+n, false, map[*desc.FileDescriptor]struct{}{})
						}
					}
				} else {
					ffld = fmd.FindFieldByName(a.name)
				}
				if ffld == nil {
					return nil, fmt.Errorf("%v: field %s not found", mc, a.name)
				}
				if err := setOptionField(mc, fdm, ffld, a.val); err != nil {
					return nil, err
				}
			}
			return fdm, nil
		}
		return nil, fmt.Errorf("%v: expecting message, got %s", mc, valueKind(val))
	case dpb.FieldDescriptorProto_TYPE_BOOL:
		if b, ok := val.(bool); ok {
			return b, nil
		}
		return nil, fmt.Errorf("%v: expecting bool, got %s", mc, valueKind(val))
	case dpb.FieldDescriptorProto_TYPE_BYTES:
		if b, ok := val.([]byte); ok {
			return b, nil
		}
		if str, ok := val.(string); ok {
			return []byte(str), nil
		}
		return nil, fmt.Errorf("%v: expecting bytes, got %s", mc, valueKind(val))
	case dpb.FieldDescriptorProto_TYPE_STRING:
		if b, ok := val.([]byte); ok {
			return string(b), nil
		}
		if str, ok := val.(string); ok {
			return str, nil
		}
		return nil, fmt.Errorf("%v: expecting string, got %s", mc, valueKind(val))
	case dpb.FieldDescriptorProto_TYPE_INT32, dpb.FieldDescriptorProto_TYPE_SINT32, dpb.FieldDescriptorProto_TYPE_SFIXED32:
		if i, ok := val.(int64); ok {
			if i > math.MaxInt32 || i < math.MinInt32 {
				return nil, fmt.Errorf("%v: value %d is out of range for int32", mc, i)
			}
			return int32(i), nil
		}
		if ui, ok := val.(uint64); ok {
			if ui > math.MaxInt32 {
				return nil, fmt.Errorf("%v: value %d is out of range for int32", mc, ui)
			}
			return int32(ui), nil
		}
		return nil, fmt.Errorf("%v: expecting int32, got %s", mc, valueKind(val))
	case dpb.FieldDescriptorProto_TYPE_UINT32, dpb.FieldDescriptorProto_TYPE_FIXED32:
		if i, ok := val.(int64); ok {
			if i > math.MaxUint32 || i < 0 {
				return nil, fmt.Errorf("%v: value %d is out of range for uint32", mc, i)
			}
			return uint32(i), nil
		}
		if ui, ok := val.(uint64); ok {
			if ui > math.MaxUint32 {
				return nil, fmt.Errorf("%v: value %d is out of range for uint32", mc, ui)
			}
			return uint32(ui), nil
		}
		return nil, fmt.Errorf("%v: expecting uint32, got %s", mc, valueKind(val))
	case dpb.FieldDescriptorProto_TYPE_INT64, dpb.FieldDescriptorProto_TYPE_SINT64, dpb.FieldDescriptorProto_TYPE_SFIXED64:
		if i, ok := val.(int64); ok {
			return i, nil
		}
		if ui, ok := val.(uint64); ok {
			if ui > math.MaxInt64 {
				return nil, fmt.Errorf("%v: value %d is out of range for int64", mc, ui)
			}
			return int64(ui), nil
		}
		return nil, fmt.Errorf("%v: expecting int64, got %s", mc, valueKind(val))
	case dpb.FieldDescriptorProto_TYPE_UINT64, dpb.FieldDescriptorProto_TYPE_FIXED64:
		if i, ok := val.(int64); ok {
			if i < 0 {
				return nil, fmt.Errorf("%v: value %d is out of range for uint64", mc, i)
			}
			return uint64(i), nil
		}
		if ui, ok := val.(uint64); ok {
			return ui, nil
		}
		return nil, fmt.Errorf("%v: expecting uint64, got %s", mc, valueKind(val))
	case dpb.FieldDescriptorProto_TYPE_DOUBLE:
		if d, ok := val.(float64); ok {
			return d, nil
		}
		if i, ok := val.(int64); ok {
			return float64(i), nil
		}
		if u, ok := val.(uint64); ok {
			return float64(u), nil
		}
		return nil, fmt.Errorf("%v: expecting double, got %s", mc, valueKind(val))
	case dpb.FieldDescriptorProto_TYPE_FLOAT:
		if d, ok := val.(float64); ok {
			if (d > math.MaxFloat32 || d < -math.MaxFloat32) && !math.IsInf(d, 1) && !math.IsInf(d, -1) && !math.IsNaN(d) {
				return nil, fmt.Errorf("%v: value %f is out of range for float", mc, d)
			}
			return float32(d), nil
		}
		if i, ok := val.(int64); ok {
			return float32(i), nil
		}
		if u, ok := val.(uint64); ok {
			return float32(u), nil
		}
		return nil, fmt.Errorf("%v: expecting float, got %s", mc, valueKind(val))
	default:
		return nil, fmt.Errorf("%v: unrecognized field type: %s", mc, fld.GetType())
	}
}
