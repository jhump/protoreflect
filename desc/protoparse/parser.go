package protoparse

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/golang/protobuf/proto"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"

	"github.com/jhump/protoreflect/desc"
)

//go:generate goyacc -o proto.y.go -p proto proto.y

const (
	maxTag = 536870911 // 2^29 - 1

	specialReservedStart = 19000
	specialReservedEnd   = 19999
)

func init() {
	protoErrorVerbose = true
}

// FileAccessor is an abstraction for opening proto source files. It takes the
// name of the file to open and returns either the input reader or an error.
type FileAccessor func(filename string) (io.ReadCloser, error)

// Parser parses proto source into descriptors.
type Parser struct {
	// The paths used to search for dependencies that are referenced in import
	// statements in proto source files. If no import paths are provided then
	// "." (current directory) is assumed to be the only import path.
	ImportPaths []string

	// If true, the supplied file names/paths need not necessarily match how the
	// files are referenced in import statements. The parser will attempt to
	// match import statements to supplied paths, "guessing" the import paths
	// for the files. Note that this inference is not perfect and link errors
	// could result. It works best when all proto files are organized such that
	// a single import path can be inferred (e.g. all files under a single tree
	// with import statements all being relative to the root of this tree).
	InferImportPaths bool

	// Used to create a reader for a given filename, when loading proto source
	// file contents. If unset, os.Open is used, and relative paths are thus
	// relative to the process's current working directory.
	Accessor FileAccessor
}

// ParseFiles parses the named files into descriptors. The returned slice has
// the same number of entries as the give filenames, in the same order. So the
// first returned descriptor corresponds to the first given name, and so on.
//
// All dependencies for all specified files (including transitive dependencies)
// must be accessible via the parser's Accessor or a link error will occur. The
// exception to this rule is that files can import standard "google/protobuf/*.proto"
// files without needing to supply sources for these files. Like protoc, this
// parser has a built-in version of these files it can use if they aren't
// explicitly supplied.
func (p Parser) ParseFiles(filenames ...string) ([]*desc.FileDescriptor, error) {
	accessor := p.Accessor
	if accessor == nil {
		accessor = func(name string) (io.ReadCloser, error) {
			return os.Open(name)
		}
	}
	paths := p.ImportPaths
	if len(paths) > 0 {
		acc := accessor
		accessor = func(name string) (io.ReadCloser, error) {
			var ret error
			for _, path := range paths {
				f, err := acc(filepath.Join(path, name))
				if err != nil && ret == nil {
					ret = err
					continue
				}
				return f, nil
			}
			return nil, ret
		}
	}

	protos := map[string]*parseResult{}
	err := parseProtoFiles(accessor, filenames, protos)
	if err != nil {
		return nil, err
	}
	if p.InferImportPaths {
		protos = fixupFilenames(protos)
	}
	linkedProtos, err := newLinker(protos).linkFiles()
	if err != nil {
		return nil, err
	}
	fds := make([]*desc.FileDescriptor, len(filenames))
	for i, name := range filenames {
		fds[i] = linkedProtos[name]
	}
	return fds, nil
}

func fixupFilenames(protos map[string]*parseResult) map[string]*parseResult {
	// In the event that the given filenames (keys in the supplied map) do not
	// match the actual paths used in 'import' statements in the files, we try
	// to revise names in the protos so that they will match and be linkable.
	revisedProtos := map[string]*parseResult{}

	protoPaths := map[string]struct{}{}
	// TODO: this is O(n^2) but could likely be O(n) with a clever data structure (prefix tree that is indexed backwards?)
	importCandidates := map[string]map[string]struct{}{}
	candidatesAvailable := map[string]struct{}{}
	for name := range protos {
		candidatesAvailable[name] = struct{}{}
		for _, f := range protos {
			for _, imp := range f.fd.Dependency {
				if strings.HasSuffix(name, imp) {
					candidates := importCandidates[imp]
					if candidates == nil {
						candidates = map[string]struct{}{}
						importCandidates[imp] = candidates
					}
					candidates[name] = struct{}{}
				}
			}
		}
	}
	for imp, candidates := range importCandidates {
		// if we found multiple possible candidates, use the one that is an exact match
		// if it exists, and otherwise, guess that it's the shortest path (fewest elements)
		var best string
		for c := range candidates {
			if _, ok := candidatesAvailable[c]; !ok {
				// already used this candidate and re-written its filename accordingly
				continue
			}
			if c == imp {
				// exact match!
				best = c
				break
			}
			if best == "" {
				best = c
			} else {
				// HACK: we can't actually tell which files is supposed to match
				// this import, so arbitrarily pick the "shorter" one (fewest
				// path elements) or, on a tie, the lexically earlier one
				minLen := strings.Count(best, string(filepath.Separator))
				cLen := strings.Count(c, string(filepath.Separator))
				if cLen < minLen || (cLen == minLen && c < best) {
					best = c
				}
			}
		}
		if best != "" {
			prefix := best[:len(best)-len(imp)]
			if len(prefix) > 0 {
				protoPaths[prefix] = struct{}{}
			}
			f := protos[best]
			f.fd.Name = proto.String(imp)
			revisedProtos[imp] = f
			delete(candidatesAvailable, best)
		}
	}

	if len(candidatesAvailable) == 0 {
		return revisedProtos
	}

	if len(protoPaths) == 0 {
		for c := range candidatesAvailable {
			revisedProtos[c] = protos[c]
		}
		return revisedProtos
	}

	// Any remaining candidates are entry-points (not imported by others), so
	// the best bet to "fixing" their file name is to see if they're in one of
	// the proto paths we found, and if so strip that prefix.
	protoPathStrs := make([]string, len(protoPaths))
	i := 0
	for p := range protoPaths {
		protoPathStrs[i] = p
		i++
	}
	sort.Strings(protoPathStrs)
	// we look at paths in reverse order, so we'll use a longer proto path if
	// there is more than one match
	for c := range candidatesAvailable {
		var imp string
		for i := len(protoPathStrs) - 1; i >= 0; i-- {
			p := protoPathStrs[i]
			if strings.HasPrefix(c, p) {
				imp = c[len(p):]
				break
			}
		}
		if imp != "" {
			f := protos[c]
			f.fd.Name = proto.String(imp)
			revisedProtos[imp] = f
		} else {
			revisedProtos[c] = protos[c]
		}
	}

	return revisedProtos
}

func parseProtoFiles(acc FileAccessor, filenames []string, parsed map[string]*parseResult) error {
	for _, name := range filenames {
		if _, ok := parsed[name]; ok {
			continue
		}
		in, err := acc(name)
		if err != nil {
			if d, ok := standardImports[name]; ok {
				parsed[name] = &parseResult{fd: d}
				continue
			}
			return err
		}
		func() {
			defer in.Close()
			parsed[name], err = parseProto(name, in)
		}()
		if err != nil {
			return err
		}
		err = parseProtoFiles(acc, parsed[name].fd.Dependency, parsed)
		if err != nil {
			return fmt.Errorf("failed to load imports for %q: %s", name, err)
		}
	}
	return nil
}

func parseProtoFile(filename string) (*parseResult, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parseProto(filename, f)
}

type parseResult struct {
	// the parsed file descriptor
	fd *dpb.FileDescriptorProto

	// a map of elements in the descriptor to nodes in the AST
	// (for extracting position information when validating the descriptor)
	nodes map[interface{}]node

	// a map of aggregate option values to their ASTs
	aggregates map[string][]*aggregateEntryNode
}

func parseProto(filename string, r io.Reader) (*parseResult, error) {
	lx := newLexer(r)
	lx.filename = filename
	protoParse(lx)
	if lx.err != nil {
		if _, ok := lx.err.(ErrorWithSourcePos); ok {
			return nil, lx.err
		} else {
			return nil, ErrorWithSourcePos{Pos: lx.prev(), Underlying: lx.err}
		}
	}

	if res, err := createParseResult(filename, lx.res); err != nil {
		return nil, err
	} else if err := basicValidate(res); err != nil {
		return nil, err
	} else {
		return res, nil
	}
}

func createParseResult(filename string, file *fileNode) (*parseResult, error) {
	res := &parseResult{
		nodes:      map[interface{}]node{},
		aggregates: map[string][]*aggregateEntryNode{},
	}
	var err error
	res.fd, err = res.asFileDescriptor(filename, file)
	return res, err
}

func (r *parseResult) asFileDescriptor(filename string, file *fileNode) (*dpb.FileDescriptorProto, error) {
	fd := &dpb.FileDescriptorProto{Name: proto.String(filename)}
	r.nodes[fd] = file

	isProto3 := false
	if file.syntax != nil {
		fd.Syntax = proto.String(file.syntax.syntax.val)
		isProto3 = file.syntax.syntax.val == "proto3"
	}

	for _, decl := range file.decls {
		if decl.enum != nil {
			fd.EnumType = append(fd.EnumType, r.asEnumDescriptor(decl.enum))
		} else if decl.extend != nil {
			r.addExtensions(decl.extend, &fd.Extension, &fd.MessageType, isProto3)
		} else if decl.imp != nil {
			file.imports = append(file.imports, decl.imp)
			index := len(fd.Dependency)
			fd.Dependency = append(fd.Dependency, decl.imp.name.val)
			if decl.imp.public {
				fd.PublicDependency = append(fd.PublicDependency, int32(index))
			} else if decl.imp.weak {
				fd.WeakDependency = append(fd.WeakDependency, int32(index))
			}
		} else if decl.message != nil {
			fd.MessageType = append(fd.MessageType, r.asMessageDescriptor(decl.message, isProto3))
		} else if decl.option != nil {
			if fd.Options == nil {
				fd.Options = &dpb.FileOptions{}
			}
			fd.Options.UninterpretedOption = append(fd.Options.UninterpretedOption, r.asUninterpretedOption(decl.option))
		} else if decl.service != nil {
			fd.Service = append(fd.Service, r.asServiceDescriptor(decl.service))
		} else if decl.pkg != nil {
			if fd.Package != nil {
				return nil, ErrorWithSourcePos{Pos: decl.pkg.start(), Underlying: errors.New("files should have only one package declaration")}
			}
			file.pkg = decl.pkg
			fd.Package = proto.String(decl.pkg.name.val)
		}
	}
	return fd, nil
}

func (r *parseResult) asUninterpretedOptions(nodes []*optionNode) []*dpb.UninterpretedOption {
	opts := make([]*dpb.UninterpretedOption, len(nodes))
	for i, n := range nodes {
		opts[i] = r.asUninterpretedOption(n)
	}
	return opts
}

func (r *parseResult) asUninterpretedOption(node *optionNode) *dpb.UninterpretedOption {
	opt := &dpb.UninterpretedOption{Name: r.asUninterpretedOptionName(node.name.parts)}
	r.nodes[opt] = node

	switch val := node.val.value().(type) {
	case bool:
		if val {
			opt.IdentifierValue = proto.String("true")
		} else {
			opt.IdentifierValue = proto.String("false")
		}
	case int64:
		opt.NegativeIntValue = proto.Int64(val)
	case uint64:
		opt.PositiveIntValue = proto.Uint64(val)
	case float64:
		opt.DoubleValue = proto.Float64(val)
	case string:
		opt.StringValue = []byte(val)
	case identifier:
		opt.IdentifierValue = proto.String(string(val))
	case []*aggregateEntryNode:
		var buf bytes.Buffer
		aggToString(val, &buf)
		aggStr := buf.String()
		opt.AggregateValue = proto.String(aggStr)
		r.aggregates[aggStr] = val
	}
	return opt
}

func (r *parseResult) asUninterpretedOptionName(parts []*optionNamePartNode) []*dpb.UninterpretedOption_NamePart {
	ret := make([]*dpb.UninterpretedOption_NamePart, len(parts))
	for i, part := range parts {
		txt := part.text.val
		if !part.isExtension {
			txt = part.text.val[part.offset : part.offset+part.length]
		}
		np := &dpb.UninterpretedOption_NamePart{
			NamePart:    proto.String(txt),
			IsExtension: proto.Bool(part.isExtension),
		}
		r.nodes[np] = part
		ret[i] = np
	}
	return ret
}

func (r *parseResult) addExtensions(ext *extendNode, flds *[]*dpb.FieldDescriptorProto, msgs *[]*dpb.DescriptorProto, isProto3 bool) {
	extendee := ext.extendee.val
	for _, decl := range ext.decls {
		if decl.field != nil {
			fd := r.asFieldDescriptor(decl.field)
			fd.Extendee = proto.String(extendee)
			*flds = append(*flds, fd)
		} else if decl.group != nil {
			fd, md := r.asGroupDescriptors(decl.group, isProto3)
			fd.Extendee = proto.String(extendee)
			*flds = append(*flds, fd)
			*msgs = append(*msgs, md)
		}
	}
}

func asLabel(lbl *labelNode) *dpb.FieldDescriptorProto_Label {
	if lbl == nil {
		return nil
	}
	switch {
	case lbl.repeated:
		return dpb.FieldDescriptorProto_LABEL_REPEATED.Enum()
	case lbl.required:
		return dpb.FieldDescriptorProto_LABEL_REQUIRED.Enum()
	default:
		return dpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()
	}
}

func (r *parseResult) asFieldDescriptor(node *fieldNode) *dpb.FieldDescriptorProto {
	fd := newFieldDescriptor(node.name.val, node.fldType.val, int32(node.tag.val), asLabel(node.label))
	r.nodes[fd] = node
	if len(node.options) > 0 {
		fd.Options = &dpb.FieldOptions{UninterpretedOption: r.asUninterpretedOptions(node.options)}
	}
	return fd
}

func newFieldDescriptor(name string, fieldType string, tag int32, lbl *dpb.FieldDescriptorProto_Label) *dpb.FieldDescriptorProto {
	fd := &dpb.FieldDescriptorProto{
		Name:     proto.String(name),
		JsonName: proto.String(jsonName(name)),
		Number:   proto.Int32(tag),
		Label:    lbl,
	}
	switch fieldType {
	case "double":
		fd.Type = dpb.FieldDescriptorProto_TYPE_DOUBLE.Enum()
	case "float":
		fd.Type = dpb.FieldDescriptorProto_TYPE_FLOAT.Enum()
	case "int32":
		fd.Type = dpb.FieldDescriptorProto_TYPE_INT32.Enum()
	case "int64":
		fd.Type = dpb.FieldDescriptorProto_TYPE_INT64.Enum()
	case "uint32":
		fd.Type = dpb.FieldDescriptorProto_TYPE_UINT32.Enum()
	case "uint64":
		fd.Type = dpb.FieldDescriptorProto_TYPE_UINT64.Enum()
	case "sint32":
		fd.Type = dpb.FieldDescriptorProto_TYPE_SINT32.Enum()
	case "sint64":
		fd.Type = dpb.FieldDescriptorProto_TYPE_SINT64.Enum()
	case "fixed32":
		fd.Type = dpb.FieldDescriptorProto_TYPE_FIXED32.Enum()
	case "fixed64":
		fd.Type = dpb.FieldDescriptorProto_TYPE_FIXED64.Enum()
	case "sfixed32":
		fd.Type = dpb.FieldDescriptorProto_TYPE_SFIXED32.Enum()
	case "sfixed64":
		fd.Type = dpb.FieldDescriptorProto_TYPE_SFIXED64.Enum()
	case "bool":
		fd.Type = dpb.FieldDescriptorProto_TYPE_BOOL.Enum()
	case "string":
		fd.Type = dpb.FieldDescriptorProto_TYPE_STRING.Enum()
	case "bytes":
		fd.Type = dpb.FieldDescriptorProto_TYPE_BYTES.Enum()
	default:
		// NB: we don't have enough info to determine whether this is an enum or a message type,
		// so we'll change it to enum later once we can ascertain if it's an enum reference
		fd.Type = dpb.FieldDescriptorProto_TYPE_MESSAGE.Enum()
		fd.TypeName = proto.String(fieldType)
	}
	return fd
}

func (r *parseResult) asGroupDescriptors(group *groupNode, isProto3 bool) (*dpb.FieldDescriptorProto, *dpb.DescriptorProto) {
	fieldName := strings.ToLower(group.name.val)
	fd := &dpb.FieldDescriptorProto{
		Name:     proto.String(fieldName),
		JsonName: proto.String(jsonName(fieldName)),
		Number:   proto.Int32(int32(group.tag.val)),
		Label:    asLabel(group.label),
		Type:     dpb.FieldDescriptorProto_TYPE_GROUP.Enum(),
		TypeName: proto.String(group.name.val),
	}
	r.nodes[fd] = group
	md := &dpb.DescriptorProto{Name: proto.String(group.name.val)}
	r.nodes[md] = group
	r.addMessageDecls(md, &group.reserved, group.decls, isProto3)
	return fd, md
}

func (r *parseResult) asMapDescriptors(mapField *mapFieldNode, isProto3 bool) (*dpb.FieldDescriptorProto, *dpb.DescriptorProto) {
	var lbl *dpb.FieldDescriptorProto_Label
	if !isProto3 {
		lbl = dpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()
	}
	keyFd := newFieldDescriptor("key", mapField.keyType.val, 1, lbl)
	r.nodes[keyFd] = mapField.keyField()
	valFd := newFieldDescriptor("value", mapField.valueType.val, 2, lbl)
	r.nodes[valFd] = mapField.valueField()
	entryName := initCap(jsonName(mapField.name.val)) + "Entry"
	fd := newFieldDescriptor(mapField.name.val, entryName, int32(mapField.tag.val), dpb.FieldDescriptorProto_LABEL_REPEATED.Enum())
	if len(mapField.options) > 0 {
		fd.Options = &dpb.FieldOptions{UninterpretedOption: r.asUninterpretedOptions(mapField.options)}
	}
	r.nodes[fd] = mapField
	md := &dpb.DescriptorProto{
		Name:    proto.String(entryName),
		Options: &dpb.MessageOptions{MapEntry: proto.Bool(true)},
		Field:   []*dpb.FieldDescriptorProto{keyFd, valFd},
	}
	r.nodes[md] = mapField
	return fd, md
}

func (r *parseResult) asExtensionRanges(node *extensionRangeNode) []*dpb.DescriptorProto_ExtensionRange {
	opts := r.asUninterpretedOptions(node.options)
	ers := make([]*dpb.DescriptorProto_ExtensionRange, len(node.ranges))
	for i, rng := range node.ranges {
		er := &dpb.DescriptorProto_ExtensionRange{
			Start: proto.Int32(int32(rng.st.val)),
			End:   proto.Int32(int32(rng.en.val + 1)),
		}
		if len(opts) > 0 {
			er.Options = &dpb.ExtensionRangeOptions{UninterpretedOption: opts}
		}
		r.nodes[er] = rng
		ers[i] = er
	}
	return ers
}

func (r *parseResult) asEnumValue(ev *enumValueNode) *dpb.EnumValueDescriptorProto {
	var num int32
	if ev.number != nil {
		num = int32(ev.number.val)
	} else {
		num = int32(ev.numberN.val)
	}
	evd := &dpb.EnumValueDescriptorProto{Name: proto.String(ev.name.val), Number: proto.Int32(num)}
	r.nodes[evd] = ev
	if len(ev.options) > 0 {
		evd.Options = &dpb.EnumValueOptions{UninterpretedOption: r.asUninterpretedOptions(ev.options)}
	}
	return evd
}

func (r *parseResult) asMethodDescriptor(node *methodNode) *dpb.MethodDescriptorProto {
	md := &dpb.MethodDescriptorProto{
		Name:       proto.String(node.name.val),
		InputType:  proto.String(node.input.msgType.val),
		OutputType: proto.String(node.output.msgType.val),
	}
	r.nodes[md] = node
	if node.input.stream {
		md.ClientStreaming = proto.Bool(true)
	}
	if node.output.stream {
		md.ServerStreaming = proto.Bool(true)
	}
	if len(node.options) > 0 {
		md.Options = &dpb.MethodOptions{UninterpretedOption: r.asUninterpretedOptions(node.options)}
	}
	return md
}

func (r *parseResult) asEnumDescriptor(en *enumNode) *dpb.EnumDescriptorProto {
	ed := &dpb.EnumDescriptorProto{Name: proto.String(en.name.val)}
	r.nodes[ed] = en
	for _, decl := range en.decls {
		if decl.option != nil {
			if ed.Options == nil {
				ed.Options = &dpb.EnumOptions{}
			}
			ed.Options.UninterpretedOption = append(ed.Options.UninterpretedOption, r.asUninterpretedOption(decl.option))
		} else if decl.value != nil {
			ed.Value = append(ed.Value, r.asEnumValue(decl.value))
		}
	}
	return ed
}

func (r *parseResult) asMessageDescriptor(node *messageNode, isProto3 bool) *dpb.DescriptorProto {
	msgd := &dpb.DescriptorProto{Name: proto.String(node.name.val)}
	r.nodes[msgd] = node
	r.addMessageDecls(msgd, &node.reserved, node.decls, isProto3)
	return msgd
}

func (r *parseResult) addMessageDecls(msgd *dpb.DescriptorProto, reservedNames *[]*stringLiteralNode, decls []*messageElement, isProto3 bool) {
	for _, decl := range decls {
		if decl.enum != nil {
			msgd.EnumType = append(msgd.EnumType, r.asEnumDescriptor(decl.enum))
		} else if decl.extend != nil {
			r.addExtensions(decl.extend, &msgd.Extension, &msgd.NestedType, isProto3)
		} else if decl.extensionRange != nil {
			msgd.ExtensionRange = append(msgd.ExtensionRange, r.asExtensionRanges(decl.extensionRange)...)
		} else if decl.field != nil {
			msgd.Field = append(msgd.Field, r.asFieldDescriptor(decl.field))
		} else if decl.mapField != nil {
			fd, md := r.asMapDescriptors(decl.mapField, isProto3)
			msgd.Field = append(msgd.Field, fd)
			msgd.NestedType = append(msgd.NestedType, md)
		} else if decl.group != nil {
			fd, md := r.asGroupDescriptors(decl.group, isProto3)
			msgd.Field = append(msgd.Field, fd)
			msgd.NestedType = append(msgd.NestedType, md)
		} else if decl.oneOf != nil {
			oodIndex := len(msgd.OneofDecl)
			ood := &dpb.OneofDescriptorProto{Name: proto.String(decl.oneOf.name.val)}
			r.nodes[ood] = decl.oneOf
			msgd.OneofDecl = append(msgd.OneofDecl, ood)
			for _, oodecl := range decl.oneOf.decls {
				if oodecl.option != nil {
					if ood.Options != nil {
						ood.Options = &dpb.OneofOptions{}
					}
					ood.Options.UninterpretedOption = append(ood.Options.UninterpretedOption, r.asUninterpretedOption(oodecl.option))
				} else if oodecl.field != nil {
					fd := r.asFieldDescriptor(oodecl.field)
					fd.OneofIndex = proto.Int32(int32(oodIndex))
					msgd.Field = append(msgd.Field, fd)
				}
			}
		} else if decl.option != nil {
			if msgd.Options == nil {
				msgd.Options = &dpb.MessageOptions{}
			}
			msgd.Options.UninterpretedOption = append(msgd.Options.UninterpretedOption, r.asUninterpretedOption(decl.option))
		} else if decl.nested != nil {
			msgd.NestedType = append(msgd.NestedType, r.asMessageDescriptor(decl.nested, isProto3))
		} else if decl.reserved != nil {
			for _, n := range decl.reserved.names {
				*reservedNames = append(*reservedNames, n)
				msgd.ReservedName = append(msgd.ReservedName, n.val)
			}
			for _, rng := range decl.reserved.ranges {
				msgd.ReservedRange = append(msgd.ReservedRange, r.asReservedRange(rng))
			}
		}
	}
}

func (r *parseResult) asReservedRange(rng *rangeNode) *dpb.DescriptorProto_ReservedRange {
	rr := &dpb.DescriptorProto_ReservedRange{
		Start: proto.Int32(int32(rng.st.val)),
		End:   proto.Int32(int32(rng.en.val + 1)),
	}
	r.nodes[rr] = rng
	return rr
}

func (r *parseResult) asServiceDescriptor(svc *serviceNode) *dpb.ServiceDescriptorProto {
	sd := &dpb.ServiceDescriptorProto{Name: proto.String(svc.name.val)}
	r.nodes[sd] = svc
	for _, decl := range svc.decls {
		if decl.option != nil {
			if sd.Options == nil {
				sd.Options = &dpb.ServiceOptions{}
			}
			sd.Options.UninterpretedOption = append(sd.Options.UninterpretedOption, r.asUninterpretedOption(decl.option))
		} else if decl.rpc != nil {
			sd.Method = append(sd.Method, r.asMethodDescriptor(decl.rpc))
		}
	}
	return sd
}

func toNameParts(ident *identNode, offset int) []*optionNamePartNode {
	parts := strings.Split(ident.val[offset:], ".")
	ret := make([]*optionNamePartNode, len(parts))
	for i, p := range parts {
		ret[i] = &optionNamePartNode{text: ident, offset: offset, length: len(p)}
		ret[i].setRange(ident, ident)
		offset += len(p) + 1
	}
	return ret
}

func checkUint64InInt32Range(lex protoLexer, v uint64) {
	if v > math.MaxInt32 {
		lex.Error(fmt.Sprintf("constant %d is out of range for int32 (%d to %d)", v, math.MinInt32, math.MaxInt32))
	}
}

func checkInt64InInt32Range(lex protoLexer, v int64) {
	if v > math.MaxInt32 || v < math.MinInt32 {
		lex.Error(fmt.Sprintf("constant %d is out of range for int32 (%d to %d)", v, math.MinInt32, math.MaxInt32))
	}
}

func checkTag(lex protoLexer, v uint64) {
	if v > maxTag {
		lex.Error(fmt.Sprintf("tag number %d is higher than max allowed tag number (%d)", v, maxTag))
	} else if v >= specialReservedStart && v <= specialReservedEnd {
		lex.Error(fmt.Sprintf("tag number %d is in disallowed reserved range %d-%d", v, specialReservedStart, specialReservedEnd))
	}
}

func jsonName(name string) string {
	var js []rune
	nextUpper := false
	for i, r := range name {
		if r == '_' {
			nextUpper = true
			continue
		}
		if i == 0 {
			// start lower-case
			js = append(js, unicode.ToLower(r))
		} else if nextUpper {
			nextUpper = false
			js = append(js, unicode.ToUpper(r))
		} else {
			js = append(js, r)
		}
	}
	return string(js)
}

func initCap(name string) string {
	return string(unicode.ToUpper(rune(name[0]))) + name[1:]
}

func aggToString(agg []*aggregateEntryNode, buf *bytes.Buffer) {
	buf.WriteString("{")
	for _, a := range agg {
		buf.WriteString(" ")
		buf.WriteString(a.name.value())
		if v, ok := a.val.(*aggregateLiteralNode); ok {
			aggToString(v.elements, buf)
		} else {
			buf.WriteString(": ")
			elementToString(v, buf)
		}
	}
	buf.WriteString(" }")
}

func elementToString(v interface{}, buf *bytes.Buffer) {
	switch v := v.(type) {
	case bool, int64, uint64, identifier:
		fmt.Fprintf(buf, "%v", v)
	case float64:
		if math.IsInf(v, 1) {
			buf.WriteString(": inf")
		} else if math.IsInf(v, -1) {
			buf.WriteString(": -inf")
		} else if math.IsNaN(v) {
			buf.WriteString(": nan")
		} else {
			fmt.Fprintf(buf, ": %v", v)
		}
	case string:
		buf.WriteRune('"')
		writeEscapedBytes(buf, []byte(v))
		buf.WriteRune('"')
	case []valueNode:
		buf.WriteString(": [")
		first := true
		for _, e := range v {
			if first {
				first = false
			} else {
				buf.WriteString(", ")
			}
			elementToString(e.value(), buf)
		}
		buf.WriteString("]")
	case []*aggregateEntryNode:
		aggToString(v, buf)
	}
}

func writeEscapedBytes(buf *bytes.Buffer, b []byte) {
	for _, c := range b {
		switch c {
		case '\n':
			buf.WriteString("\\n")
		case '\r':
			buf.WriteString("\\r")
		case '\t':
			buf.WriteString("\\t")
		case '"':
			buf.WriteString("\\\"")
		case '\'':
			buf.WriteString("\\'")
		case '\\':
			buf.WriteString("\\\\")
		default:
			if c >= 0x20 && c <= 0x7f && c != '"' && c != '\\' {
				// simple printable characters
				buf.WriteByte(c)
			} else {
				// use octal escape for all other values
				buf.WriteRune('\\')
				buf.WriteByte('0' + ((c >> 6) & 0x7))
				buf.WriteByte('0' + ((c >> 3) & 0x7))
				buf.WriteByte('0' + (c & 0x7))
			}
		}
	}
}

func basicValidate(res *parseResult) error {
	fd := res.fd
	// TODO: include position information in errors
	if fd.Syntax != nil && fd.GetSyntax() != "proto2" && fd.GetSyntax() != "proto3" {
		return fmt.Errorf(`file %q: syntax must be "proto2" or "proto3", instead found %q`, fd.GetName(), fd.GetSyntax())
	}
	isProto3 := fd.GetSyntax() == "proto3"

	for _, md := range fd.MessageType {
		if err := validateMessage(fd, isProto3, "", md); err != nil {
			return err
		}
	}

	for _, ed := range fd.EnumType {
		if err := validateEnum(fd, isProto3, "", ed); err != nil {
			return err
		}
	}

	for _, fld := range fd.Extension {
		if err := validateField(fd, isProto3, "", fld); err != nil {
			return err
		}
	}
	return nil
}

func validateMessage(fd *dpb.FileDescriptorProto, isProto3 bool, prefix string, md *dpb.DescriptorProto) error {
	nextPrefix := md.GetName() + "."

	if md.GetOptions().GetMapEntry() && !isProto3 {
		// we build map fields without a label, but it should
		// instead be "optional" for proto2 messages
		for _, fld := range md.Field {
			if fld.Label == nil {
				fld.Label = dpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()
			}
		}
	}

	for _, fld := range md.Field {
		if err := validateField(fd, isProto3, nextPrefix, fld); err != nil {
			return err
		}
	}
	for _, fld := range md.Extension {
		if err := validateField(fd, isProto3, nextPrefix, fld); err != nil {
			return err
		}
	}
	for _, ed := range md.EnumType {
		if err := validateEnum(fd, isProto3, nextPrefix, ed); err != nil {
			return err
		}
	}
	for _, nmd := range md.NestedType {
		if err := validateMessage(fd, isProto3, nextPrefix, nmd); err != nil {
			return err
		}
	}

	if isProto3 && len(md.ExtensionRange) > 0 {
		return fmt.Errorf("file %q: message %s%s: extension ranges are not allowed in proto3", fd.GetName(), prefix, md.GetName())
	}

	if index, err := findOption(md.Options.GetUninterpretedOption(), "map_entry"); err != nil {
		return fmt.Errorf("file %q: message %s%s: %s", fd.GetName(), prefix, md.GetName(), err)
	} else if index >= 0 {
		opt := md.Options.UninterpretedOption[index]
		md.Options.UninterpretedOption = removeOption(md.Options.UninterpretedOption, index)
		valid := false
		if opt.IdentifierValue != nil {
			if opt.GetIdentifierValue() == "true" {
				return fmt.Errorf("file %q: message %s%s: map_entry option should not be set explicitly; use map type instead", fd.GetName(), prefix, md.GetName())
			} else if opt.GetIdentifierValue() == "false" {
				md.Options.MapEntry = proto.Bool(false)
				valid = true
			}
		}
		if !valid {
			return fmt.Errorf("file %q: message %s%s: expecting bool value for map_entry option", fd.GetName(), prefix, md.GetName())
		}
	}

	// reserved ranges should not overlap
	rsvd := make(tagRanges, len(md.ReservedRange))
	for i, r := range md.ReservedRange {
		rsvd[i] = tagRange{Start: r.GetStart(), End: r.GetEnd()}
	}
	sort.Sort(rsvd)
	for i := 1; i < len(rsvd); i++ {
		if rsvd[i].Start < rsvd[i-1].End {
			return fmt.Errorf("file %s: message %s%s: reserved ranges overlap: %d to %d and %d to %d", fd.GetName(), prefix, md.GetName(), rsvd[i-1].Start, rsvd[i-1].End-1, rsvd[i].Start, rsvd[i].End-1)
		}
	}

	// extensions ranges should not overlap
	exts := make(tagRanges, len(md.ExtensionRange))
	for i, r := range md.ExtensionRange {
		exts[i] = tagRange{Start: r.GetStart(), End: r.GetEnd()}
	}
	sort.Sort(exts)
	for i := 1; i < len(exts); i++ {
		if exts[i].Start < exts[i-1].End {
			return fmt.Errorf("file %s: message %s%s: extension ranges overlap: %d to %d and %d to %d", fd.GetName(), prefix, md.GetName(), exts[i-1].Start, exts[i-1].End-1, exts[i].Start, exts[i].End-1)
		}
	}

	// see if any extension range overlaps any reserved range
	var i, j int // i indexes rsvd; j indexes exts
	for i < len(rsvd) && j < len(exts) {
		if rsvd[i].Start >= exts[j].Start && rsvd[i].Start < exts[j].End ||
			exts[j].Start >= rsvd[i].Start && exts[j].Start < rsvd[i].End {
			// ranges overlap
			return fmt.Errorf("file %s: message %s%s: extension range %d to %d overlaps reserved range %d to %d", fd.GetName(), prefix, md.GetName(), exts[j].Start, exts[j].End-1, rsvd[i].Start, rsvd[i].End-1)
		}
		if rsvd[i].Start < exts[j].Start {
			i++
		} else {
			j++
		}
	}

	// now, check that fields don't re-use tags and don't try to use extension
	// or reserved ranges or reserved names
	rsvdNames := map[string]struct{}{}
	for _, n := range md.ReservedName {
		rsvdNames[n] = struct{}{}
	}
	fieldTags := map[int32]string{}
	for _, fld := range md.Field {
		if _, ok := rsvdNames[fld.GetName()]; ok {
			return fmt.Errorf("file %s: message %s%s: field %s is using a reserved name", fd.GetName(), prefix, md.GetName(), fld.GetName())
		}
		if existing := fieldTags[fld.GetNumber()]; existing != "" {
			return fmt.Errorf("file %s: message %s%s: fields %s and %s both have the same tag %d", fd.GetName(), prefix, md.GetName(), existing, fld.GetName(), fld.GetNumber())
		}
		fieldTags[fld.GetNumber()] = fld.GetName()
		// check reserved ranges
		r := sort.Search(len(rsvd), func(index int) bool { return rsvd[index].End > fld.GetNumber() })
		if r < len(rsvd) && rsvd[r].Start <= fld.GetNumber() {
			return fmt.Errorf("file %s: message %s%s: field %s is using tag %d which is in reserved range %d to %d", fd.GetName(), prefix, md.GetName(), fld.GetName(), fld.GetNumber(), rsvd[r].Start, rsvd[r].End-1)
		}
		// and check extension ranges
		e := sort.Search(len(exts), func(index int) bool { return exts[index].End > fld.GetNumber() })
		if e < len(exts) && exts[e].Start <= fld.GetNumber() {
			return fmt.Errorf("file %s: message %s%s: field %s is using tag %d which is in extension range %d to %d", fd.GetName(), prefix, md.GetName(), fld.GetName(), fld.GetNumber(), exts[e].Start, exts[e].End-1)
		}
	}

	return nil
}

func validateEnum(fd *dpb.FileDescriptorProto, isProto3 bool, prefix string, ed *dpb.EnumDescriptorProto) error {
	if index, err := findOption(ed.Options.GetUninterpretedOption(), "allow_alias"); err != nil {
		return fmt.Errorf("file %q: enum %s%s: %s", fd.GetName(), prefix, ed.GetName(), err)
	} else if index >= 0 {
		opt := ed.Options.UninterpretedOption[index]
		ed.Options.UninterpretedOption = removeOption(ed.Options.UninterpretedOption, index)
		valid := false
		if opt.IdentifierValue != nil {
			if opt.GetIdentifierValue() == "true" {
				ed.Options.AllowAlias = proto.Bool(true)
				valid = true
			} else if opt.GetIdentifierValue() == "false" {
				ed.Options.AllowAlias = proto.Bool(false)
				valid = true
			}
		}
		if !valid {
			return fmt.Errorf("file %q: enum %s%s: expecting bool value for allow_alias option", fd.GetName(), prefix, ed.GetName())
		}
	}

	if isProto3 && ed.Value[0].GetNumber() != 0 {
		return fmt.Errorf("file %q: enum %s%s: proto3 requires that first value in enum have numeric value of 0", fd.GetName(), prefix, ed.GetName())
	}

	if !ed.Options.GetAllowAlias() {
		// make sure all value numbers are distinct
		vals := map[int32]string{}
		for _, evd := range ed.Value {
			if existing := vals[evd.GetNumber()]; existing != "" {
				return fmt.Errorf("file %s: enum %s%s: values %s and %s both have the same numeric value %d; use allow_alias option if intentional", fd.GetName(), prefix, ed.GetName(), existing, evd.GetName(), evd.GetNumber())
			}
			vals[evd.GetNumber()] = evd.GetName()
		}
	}

	return nil
}

func validateField(fd *dpb.FileDescriptorProto, isProto3 bool, prefix string, fld *dpb.FieldDescriptorProto) error {
	if isProto3 {
		if fld.GetType() == dpb.FieldDescriptorProto_TYPE_GROUP {
			return fmt.Errorf("file %q: group %s%s: groups are not allowed in proto3", fd.GetName(), prefix, fld.GetTypeName())
		}
		if fld.Label != nil && fld.GetLabel() != dpb.FieldDescriptorProto_LABEL_REPEATED {
			return fmt.Errorf("file %q: field %s%s: field has label %v, but proto3 should omit labels other than 'repeated'",
				fd.GetName(), prefix, fld.GetName(), fld.GetLabel())
		}
		if index, err := findOption(fld.Options.GetUninterpretedOption(), "default"); err != nil {
			return fmt.Errorf("file %q: field %s%s: %s", fd.GetName(), prefix, fld.GetName(), err)
		} else if index >= 0 {
			return fmt.Errorf("file %q: field %s%s: default values are not allowed in proto3", fd.GetName(), prefix, fld.GetName())
		}
	} else {
		if fld.Label == nil && fld.OneofIndex == nil {
			return fmt.Errorf("file %q: field %s%s: field has no label, but proto2 must indicate 'optional' or 'required'",
				fd.GetName(), prefix, fld.GetName())
		}
		if fld.GetExtendee() != "" && fld.Label != nil && fld.GetLabel() == dpb.FieldDescriptorProto_LABEL_REQUIRED {
			return fmt.Errorf("file %q: field %s%s: extension fields cannot be 'required'", fd.GetName(), prefix, fld.GetName())
		}
	}

	// process json_name pseudo-option
	if index, err := findOption(fld.Options.GetUninterpretedOption(), "json_name"); err != nil {
		return fmt.Errorf("file %q: field %s%s: %s", fd.GetName(), prefix, fld.GetName(), err)
	} else if index >= 0 {
		opt := fld.Options.UninterpretedOption[index]
		if len(fld.Options.UninterpretedOption) == 1 {
			// this was the only option and it's been hoisted out, so clean up
			fld.Options = nil
		} else {
			fld.Options.UninterpretedOption = removeOption(fld.Options.UninterpretedOption, index)
		}
		if opt.StringValue == nil {
			return fmt.Errorf("file %q: field %s%s: expecting string value for json_name option", fd.GetName(), prefix, fld.GetName())
		}
		fld.JsonName = proto.String(string(opt.StringValue))
	}

	// finally, set any missing label to optional
	if fld.Label == nil {
		fld.Label = dpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()
	}
	return nil
}

func findOption(opts []*dpb.UninterpretedOption, name string) (int, error) {
	found := -1
	for i, opt := range opts {
		if len(opt.Name) != 1 {
			continue
		}
		if opt.Name[0].GetIsExtension() || opt.Name[0].GetNamePart() != name {
			continue
		}
		if found >= 0 {
			return -1, fmt.Errorf("option %s cannot be defined more than once", name)
		}
		found = i
	}
	return found, nil
}

func removeOption(uo []*dpb.UninterpretedOption, indexToRemove int) []*dpb.UninterpretedOption {
	if indexToRemove == 0 {
		return uo[1:]
	} else if int(indexToRemove) == len(uo)-1 {
		return uo[:len(uo)-1]
	} else {
		return append(uo[:indexToRemove], uo[indexToRemove+1:]...)
	}
}

type tagRange struct {
	Start int32
	End   int32
}

type tagRanges []tagRange

func (r tagRanges) Len() int {
	return len(r)
}

func (r tagRanges) Less(i, j int) bool {
	return r[i].Start < r[j].Start ||
		(r[i].Start == r[j].Start && r[i].End < r[j].End)
}

func (r tagRanges) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}
