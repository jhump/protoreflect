package internal

import (
	"unicode"
	"unicode/utf8"

	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
)

const (
	MaxTag = 536870911 // 2^29 - 1

	SpecialReservedStart = 19000
	SpecialReservedEnd   = 19999

	// NB: It would be nice to use constants from generated code instead of hard-coding these here.
	// But code-gen does not emit these as constants anywhere. The only places they appear in generated
	// code are struct tags on fields of the generated descriptor protos.
	File_packageTag           = 2
	File_dependencyTag        = 3
	File_messagesTag          = 4
	File_enumsTag             = 5
	File_servicesTag          = 6
	File_extensionsTag        = 7
	File_syntaxTag            = 12
	Message_nameTag           = 1
	Message_fieldsTag         = 2
	Message_nestedMessagesTag = 3
	Message_enumsTag          = 4
	Message_extensionRangeTag = 5
	Message_extensionsTag     = 6
	Message_oneOfsTag         = 8
	Message_reservedRangeTag  = 9
	Message_reservedNameTag   = 10
	Field_nameTag             = 1
	Field_extendeeTag         = 2
	Field_numberTag           = 3
	Field_labelTag            = 4
	Field_typeTag             = 5
	OneOf_nameTag             = 1
	Enum_nameTag              = 1
	Enum_valuesTag            = 2
	EnumVal_nameTag           = 1
	EnumVal_numberTag         = 2
	Service_nameTag           = 1
	Service_methodsTag        = 2
	Method_nameTag            = 1
	Method_inputTag           = 2
	Method_outputTag          = 3
)

func JsonName(name string) string {
	var js []rune
	nextUpper := false
	for i, r := range name {
		if r == '_' {
			nextUpper = true
			continue
		}
		if i == 0 {
			js = append(js, r)
		} else if nextUpper {
			nextUpper = false
			js = append(js, unicode.ToUpper(r))
		} else {
			js = append(js, r)
		}
	}
	return string(js)
}

func InitCap(name string) string {
	r, sz := utf8.DecodeRuneInString(name)
	return string(unicode.ToUpper(r)) + name[sz:]
}

type SourceInfoMap map[interface{}]*dpb.SourceCodeInfo_Location

func (m SourceInfoMap) Get(path []int32) *dpb.SourceCodeInfo_Location {
	return m[asMapKey(path)]
}

func (m SourceInfoMap) Put(path []int32, loc *dpb.SourceCodeInfo_Location) {
	m[asMapKey(path)] = loc
}

func asMapKey(slice []int32) interface{} {
	// NB: arrays should be usable as map keys, but this does not
	// work due to a bug: https://github.com/golang/go/issues/22605
	//rv := reflect.ValueOf(slice)
	//arrayType := reflect.ArrayOf(rv.Len(), rv.Type().Elem())
	//array := reflect.New(arrayType).Elem()
	//reflect.Copy(array, rv)
	//return array.Interface()

	b := make([]byte, len(slice)*4)
	for i, s := range slice {
		j := i * 4
		b[j] = byte(s)
		b[j+1] = byte(s >> 8)
		b[j+2] = byte(s >> 16)
		b[j+3] = byte(s >> 24)
	}
	return string(b)
}

func CreateSourceInfoMap(fd *dpb.FileDescriptorProto) SourceInfoMap {
	res := SourceInfoMap{}
	for _, l := range fd.GetSourceCodeInfo().GetLocation() {
		res.Put(l.Path, l)
	}
	return res
}
