package protoparse

import (
	"bytes"
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

	protos := map[string]*dpb.FileDescriptorProto{}
	aggregates := map[string][]*aggregate{}
	err := parseProtoFiles(accessor, filenames, protos, aggregates)
	if err != nil {
		return nil, err
	}
	if p.InferImportPaths {
		protos = fixupFilenames(protos)
	}
	linkedProtos, err := newLinker(protos, aggregates).linkFiles()
	if err != nil {
		return nil, err
	}
	fds := make([]*desc.FileDescriptor, len(filenames))
	for i, name := range filenames {
		fds[i] = linkedProtos[name]
	}
	return fds, nil
}

func fixupFilenames(protos map[string]*dpb.FileDescriptorProto) map[string]*dpb.FileDescriptorProto {
	// In the event that the given filenames (keys in the supplied map) do not
	// match the actual paths used in 'import' statements in the files, we try
	// to revise names in the protos so that they will match and be linkable.
	revisedProtos := map[string]*dpb.FileDescriptorProto{}

	protoPaths := map[string]struct{}{}
	// TODO: this is O(n^2) but could likely be O(n) with a clever data structure (prefix tree that is indexed backwards?)
	importCandidates := map[string]map[string]struct{}{}
	candidatesAvailable := map[string]struct{}{}
	for name := range protos {
		candidatesAvailable[name] = struct{}{}
		for _, fd := range protos {
			for _, imp := range fd.Dependency {
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
			fd := protos[best]
			fd.Name = proto.String(imp)
			revisedProtos[imp] = fd
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
			fd := protos[c]
			fd.Name = proto.String(imp)
			revisedProtos[imp] = fd
		} else {
			revisedProtos[c] = protos[c]
		}
	}

	return revisedProtos
}

func parseProtoFiles(acc FileAccessor, filenames []string, parsed map[string]*dpb.FileDescriptorProto, aggregates map[string][]*aggregate) error {
	for _, name := range filenames {
		if _, ok := parsed[name]; ok {
			continue
		}
		in, err := acc(name)
		if err != nil {
			if d, ok := standardImports[name]; ok {
				parsed[name] = d
				continue
			}
			return err
		}
		func() {
			defer in.Close()
			parsed[name], err = parseProto(name, in, aggregates)
		}()
		if err != nil {
			return err
		}
		err = parseProtoFiles(acc, parsed[name].Dependency, parsed, aggregates)
		if err != nil {
			return fmt.Errorf("failed to load imports for %q: %s", name, err)
		}
	}
	return nil
}

func parseProtoFile(filename string) (*dpb.FileDescriptorProto, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parseProto(filename, f, map[string][]*aggregate{})
}

func parseProto(filename string, r io.Reader, aggregates map[string][]*aggregate) (*dpb.FileDescriptorProto, error) {
	lx := newLexer(r)
	lx.aggregates = aggregates
	protoParse(lx)
	if lx.err != nil {
		if lx.prevLineNo == lx.lineNo && lx.prevColNo != lx.colNo {
			return nil, fmt.Errorf("file %s: line %d, col %d-%d: %s", filename, lx.lineNo, lx.prevColNo, lx.colNo, lx.err)
		} else {
			return nil, fmt.Errorf("file %s: line %d, col %d: %s", filename, lx.lineNo, lx.prevColNo, lx.err)
		}
	}
	lx.res.Name = proto.String(filename)
	if err := basicValidate(lx.res); err != nil {
		return nil, err
	}
	return lx.res, nil
}

type importSpec struct {
	name         string
	weak, public bool
}

type fileDecl struct {
	importSpec  *importSpec
	packageName string
	option      *dpb.UninterpretedOption
	message     *dpb.DescriptorProto
	enum        *dpb.EnumDescriptorProto
	extend      *extendBlock
	service     *dpb.ServiceDescriptorProto
}

type groupDesc struct {
	field *dpb.FieldDescriptorProto
	msg   *dpb.DescriptorProto
}

type oneofDesc struct {
	name    string
	fields  []*dpb.FieldDescriptorProto
	options []*dpb.UninterpretedOption
}

type extendBlock struct {
	fields []*dpb.FieldDescriptorProto
	msgs   []*dpb.DescriptorProto
}

type reservedFields struct {
	tags  []tagRange
	names []string
}

type enumDecl struct {
	option *dpb.UninterpretedOption
	val    *dpb.EnumValueDescriptorProto
}

type msgDecl struct {
	option     *dpb.UninterpretedOption
	fld        *dpb.FieldDescriptorProto
	grp        *groupDesc
	oneof      *oneofDesc
	enum       *dpb.EnumDescriptorProto
	msg        *dpb.DescriptorProto
	extend     *extendBlock
	extensions []*dpb.DescriptorProto_ExtensionRange
	reserved   *reservedFields
}

type serviceDecl struct {
	option *dpb.UninterpretedOption
	rpc    *dpb.MethodDescriptorProto
}

type rpcType struct {
	msgType string
	stream  bool
}

type option struct {
	name []*dpb.UninterpretedOption_NamePart
	val  interface{}
}

type aggregate struct {
	name string
	val  interface{}
}

type identifier string

func fileDeclsToProto(decls []*fileDecl) *dpb.FileDescriptorProto {
	fd := &dpb.FileDescriptorProto{}
	for _, decl := range decls {
		if decl.enum != nil {
			fd.EnumType = append(fd.EnumType, decl.enum)
		} else if decl.extend != nil {
			fd.Extension = append(fd.Extension, decl.extend.fields...)
			fd.MessageType = append(fd.MessageType, decl.extend.msgs...)
		} else if decl.importSpec != nil {
			index := len(fd.Dependency)
			fd.Dependency = append(fd.Dependency, decl.importSpec.name)
			if decl.importSpec.public {
				fd.PublicDependency = append(fd.PublicDependency, int32(index))
			} else if decl.importSpec.weak {
				fd.WeakDependency = append(fd.WeakDependency, int32(index))
			}
		} else if decl.message != nil {
			fd.MessageType = append(fd.MessageType, decl.message)
		} else if decl.option != nil {
			if fd.Options == nil {
				fd.Options = &dpb.FileOptions{}
			}
			fd.Options.UninterpretedOption = append(fd.Options.UninterpretedOption, decl.option)
		} else if decl.service != nil {
			fd.Service = append(fd.Service, decl.service)
		} else if decl.packageName != "" {
			fd.Package = proto.String(decl.packageName)
		}
	}
	return fd
}

func asOption(ctx *protoLex, name []*dpb.UninterpretedOption_NamePart, val interface{}) *dpb.UninterpretedOption {
	opt := &dpb.UninterpretedOption{Name: name}
	switch val := val.(type) {
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
	case []*aggregate:
		var buf bytes.Buffer
		aggToString(val, &buf)
		aggStr := buf.String()
		opt.AggregateValue = proto.String(aggStr)
		if ctx.aggregates != nil {
			ctx.aggregates[aggStr] = val
		}
	}
	return opt
}

func toNameParts(ident string) []*dpb.UninterpretedOption_NamePart {
	parts := strings.Split(ident, ".")
	ret := make([]*dpb.UninterpretedOption_NamePart, len(parts))
	for i, p := range parts {
		ret[i] = &dpb.UninterpretedOption_NamePart{NamePart: proto.String(p)}
	}
	return ret
}

func asFieldDescriptor(label *dpb.FieldDescriptorProto_Label, typ, name string, tag int32, opts []*dpb.UninterpretedOption) *dpb.FieldDescriptorProto {
	fd := &dpb.FieldDescriptorProto{
		Name:     proto.String(name),
		JsonName: proto.String(jsonName(name)),
		Number:   proto.Int32(tag),
		Label:    label,
	}
	if len(opts) > 0 {
		fd.Options = &dpb.FieldOptions{UninterpretedOption: opts}
	}
	switch typ {
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
		fd.TypeName = proto.String(typ)
	}
	return fd
}

func asGroupDescriptor(lex protoLexer, label dpb.FieldDescriptorProto_Label, name string, tag int32, body []*msgDecl) *groupDesc {
	if !unicode.IsUpper(rune(name[0])) {
		lex.Error(fmt.Sprintf("group %s should have a name that starts with a capital letter", name))
	}
	fieldName := strings.ToLower(name)
	fd := &dpb.FieldDescriptorProto{
		Name:     proto.String(fieldName),
		JsonName: proto.String(jsonName(fieldName)),
		Number:   proto.Int32(tag),
		Label:    label.Enum(),
		Type:     dpb.FieldDescriptorProto_TYPE_GROUP.Enum(),
		TypeName: proto.String(name),
	}
	md := msgDeclsToProto(name, body)
	return &groupDesc{field: fd, msg: md}
}

func asMapField(keyType, valType, name string, tag int32, opts []*dpb.UninterpretedOption) *groupDesc {
	keyFd := asFieldDescriptor(dpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), keyType, "key", 1, nil)
	valFd := asFieldDescriptor(dpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), valType, "value", 2, nil)
	entryName := initCap(jsonName(name)) + "Entry"
	fd := asFieldDescriptor(dpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), entryName, name, tag, opts)
	md := &dpb.DescriptorProto{
		Name:    proto.String(entryName),
		Options: &dpb.MessageOptions{MapEntry: proto.Bool(true)},
		Field:   []*dpb.FieldDescriptorProto{keyFd, valFd},
	}
	return &groupDesc{field: fd, msg: md}
}

func asExtensionRanges(ranges []tagRange, opts []*dpb.UninterpretedOption) []*dpb.DescriptorProto_ExtensionRange {
	ers := make([]*dpb.DescriptorProto_ExtensionRange, len(ranges))
	for i, r := range ranges {
		ers[i] = &dpb.DescriptorProto_ExtensionRange{Start: proto.Int32(r.Start), End: proto.Int32(r.End)}
		if len(opts) > 0 {
			ers[i].Options = &dpb.ExtensionRangeOptions{UninterpretedOption: opts}
		}
	}
	return ers
}

func asEnumValue(name string, val int32, opts []*dpb.UninterpretedOption) *dpb.EnumValueDescriptorProto {
	evd := &dpb.EnumValueDescriptorProto{Name: proto.String(name), Number: proto.Int32(val)}
	if len(opts) > 0 {
		evd.Options = &dpb.EnumValueOptions{UninterpretedOption: opts}
	}
	return evd
}

func asMethodDescriptor(name string, req, resp *rpcType, opts []*dpb.UninterpretedOption) *dpb.MethodDescriptorProto {
	md := &dpb.MethodDescriptorProto{
		Name:       proto.String(name),
		InputType:  proto.String(req.msgType),
		OutputType: proto.String(resp.msgType),
	}
	if req.stream {
		md.ClientStreaming = proto.Bool(true)
	}
	if resp.stream {
		md.ServerStreaming = proto.Bool(true)
	}
	if len(opts) > 0 {
		md.Options = &dpb.MethodOptions{UninterpretedOption: opts}
	}
	return md
}

func enumDeclsToProto(name string, decls []*enumDecl) *dpb.EnumDescriptorProto {
	ed := &dpb.EnumDescriptorProto{Name: proto.String(name)}
	for _, decl := range decls {
		if decl.option != nil {
			if ed.Options == nil {
				ed.Options = &dpb.EnumOptions{}
			}
			ed.Options.UninterpretedOption = append(ed.Options.UninterpretedOption, decl.option)
		} else if decl.val != nil {
			ed.Value = append(ed.Value, decl.val)
		}
	}
	return ed
}

func msgDeclsToProto(name string, decls []*msgDecl) *dpb.DescriptorProto {
	msgd := &dpb.DescriptorProto{Name: proto.String(name)}
	for _, decl := range decls {
		if decl.enum != nil {
			msgd.EnumType = append(msgd.EnumType, decl.enum)
		} else if decl.extend != nil {
			msgd.Extension = append(msgd.Extension, decl.extend.fields...)
			msgd.NestedType = append(msgd.NestedType, decl.extend.msgs...)
		} else if decl.extensions != nil {
			msgd.ExtensionRange = append(msgd.ExtensionRange, decl.extensions...)
		} else if decl.fld != nil {
			msgd.Field = append(msgd.Field, decl.fld)
		} else if decl.grp != nil {
			msgd.Field = append(msgd.Field, decl.grp.field)
			msgd.NestedType = append(msgd.NestedType, decl.grp.msg)
		} else if decl.oneof != nil {
			oodIndex := len(msgd.OneofDecl)
			ood := &dpb.OneofDescriptorProto{Name: proto.String(decl.oneof.name)}
			if len(decl.oneof.options) > 0 {
				ood.Options = &dpb.OneofOptions{UninterpretedOption: decl.oneof.options}
			}
			msgd.OneofDecl = append(msgd.OneofDecl, ood)
			for _, fd := range decl.oneof.fields {
				fd.OneofIndex = proto.Int32(int32(oodIndex))
			}
			msgd.Field = append(msgd.Field, decl.oneof.fields...)
		} else if decl.option != nil {
			if msgd.Options == nil {
				msgd.Options = &dpb.MessageOptions{}
			}
			msgd.Options.UninterpretedOption = append(msgd.Options.UninterpretedOption, decl.option)
		} else if decl.msg != nil {
			msgd.NestedType = append(msgd.NestedType, decl.msg)
		} else if decl.reserved != nil {
			if len(decl.reserved.names) > 0 {
				msgd.ReservedName = append(msgd.ReservedName, decl.reserved.names...)
			}
			if len(decl.reserved.tags) > 0 {
				for _, r := range decl.reserved.tags {
					msgd.ReservedRange = append(msgd.ReservedRange, &dpb.DescriptorProto_ReservedRange{Start: proto.Int32(r.Start), End: proto.Int32(r.End)})
				}
			}
		}
	}
	return msgd
}

func svcDeclsToProto(name string, decls []*serviceDecl) *dpb.ServiceDescriptorProto {
	sd := &dpb.ServiceDescriptorProto{Name: proto.String(name)}
	for _, decl := range decls {
		if decl.option != nil {
			if sd.Options == nil {
				sd.Options = &dpb.ServiceOptions{}
			}
			sd.Options.UninterpretedOption = append(sd.Options.UninterpretedOption, decl.option)
		} else if decl.rpc != nil {
			sd.Method = append(sd.Method, decl.rpc)
		}
	}
	return sd
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

func aggToString(agg []*aggregate, buf *bytes.Buffer) {
	buf.WriteString("{")
	for _, a := range agg {
		buf.WriteString(" ")
		buf.WriteString(a.name)
		if v, ok := a.val.([]*aggregate); ok {
			aggToString(v, buf)
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
	case []interface{}:
		buf.WriteString(": [")
		first := true
		for e := range v {
			if first {
				first = false
			} else {
				buf.WriteString(", ")
			}
			elementToString(e, buf)
		}
		buf.WriteString("]")
	case []*aggregate:
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

func basicValidate(fd *dpb.FileDescriptorProto) error {
	// TODO: track syntax during parse so we can then apply validations at parse time instead of in post-process
	// (this will allow us to include location information: e.g. line number in file where error is)
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
