package protoparse

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/internal"
	"github.com/jhump/protoreflect/desc/protoparse/ast"
)

type linker struct {
	files          map[string]*parseResult
	filenames      []string
	errs           *errorHandler
	descriptorPool map[*descriptorpb.FileDescriptorProto]map[string]proto.Message
	extensions     map[string]map[int32]string
	descriptors    map[proto.Message]protoreflect.Descriptor
}

func newLinker(files *parseResults, errs *errorHandler) *linker {
	return &linker{
		files:          files.resultsByFilename,
		filenames:      files.filenames,
		errs:           errs,
		descriptorPool: map[*descriptorpb.FileDescriptorProto]map[string]proto.Message{},
		extensions:     map[string]map[int32]string{},
		descriptors:    map[proto.Message]protoreflect.Descriptor{},
	}
}

func (l *linker) linkFiles() error {
	// First, we put all symbols into a single pool, which lets us ensure there
	// are no duplicate symbols and will also let us resolve and revise all type
	// references in next step.
	if err := l.createDescriptorPool(); err != nil {
		return err
	}

	// After we've populated the pool, we can now try to resolve all type
	// references. All references must be checked for correct type, any fields
	// with enum types must be corrected (since we parse them as if they are
	// message references since we don't actually know message or enum until
	// link time), and references will be re-written to be fully-qualified
	// references (e.g. start with a dot ".").
	if err := l.resolveReferences(); err != nil {
		return err
	}

	if err := l.errs.getError(); err != nil {
		// we won't be able to create real descriptors if we've encountered
		// errors up to this point, so bail at this point
		return err
	}

	// Now that we have linked descriptors, we can interpret any uninterpreted
	// options that remain.
	for _, r := range l.files {
		if err := l.interpretFileOptions(r); err != nil {
			return err
		}
		// we should now have any message_set_wire_format options
		// parsed and can do further validation
		if err := l.checkMessageSets(r); err != nil {
			return err
		}
	}

	// When Parser calls linkFiles, it does not check errs again, and it expects that linkFiles
	// will return all errors it should process. If the ErrorReporter handles all errors itself
	// and always returns nil, we should get ErrInvalidSource here, and need to propagate this
	if err := l.errs.getError(); err != nil {
		return err
	}
	return nil
}

func (l *linker) createDescriptorPool() error {
	for _, filename := range l.filenames {
		r := l.files[filename]
		fd := r.fd
		pool := map[string]proto.Message{}
		l.descriptorPool[fd] = pool
		prefix := fd.GetPackage()
		if prefix != "" {
			prefix += "."
		}
		for _, md := range fd.MessageType {
			if err := addMessageToPool(r, pool, l.errs, prefix, md); err != nil {
				return err
			}
		}
		for _, fld := range fd.Extension {
			if err := addFieldToPool(r, pool, l.errs, prefix, fld); err != nil {
				return err
			}
		}
		for _, ed := range fd.EnumType {
			if err := addEnumToPool(r, pool, l.errs, prefix, ed); err != nil {
				return err
			}
		}
		for _, sd := range fd.Service {
			if err := addServiceToPool(r, pool, l.errs, prefix, sd); err != nil {
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
	for _, filename := range l.filenames {
		f := l.files[filename].fd
		p := l.descriptorPool[f]
		keys := make([]string, 0, len(p))
		for k := range p {
			keys = append(keys, k)
		}
		sort.Strings(keys) // for deterministic error reporting
		for _, k := range keys {
			v := p[k]
			if e, ok := pool[k]; ok {
				desc1 := e.msg
				file1 := e.file
				desc2 := v
				file2 := f.GetName()
				if file2 < file1 {
					file1, file2 = file2, file1
					desc1, desc2 = desc2, desc1
				}
				node := l.files[file2].nodes[desc2]
				if err := l.errs.handleErrorWithPos(node.Start(), "duplicate symbol %s: already defined as %s in %q", k, descriptorType(desc1), file1); err != nil {
					return err
				}
			}
			pool[k] = entry{file: f.GetName(), msg: v}
		}
	}

	return nil
}

func addMessageToPool(r *parseResult, pool map[string]proto.Message, errs *errorHandler, prefix string, md *descriptorpb.DescriptorProto) error {
	fqn := prefix + md.GetName()
	if err := addToPool(r, pool, errs, fqn, md); err != nil {
		return err
	}
	prefix = fqn + "."
	for _, fld := range md.Field {
		if err := addFieldToPool(r, pool, errs, prefix, fld); err != nil {
			return err
		}
	}
	for _, fld := range md.Extension {
		if err := addFieldToPool(r, pool, errs, prefix, fld); err != nil {
			return err
		}
	}
	for _, nmd := range md.NestedType {
		if err := addMessageToPool(r, pool, errs, prefix, nmd); err != nil {
			return err
		}
	}
	for _, ed := range md.EnumType {
		if err := addEnumToPool(r, pool, errs, prefix, ed); err != nil {
			return err
		}
	}
	return nil
}

func addFieldToPool(r *parseResult, pool map[string]proto.Message, errs *errorHandler, prefix string, fld *descriptorpb.FieldDescriptorProto) error {
	fqn := prefix + fld.GetName()
	return addToPool(r, pool, errs, fqn, fld)
}

func addEnumToPool(r *parseResult, pool map[string]proto.Message, errs *errorHandler, prefix string, ed *descriptorpb.EnumDescriptorProto) error {
	fqn := prefix + ed.GetName()
	if err := addToPool(r, pool, errs, fqn, ed); err != nil {
		return err
	}
	for _, evd := range ed.Value {
		vfqn := fqn + "." + evd.GetName()
		if err := addToPool(r, pool, errs, vfqn, evd); err != nil {
			return err
		}
	}
	return nil
}

func addServiceToPool(r *parseResult, pool map[string]proto.Message, errs *errorHandler, prefix string, sd *descriptorpb.ServiceDescriptorProto) error {
	fqn := prefix + sd.GetName()
	if err := addToPool(r, pool, errs, fqn, sd); err != nil {
		return err
	}
	for _, mtd := range sd.Method {
		mfqn := fqn + "." + mtd.GetName()
		if err := addToPool(r, pool, errs, mfqn, mtd); err != nil {
			return err
		}
	}
	return nil
}

func addToPool(r *parseResult, pool map[string]proto.Message, errs *errorHandler, fqn string, dsc proto.Message) error {
	if d, ok := pool[fqn]; ok {
		node := r.nodes[dsc]
		if err := errs.handleErrorWithPos(node.Start(), "duplicate symbol %s: already defined as %s", fqn, descriptorType(d)); err != nil {
			return err
		}
	}
	pool[fqn] = dsc
	return nil
}

func descriptorType(m proto.Message) string {
	switch m := m.(type) {
	case *descriptorpb.DescriptorProto:
		return "message"
	case *descriptorpb.DescriptorProto_ExtensionRange:
		return "extension range"
	case *descriptorpb.FieldDescriptorProto:
		if m.GetExtendee() == "" {
			return "field"
		} else {
			return "extension"
		}
	case *descriptorpb.EnumDescriptorProto:
		return "enum"
	case *descriptorpb.EnumValueDescriptorProto:
		return "enum value"
	case *descriptorpb.ServiceDescriptorProto:
		return "service"
	case *descriptorpb.MethodDescriptorProto:
		return "method"
	case *descriptorpb.FileDescriptorProto:
		return "file"
	default:
		// shouldn't be possible
		return fmt.Sprintf("%T", m)
	}
}

func (l *linker) resolveReferences() error {
	for _, filename := range l.filenames {
		r := l.files[filename]
		fd := r.fd
		prefix := fd.GetPackage()
		scopes := []scope{fileScope(fd, l)}
		if prefix != "" {
			prefix += "."
		}
		if fd.Options != nil {
			if err := l.resolveOptions(r, fd, "file", fd.GetName(), proto.MessageName(fd.Options), fd.Options.UninterpretedOption, scopes); err != nil {
				return err
			}
		}
		for _, md := range fd.MessageType {
			if err := l.resolveMessageTypes(r, fd, prefix, md, scopes); err != nil {
				return err
			}
		}
		for _, fld := range fd.Extension {
			if err := l.resolveFieldTypes(r, fd, prefix, fld, scopes); err != nil {
				return err
			}
		}
		for _, ed := range fd.EnumType {
			if err := l.resolveEnumTypes(r, fd, prefix, ed, scopes); err != nil {
				return err
			}
		}
		for _, sd := range fd.Service {
			if err := l.resolveServiceTypes(r, fd, prefix, sd, scopes); err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *linker) resolveEnumTypes(r *parseResult, fd *descriptorpb.FileDescriptorProto, prefix string, ed *descriptorpb.EnumDescriptorProto, scopes []scope) error {
	enumFqn := prefix + ed.GetName()
	if ed.Options != nil {
		if err := l.resolveOptions(r, fd, "enum", enumFqn, proto.MessageName(ed.Options), ed.Options.UninterpretedOption, scopes); err != nil {
			return err
		}
	}
	for _, evd := range ed.Value {
		if evd.Options != nil {
			evFqn := enumFqn + "." + evd.GetName()
			if err := l.resolveOptions(r, fd, "enum value", evFqn, proto.MessageName(evd.Options), evd.Options.UninterpretedOption, scopes); err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *linker) resolveMessageTypes(r *parseResult, fd *descriptorpb.FileDescriptorProto, prefix string, md *descriptorpb.DescriptorProto, scopes []scope) error {
	fqn := prefix + md.GetName()
	scope := messageScope(fqn, isProto3(fd), l.descriptorPool[fd])
	scopes = append(scopes, scope)
	prefix = fqn + "."

	if md.Options != nil {
		if err := l.resolveOptions(r, fd, "message", fqn, proto.MessageName(md.Options), md.Options.UninterpretedOption, scopes); err != nil {
			return err
		}
	}

	for _, nmd := range md.NestedType {
		if err := l.resolveMessageTypes(r, fd, prefix, nmd, scopes); err != nil {
			return err
		}
	}
	for _, ned := range md.EnumType {
		if err := l.resolveEnumTypes(r, fd, prefix, ned, scopes); err != nil {
			return err
		}
	}
	for _, fld := range md.Field {
		if err := l.resolveFieldTypes(r, fd, prefix, fld, scopes); err != nil {
			return err
		}
	}
	for _, fld := range md.Extension {
		if err := l.resolveFieldTypes(r, fd, prefix, fld, scopes); err != nil {
			return err
		}
	}
	for _, er := range md.ExtensionRange {
		if er.Options != nil {
			erName := fmt.Sprintf("%s:%d-%d", fqn, er.GetStart(), er.GetEnd()-1)
			if err := l.resolveOptions(r, fd, "extension range", erName, proto.MessageName(er.Options), er.Options.UninterpretedOption, scopes); err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *linker) resolveFieldTypes(r *parseResult, fd *descriptorpb.FileDescriptorProto, prefix string, fld *descriptorpb.FieldDescriptorProto, scopes []scope) error {
	thisName := prefix + fld.GetName()
	scope := fmt.Sprintf("field %s", thisName)
	node := r.getFieldNode(fld)
	elemType := "field"
	if fld.GetExtendee() != "" {
		elemType = "extension"
		fqn, dsc, _ := l.resolve(fd, fld.GetExtendee(), isMessage, scopes)
		if dsc == nil {
			return l.errs.handleErrorWithPos(node.FieldExtendee().Start(), "unknown extendee type %s", fld.GetExtendee())
		}
		extd, ok := dsc.(*descriptorpb.DescriptorProto)
		if !ok {
			otherType := descriptorType(dsc)
			return l.errs.handleErrorWithPos(node.FieldExtendee().Start(), "extendee is invalid: %s is a %s, not a message", fqn, otherType)
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
			if err := l.errs.handleErrorWithPos(node.FieldTag().Start(), "%s: tag %d is not in valid range for extended type %s", scope, tag, fqn); err != nil {
				return err
			}
		} else {
			// make sure tag is not a duplicate
			usedExtTags := l.extensions[fqn]
			if usedExtTags == nil {
				usedExtTags = map[int32]string{}
				l.extensions[fqn] = usedExtTags
			}
			if other := usedExtTags[fld.GetNumber()]; other != "" {
				if err := l.errs.handleErrorWithPos(node.FieldTag().Start(), "%s: duplicate extension: %s and %s are both using tag %d", scope, other, thisName, fld.GetNumber()); err != nil {
					return err
				}
			} else {
				usedExtTags[fld.GetNumber()] = thisName
			}
		}
	}

	if fld.Options != nil {
		if err := l.resolveOptions(r, fd, elemType, thisName, proto.MessageName(fld.Options), fld.Options.UninterpretedOption, scopes); err != nil {
			return err
		}
	}

	if fld.GetTypeName() == "" {
		// scalar type; no further resolution required
		return nil
	}

	fqn, dsc, proto3 := l.resolve(fd, fld.GetTypeName(), isType, scopes)
	if dsc == nil {
		return l.errs.handleErrorWithPos(node.FieldType().Start(), "%s: unknown type %s", scope, fld.GetTypeName())
	}
	switch dsc := dsc.(type) {
	case *descriptorpb.DescriptorProto:
		fld.TypeName = proto.String("." + fqn)
		// if type was tentatively unset, we now know it's actually a message
		if fld.Type == nil {
			fld.Type = descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum()
		}
	case *descriptorpb.EnumDescriptorProto:
		if fld.GetExtendee() == "" && isProto3(fd) && !proto3 {
			// fields in a proto3 message cannot refer to proto2 enums
			return l.errs.handleErrorWithPos(node.FieldType().Start(), "%s: cannot use proto2 enum %s in a proto3 message", scope, fld.GetTypeName())
		}
		fld.TypeName = proto.String("." + fqn)
		// the type was tentatively unset, but now we know it's actually an enum
		fld.Type = descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum()
	default:
		otherType := descriptorType(dsc)
		return l.errs.handleErrorWithPos(node.FieldType().Start(), "%s: invalid type: %s is a %s, not a message or enum", scope, fqn, otherType)
	}
	return nil
}

func (l *linker) resolveServiceTypes(r *parseResult, fd *descriptorpb.FileDescriptorProto, prefix string, sd *descriptorpb.ServiceDescriptorProto, scopes []scope) error {
	thisName := prefix + sd.GetName()
	if sd.Options != nil {
		if err := l.resolveOptions(r, fd, "service", thisName, proto.MessageName(sd.Options), sd.Options.UninterpretedOption, scopes); err != nil {
			return err
		}
	}

	for _, mtd := range sd.Method {
		if mtd.Options != nil {
			if err := l.resolveOptions(r, fd, "method", thisName+"."+mtd.GetName(), proto.MessageName(mtd.Options), mtd.Options.UninterpretedOption, scopes); err != nil {
				return err
			}
		}
		scope := fmt.Sprintf("method %s.%s", thisName, mtd.GetName())
		node := r.getMethodNode(mtd)
		fqn, dsc, _ := l.resolve(fd, mtd.GetInputType(), isMessage, scopes)
		if dsc == nil {
			if err := l.errs.handleErrorWithPos(node.GetInputType().Start(), "%s: unknown request type %s", scope, mtd.GetInputType()); err != nil {
				return err
			}
		} else if _, ok := dsc.(*descriptorpb.DescriptorProto); !ok {
			otherType := descriptorType(dsc)
			if err := l.errs.handleErrorWithPos(node.GetInputType().Start(), "%s: invalid request type: %s is a %s, not a message", scope, fqn, otherType); err != nil {
				return err
			}
		} else {
			mtd.InputType = proto.String("." + fqn)
		}

		fqn, dsc, _ = l.resolve(fd, mtd.GetOutputType(), isMessage, scopes)
		if dsc == nil {
			if err := l.errs.handleErrorWithPos(node.GetOutputType().Start(), "%s: unknown response type %s", scope, mtd.GetOutputType()); err != nil {
				return err
			}
		} else if _, ok := dsc.(*descriptorpb.DescriptorProto); !ok {
			otherType := descriptorType(dsc)
			if err := l.errs.handleErrorWithPos(node.GetOutputType().Start(), "%s: invalid response type: %s is a %s, not a message", scope, fqn, otherType); err != nil {
				return err
			}
		} else {
			mtd.OutputType = proto.String("." + fqn)
		}
	}
	return nil
}

func (l *linker) resolveOptions(r *parseResult, fd *descriptorpb.FileDescriptorProto, elemType, elemName, optType string, opts []*descriptorpb.UninterpretedOption, scopes []scope) error {
	var scope string
	if elemType != "file" {
		scope = fmt.Sprintf("%s %s: ", elemType, elemName)
	}
opts:
	for _, opt := range opts {
		for _, nm := range opt.Name {
			if nm.GetIsExtension() {
				node := r.getOptionNamePartNode(nm)
				fqn, dsc, _ := l.resolve(fd, nm.GetNamePart(), isField, scopes)
				if dsc == nil {
					if err := l.errs.handleErrorWithPos(node.Start(), "%sunknown extension %s", scope, nm.GetNamePart()); err != nil {
						return err
					}
					continue opts
				}
				if ext, ok := dsc.(*descriptorpb.FieldDescriptorProto); !ok {
					otherType := descriptorType(dsc)
					if err := l.errs.handleErrorWithPos(node.Start(), "%sinvalid extension: %s is a %s, not an extension", scope, nm.GetNamePart(), otherType); err != nil {
						return err
					}
					continue opts
				} else if ext.GetExtendee() == "" {
					if err := l.errs.handleErrorWithPos(node.Start(), "%sinvalid extension: %s is a field but not an extension", scope, nm.GetNamePart()); err != nil {
						return err
					}
					continue opts
				}
				nm.NamePart = proto.String("." + fqn)
			}
		}
	}
	return nil
}

func (l *linker) resolve(fd *descriptorpb.FileDescriptorProto, name string, allowed func(proto.Message) bool, scopes []scope) (fqn string, element proto.Message, proto3 bool) {
	if strings.HasPrefix(name, ".") {
		// already fully-qualified
		d, proto3 := l.findSymbol(fd, name[1:], false, map[*descriptorpb.FileDescriptorProto]struct{}{})
		if d != nil {
			return name[1:], d, proto3
		}
	} else {
		// unqualified, so we look in the enclosing (last) scope first and move
		// towards outermost (first) scope, trying to resolve the symbol
		var bestGuess proto.Message
		var bestGuessFqn string
		var bestGuessProto3 bool
		for i := len(scopes) - 1; i >= 0; i-- {
			fqn, d, proto3 := scopes[i](name)
			if d != nil {
				if allowed(d) {
					return fqn, d, proto3
				} else if bestGuess == nil {
					bestGuess = d
					bestGuessFqn = fqn
					bestGuessProto3 = proto3
				}
			}
		}
		// we return best guess, even though it was not an allowed kind of
		// descriptor, so caller can print a better error message (e.g.
		// indicating that the name was found but that it's the wrong type)
		return bestGuessFqn, bestGuess, bestGuessProto3
	}
	return "", nil, false
}

func isField(m proto.Message) bool {
	_, ok := m.(*descriptorpb.FieldDescriptorProto)
	return ok
}

func isMessage(m proto.Message) bool {
	_, ok := m.(*descriptorpb.DescriptorProto)
	return ok
}

func isType(m proto.Message) bool {
	switch m.(type) {
	case *descriptorpb.DescriptorProto, *descriptorpb.EnumDescriptorProto:
		return true
	}
	return false
}

// scope represents a lexical scope in a proto file in which messages and enums
// can be declared.
type scope func(symbol string) (fqn string, element proto.Message, proto3 bool)

func fileScope(fd *descriptorpb.FileDescriptorProto, l *linker) scope {
	// we search symbols in this file, but also symbols in other files that have
	// the same package as this file or a "parent" package (in protobuf,
	// packages are a hierarchy like C++ namespaces)
	prefixes := internal.CreatePrefixList(fd.GetPackage())
	return func(name string) (string, proto.Message, bool) {
		for _, prefix := range prefixes {
			var n string
			if prefix == "" {
				n = name
			} else {
				n = prefix + "." + name
			}
			d, proto3 := l.findSymbol(fd, n, false, map[*descriptorpb.FileDescriptorProto]struct{}{})
			if d != nil {
				return n, d, proto3
			}
		}
		return "", nil, false
	}
}

func messageScope(messageName string, proto3 bool, filePool map[string]proto.Message) scope {
	return func(name string) (string, proto.Message, bool) {
		n := messageName + "." + name
		if d, ok := filePool[n]; ok {
			return n, d, proto3
		}
		return "", nil, false
	}
}

// TODO: consolidate with findElement
func (l *linker) findSymbol(fd *descriptorpb.FileDescriptorProto, name string, public bool, checked map[*descriptorpb.FileDescriptorProto]struct{}) (element proto.Message, proto3 bool) {
	fd, d := l.findElement(fd, name, public, checked)
	return d, isProto3(fd)
}

func isProto3(fd *descriptorpb.FileDescriptorProto) bool {
	return fd.GetSyntax() == "proto3"
}

func (l *linker) createdLinkedDescriptors() (map[string]*desc.FileDescriptor, error) {
	names := make([]string, 0, len(l.files))
	for name := range l.files {
		names = append(names, name)
	}
	sort.Strings(names)
	linked := map[string]*desc.FileDescriptor{}
	for _, name := range names {
		if _, err := l.linkFile(name, nil, nil, linked); err != nil {
			return nil, err
		}
	}
	return linked, nil
}

func (l *linker) linkFile(name string, rootImportLoc *SourcePos, seen []string, linked map[string]*desc.FileDescriptor) (*desc.FileDescriptor, error) {
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
				_, _ = fmt.Fprintf(&msg, "%q", s)
			}
			_, _ = fmt.Fprintf(&msg, " -> %q", name)
			return nil, ErrorWithSourcePos{
				Underlying: fmt.Errorf("cycle found in imports: %s", msg.String()),
				Pos:        rootImportLoc,
			}
		}
	}
	seen = append(seen, name)

	if lfd, ok := linked[name]; ok {
		// already linked
		return lfd, nil
	}
	r := l.files[name]
	if r == nil {
		importer := seen[len(seen)-2] // len-1 is *this* file, before that is the one that imported it
		return nil, fmt.Errorf("no descriptor found for %q, imported by %q", name, importer)
	}
	var deps []*desc.FileDescriptor
	if rootImportLoc == nil {
		// try to find a source location for this "root" import
		decl := r.getFileNode(r.fd)
		fnode, ok := decl.(*ast.FileNode)
		if ok {
			for _, decl := range fnode.Decls {
				if dep, ok := decl.(*ast.ImportNode); ok {
					ldep, err := l.linkFile(dep.Name.AsString(), dep.Name.Start(), seen, linked)
					if err != nil {
						return nil, err
					}
					deps = append(deps, ldep)
				}
			}
		} else {
			// no AST? just use the descriptor
			for _, dep := range r.fd.Dependency {
				ldep, err := l.linkFile(dep, decl.Start(), seen, linked)
				if err != nil {
					return nil, err
				}
				deps = append(deps, ldep)
			}
		}
	} else {
		// we can just use the descriptor since we don't need source location
		// (we'll just attribute any import cycles found to the "root" import)
		for _, dep := range r.fd.Dependency {
			ldep, err := l.linkFile(dep, rootImportLoc, seen, linked)
			if err != nil {
				return nil, err
			}
			deps = append(deps, ldep)
		}
	}
	lfd, err := desc.CreateFileDescriptor(r.fd, deps...)
	if err != nil {
		return nil, fmt.Errorf("error linking %q: %s", name, err)
	}
	linked[name] = lfd
	return lfd, nil
}

func (l *linker) checkMessageSets(res *parseResult) error {
	prefix := res.fd.GetPackage()
	if prefix != "" {
		prefix = "."
	}
	for _, fld := range res.fd.GetExtension() {
		if err := l.checkMessageSetsExtension(fld, prefix+fld.GetName(), res); err != nil {
			return err
		}
	}
	for _, md := range res.fd.GetMessageType() {
		if err := l.checkMessageSetsMessage(md, prefix+md.GetName(), res); err != nil {
			return err
		}
	}
	return nil
}

func (l *linker) checkMessageSetsMessage(md *descriptorpb.DescriptorProto, fqn string, res *parseResult) error {
	if md.Options.GetMessageSetWireFormat() {
		// Message set: must have at least one extension range and zero fields
		if len(md.Field) > 0 {
			pos := res.nodes[md].Start()
			return errorWithPos(pos, "message %q is a message set so should have only extensions, not regular fields", md.GetName())
		}
		if len(md.ExtensionRange) == 0 {
			pos := res.nodes[md].Start()
			return errorWithPos(pos, "message %q is a message set so should have only extensions, but has no extension ranges", md.GetName())
		}
	}

	for _, fld := range md.GetExtension() {
		if err := l.checkMessageSetsExtension(fld, fqn+"."+fld.GetName(), res); err != nil {
			return err
		}
	}
	for _, nmd := range md.GetNestedType() {
		if err := l.checkMessageSetsMessage(nmd, fqn+"."+md.GetName(), res); err != nil {
			return err
		}
	}
	return nil
}

func (l *linker) checkMessageSetsExtension(fld *descriptorpb.FieldDescriptorProto, fqn string, res *parseResult) error {
	// NB: This is kind of gross that we don't enforce this in validateBasic(). But it would
	// require doing some minimal linking there (to identify the extendee and locate its
	// descriptor). To keep the code simpler, we just wait until things are fully linked.

	// In validateBasic() we just made sure these were within bounds for any message. But
	// now that things are linked, we can check if the extendee is messageset wire format
	// and, if not, enforce tighter limit.

	md := l.findMessageType(res.fd, fld.GetExtendee())
	if md.Options().(*descriptorpb.MessageOptions).GetMessageSetWireFormat() {
		// Message set: extensions must be optional messages
		if fld.GetLabel() != descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL {
			pos := res.nodes[fld].(ast.FieldDeclNode).FieldLabel().Start()
			return errorWithPos(pos, "field %q extends message set so must be an optional message", fld.GetName())
		}
		if fld.GetType() != descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
			pos := res.nodes[fld].(ast.FieldDeclNode).FieldType().Start()
			return errorWithPos(pos, "field %q extends message set so must be an optional message", fld.GetName())
		}
	} else if fld.GetNumber() > internal.MaxNormalTag {
		// NOT a message set, which means more restrictive range of tags
		pos := res.nodes[fld].(ast.FieldDeclNode).FieldTag().Start()
		return errorWithPos(pos, "tag number %d is higher than max allowed tag number (%d)", fld.GetNumber(), internal.MaxNormalTag)
	}
	return nil
}
