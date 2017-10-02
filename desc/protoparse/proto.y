%{

package protoparse

import (
	"fmt"
	"math"

	"github.com/golang/protobuf/proto"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
)

%}

// fields inside this union end up as the fields in a structure known
// as ${PREFIX}SymType, of which a reference is passed to the lexer.
%union{
	str      string
	b        bool
	i        int64
	ui       uint64
	f        float64
	u        interface{}
	sl       []interface{}
	names    []string
	agg      []*aggregate
	fd       *dpb.FileDescriptorProto
	msgd     *dpb.DescriptorProto
	fldd     *dpb.FieldDescriptorProto
	end      *dpb.EnumDescriptorProto
	envd     *dpb.EnumValueDescriptorProto
	sd       *dpb.ServiceDescriptorProto
	mtd      *dpb.MethodDescriptorProto
	opts     []*dpb.UninterpretedOption
	optNm    []*dpb.UninterpretedOption_NamePart
	imprt    *importSpec
	extend   *extendBlock
	grpd     *groupDesc
	ood      *oneofDesc
	fileDecs []*fileDecl
	msgDecs  []*msgDecl
	enDecs   []*enumDecl
	svcDecs  []*serviceDecl
	rpcType  *rpcType
	resvd    *reservedFields
	rngs     []tagRange
	extRngs  []*dpb.DescriptorProto_ExtensionRange
}

// any non-terminal which returns a value needs a type, which is
// really a field name in the above union struct
%type <str>      name syntax package ident typeIdent keyType aggName
%type <i>        negIntLit
%type <ui>       intLit
%type <f>        floatLit
%type <u>        constant scalarConstant
%type <sl>       constantList
%type <names>    fieldNames
%type <agg>      aggregate aggFields aggField
%type <fd>       file
%type <msgd>     message
%type <fldd>     field oneofField
%type <end>      enum
%type <envd>     enumField
%type <sd>       service
%type <mtd>      rpc
%type <opts>     option fieldOption fieldOptions rpcOption rpcOptions
%type <optNm>    optionName optionNameRest optionNameComponent
%type <imprt>    import
%type <extend>   extend
%type <grpd>     group mapField
%type <ood>      oneof
%type <fileDecs> fileDecl fileDecls
%type <msgDecs>  messageItem messageBody extendItem extendBody oneofItem oneofBody
%type <enDecs>   enumItem enumBody
%type <svcDecs>  serviceItem serviceBody
%type <rpcType>  rpcType
%type <resvd>    reserved
%type <rngs>     range ranges
%type <extRngs>  extensions

// same for terminals
%token <str> _SYNTAX _IMPORT _WEAK _PUBLIC _PACKAGE _OPTION _TRUE _FALSE _INF _NAN _REPEATED _OPTIONAL _REQUIRED
%token <str> _DOUBLE _FLOAT _INT32 _INT64 _UINT32 _UINT64 _SINT32 _SINT64 _FIXED32 _FIXED64 _SFIXED32 _SFIXED64
%token <str> _BOOL _STRING _BYTES _GROUP _ONEOF _MAP _EXTENSIONS _TO _MAX _RESERVED _ENUM _MESSAGE _EXTEND
%token <str> _SERVICE _RPC _STREAM _RETURNS _NAME _FQNAME _TYPENAME _STRING_LIT
%token <ui>  _INT_LIT
%token <f>   _FLOAT_LIT
%token <u>   _ERROR

%%

file : syntax {
		$$ = &dpb.FileDescriptorProto{}
		$$.Syntax = proto.String($1)
		protolex.(*protoLex).res = $$
	}
	| fileDecls  {
		$$ = fileDeclsToProto($1)
		protolex.(*protoLex).res = $$
	}
	| syntax fileDecls {
		$$ = fileDeclsToProto($2)
		$$.Syntax = proto.String($1)
		protolex.(*protoLex).res = $$
	}
	| {
		$$ = &dpb.FileDescriptorProto{}
	}

fileDecls : fileDecls fileDecl {
		$$ = append($1, $2...)
	}
	| fileDecl

fileDecl : import {
		$$ = []*fileDecl{ { importSpec: $1 } }
	}
	| package {
		$$ = []*fileDecl{ { packageName: $1 } }
	}
	| option {
		$$ = []*fileDecl{ { option: $1[0] } }
	}
	| message {
		$$ = []*fileDecl{ { message: $1 } }
	}
	| enum {
		$$ = []*fileDecl{ { enum: $1 } }
	}
	| extend {
		$$ = []*fileDecl{ { extend: $1 } }
	}
	| service {
		$$ = []*fileDecl{ { service: $1 } }
	}
	| ';' {
		$$ = nil
	}

syntax : _SYNTAX '=' _STRING_LIT ';' {
		if $3 != "proto2" && $3 != "proto3" {
			protolex.Error("syntax value must be 'proto2' or 'proto3'")
		}
		$$ = $3
	}

import : _IMPORT _STRING_LIT ';' {
		$$ = &importSpec{ name: $2 }
	}
	| _IMPORT _WEAK _STRING_LIT ';' {
		$$ = &importSpec{ name: $3, weak: true }
	}
	| _IMPORT _PUBLIC _STRING_LIT ';' {
		$$ = &importSpec{ name: $3, public: true }
	}

package : _PACKAGE ident ';' {
		$$ = $2
	}

ident : name
	| _FQNAME

option : _OPTION optionName '=' constant ';' {
		$$ = []*dpb.UninterpretedOption{asOption(protolex.(*protoLex), $2, $4)}
	}

optionName : ident {
		$$ = toNameParts($1)
	}
	| '(' typeIdent ')' {
		$$ = []*dpb.UninterpretedOption_NamePart{{NamePart: proto.String($2), IsExtension: proto.Bool(true)}}
	}
	| '(' typeIdent ')' optionNameRest {
		on := []*dpb.UninterpretedOption_NamePart{{NamePart: proto.String($2), IsExtension: proto.Bool(true)}}
		$$ = append(on, $4...)
	}

optionNameRest : optionNameComponent
	| optionNameComponent optionNameRest {
		$$ = append($1, $2...)
	}

optionNameComponent : _TYPENAME {
		$$ = toNameParts($1[1:] /* exclude leading dot */)
	}
	| '.' '(' typeIdent ')' {
		$$ = []*dpb.UninterpretedOption_NamePart{{NamePart: proto.String($3), IsExtension: proto.Bool(true)}}
	}

constant : scalarConstant
	| aggregate {
		$$ = $1
	}

scalarConstant : _STRING_LIT {
		$$ = $1
	}
	| intLit {
		$$ = $1
	}
	| negIntLit {
		$$ = $1
	}
	| floatLit {
		$$ = $1
	}
	| name {
		if $1 == "true" {
			$$ = true
		} else if $1 == "false" {
			$$ = false
		} else if $1 == "inf" {
			$$ = math.Inf(1)
		} else if $1 == "nan" {
			$$ = math.NaN()
		} else {
			$$ = identifier($1)
		}
	}

intLit : _INT_LIT
	| '+' _INT_LIT {
		$$ = $2
	}

negIntLit : '-' _INT_LIT {
		if $2 > math.MaxInt64 + 1 {
			protolex.Error(fmt.Sprintf("numeric constant %d would underflow (allowed range is %d to %d)", $2, int64(math.MinInt64), int64(math.MaxInt64)))
		}
		$$ = -int64($2)
	}

floatLit : _FLOAT_LIT
	| '-' _FLOAT_LIT {
		$$ = -$2
	}
	| '+' _FLOAT_LIT {
		$$ = $2
	}
	| '+' _INF {
		$$ = math.Inf(1)
	}
	| '-' _INF {
		$$ = math.Inf(-1)
	}

aggregate : '{' aggFields '}' {
		$$ = $2
	}

aggFields : aggField
	| aggFields aggField {
		$$ = append($1, $2...)
	}
	| aggFields ',' aggField {
		$$ = append($1, $3...)
	}
	| {
		$$ = nil
	}

aggField : aggName ':' scalarConstant {
		$$ = []*aggregate{{name: $1, val: $3}}
	}
	| aggName ':' '[' ']' {
		$$ = []*aggregate{{name: $1, val: []interface{}(nil)}}
	}
	| aggName ':' '[' constantList ']' {
		$$ = []*aggregate{{name: $1, val: $4}}
	}
	| aggName ':' aggregate {
		$$ = []*aggregate{{name: $1, val: $3}}
	}
	| aggName aggregate {
		$$ = []*aggregate{{name: $1, val: $2}}
	}
	| aggName ':' '<' aggFields '>' {
		$$ = []*aggregate{{name: $1, val: $4}}
	}
	| aggName '<' aggFields '>' {
		$$ = []*aggregate{{name: $1, val: $3}}
	}

aggName : _NAME
	| '[' ident ']' {
		$$ = "[" + $2 + "]"
	}

constantList : constant {
		$$ = []interface{}{ $1 }
	}
	| constantList ',' constant {
		$$ = append($1, $3)
	}
	| '<' aggFields '>' {
		$$ = []interface{}{ $2 }
	}
	| constantList ','  '<' aggFields '>' {
		$$ = append($1, $4)
	}

typeIdent : ident
	| _TYPENAME

field : _REQUIRED typeIdent name '=' _INT_LIT ';' {
		checkTag(protolex, $5)
		$$ = asFieldDescriptor(dpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), $2, $3, int32($5), nil)
	}
	| _OPTIONAL typeIdent name '=' _INT_LIT ';' {
		checkTag(protolex, $5)
		$$ = asFieldDescriptor(dpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), $2, $3, int32($5), nil)
	}
	| _REPEATED typeIdent name '=' _INT_LIT ';' {
		checkTag(protolex, $5)
		$$ = asFieldDescriptor(dpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), $2, $3, int32($5), nil)
	}
	| typeIdent name '=' _INT_LIT ';' {
		checkTag(protolex, $4)
		$$ = asFieldDescriptor(nil, $1, $2, int32($4), nil)
	}
	| _REQUIRED typeIdent name '=' _INT_LIT '[' fieldOptions ']' ';' {
		checkTag(protolex, $5)
		$$ = asFieldDescriptor(dpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), $2, $3, int32($5), $7)
	}
	| _OPTIONAL typeIdent name '=' _INT_LIT '[' fieldOptions ']' ';' {
		checkTag(protolex, $5)
		$$ = asFieldDescriptor(dpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), $2, $3, int32($5), $7)
	}
	| _REPEATED typeIdent name '=' _INT_LIT '[' fieldOptions ']' ';' {
		checkTag(protolex, $5)
		$$ = asFieldDescriptor(dpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), $2, $3, int32($5), $7)
	}
	| typeIdent name '=' _INT_LIT '[' fieldOptions ']' ';' {
		checkTag(protolex, $4)
		$$ = asFieldDescriptor(nil, $1, $2, int32($4), $6)
	}

fieldOptions : fieldOptions ',' fieldOption {
		$$ = append($1, $3...)
	}
	| fieldOption

fieldOption: optionName '=' constant {
		$$ = []*dpb.UninterpretedOption{asOption(protolex.(*protoLex), $1, $3)}
	}

group : _REQUIRED _GROUP name '=' _INT_LIT '{' messageBody '}' {
		checkTag(protolex, $5)
		$$ = asGroupDescriptor(protolex, dpb.FieldDescriptorProto_LABEL_REQUIRED, $3, int32($5), $7)
	}
	| _OPTIONAL _GROUP name '=' _INT_LIT '{' messageBody '}' {
		checkTag(protolex, $5)
		$$ = asGroupDescriptor(protolex, dpb.FieldDescriptorProto_LABEL_OPTIONAL, $3, int32($5), $7)
	}
	| _REPEATED _GROUP name '=' _INT_LIT '{' messageBody '}' {
		checkTag(protolex, $5)
		$$ = asGroupDescriptor(protolex, dpb.FieldDescriptorProto_LABEL_REPEATED, $3, int32($5), $7)
	}

oneof : _ONEOF name '{' oneofBody '}' {
		if len($4) == 0 {
			protolex.Error(fmt.Sprintf("oneof must contain at least one field"))
		}
		$$ = &oneofDesc{name: $2}
		for _, i := range $4 {
			if i.fld != nil {
				$$.fields = append($$.fields, i.fld)
			} else if i.option != nil {
				$$.options = append($$.options, i.option)
			}
		}
	}

oneofBody : oneofBody oneofItem {
		$$ = append($1, $2...)
	}
	| oneofItem
	| {
		$$ = nil
	}

oneofItem : option {
		$$ = []*msgDecl{ { option: $1[0] } }
	}
	| oneofField {
		$$ = []*msgDecl{ { fld: $1 } }
	}
	| ';' {
		$$ = nil
	}

oneofField : typeIdent name '=' _INT_LIT ';' {
		checkTag(protolex, $4)
		$$ = asFieldDescriptor(nil, $1, $2, int32($4), nil)
	}
	| typeIdent name '=' _INT_LIT '[' fieldOptions ']' ';' {
		checkTag(protolex, $4)
		$$ = asFieldDescriptor(nil, $1, $2, int32($4), $6)
	}

mapField : _MAP '<' keyType ',' typeIdent '>' name '=' _INT_LIT ';' {
		checkTag(protolex, $9)
		$$ = asMapField($3, $5, $7, int32($9), nil)
	}
	| _MAP '<' keyType ',' typeIdent '>' name '=' _INT_LIT '[' fieldOptions ']' ';' {
		checkTag(protolex, $9)
		$$ = asMapField($3, $5, $7, int32($9), $11)
	}

keyType : _INT32
	| _INT64
	| _UINT32
	| _UINT64
	| _SINT32
	| _SINT64
	| _FIXED32
	| _FIXED64
	| _SFIXED32
	| _SFIXED64
	| _BOOL
	| _STRING

extensions : _EXTENSIONS ranges ';' {
		$$ = asExtensionRanges($2, nil)
	}
	| _EXTENSIONS ranges '[' fieldOptions ']' ';' {
		$$ = asExtensionRanges($2, $4)
	}

ranges : ranges ',' range {
		$$ = append($1, $3...)
	}
	| range

range : _INT_LIT {
		if $1 > maxTag {
			protolex.Error(fmt.Sprintf("range includes out-of-range tag: %d (should be between 0 and %d)", $1, maxTag))
		}
		$$ = []tagRange{{Start: int32($1), End: int32($1)+1}}
	}
	| _INT_LIT _TO _INT_LIT {
		if $1 > maxTag {
			protolex.Error(fmt.Sprintf("range start is out-of-range tag: %d (should be between 0 and %d)", $1, maxTag))
		}
		if $3 > maxTag {
			protolex.Error(fmt.Sprintf("range end is out-of-range tag: %d (should be between 0 and %d)", $3, maxTag))
		}
		if $1 > $3 {
			protolex.Error(fmt.Sprintf("range, %d to %d, is invalid: start must be <= end", $1, $3))
		}
		$$ = []tagRange{{Start: int32($1), End: int32($3)+1}}
	}
	| _INT_LIT _TO _MAX {
		if $1 > maxTag {
			protolex.Error(fmt.Sprintf("range start is out-of-range tag: %d (should be between 0 and %d)", $1, maxTag))
		}
		$$ = []tagRange{{Start: int32($1), End: maxTag+1}}
	}

reserved : _RESERVED ranges ';' {
		$$ = &reservedFields{ tags: $2 }
	}
	| _RESERVED fieldNames ';' {
		rsvd := map[string]struct{}{}
		for _, n := range $2 {
			if _, ok := rsvd[n]; ok {
				protolex.Error(fmt.Sprintf("field %q is reserved multiple times", n))
				break
			}
			rsvd[n] = struct{}{}
		}
		$$ = &reservedFields{ names: $2 }
	}

fieldNames : fieldNames ',' _STRING_LIT {
		$$ = append($1, $3)
	}
	| _STRING_LIT {
		$$ = []string{$1}
	}

enum : _ENUM name '{' enumBody '}' {
		if len($4) == 0 {
			protolex.Error(fmt.Sprintf("enums must define at least one value"))
		}
		$$ = enumDeclsToProto($2, $4)
	}

enumBody : enumBody enumItem {
		$$ = append($1, $2...)
	}
	| enumItem
	| {
		$$ = nil
	}

enumItem : option {
		$$ = []*enumDecl{{ option: $1[0] }}
	}
	| enumField {
		$$ = []*enumDecl{{ val: $1 }}
	}
	| ';' {
		$$ = nil
	}

enumField : name '=' intLit ';' {
		checkUint64InInt32Range(protolex, $3)
		$$ = asEnumValue($1, int32($3), nil)
	}
	|  name '=' intLit '[' fieldOptions ']' ';' {
		checkUint64InInt32Range(protolex, $3)
		$$ = asEnumValue($1, int32($3), $5)
	}
	| name '=' negIntLit ';' {
		checkInt64InInt32Range(protolex, $3)
		$$ = asEnumValue($1, int32($3), nil)
	}
	|  name '=' negIntLit '[' fieldOptions ']' ';' {
		checkInt64InInt32Range(protolex, $3)
		$$ = asEnumValue($1, int32($3), $5)
	}

message : _MESSAGE name '{' messageBody '}' {
		$$ = msgDeclsToProto($2, $4)
	}

messageBody : messageBody messageItem {
		$$ = append($1, $2...)
	}
	| messageItem
	| {
		$$ = nil
	}

messageItem : field {
		$$ = []*msgDecl{{ fld: $1 }}
	}
	| enum {
		$$ = []*msgDecl{{ enum: $1 }}
	}
	| message {
		$$ = []*msgDecl{{ msg: $1 }}
	}
	| extend {
		$$ = []*msgDecl{{ extend: $1 }}
	}
	| extensions {
		$$ = []*msgDecl{{ extensions: $1 }}
	}
	| group {
		$$ = []*msgDecl{{ grp: $1 }}
	}
	| option {
		$$ = []*msgDecl{{ option: $1[0] }}
	}
	| oneof {
		$$ = []*msgDecl{{ oneof: $1 }}
	}
	| mapField {
		$$ = []*msgDecl{{ grp: $1 }}
	}
	| reserved {
		$$ = []*msgDecl{{ reserved: $1 }}
	}
	| ';' {
		$$ = nil
	}

extend : _EXTEND typeIdent '{' extendBody '}' {
		if len($4) == 0 {
			protolex.Error(fmt.Sprintf("extend sections must define at least one extension"))
		}
		$$ = &extendBlock{}
		for _, i := range $4 {
			var fd *dpb.FieldDescriptorProto
			if i.fld != nil {
				fd = i.fld
			} else if i.grp != nil {
				fd = i.grp.field
				$$.msgs = append($$.msgs, i.grp.msg)
			}
			fd.Extendee = proto.String($2)
			$$.fields = append($$.fields, fd)
		}
	}

extendBody : extendBody extendItem {
		$$ = append($1, $2...)
	}
	| extendItem
	| {
		$$ = nil
	}

extendItem : field {
		$$ = []*msgDecl{{ fld: $1 }}
	}
	| group {
		$$ = []*msgDecl{{ grp: $1 }}
	}
	| ';' {
		$$ = nil
	}

service : _SERVICE name '{' serviceBody '}' {
		$$ = svcDeclsToProto($2, $4)
	}

serviceBody : serviceBody serviceItem {
		$$ = append($1, $2...)
	}
	| serviceItem
	| {
		$$ = nil
	}

// NB: doc suggests support for "stream" declaration, separate from "rpc", but
// it does not appear to be supported in protoc (doc is likely from grammar for
// Google-internal version of protoc, with support for streaming stubby)
serviceItem : option {
		$$ = []*serviceDecl{{option: $1[0]}}
	}
	| rpc {
		$$ = []*serviceDecl{{rpc: $1}}
	}
	| ';' {
		$$ = nil
	}

rpc : _RPC name '(' rpcType ')' _RETURNS '(' rpcType ')' ';' {
		$$ = asMethodDescriptor($2, $4, $8, nil)
	}
	| _RPC name '(' rpcType ')' _RETURNS '(' rpcType ')' '{' rpcOptions '}' {
		$$ = asMethodDescriptor($2, $4, $8, $11)
	}

rpcType : _STREAM typeIdent {
		$$ = &rpcType{msgType: $2, stream: true}
	}
	| typeIdent {
		$$ = &rpcType{msgType: $1}
	}

rpcOptions : rpcOptions rpcOption {
		$$ = append($1, $2...)
	}
	| rpcOption
	| {
		$$ = nil
	}

rpcOption : option {
		$$ = $1
	}
	| ';' {
		$$ = nil
	}

name : _NAME
	| _SYNTAX
	| _IMPORT
	| _WEAK
	| _PUBLIC
	| _PACKAGE
	| _OPTION
	| _TRUE
	| _FALSE
	| _INF
	| _NAN
	| _REPEATED
	| _OPTIONAL
	| _REQUIRED
	| _DOUBLE
	| _FLOAT
	| _INT32
	| _INT64
	| _UINT32
	| _UINT64
	| _SINT32
	| _SINT64
	| _FIXED32
	| _FIXED64
	| _SFIXED32
	| _SFIXED64
	| _BOOL
	| _STRING
	| _BYTES
	| _GROUP
	| _ONEOF
	| _MAP
	| _EXTENSIONS
	| _TO
	| _MAX
	| _RESERVED
	| _ENUM
	| _MESSAGE
	| _EXTEND
	| _SERVICE
	| _RPC
	| _STREAM
	| _RETURNS

%%
