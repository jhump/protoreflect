package protobuilder

import (
	"errors"
	"fmt"
	"iter"
	"sort"
	"strings"
	"sync/atomic"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/v2/internal"
	"github.com/jhump/protoreflect/v2/protodescs"
	"github.com/jhump/protoreflect/v2/protomessage"
	"github.com/jhump/protoreflect/v2/protoresolve"
)

var uniqueFileCounter uint64

func uniqueFilePath() string {
	i := atomic.AddUint64(&uniqueFileCounter, 1)
	return fmt.Sprintf("{generated-file-%04x}.proto", i)
}

func makeUnique(name string, existingNames map[string]struct{}) string {
	i := 1
	n := name
	for {
		if _, ok := existingNames[n]; !ok {
			return n
		}
		n = fmt.Sprintf("%s(%d)", name, i)
		i++
	}
}

// FileBuilder is a builder used to construct a protoreflect.FileDescriptor. This is the
// root of the hierarchy. All other descriptors belong to a file, and thus all
// other builders also belong to a file.
//
// If a builder is *not* associated with a file, the resulting descriptor will
// be associated with a synthesized file that contains only the built descriptor
// and its ancestors. This means that such descriptors will have no associated
// package name.
//
// To create a new FileBuilder, use NewFile.
type FileBuilder struct {
	path string

	Syntax protoreflect.Syntax
	// Edition indicates which edition this file uses. It may only be
	// set when Syntax is unset or set to protoreflect.Editions.
	Edition descriptorpb.Edition

	Package protoreflect.FullName
	Options *descriptorpb.FileOptions

	comments        Comments
	SyntaxComments  Comments
	PackageComments Comments

	messages   []*MessageBuilder
	extensions []*FieldBuilder
	enums      []*EnumBuilder
	services   []*ServiceBuilder
	symbols    map[protoreflect.Name]Builder

	origExts        protoregistry.Types
	explicitDeps    map[*FileBuilder]bool
	explicitImports map[protoreflect.FileDescriptor]bool
}

var _ Builder = (*FileBuilder)(nil)

// NewFile creates a new FileBuilder for a file with the given path. The
// path can be blank, which indicates a unique path should be generated for it.
func NewFile(path string) *FileBuilder {
	return &FileBuilder{
		path:    path,
		symbols: map[protoreflect.Name]Builder{},
	}
}

// FromFile returns a FileBuilder that is effectively a copy of the given
// descriptor. Note that builders do not retain full source code info, even if
// the given descriptor included it. Instead, comments are extracted from the
// given descriptor's source info (if present) and, when built, the resulting
// descriptor will have just the comment info (no location information).
func FromFile(fd protoreflect.FileDescriptor) (*FileBuilder, error) {
	fb := NewFile(fd.Path())

	fb.Syntax = fd.Syntax()
	var path []int32
	if fb.Syntax == protoreflect.Editions {
		fb.Edition = protodescs.GetEdition(fd, nil)
		path = []int32{internal.FileEditionTag}
	} else {
		path = []int32{internal.FileSyntaxTag}
	}
	setComments(&fb.SyntaxComments, fd.SourceLocations().ByPath(path))

	fb.Package = fd.Package()
	path = []int32{internal.FilePackageTag}
	setComments(&fb.PackageComments, fd.SourceLocations().ByPath(path))

	var err error
	fb.Options, err = protomessage.As[*descriptorpb.FileOptions](fd.Options())
	if err != nil {
		return nil, err
	}
	setComments(&fb.comments, fd.SourceLocations().ByPath(protoreflect.SourcePath{}))

	// add imports explicitly
	imps := fd.Imports()
	for i, length := 0, imps.Len(); i < length; i++ {
		imp := imps.Get(i).FileDescriptor
		fb.AddImportedDependency(imp)
		if err := fb.addExtensionsFromImport(imp); err != nil {
			return nil, err
		}
	}

	localMessages := map[protoreflect.MessageDescriptor]*MessageBuilder{}
	localEnums := map[protoreflect.EnumDescriptor]*EnumBuilder{}

	msgs := fd.Messages()
	for i, length := 0, msgs.Len(); i < length; i++ {
		msg := msgs.Get(i)
		if mb, err := fromMessage(msg, localMessages, localEnums); err != nil {
			return nil, err
		} else if err := fb.TryAddMessage(mb); err != nil {
			return nil, err
		}
	}
	enums := fd.Enums()
	for i, length := 0, enums.Len(); i < length; i++ {
		enum := enums.Get(i)
		if eb, err := fromEnum(enum, localEnums); err != nil {
			return nil, err
		} else if err := fb.TryAddEnum(eb); err != nil {
			return nil, err
		}
	}
	exts := fd.Extensions()
	for i, length := 0, exts.Len(); i < length; i++ {
		ext := exts.Get(i)
		if exb, err := fromField(ext); err != nil {
			return nil, err
		} else if err := fb.TryAddExtension(exb); err != nil {
			return nil, err
		}
	}
	svcs := fd.Services()
	for i, length := 0, svcs.Len(); i < length; i++ {
		svc := svcs.Get(i)
		if sb, err := fromService(svc); err != nil {
			return nil, err
		} else if err := fb.TryAddService(sb); err != nil {
			return nil, err
		}
	}

	// we've converted everything, so now we update all foreign type references
	// to be local type references if possible
	for _, mb := range fb.messages {
		updateLocalRefsInMessage(mb, localMessages, localEnums)
	}
	for _, exb := range fb.extensions {
		updateLocalRefsInField(exb, localMessages, localEnums)
	}
	for _, sb := range fb.services {
		for _, mtb := range sb.methods {
			updateLocalRefsInRpcType(mtb.ReqType, localMessages)
			updateLocalRefsInRpcType(mtb.RespType, localMessages)
		}
	}

	return fb, nil
}

func updateLocalRefsInMessage(mb *MessageBuilder, localMessages map[protoreflect.MessageDescriptor]*MessageBuilder, localEnums map[protoreflect.EnumDescriptor]*EnumBuilder) {
	for _, b := range mb.fieldsAndOneofs {
		if flb, ok := b.(*FieldBuilder); ok {
			updateLocalRefsInField(flb, localMessages, localEnums)
		} else {
			oob := b.(*OneofBuilder)
			for _, flb := range oob.choices {
				updateLocalRefsInField(flb, localMessages, localEnums)
			}
		}
	}
	for _, nmb := range mb.nestedMessages {
		updateLocalRefsInMessage(nmb, localMessages, localEnums)
	}
	for _, exb := range mb.nestedExtensions {
		updateLocalRefsInField(exb, localMessages, localEnums)
	}
}

func updateLocalRefsInField(flb *FieldBuilder, localMessages map[protoreflect.MessageDescriptor]*MessageBuilder, localEnums map[protoreflect.EnumDescriptor]*EnumBuilder) {
	if flb.fieldType.foreignMsgType != nil {
		if mb, ok := localMessages[flb.fieldType.foreignMsgType]; ok {
			flb.fieldType.foreignMsgType = nil
			flb.fieldType.localMsgType = mb
		}
	}
	if flb.fieldType.foreignEnumType != nil {
		if eb, ok := localEnums[flb.fieldType.foreignEnumType]; ok {
			flb.fieldType.foreignEnumType = nil
			flb.fieldType.localEnumType = eb
		}
	}
	if flb.foreignExtendee != nil {
		if mb, ok := localMessages[flb.foreignExtendee]; ok {
			flb.foreignExtendee = nil
			flb.localExtendee = mb
		}
	}
	if flb.msgType != nil {
		updateLocalRefsInMessage(flb.msgType, localMessages, localEnums)
	}
}

func updateLocalRefsInRpcType(rpcType *RpcType, localMessages map[protoreflect.MessageDescriptor]*MessageBuilder) {
	if rpcType.foreignType != nil {
		if mb, ok := localMessages[rpcType.foreignType]; ok {
			rpcType.foreignType = nil
			rpcType.localType = mb
		}
	}
}

// Name implements the Builder interface. However, files do not have
// names, they have paths. So this method always returns the empty
// string. Use Path instead.
// instead.
func (fb *FileBuilder) Name() protoreflect.Name {
	return ""
}

// SetName implements the Builder interface. However, files do not have
// names, they have paths. So this method always panics. Use SetPath
// instead.
func (fb *FileBuilder) SetName(newName protoreflect.Name) *FileBuilder {
	if err := fb.TrySetName(newName); err != nil {
		panic(err)
	}
	return fb
}

// TrySetName implements the Builder interface. However, files do not have
// names, they have paths. So this method always returns an error. Use
// SetPath instead.
func (fb *FileBuilder) TrySetName(_ protoreflect.Name) error {
	return errors.New("can't set name on FileBuilder; use SetPath instead")
}

// Path returns the path of the file. It may include relative path
// information, too.
func (fb *FileBuilder) Path() string {
	return fb.path
}

// SetPath changes this file's path, returning the file builder for method
// chaining.
func (fb *FileBuilder) SetPath(path string) *FileBuilder {
	fb.path = path
	return fb
}

// Parent always returns nil since files are the roots of builder
// hierarchies.
func (fb *FileBuilder) Parent() Builder {
	return nil
}

func (fb *FileBuilder) setParent(parent Builder) {
	if parent != nil {
		panic("files cannot have parent elements")
	}
}

// Comments returns comments associated with the file itself and not any
// particular element therein. (Note that such a comment will not be rendered by
// the protoprint package.)
func (fb *FileBuilder) Comments() *Comments {
	return &fb.comments
}

// SetComments sets the comments associated with the file itself, not any
// particular element therein. (Note that such a comment will not be rendered by
// the protoprint package.) This method returns the file, for method chaining.
func (fb *FileBuilder) SetComments(c Comments) *FileBuilder {
	fb.comments = c
	return fb
}

// SetSyntaxComments sets the comments associated with the syntax declaration
// element (which, if present, is required to be the first element in a proto
// file). This method returns the file, for method chaining.
func (fb *FileBuilder) SetSyntaxComments(c Comments) *FileBuilder {
	fb.SyntaxComments = c
	return fb
}

// SetPackageComments sets the comments associated with the package declaration
// element. (This comment will not be rendered if the file's declared package is
// empty.) This method returns the file, for method chaining.
func (fb *FileBuilder) SetPackageComments(c Comments) *FileBuilder {
	fb.PackageComments = c
	return fb
}

// ParentFile implements the Builder interface and always returns this file.
func (fb *FileBuilder) ParentFile() *FileBuilder {
	return fb
}

// Children returns builders for all nested elements, including all top-level
// messages, enums, extensions, and services.
func (fb *FileBuilder) Children() iter.Seq[Builder] {
	return func(yield func(Builder) bool) {
		for _, mb := range fb.messages {
			if !yield(mb) {
				return
			}
		}
		for _, exb := range fb.extensions {
			if !yield(exb) {
				return
			}
		}
		for _, eb := range fb.enums {
			if !yield(eb) {
				return
			}
		}
		for _, sb := range fb.services {
			if !yield(sb) {
				return
			}
		}
	}
}

func (fb *FileBuilder) findChild(name protoreflect.Name) Builder {
	child := fb.symbols[name]
	if child != nil {
		return child
	}
	// Enum values are in the scope of the enclosing element, not the
	// enum itself. So we have to look here in the file for values of
	// any top-level enums
	for _, eb := range fb.enums {
		child = eb.findChild(name)
		if child != nil {
			return child
		}
	}
	return nil
}

func (fb *FileBuilder) removeChild(b Builder) {
	if p, ok := b.Parent().(*FileBuilder); !ok || p != fb {
		return
	}

	switch b.(type) {
	case *MessageBuilder:
		fb.messages = deleteBuilder(b.Name(), fb.messages).([]*MessageBuilder)
	case *FieldBuilder:
		fb.extensions = deleteBuilder(b.Name(), fb.extensions).([]*FieldBuilder)
	case *EnumBuilder:
		fb.enums = deleteBuilder(b.Name(), fb.enums).([]*EnumBuilder)
	case *ServiceBuilder:
		fb.services = deleteBuilder(b.Name(), fb.services).([]*ServiceBuilder)
	}
	delete(fb.symbols, b.Name())
	b.setParent(nil)
}

func (fb *FileBuilder) renamedChild(b Builder, oldName protoreflect.Name) error {
	if p, ok := b.Parent().(*FileBuilder); !ok || p != fb {
		return nil
	}

	if err := fb.addSymbol(b); err != nil {
		return err
	}
	delete(fb.symbols, oldName)
	return nil
}

func (fb *FileBuilder) addSymbol(b Builder) error {
	if ex, ok := fb.symbols[b.Name()]; ok {
		return fmt.Errorf("file %q already contains element (%T) named %q", fb.Name(), ex, b.Name())
	}
	fb.symbols[b.Name()] = b
	return nil
}

func (fb *FileBuilder) findFullyQualifiedElement(fqn protoreflect.FullName) Builder {
	if fb.Package != "" {
		if !strings.HasPrefix(string(fqn), string(fb.Package+".")) {
			return nil
		}
		fqn = fqn[len(fb.Package)+1:]
	}
	names := strings.Split(string(fqn), ".")
	var b Builder = fb
	for b != nil && len(names) > 0 {
		b = b.findChild(protoreflect.Name(names[0]))
		names = names[1:]
	}
	return b
}

// GetMessage returns the top-level message with the given name. If no such
// message exists in the file, nil is returned.
func (fb *FileBuilder) GetMessage(name protoreflect.Name) *MessageBuilder {
	b := fb.symbols[name]
	if mb, ok := b.(*MessageBuilder); ok {
		return mb
	}
	return nil
}

// RemoveMessage removes the top-level message with the given name. If no such
// message exists in the file, this is a no-op. This returns the file builder,
// for method chaining.
func (fb *FileBuilder) RemoveMessage(name protoreflect.Name) *FileBuilder {
	fb.TryRemoveMessage(name)
	return fb
}

// TryRemoveMessage removes the top-level message with the given name and
// returns false if the file has no such message.
func (fb *FileBuilder) TryRemoveMessage(name protoreflect.Name) bool {
	b := fb.symbols[name]
	if mb, ok := b.(*MessageBuilder); ok {
		fb.removeChild(mb)
		return true
	}
	return false
}

// AddMessage adds the given message to this file. If an error prevents the
// message from being added, this method panics. This returns the file builder,
// for method chaining.
func (fb *FileBuilder) AddMessage(mb *MessageBuilder) *FileBuilder {
	if err := fb.TryAddMessage(mb); err != nil {
		panic(err)
	}
	return fb
}

// TryAddMessage adds the given message to this file, returning any error that
// prevents the message from being added (such as a name collision with another
// element already added to the file).
func (fb *FileBuilder) TryAddMessage(mb *MessageBuilder) error {
	if err := fb.addSymbol(mb); err != nil {
		return err
	}
	Unlink(mb)
	mb.setParent(fb)
	fb.messages = append(fb.messages, mb)
	return nil
}

// GetExtension returns the top-level extension with the given name. If no such
// extension exists in the file, nil is returned.
func (fb *FileBuilder) GetExtension(name protoreflect.Name) *FieldBuilder {
	b := fb.symbols[name]
	if exb, ok := b.(*FieldBuilder); ok {
		return exb
	}
	return nil
}

// RemoveExtension removes the top-level extension with the given name. If no
// such extension exists in the file, this is a no-op. This returns the file
// builder, for method chaining.
func (fb *FileBuilder) RemoveExtension(name protoreflect.Name) *FileBuilder {
	fb.TryRemoveExtension(name)
	return fb
}

// TryRemoveExtension removes the top-level extension with the given name and
// returns false if the file has no such extension.
func (fb *FileBuilder) TryRemoveExtension(name protoreflect.Name) bool {
	b := fb.symbols[name]
	if exb, ok := b.(*FieldBuilder); ok {
		fb.removeChild(exb)
		return true
	}
	return false
}

// AddExtension adds the given extension to this file. If an error prevents the
// extension from being added, this method panics. This returns the file
// builder, for method chaining.
func (fb *FileBuilder) AddExtension(exb *FieldBuilder) *FileBuilder {
	if err := fb.TryAddExtension(exb); err != nil {
		panic(err)
	}
	return fb
}

// TryAddExtension adds the given extension to this file, returning any error
// that prevents the extension from being added (such as a name collision with
// another element already added to the file).
func (fb *FileBuilder) TryAddExtension(exb *FieldBuilder) error {
	if !exb.IsExtension() {
		return fmt.Errorf("field %s is not an extension", exb.Name())
	}
	if err := fb.addSymbol(exb); err != nil {
		return err
	}
	Unlink(exb)
	exb.setParent(fb)
	fb.extensions = append(fb.extensions, exb)
	return nil
}

// GetEnum returns the top-level enum with the given name. If no such enum
// exists in the file, nil is returned.
func (fb *FileBuilder) GetEnum(name protoreflect.Name) *EnumBuilder {
	b := fb.symbols[name]
	if eb, ok := b.(*EnumBuilder); ok {
		return eb
	}
	return nil
}

// RemoveEnum removes the top-level enum with the given name. If no such enum
// exists in the file, this is a no-op. This returns the file builder, for
// method chaining.
func (fb *FileBuilder) RemoveEnum(name protoreflect.Name) *FileBuilder {
	fb.TryRemoveEnum(name)
	return fb
}

// TryRemoveEnum removes the top-level enum with the given name and returns
// false if the file has no such enum.
func (fb *FileBuilder) TryRemoveEnum(name protoreflect.Name) bool {
	b := fb.symbols[name]
	if eb, ok := b.(*EnumBuilder); ok {
		fb.removeChild(eb)
		return true
	}
	return false
}

// AddEnum adds the given enum to this file. If an error prevents the enum from
// being added, this method panics. This returns the file builder, for method
// chaining.
func (fb *FileBuilder) AddEnum(eb *EnumBuilder) *FileBuilder {
	if err := fb.TryAddEnum(eb); err != nil {
		panic(err)
	}
	return fb
}

// TryAddEnum adds the given enum to this file, returning any error that
// prevents the enum from being added (such as a name collision with another
// element already added to the file).
func (fb *FileBuilder) TryAddEnum(eb *EnumBuilder) error {
	if err := fb.addSymbol(eb); err != nil {
		return err
	}
	Unlink(eb)
	eb.setParent(fb)
	fb.enums = append(fb.enums, eb)
	return nil
}

// GetService returns the top-level service with the given name. If no such
// service exists in the file, nil is returned.
func (fb *FileBuilder) GetService(name protoreflect.Name) *ServiceBuilder {
	b := fb.symbols[name]
	if sb, ok := b.(*ServiceBuilder); ok {
		return sb
	}
	return nil
}

// RemoveService removes the top-level service with the given name. If no such
// service exists in the file, this is a no-op. This returns the file builder,
// for method chaining.
func (fb *FileBuilder) RemoveService(name protoreflect.Name) *FileBuilder {
	fb.TryRemoveService(name)
	return fb
}

// TryRemoveService removes the top-level service with the given name and
// returns false if the file has no such service.
func (fb *FileBuilder) TryRemoveService(name protoreflect.Name) bool {
	b := fb.symbols[name]
	if sb, ok := b.(*ServiceBuilder); ok {
		fb.removeChild(sb)
		return true
	}
	return false
}

// AddService adds the given service to this file. If an error prevents the
// service from being added, this method panics. This returns the file builder,
// for method chaining.
func (fb *FileBuilder) AddService(sb *ServiceBuilder) *FileBuilder {
	if err := fb.TryAddService(sb); err != nil {
		panic(err)
	}
	return fb
}

// TryAddService adds the given service to this file, returning any error that
// prevents the service from being added (such as a name collision with another
// element already added to the file).
func (fb *FileBuilder) TryAddService(sb *ServiceBuilder) error {
	if err := fb.addSymbol(sb); err != nil {
		return err
	}
	Unlink(sb)
	sb.setParent(fb)
	fb.services = append(fb.services, sb)
	return nil
}

func (fb *FileBuilder) addExtensionsFromImport(dep protoreflect.FileDescriptor) error {
	if err := protoresolve.RegisterTypesInFile(dep, &fb.origExts, protoresolve.TypeKindExtension); err != nil {
		return err
	}
	// we also add any extensions from this dependency's "public" imports since
	// they are also visible to the importing file
	imps := dep.Imports()
	for i, length := 0, imps.Len(); i < length; i++ {
		imp := imps.Get(i)
		if !imp.IsPublic {
			continue
		}
		if err := fb.addExtensionsFromImport(imp.FileDescriptor); err != nil {
			return err
		}
	}
	return nil
}

// AddDependency adds the given file as an explicit import. Normally,
// dependencies can be inferred during the build process by finding the files
// for all referenced types (such as message and enum types used in this file).
// However, this does not work for custom options, which must be known in order
// to be interpretable. And they aren't known unless an explicit import is added
// for the file that contains the custom options.
//
// Knowledge of custom options can also be provided by using BuilderOptions with
// an [protoresolve.ExtensionTypeResolver], when building the file.
func (fb *FileBuilder) AddDependency(dep *FileBuilder) *FileBuilder {
	return fb.addDep(dep, false)
}

// AddOptionDependency adds the given file as an explicit option import. This is
// just like AddDependency but it is for options-only imports. When the file uses
// Edition 2024 or newer, these dependencies appear in a different part of the file
// descriptor so that they don't establish a hard runtime requirement for the
// dependency. In source, they use "import option" statements instead of normal
// "import" statements.
func (fb *FileBuilder) AddOptionDependency(dep *FileBuilder) *FileBuilder {
	return fb.addDep(dep, true)
}

func (fb *FileBuilder) addDep(dep *FileBuilder, optionOnly bool) *FileBuilder {
	if fb.explicitDeps == nil {
		fb.explicitDeps = map[*FileBuilder]bool{}
	}
	fb.explicitDeps[dep] = optionOnly
	return fb
}

// AddImportedDependency adds the given file as an explicit import. Normally,
// dependencies can be inferred during the build process by finding the files
// for all referenced types (such as message and enum types used in this file).
// However, this may not work for custom options, which must be known in order
// to be interpretable. And they may not be known unless an explicit import is
// added for the file that contains the custom options.
//
// Knowledge of custom options can also be provided by using BuilderOptions with
// an [protoresolve.ExtensionTypeResolver], when building the file.
func (fb *FileBuilder) AddImportedDependency(dep protoreflect.FileDescriptor) *FileBuilder {
	return fb.addImport(dep, false)
}

// AddImportedOptionDependency adds the given file as an explicit option import.
// This is just like AddImportedDependency but it is for options-only imports. When
// the file uses Edition 2024 or newer, these dependencies appear in a different
// part of the file descriptor so that they don't establish a hard runtime
// requirement for the dependency. In source, they use "import option" statements
// instead of normal "import" statements.
func (fb *FileBuilder) AddImportedOptionDependency(dep protoreflect.FileDescriptor) *FileBuilder {
	return fb.addImport(dep, true)
}

func (fb *FileBuilder) addImport(dep protoreflect.FileDescriptor, optionOnly bool) *FileBuilder {
	if fb.explicitImports == nil {
		fb.explicitImports = map[protoreflect.FileDescriptor]bool{}
	}
	fb.explicitImports[dep] = optionOnly
	return fb
}

// PruneUnusedDependencies removes all imports that are not actually used in the
// file. Note that this undoes any calls to AddDependency or AddImportedDependency
// which means that custom options may be missing from the resulting built
// descriptor unless BuilderOptions are used that include an extension resolver with
// knowledge of all custom options.
//
// When FromFile is used to create a FileBuilder from an existing descriptor, all
// imports are usually preserved in any subsequent built descriptor. But this method
// can be used to remove imports from the original file, like if mutations are made
// to the file's contents such that not all imports are needed anymore. When FromFile
// is used, any custom options present in the original descriptor will be correctly
// retained. If the file is mutated such that new custom options are added to the file,
// they may be missing unless AddImportedDependency is called after pruning OR
// BuilderOptions are used that include an ExtensionRegistry with knowledge of the
// new custom options.
func (fb *FileBuilder) PruneUnusedDependencies() *FileBuilder {
	fb.explicitImports = nil
	fb.explicitDeps = nil
	return fb
}

// SetOptions sets the file options for this file and returns the file, for
// method chaining.
func (fb *FileBuilder) SetOptions(options *descriptorpb.FileOptions) *FileBuilder {
	fb.Options = options
	return fb
}

// SetPackageName sets the name of the package for this file and returns the
// file, for method chaining.
func (fb *FileBuilder) SetPackageName(pkg protoreflect.FullName) *FileBuilder {
	fb.Package = pkg
	return fb
}

// SetSyntax sets whether this file is declared to use "proto3" syntax or not
// and returns the file, for method chaining. To set the syntax of the file
// to protoreflect.Editions, use SetEdition instead.
func (fb *FileBuilder) SetSyntax(syntax protoreflect.Syntax) *FileBuilder {
	fb.Syntax = syntax
	return fb
}

// SetEdition sets the edition for this file. Setting it to unknown, legacy,
// max, or a test-only value will result in an error when this file is built.
func (fb *FileBuilder) SetEdition(edition descriptorpb.Edition) *FileBuilder {
	fb.Syntax = protoreflect.Editions
	fb.Edition = edition
	return fb
}

func (fb *FileBuilder) buildProto(deps, optionDeps []protoreflect.FileDescriptor) (*descriptorpb.FileDescriptorProto, error) {
	filePath := fb.path
	if filePath == "" {
		filePath = uniqueFilePath()
	}
	var syntax *string
	var edition *descriptorpb.Edition
	switch fb.Syntax {
	case protoreflect.Proto3:
		syntax = proto.String("proto3")
	case protoreflect.Proto2:
		syntax = proto.String("proto2")
	case 0: // default (unset) is proto2 unless an edition was specified
		if fb.Edition == 0 {
			syntax = proto.String("proto2")
			break
		}
		fallthrough
	case protoreflect.Editions:
		switch {
		case fb.Edition < descriptorpb.Edition_EDITION_PROTO2 ||
			fb.Edition >= descriptorpb.Edition_EDITION_MAX ||
			descriptorpb.Edition_name[int32(fb.Edition)] == "" ||
			strings.HasSuffix(fb.Edition.String(), "_TEST_ONLY") ||
			// TODO: reference generated constant for this once there's a release of protobuf-go with it
			fb.Edition.String() == "EDITION_UNSTABLE":
			return nil, fmt.Errorf("builder contains unknown or invalid edition: %v", fb.Edition)
		case fb.Edition > maxSupportedEdition:
			return nil, fmt.Errorf("builder uses an edition that is not yet supported: %v", fb.Edition)
		case fb.Edition == descriptorpb.Edition_EDITION_PROTO2 && fb.Syntax == 0:
			// Edition set to proto2 instead of syntax? We'll allow it.
			syntax = proto.String("proto2")
		case fb.Edition == descriptorpb.Edition_EDITION_PROTO3 && fb.Syntax == 0:
			// Edition set to proto3 instead of syntax? We'll allow it.
			syntax = proto.String("proto3")
		case fb.Edition == descriptorpb.Edition_EDITION_PROTO2 || fb.Edition == descriptorpb.Edition_EDITION_PROTO3:
			return nil, fmt.Errorf("builder indicates syntax editions but edition %v; set syntax instead", fb.Edition)
		default:
			syntax = proto.String("editions")
			edition = fb.Edition.Enum()
		}
	default:
		return nil, fmt.Errorf("builder contains unknown syntax: %v", fb.Syntax)
	}
	var pkg *string
	if fb.Package != "" {
		pkg = proto.String(string(fb.Package))
	}

	path := make([]int32, 0, 10)
	sourceInfo := descriptorpb.SourceCodeInfo{}
	addCommentsTo(&sourceInfo, path, &fb.comments)
	addCommentsTo(&sourceInfo, append(path, internal.FileSyntaxTag), &fb.SyntaxComments)
	addCommentsTo(&sourceInfo, append(path, internal.FilePackageTag), &fb.PackageComments)

	var imports, optionImports []string
	if edition != nil && *edition >= descriptorpb.Edition_EDITION_2024 {
		// We can use option-only imports.
		imports = make([]string, 0, len(deps))
		for _, dep := range deps {
			imports = append(imports, dep.Path())
		}
		sort.Strings(imports)
		optionImports = make([]string, 0, len(optionDeps))
		for _, optionDep := range optionDeps {
			optionImports = append(optionImports, optionDep.Path())
		}
		sort.Strings(optionImports)
	} else {
		// We can't use option-only imports, so put them all together
		// as normal imports.
		imports = make([]string, 0, len(deps)+len(optionDeps))
		for _, dep := range deps {
			imports = append(imports, dep.Path())
		}
		for _, dep := range optionDeps {
			imports = append(imports, dep.Path())
		}
		sort.Strings(imports)
	}

	messages := make([]*descriptorpb.DescriptorProto, 0, len(fb.messages))
	for _, mb := range fb.messages {
		path := append(path, internal.FileMessagesTag, int32(len(messages)))
		md, err := mb.buildProto(path, &sourceInfo)
		if err != nil {
			return nil, err
		}
		messages = append(messages, md)
	}

	enums := make([]*descriptorpb.EnumDescriptorProto, 0, len(fb.enums))
	for _, eb := range fb.enums {
		path := append(path, internal.FileEnumsTag, int32(len(enums)))
		ed, err := eb.buildProto(path, &sourceInfo)
		if err != nil {
			return nil, err
		}
		enums = append(enums, ed)
	}

	extensions := make([]*descriptorpb.FieldDescriptorProto, 0, len(fb.extensions))
	for _, exb := range fb.extensions {
		path := append(path, internal.FileExtensionsTag, int32(len(extensions)))
		exd, err := exb.buildProto(path, &sourceInfo, isExtendeeMessageSet(exb))
		if err != nil {
			return nil, err
		}
		extensions = append(extensions, exd)
	}

	services := make([]*descriptorpb.ServiceDescriptorProto, 0, len(fb.services))
	for _, sb := range fb.services {
		path := append(path, internal.FileServicesTag, int32(len(services)))
		sd, err := sb.buildProto(path, &sourceInfo)
		if err != nil {
			return nil, err
		}
		services = append(services, sd)
	}

	return &descriptorpb.FileDescriptorProto{
		Name:             proto.String(filePath),
		Package:          pkg,
		Dependency:       imports,
		OptionDependency: optionImports,
		Options:          fb.Options,
		Syntax:           syntax,
		Edition:          edition,
		MessageType:      messages,
		EnumType:         enums,
		Extension:        extensions,
		Service:          services,
		SourceCodeInfo:   &sourceInfo,
	}, nil
}

func isExtendeeMessageSet(flb *FieldBuilder) bool {
	if flb.localExtendee != nil {
		return flb.localExtendee.Options.GetMessageSetWireFormat()
	}
	opts, _ := protomessage.As[*descriptorpb.MessageOptions](flb.foreignExtendee.Options())
	return opts.GetMessageSetWireFormat()
}

// Build constructs a file descriptor based on the contents of this file
// builder. If there are any problems constructing the descriptor, including
// resolving symbols referenced by the builder or failing to meet certain
// validation rules, an error is returned.
func (fb *FileBuilder) Build() (protoreflect.FileDescriptor, error) {
	fd, err := fb.BuildDescriptor()
	if err != nil {
		return nil, err
	}
	return fd.(protoreflect.FileDescriptor), nil
}

// BuildDescriptor constructs a file descriptor based on the contents of this
// file builder. Most usages will prefer Build() instead, whose return type is a
// concrete descriptor type. This method is present to satisfy the Builder
// interface.
func (fb *FileBuilder) BuildDescriptor() (protoreflect.Descriptor, error) {
	return doBuild(fb, BuilderOptions{})
}
