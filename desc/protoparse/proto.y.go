//line proto.y:2
package protoparse

import __yyfmt__ "fmt"

//line proto.y:3
import (
	"fmt"
	"math"

	"github.com/golang/protobuf/proto"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
)

//line proto.y:17
type protoSymType struct {
	yys      int
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

const _SYNTAX = 57346
const _IMPORT = 57347
const _WEAK = 57348
const _PUBLIC = 57349
const _PACKAGE = 57350
const _OPTION = 57351
const _TRUE = 57352
const _FALSE = 57353
const _INF = 57354
const _NAN = 57355
const _REPEATED = 57356
const _OPTIONAL = 57357
const _REQUIRED = 57358
const _DOUBLE = 57359
const _FLOAT = 57360
const _INT32 = 57361
const _INT64 = 57362
const _UINT32 = 57363
const _UINT64 = 57364
const _SINT32 = 57365
const _SINT64 = 57366
const _FIXED32 = 57367
const _FIXED64 = 57368
const _SFIXED32 = 57369
const _SFIXED64 = 57370
const _BOOL = 57371
const _STRING = 57372
const _BYTES = 57373
const _GROUP = 57374
const _ONEOF = 57375
const _MAP = 57376
const _EXTENSIONS = 57377
const _TO = 57378
const _MAX = 57379
const _RESERVED = 57380
const _ENUM = 57381
const _MESSAGE = 57382
const _EXTEND = 57383
const _SERVICE = 57384
const _RPC = 57385
const _STREAM = 57386
const _RETURNS = 57387
const _NAME = 57388
const _FQNAME = 57389
const _TYPENAME = 57390
const _STRING_LIT = 57391
const _INT_LIT = 57392
const _FLOAT_LIT = 57393
const _ERROR = 57394

var protoToknames = [...]string{
	"$end",
	"error",
	"$unk",
	"_SYNTAX",
	"_IMPORT",
	"_WEAK",
	"_PUBLIC",
	"_PACKAGE",
	"_OPTION",
	"_TRUE",
	"_FALSE",
	"_INF",
	"_NAN",
	"_REPEATED",
	"_OPTIONAL",
	"_REQUIRED",
	"_DOUBLE",
	"_FLOAT",
	"_INT32",
	"_INT64",
	"_UINT32",
	"_UINT64",
	"_SINT32",
	"_SINT64",
	"_FIXED32",
	"_FIXED64",
	"_SFIXED32",
	"_SFIXED64",
	"_BOOL",
	"_STRING",
	"_BYTES",
	"_GROUP",
	"_ONEOF",
	"_MAP",
	"_EXTENSIONS",
	"_TO",
	"_MAX",
	"_RESERVED",
	"_ENUM",
	"_MESSAGE",
	"_EXTEND",
	"_SERVICE",
	"_RPC",
	"_STREAM",
	"_RETURNS",
	"_NAME",
	"_FQNAME",
	"_TYPENAME",
	"_STRING_LIT",
	"_INT_LIT",
	"_FLOAT_LIT",
	"_ERROR",
	"';'",
	"'='",
	"'('",
	"')'",
	"'.'",
	"'+'",
	"'-'",
	"'{'",
	"'}'",
	"','",
	"':'",
	"'['",
	"']'",
	"'<'",
	"'>'",
}
var protoStatenames = [...]string{}

const protoEofCode = 1
const protoErrCode = 2
const protoInitialStackSize = 16

//line proto.y:706

//line yacctab:1
var protoExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
}

const protoPrivate = 57344

const protoLast = 1595

var protoAct = [...]int{

	118, 8, 268, 8, 8, 361, 250, 79, 126, 111,
	251, 257, 178, 110, 98, 28, 97, 100, 101, 154,
	117, 147, 8, 27, 74, 136, 153, 164, 78, 96,
	112, 177, 142, 76, 77, 156, 81, 252, 104, 156,
	315, 196, 286, 156, 198, 364, 286, 156, 181, 353,
	241, 195, 262, 157, 73, 195, 346, 157, 156, 195,
	321, 157, 228, 195, 300, 157, 286, 286, 276, 339,
	337, 229, 286, 193, 195, 335, 157, 286, 286, 286,
	328, 317, 316, 298, 88, 286, 297, 209, 287, 354,
	342, 137, 156, 148, 307, 309, 211, 304, 210, 306,
	355, 343, 301, 103, 284, 308, 140, 266, 305, 356,
	157, 16, 144, 302, 264, 285, 357, 227, 267, 303,
	169, 105, 143, 213, 333, 265, 211, 16, 92, 232,
	233, 234, 170, 172, 174, 91, 137, 90, 78, 74,
	89, 352, 176, 77, 76, 151, 14, 148, 180, 15,
	16, 140, 295, 277, 109, 150, 201, 16, 16, 344,
	314, 186, 144, 190, 288, 199, 182, 192, 191, 73,
	197, 363, 143, 194, 189, 166, 248, 247, 246, 365,
	18, 17, 19, 20, 167, 245, 202, 203, 204, 205,
	206, 207, 151, 200, 13, 4, 14, 244, 243, 15,
	16, 363, 150, 318, 208, 230, 231, 187, 87, 23,
	242, 238, 103, 236, 258, 235, 367, 358, 74, 349,
	348, 347, 261, 341, 253, 240, 332, 331, 312, 152,
	18, 17, 19, 20, 95, 94, 93, 86, 83, 255,
	351, 163, 329, 270, 13, 160, 184, 179, 25, 26,
	283, 282, 254, 103, 281, 280, 279, 258, 278, 161,
	194, 158, 179, 249, 263, 261, 275, 273, 290, 85,
	84, 292, 293, 74, 294, 74, 82, 291, 296, 161,
	162, 212, 116, 158, 159, 115, 11, 3, 11, 11,
	21, 24, 310, 74, 74, 194, 121, 311, 146, 113,
	10, 299, 10, 10, 103, 135, 256, 11, 141, 322,
	74, 119, 324, 74, 103, 326, 74, 323, 313, 194,
	325, 10, 120, 327, 6, 165, 114, 9, 319, 9,
	9, 330, 360, 169, 5, 169, 345, 169, 22, 149,
	12, 138, 270, 259, 1, 183, 272, 334, 9, 102,
	350, 74, 155, 214, 194, 7, 22, 2, 362, 0,
	0, 362, 359, 74, 0, 0, 366, 31, 32, 33,
	34, 35, 36, 37, 38, 39, 40, 41, 42, 43,
	44, 45, 46, 47, 48, 49, 50, 51, 52, 53,
	54, 55, 56, 57, 58, 59, 60, 61, 62, 63,
	64, 65, 66, 67, 68, 69, 70, 71, 72, 30,
	0, 0, 99, 105, 108, 0, 0, 0, 0, 0,
	0, 106, 107, 104, 0, 0, 0, 0, 271, 274,
	31, 32, 33, 34, 35, 36, 37, 38, 39, 40,
	41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
	51, 52, 53, 54, 55, 56, 57, 58, 59, 60,
	61, 62, 63, 64, 65, 66, 67, 68, 69, 70,
	71, 72, 30, 0, 0, 99, 105, 108, 0, 0,
	0, 0, 0, 0, 106, 107, 104, 0, 0, 0,
	237, 0, 239, 31, 32, 33, 34, 35, 36, 37,
	38, 39, 40, 41, 42, 43, 44, 45, 46, 47,
	48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
	58, 59, 60, 61, 62, 63, 64, 65, 66, 67,
	68, 69, 70, 71, 72, 30, 0, 0, 99, 105,
	108, 0, 0, 0, 0, 0, 0, 106, 107, 104,
	0, 0, 0, 0, 0, 320, 31, 32, 33, 34,
	35, 36, 37, 38, 39, 40, 41, 42, 43, 44,
	45, 46, 47, 48, 49, 50, 51, 52, 53, 54,
	55, 56, 57, 58, 59, 60, 61, 62, 63, 64,
	65, 66, 67, 68, 69, 70, 71, 72, 30, 0,
	0, 99, 105, 108, 0, 0, 0, 0, 0, 0,
	106, 107, 104, 31, 32, 33, 34, 35, 131, 37,
	38, 39, 40, 125, 124, 123, 44, 45, 46, 47,
	48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
	58, 59, 132, 133, 130, 63, 64, 134, 127, 128,
	129, 69, 70, 71, 72, 30, 29, 80, 0, 0,
	0, 0, 122, 0, 0, 0, 0, 0, 0, 0,
	340, 31, 32, 33, 34, 35, 131, 37, 38, 39,
	40, 125, 124, 123, 44, 45, 46, 47, 48, 49,
	50, 51, 52, 53, 54, 55, 56, 57, 58, 59,
	132, 133, 130, 63, 64, 134, 127, 128, 129, 69,
	70, 71, 72, 30, 29, 80, 0, 0, 0, 0,
	122, 0, 0, 0, 0, 0, 0, 0, 338, 31,
	32, 33, 34, 35, 131, 37, 38, 39, 40, 125,
	124, 123, 44, 45, 46, 47, 48, 49, 50, 51,
	52, 53, 54, 55, 56, 57, 58, 59, 132, 133,
	130, 63, 64, 134, 127, 128, 129, 69, 70, 71,
	72, 30, 29, 80, 0, 0, 0, 0, 122, 0,
	0, 0, 0, 0, 0, 0, 336, 31, 32, 33,
	34, 35, 131, 37, 38, 39, 40, 41, 42, 43,
	44, 45, 46, 47, 48, 49, 50, 51, 52, 53,
	54, 55, 56, 57, 58, 59, 60, 61, 62, 63,
	64, 65, 66, 67, 68, 69, 70, 71, 72, 30,
	29, 80, 0, 0, 0, 0, 260, 0, 0, 0,
	0, 0, 0, 0, 289, 31, 32, 33, 34, 35,
	36, 37, 38, 39, 40, 125, 124, 123, 44, 45,
	46, 47, 48, 49, 50, 51, 52, 53, 54, 55,
	56, 57, 58, 59, 60, 61, 62, 63, 64, 65,
	66, 67, 68, 69, 70, 71, 72, 30, 29, 80,
	0, 0, 0, 0, 145, 0, 0, 0, 0, 0,
	0, 0, 188, 31, 32, 33, 34, 35, 131, 37,
	38, 39, 40, 125, 124, 123, 44, 45, 46, 47,
	48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
	58, 59, 132, 133, 130, 63, 64, 134, 127, 128,
	129, 69, 70, 71, 72, 30, 29, 80, 0, 0,
	0, 0, 122, 0, 0, 0, 0, 0, 0, 0,
	168, 31, 32, 33, 34, 35, 131, 37, 38, 39,
	40, 41, 42, 43, 44, 45, 46, 47, 48, 49,
	50, 51, 52, 53, 54, 55, 56, 57, 58, 59,
	60, 61, 62, 63, 64, 65, 66, 67, 68, 69,
	70, 71, 72, 30, 0, 0, 0, 0, 0, 0,
	139, 0, 0, 0, 0, 0, 0, 0, 185, 31,
	32, 33, 34, 35, 36, 37, 38, 39, 40, 41,
	42, 43, 44, 45, 46, 47, 48, 49, 50, 51,
	52, 53, 54, 55, 56, 57, 58, 59, 60, 61,
	62, 63, 64, 65, 66, 67, 68, 69, 70, 71,
	72, 30, 29, 0, 0, 0, 0, 0, 0, 0,
	75, 31, 32, 33, 34, 35, 131, 37, 38, 39,
	40, 125, 124, 123, 44, 45, 46, 47, 48, 49,
	50, 51, 52, 53, 54, 55, 56, 57, 58, 59,
	132, 133, 130, 63, 64, 134, 127, 128, 129, 69,
	70, 71, 72, 30, 29, 80, 0, 0, 0, 0,
	122, 31, 32, 33, 34, 35, 131, 37, 38, 39,
	40, 41, 42, 43, 44, 45, 46, 47, 48, 49,
	50, 51, 52, 53, 54, 55, 56, 57, 58, 59,
	60, 61, 62, 63, 64, 65, 66, 67, 68, 69,
	70, 71, 72, 30, 29, 80, 0, 0, 0, 0,
	260, 31, 32, 33, 34, 35, 36, 37, 38, 39,
	40, 125, 124, 123, 44, 45, 46, 47, 48, 49,
	50, 51, 52, 53, 54, 55, 56, 57, 58, 59,
	60, 61, 62, 63, 64, 65, 66, 67, 68, 69,
	70, 71, 72, 30, 29, 80, 0, 0, 0, 0,
	145, 31, 32, 33, 34, 35, 131, 37, 38, 39,
	40, 41, 42, 43, 44, 45, 46, 47, 48, 49,
	50, 51, 52, 53, 54, 55, 56, 57, 58, 59,
	60, 61, 62, 63, 64, 65, 66, 67, 68, 69,
	70, 71, 72, 30, 0, 0, 0, 0, 0, 0,
	139, 31, 32, 33, 34, 35, 36, 37, 38, 39,
	40, 41, 42, 43, 44, 45, 46, 47, 48, 49,
	50, 51, 52, 53, 54, 55, 56, 57, 58, 59,
	60, 61, 62, 63, 64, 65, 66, 67, 68, 69,
	70, 269, 72, 30, 29, 80, 31, 32, 33, 34,
	35, 36, 37, 38, 39, 40, 41, 42, 43, 44,
	45, 46, 47, 48, 49, 50, 51, 52, 53, 54,
	55, 56, 57, 58, 59, 60, 61, 62, 63, 64,
	65, 66, 67, 68, 69, 70, 71, 72, 30, 29,
	80, 31, 32, 33, 34, 35, 36, 37, 38, 39,
	40, 41, 42, 43, 44, 45, 46, 47, 48, 49,
	50, 51, 52, 53, 54, 55, 56, 57, 58, 175,
	60, 61, 62, 63, 64, 65, 66, 67, 68, 69,
	70, 71, 72, 30, 29, 80, 31, 32, 33, 34,
	35, 36, 37, 38, 39, 40, 41, 42, 43, 44,
	45, 46, 47, 48, 49, 50, 51, 52, 53, 54,
	55, 56, 57, 58, 173, 60, 61, 62, 63, 64,
	65, 66, 67, 68, 69, 70, 71, 72, 30, 29,
	80, 31, 32, 33, 34, 35, 36, 37, 38, 39,
	40, 41, 42, 43, 44, 45, 46, 47, 48, 49,
	50, 51, 52, 53, 54, 55, 56, 57, 58, 171,
	60, 61, 62, 63, 64, 65, 66, 67, 68, 69,
	70, 71, 72, 30, 29, 80, 31, 32, 33, 34,
	35, 36, 37, 38, 39, 40, 41, 42, 43, 44,
	45, 46, 47, 48, 49, 50, 51, 52, 53, 54,
	55, 56, 57, 58, 59, 60, 61, 62, 63, 64,
	65, 66, 67, 68, 69, 70, 71, 72, 30, 29,
	31, 32, 33, 34, 35, 36, 37, 38, 39, 40,
	41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
	51, 52, 53, 54, 55, 56, 57, 58, 59, 60,
	61, 62, 63, 64, 65, 66, 67, 68, 69, 70,
	71, 72, 30, 215, 216, 217, 218, 219, 220, 221,
	222, 223, 224, 225, 226,
}
var protoPact = [...]int{

	191, -1000, 141, 141, 155, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, 242, 1492, 1015, 1536, 1536, 1312,
	1536, 141, -1000, 227, 185, 221, 220, 184, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, 154, -1000, 1312, 80, 77, 75, -1000,
	-1000, 68, 183, -1000, 182, 181, -1000, 552, 98, 1067,
	1217, 1167, 149, -1000, -1000, -1000, 176, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, 46, -1000, 233, 229, -1000, 127,
	899, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, 1447, 1402, 1357, 1536, 1536, 1536, 1312,
	212, 1015, 1536, -18, 197, 957, -1000, -1000, -1000, -1000,
	153, 841, -1000, -1000, -1000, -1000, 102, -1000, -1000, -1000,
	-1000, 1536, -1000, 12, -1000, -22, -1000, 1492, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, 127, -1000, 101, -1000, -1000,
	1536, 1536, 1536, 1536, 1536, 1536, 150, 34, -1000, 245,
	63, 1564, 64, 9, -1000, -1000, -1000, 71, -1000, -1000,
	-1000, -1000, 76, -1000, -1000, 46, 426, -1000, 46, -15,
	-1000, 1312, 144, 143, 131, 124, 123, 122, 213, -1000,
	1015, 212, 202, 1117, -10, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, 215,
	61, 54, 211, 209, 1267, -1000, -1000, 363, -1000, 46,
	1, -1000, 97, 208, 206, 205, 204, 201, 200, 51,
	23, -1000, 110, -1000, -1000, -1000, 783, -1000, -1000, -1000,
	-1000, 1536, 1312, -1000, -1000, 1015, -1000, 1015, 96, 1312,
	-1000, -1000, 21, -1000, 46, -3, -1000, -1000, 49, 59,
	44, 39, 41, 35, -1000, 1015, 1015, 175, 552, -1000,
	-1000, 106, -27, 17, 16, 158, -1000, -1000, 489, -7,
	-1000, -1000, 1015, 1067, -1000, 1015, 1067, -1000, 1015, 1067,
	15, -1000, -1000, -1000, 192, 1536, 174, 173, 69, -1000,
	46, -1000, 10, 725, 5, 667, 4, 609, 170, 37,
	105, -1000, -1000, 1267, -11, 168, -1000, 167, -1000, 166,
	-1000, -1000, -1000, 1015, 190, 85, -1000, -1000, -1000, -1000,
	-16, 36, 56, 164, -1000, 1015, -1000, 148, -1000, -20,
	118, -1000, -1000, -1000, 163, -1000, -1000, -1000,
}
var protoPgo = [...]int{

	0, 15, 357, 355, 7, 8, 353, 352, 18, 17,
	349, 29, 16, 346, 345, 14, 26, 19, 344, 326,
	30, 343, 299, 341, 340, 339, 0, 10, 6, 5,
	332, 37, 27, 325, 324, 285, 20, 322, 311, 334,
	287, 9, 13, 32, 308, 11, 306, 25, 305, 21,
	298, 2, 296, 12, 31, 282,
}
var protoR1 = [...]int{

	0, 18, 18, 18, 18, 40, 40, 39, 39, 39,
	39, 39, 39, 39, 39, 2, 34, 34, 34, 3,
	4, 4, 26, 31, 31, 31, 32, 32, 33, 33,
	11, 11, 12, 12, 12, 12, 12, 9, 9, 8,
	10, 10, 10, 10, 10, 15, 16, 16, 16, 16,
	17, 17, 17, 17, 17, 17, 17, 7, 7, 13,
	13, 13, 13, 5, 5, 20, 20, 20, 20, 20,
	20, 20, 20, 28, 28, 27, 36, 36, 36, 38,
	46, 46, 46, 45, 45, 45, 21, 21, 37, 37,
	6, 6, 6, 6, 6, 6, 6, 6, 6, 6,
	6, 6, 55, 55, 54, 54, 53, 53, 53, 52,
	52, 14, 14, 22, 48, 48, 48, 47, 47, 47,
	23, 23, 23, 23, 19, 42, 42, 42, 41, 41,
	41, 41, 41, 41, 41, 41, 41, 41, 41, 35,
	44, 44, 44, 43, 43, 43, 24, 50, 50, 50,
	49, 49, 49, 25, 25, 51, 51, 30, 30, 30,
	29, 29, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1,
}
var protoR2 = [...]int{

	0, 1, 1, 2, 0, 2, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 4, 3, 4, 4, 3,
	1, 1, 5, 1, 3, 4, 1, 2, 1, 4,
	1, 1, 1, 1, 1, 1, 1, 1, 2, 2,
	1, 2, 2, 2, 2, 3, 1, 2, 3, 0,
	3, 4, 5, 3, 2, 5, 4, 1, 3, 1,
	3, 3, 5, 1, 1, 6, 6, 6, 5, 9,
	9, 9, 8, 3, 1, 3, 8, 8, 8, 5,
	2, 1, 0, 1, 1, 1, 5, 8, 10, 13,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 3, 6, 3, 1, 1, 3, 3, 3,
	3, 3, 1, 5, 2, 1, 0, 1, 1, 1,
	4, 7, 4, 7, 5, 2, 1, 0, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 5,
	2, 1, 0, 1, 1, 1, 5, 2, 1, 0,
	1, 1, 1, 10, 12, 2, 1, 2, 1, 0,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1,
}
var protoChk = [...]int{

	-1000, -18, -2, -40, 4, -39, -34, -3, -26, -19,
	-22, -35, -24, 53, 5, 8, 9, 40, 39, 41,
	42, -40, -39, 54, 49, 6, 7, -4, -1, 47,
	46, 4, 5, 6, 7, 8, 9, 10, 11, 12,
	13, 14, 15, 16, 17, 18, 19, 20, 21, 22,
	23, 24, 25, 26, 27, 28, 29, 30, 31, 32,
	33, 34, 35, 36, 37, 38, 39, 40, 41, 42,
	43, 44, 45, -31, -4, 55, -1, -1, -5, -4,
	48, -1, 49, 53, 49, 49, 53, 54, -5, 60,
	60, 60, 60, 53, 53, 53, -11, -12, -15, 49,
	-9, -8, -10, -1, 60, 50, 58, 59, 51, 56,
	-42, -41, -20, -22, -19, -35, -55, -36, -26, -38,
	-37, -52, 53, 16, 15, 14, -5, 39, 40, 41,
	35, 9, 33, 34, 38, -48, -47, -26, -23, 53,
	-1, -44, -43, -20, -36, 53, -50, -49, -26, -25,
	53, 43, 53, -16, -17, -7, 46, 64, 50, 51,
	12, 50, 51, 12, -32, -33, 48, 57, 61, -41,
	-5, 32, -5, 32, -5, 32, -1, -54, -53, 50,
	-1, 66, -54, -14, 49, 61, -47, 54, 61, -43,
	61, -49, -1, 61, -17, 62, 63, -15, 66, -4,
	-32, 55, -1, -1, -1, -1, -1, -1, 54, 53,
	64, 62, 36, 60, -6, 19, 20, 21, 22, 23,
	24, 25, 26, 27, 28, 29, 30, 53, 53, 62,
	-9, -8, 58, 59, 55, -17, -12, 64, -15, 66,
	-16, 65, -5, 54, 54, 54, 54, 54, 54, 50,
	-28, -27, -31, -53, 50, 37, -46, -45, -26, -21,
	53, -5, 62, 49, 53, 64, 53, 64, -51, 44,
	-5, 65, -13, -11, 66, -16, 67, 56, 50, 50,
	50, 50, 50, 50, 53, 64, 62, 65, 54, 61,
	-45, -1, -5, -28, -28, 56, -5, 65, 62, -16,
	67, 53, 64, 60, 53, 64, 60, 53, 64, 60,
	-28, -27, 53, -11, 54, 67, 65, 65, 45, -11,
	66, 67, -28, -42, -28, -42, -28, -42, 65, 50,
	-1, 53, 53, 55, -16, 65, 61, 65, 61, 65,
	61, 53, 53, 64, 54, -51, 67, 53, 53, 53,
	-28, 50, 56, 65, 53, 64, 53, 60, 53, -28,
	-30, -29, -26, 53, 65, 61, -29, 53,
}
var protoDef = [...]int{

	4, -2, 1, 2, 0, 6, 7, 8, 9, 10,
	11, 12, 13, 14, 0, 0, 0, 0, 0, 0,
	0, 3, 5, 0, 0, 0, 0, 0, 20, 21,
	162, 163, 164, 165, 166, 167, 168, 169, 170, 171,
	172, 173, 174, 175, 176, 177, 178, 179, 180, 181,
	182, 183, 184, 185, 186, 187, 188, 189, 190, 191,
	192, 193, 194, 195, 196, 197, 198, 199, 200, 201,
	202, 203, 204, 0, 23, 0, 0, 0, 0, 63,
	64, 0, 0, 16, 0, 0, 19, 0, 0, 127,
	116, 142, 149, 15, 17, 18, 0, 30, 31, 32,
	33, 34, 35, 36, 49, 37, 0, 0, 40, 24,
	0, 126, 128, 129, 130, 131, 132, 133, 134, 135,
	136, 137, 138, 0, 0, 0, 0, 0, 0, 0,
	194, 168, 0, 193, 197, 0, 115, 117, 118, 119,
	0, 0, 141, 143, 144, 145, 0, 148, 150, 151,
	152, 0, 22, 0, 46, 0, 57, 0, 38, 42,
	43, 39, 41, 44, 25, 26, 28, 0, 124, 125,
	0, 0, 0, 0, 0, 0, 0, 0, 105, 106,
	0, 0, 0, 0, 112, 113, 114, 0, 139, 140,
	146, 147, 0, 45, 47, 0, 0, 54, 49, 0,
	27, 0, 0, 0, 0, 0, 0, 0, 0, 102,
	0, 0, 0, 82, 0, 90, 91, 92, 93, 94,
	95, 96, 97, 98, 99, 100, 101, 109, 110, 0,
	0, 0, 0, 0, 0, 48, 50, 0, 53, 49,
	0, 58, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 74, 0, 104, 107, 108, 0, 81, 83, 84,
	85, 0, 0, 111, 120, 0, 122, 0, 0, 203,
	156, 51, 0, 59, 49, 0, 56, 29, 0, 0,
	0, 0, 0, 0, 68, 0, 0, 0, 0, 79,
	80, 0, 0, 0, 0, 0, 155, 52, 0, 0,
	55, 65, 0, 127, 66, 0, 127, 67, 0, 127,
	0, 73, 103, 75, 0, 0, 0, 0, 0, 60,
	49, 61, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 121, 123, 0, 0, 0, 76, 0, 77, 0,
	78, 72, 86, 0, 0, 0, 62, 69, 70, 71,
	0, 0, 0, 0, 88, 0, 153, 159, 87, 0,
	0, 158, 160, 161, 0, 154, 157, 89,
}
var protoTok1 = [...]int{

	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	55, 56, 3, 58, 62, 59, 57, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 63, 53,
	66, 54, 67, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 64, 3, 65, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 60, 3, 61,
}
var protoTok2 = [...]int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
	22, 23, 24, 25, 26, 27, 28, 29, 30, 31,
	32, 33, 34, 35, 36, 37, 38, 39, 40, 41,
	42, 43, 44, 45, 46, 47, 48, 49, 50, 51,
	52,
}
var protoTok3 = [...]int{
	0,
}

var protoErrorMessages = [...]struct {
	state int
	token int
	msg   string
}{}

//line yaccpar:1

/*	parser for yacc output	*/

var (
	protoDebug        = 0
	protoErrorVerbose = false
)

type protoLexer interface {
	Lex(lval *protoSymType) int
	Error(s string)
}

type protoParser interface {
	Parse(protoLexer) int
	Lookahead() int
}

type protoParserImpl struct {
	lval  protoSymType
	stack [protoInitialStackSize]protoSymType
	char  int
}

func (p *protoParserImpl) Lookahead() int {
	return p.char
}

func protoNewParser() protoParser {
	return &protoParserImpl{}
}

const protoFlag = -1000

func protoTokname(c int) string {
	if c >= 1 && c-1 < len(protoToknames) {
		if protoToknames[c-1] != "" {
			return protoToknames[c-1]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func protoStatname(s int) string {
	if s >= 0 && s < len(protoStatenames) {
		if protoStatenames[s] != "" {
			return protoStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func protoErrorMessage(state, lookAhead int) string {
	const TOKSTART = 4

	if !protoErrorVerbose {
		return "syntax error"
	}

	for _, e := range protoErrorMessages {
		if e.state == state && e.token == lookAhead {
			return "syntax error: " + e.msg
		}
	}

	res := "syntax error: unexpected " + protoTokname(lookAhead)

	// To match Bison, suggest at most four expected tokens.
	expected := make([]int, 0, 4)

	// Look for shiftable tokens.
	base := protoPact[state]
	for tok := TOKSTART; tok-1 < len(protoToknames); tok++ {
		if n := base + tok; n >= 0 && n < protoLast && protoChk[protoAct[n]] == tok {
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}
	}

	if protoDef[state] == -2 {
		i := 0
		for protoExca[i] != -1 || protoExca[i+1] != state {
			i += 2
		}

		// Look for tokens that we accept or reduce.
		for i += 2; protoExca[i] >= 0; i += 2 {
			tok := protoExca[i]
			if tok < TOKSTART || protoExca[i+1] == 0 {
				continue
			}
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}

		// If the default action is to accept or reduce, give up.
		if protoExca[i+1] != 0 {
			return res
		}
	}

	for i, tok := range expected {
		if i == 0 {
			res += ", expecting "
		} else {
			res += " or "
		}
		res += protoTokname(tok)
	}
	return res
}

func protolex1(lex protoLexer, lval *protoSymType) (char, token int) {
	token = 0
	char = lex.Lex(lval)
	if char <= 0 {
		token = protoTok1[0]
		goto out
	}
	if char < len(protoTok1) {
		token = protoTok1[char]
		goto out
	}
	if char >= protoPrivate {
		if char < protoPrivate+len(protoTok2) {
			token = protoTok2[char-protoPrivate]
			goto out
		}
	}
	for i := 0; i < len(protoTok3); i += 2 {
		token = protoTok3[i+0]
		if token == char {
			token = protoTok3[i+1]
			goto out
		}
	}

out:
	if token == 0 {
		token = protoTok2[1] /* unknown char */
	}
	if protoDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", protoTokname(token), uint(char))
	}
	return char, token
}

func protoParse(protolex protoLexer) int {
	return protoNewParser().Parse(protolex)
}

func (protorcvr *protoParserImpl) Parse(protolex protoLexer) int {
	var proton int
	var protoVAL protoSymType
	var protoDollar []protoSymType
	_ = protoDollar // silence set and not used
	protoS := protorcvr.stack[:]

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	protostate := 0
	protorcvr.char = -1
	prototoken := -1 // protorcvr.char translated into internal numbering
	defer func() {
		// Make sure we report no lookahead when not parsing.
		protostate = -1
		protorcvr.char = -1
		prototoken = -1
	}()
	protop := -1
	goto protostack

ret0:
	return 0

ret1:
	return 1

protostack:
	/* put a state and value onto the stack */
	if protoDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", protoTokname(prototoken), protoStatname(protostate))
	}

	protop++
	if protop >= len(protoS) {
		nyys := make([]protoSymType, len(protoS)*2)
		copy(nyys, protoS)
		protoS = nyys
	}
	protoS[protop] = protoVAL
	protoS[protop].yys = protostate

protonewstate:
	proton = protoPact[protostate]
	if proton <= protoFlag {
		goto protodefault /* simple state */
	}
	if protorcvr.char < 0 {
		protorcvr.char, prototoken = protolex1(protolex, &protorcvr.lval)
	}
	proton += prototoken
	if proton < 0 || proton >= protoLast {
		goto protodefault
	}
	proton = protoAct[proton]
	if protoChk[proton] == prototoken { /* valid shift */
		protorcvr.char = -1
		prototoken = -1
		protoVAL = protorcvr.lval
		protostate = proton
		if Errflag > 0 {
			Errflag--
		}
		goto protostack
	}

protodefault:
	/* default state action */
	proton = protoDef[protostate]
	if proton == -2 {
		if protorcvr.char < 0 {
			protorcvr.char, prototoken = protolex1(protolex, &protorcvr.lval)
		}

		/* look through exception table */
		xi := 0
		for {
			if protoExca[xi+0] == -1 && protoExca[xi+1] == protostate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			proton = protoExca[xi+0]
			if proton < 0 || proton == prototoken {
				break
			}
		}
		proton = protoExca[xi+1]
		if proton < 0 {
			goto ret0
		}
	}
	if proton == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			protolex.Error(protoErrorMessage(protostate, prototoken))
			Nerrs++
			if protoDebug >= 1 {
				__yyfmt__.Printf("%s", protoStatname(protostate))
				__yyfmt__.Printf(" saw %s\n", protoTokname(prototoken))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for protop >= 0 {
				proton = protoPact[protoS[protop].yys] + protoErrCode
				if proton >= 0 && proton < protoLast {
					protostate = protoAct[proton] /* simulate a shift of "error" */
					if protoChk[protostate] == protoErrCode {
						goto protostack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if protoDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", protoS[protop].yys)
				}
				protop--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if protoDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", protoTokname(prototoken))
			}
			if prototoken == protoEofCode {
				goto ret1
			}
			protorcvr.char = -1
			prototoken = -1
			goto protonewstate /* try again in the same state */
		}
	}

	/* reduction by production proton */
	if protoDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", proton, protoStatname(protostate))
	}

	protont := proton
	protopt := protop
	_ = protopt // guard against "declared and not used"

	protop -= protoR2[proton]
	// protop is now the index of $0. Perform the default action. Iff the
	// reduced production is ε, $1 is possibly out of range.
	if protop+1 >= len(protoS) {
		nyys := make([]protoSymType, len(protoS)*2)
		copy(nyys, protoS)
		protoS = nyys
	}
	protoVAL = protoS[protop+1]

	/* consult goto table to find next state */
	proton = protoR1[proton]
	protog := protoPgo[proton]
	protoj := protog + protoS[protop].yys + 1

	if protoj >= protoLast {
		protostate = protoAct[protog]
	} else {
		protostate = protoAct[protoj]
		if protoChk[protostate] != -proton {
			protostate = protoAct[protog]
		}
	}
	// dummy call; replaced with literal code
	switch protont {

	case 1:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:93
		{
			protoVAL.fd = &dpb.FileDescriptorProto{}
			protoVAL.fd.Syntax = proto.String(protoDollar[1].str)
			protolex.(*protoLex).res = protoVAL.fd
		}
	case 2:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:98
		{
			protoVAL.fd = fileDeclsToProto(protoDollar[1].fileDecs)
			protolex.(*protoLex).res = protoVAL.fd
		}
	case 3:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:102
		{
			protoVAL.fd = fileDeclsToProto(protoDollar[2].fileDecs)
			protoVAL.fd.Syntax = proto.String(protoDollar[1].str)
			protolex.(*protoLex).res = protoVAL.fd
		}
	case 4:
		protoDollar = protoS[protopt-0 : protopt+1]
		//line proto.y:107
		{
			protoVAL.fd = &dpb.FileDescriptorProto{}
		}
	case 5:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:111
		{
			protoVAL.fileDecs = append(protoDollar[1].fileDecs, protoDollar[2].fileDecs...)
		}
	case 7:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:116
		{
			protoVAL.fileDecs = []*fileDecl{{importSpec: protoDollar[1].imprt}}
		}
	case 8:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:119
		{
			protoVAL.fileDecs = []*fileDecl{{packageName: protoDollar[1].str}}
		}
	case 9:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:122
		{
			protoVAL.fileDecs = []*fileDecl{{option: protoDollar[1].opts[0]}}
		}
	case 10:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:125
		{
			protoVAL.fileDecs = []*fileDecl{{message: protoDollar[1].msgd}}
		}
	case 11:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:128
		{
			protoVAL.fileDecs = []*fileDecl{{enum: protoDollar[1].end}}
		}
	case 12:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:131
		{
			protoVAL.fileDecs = []*fileDecl{{extend: protoDollar[1].extend}}
		}
	case 13:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:134
		{
			protoVAL.fileDecs = []*fileDecl{{service: protoDollar[1].sd}}
		}
	case 14:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:137
		{
			protoVAL.fileDecs = nil
		}
	case 15:
		protoDollar = protoS[protopt-4 : protopt+1]
		//line proto.y:141
		{
			if protoDollar[3].str != "proto2" && protoDollar[3].str != "proto3" {
				protolex.Error("syntax value must be 'proto2' or 'proto3'")
			}
			protoVAL.str = protoDollar[3].str
		}
	case 16:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:148
		{
			protoVAL.imprt = &importSpec{name: protoDollar[2].str}
		}
	case 17:
		protoDollar = protoS[protopt-4 : protopt+1]
		//line proto.y:151
		{
			protoVAL.imprt = &importSpec{name: protoDollar[3].str, weak: true}
		}
	case 18:
		protoDollar = protoS[protopt-4 : protopt+1]
		//line proto.y:154
		{
			protoVAL.imprt = &importSpec{name: protoDollar[3].str, public: true}
		}
	case 19:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:158
		{
			protoVAL.str = protoDollar[2].str
		}
	case 22:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:165
		{
			protoVAL.opts = []*dpb.UninterpretedOption{asOption(protolex.(*protoLex), protoDollar[2].optNm, protoDollar[4].u)}
		}
	case 23:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:169
		{
			protoVAL.optNm = toNameParts(protoDollar[1].str)
		}
	case 24:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:172
		{
			protoVAL.optNm = []*dpb.UninterpretedOption_NamePart{{NamePart: proto.String(protoDollar[2].str), IsExtension: proto.Bool(true)}}
		}
	case 25:
		protoDollar = protoS[protopt-4 : protopt+1]
		//line proto.y:175
		{
			on := []*dpb.UninterpretedOption_NamePart{{NamePart: proto.String(protoDollar[2].str), IsExtension: proto.Bool(true)}}
			protoVAL.optNm = append(on, protoDollar[4].optNm...)
		}
	case 27:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:181
		{
			protoVAL.optNm = append(protoDollar[1].optNm, protoDollar[2].optNm...)
		}
	case 28:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:185
		{
			protoVAL.optNm = toNameParts(protoDollar[1].str[1:] /* exclude leading dot */)
		}
	case 29:
		protoDollar = protoS[protopt-4 : protopt+1]
		//line proto.y:188
		{
			protoVAL.optNm = []*dpb.UninterpretedOption_NamePart{{NamePart: proto.String(protoDollar[3].str), IsExtension: proto.Bool(true)}}
		}
	case 31:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:193
		{
			protoVAL.u = protoDollar[1].agg
		}
	case 32:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:197
		{
			protoVAL.u = protoDollar[1].str
		}
	case 33:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:200
		{
			protoVAL.u = protoDollar[1].ui
		}
	case 34:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:203
		{
			protoVAL.u = protoDollar[1].i
		}
	case 35:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:206
		{
			protoVAL.u = protoDollar[1].f
		}
	case 36:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:209
		{
			if protoDollar[1].str == "true" {
				protoVAL.u = true
			} else if protoDollar[1].str == "false" {
				protoVAL.u = false
			} else if protoDollar[1].str == "inf" {
				protoVAL.u = math.Inf(1)
			} else if protoDollar[1].str == "nan" {
				protoVAL.u = math.NaN()
			} else {
				protoVAL.u = identifier(protoDollar[1].str)
			}
		}
	case 38:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:224
		{
			protoVAL.ui = protoDollar[2].ui
		}
	case 39:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:228
		{
			if protoDollar[2].ui > math.MaxInt64+1 {
				protolex.Error(fmt.Sprintf("numeric constant %d would underflow (allowed range is %d to %d)", protoDollar[2].ui, int64(math.MinInt64), int64(math.MaxInt64)))
			}
			protoVAL.i = -int64(protoDollar[2].ui)
		}
	case 41:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:236
		{
			protoVAL.f = -protoDollar[2].f
		}
	case 42:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:239
		{
			protoVAL.f = protoDollar[2].f
		}
	case 43:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:242
		{
			protoVAL.f = math.Inf(1)
		}
	case 44:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:245
		{
			protoVAL.f = math.Inf(-1)
		}
	case 45:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:249
		{
			protoVAL.agg = protoDollar[2].agg
		}
	case 47:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:254
		{
			protoVAL.agg = append(protoDollar[1].agg, protoDollar[2].agg...)
		}
	case 48:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:257
		{
			protoVAL.agg = append(protoDollar[1].agg, protoDollar[3].agg...)
		}
	case 49:
		protoDollar = protoS[protopt-0 : protopt+1]
		//line proto.y:260
		{
			protoVAL.agg = nil
		}
	case 50:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:264
		{
			protoVAL.agg = []*aggregate{{name: protoDollar[1].str, val: protoDollar[3].u}}
		}
	case 51:
		protoDollar = protoS[protopt-4 : protopt+1]
		//line proto.y:267
		{
			protoVAL.agg = []*aggregate{{name: protoDollar[1].str, val: []interface{}(nil)}}
		}
	case 52:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:270
		{
			protoVAL.agg = []*aggregate{{name: protoDollar[1].str, val: protoDollar[4].sl}}
		}
	case 53:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:273
		{
			protoVAL.agg = []*aggregate{{name: protoDollar[1].str, val: protoDollar[3].agg}}
		}
	case 54:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:276
		{
			protoVAL.agg = []*aggregate{{name: protoDollar[1].str, val: protoDollar[2].agg}}
		}
	case 55:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:279
		{
			protoVAL.agg = []*aggregate{{name: protoDollar[1].str, val: protoDollar[4].agg}}
		}
	case 56:
		protoDollar = protoS[protopt-4 : protopt+1]
		//line proto.y:282
		{
			protoVAL.agg = []*aggregate{{name: protoDollar[1].str, val: protoDollar[3].agg}}
		}
	case 58:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:287
		{
			protoVAL.str = "[" + protoDollar[2].str + "]"
		}
	case 59:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:291
		{
			protoVAL.sl = []interface{}{protoDollar[1].u}
		}
	case 60:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:294
		{
			protoVAL.sl = append(protoDollar[1].sl, protoDollar[3].u)
		}
	case 61:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:297
		{
			protoVAL.sl = []interface{}{protoDollar[2].agg}
		}
	case 62:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:300
		{
			protoVAL.sl = append(protoDollar[1].sl, protoDollar[4].agg)
		}
	case 65:
		protoDollar = protoS[protopt-6 : protopt+1]
		//line proto.y:307
		{
			checkTag(protolex, protoDollar[5].ui)
			protoVAL.fldd = asFieldDescriptor(dpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), protoDollar[2].str, protoDollar[3].str, int32(protoDollar[5].ui), nil)
		}
	case 66:
		protoDollar = protoS[protopt-6 : protopt+1]
		//line proto.y:311
		{
			checkTag(protolex, protoDollar[5].ui)
			protoVAL.fldd = asFieldDescriptor(dpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), protoDollar[2].str, protoDollar[3].str, int32(protoDollar[5].ui), nil)
		}
	case 67:
		protoDollar = protoS[protopt-6 : protopt+1]
		//line proto.y:315
		{
			checkTag(protolex, protoDollar[5].ui)
			protoVAL.fldd = asFieldDescriptor(dpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), protoDollar[2].str, protoDollar[3].str, int32(protoDollar[5].ui), nil)
		}
	case 68:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:319
		{
			checkTag(protolex, protoDollar[4].ui)
			protoVAL.fldd = asFieldDescriptor(nil, protoDollar[1].str, protoDollar[2].str, int32(protoDollar[4].ui), nil)
		}
	case 69:
		protoDollar = protoS[protopt-9 : protopt+1]
		//line proto.y:323
		{
			checkTag(protolex, protoDollar[5].ui)
			protoVAL.fldd = asFieldDescriptor(dpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), protoDollar[2].str, protoDollar[3].str, int32(protoDollar[5].ui), protoDollar[7].opts)
		}
	case 70:
		protoDollar = protoS[protopt-9 : protopt+1]
		//line proto.y:327
		{
			checkTag(protolex, protoDollar[5].ui)
			protoVAL.fldd = asFieldDescriptor(dpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), protoDollar[2].str, protoDollar[3].str, int32(protoDollar[5].ui), protoDollar[7].opts)
		}
	case 71:
		protoDollar = protoS[protopt-9 : protopt+1]
		//line proto.y:331
		{
			checkTag(protolex, protoDollar[5].ui)
			protoVAL.fldd = asFieldDescriptor(dpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), protoDollar[2].str, protoDollar[3].str, int32(protoDollar[5].ui), protoDollar[7].opts)
		}
	case 72:
		protoDollar = protoS[protopt-8 : protopt+1]
		//line proto.y:335
		{
			checkTag(protolex, protoDollar[4].ui)
			protoVAL.fldd = asFieldDescriptor(nil, protoDollar[1].str, protoDollar[2].str, int32(protoDollar[4].ui), protoDollar[6].opts)
		}
	case 73:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:340
		{
			protoVAL.opts = append(protoDollar[1].opts, protoDollar[3].opts...)
		}
	case 75:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:345
		{
			protoVAL.opts = []*dpb.UninterpretedOption{asOption(protolex.(*protoLex), protoDollar[1].optNm, protoDollar[3].u)}
		}
	case 76:
		protoDollar = protoS[protopt-8 : protopt+1]
		//line proto.y:349
		{
			checkTag(protolex, protoDollar[5].ui)
			protoVAL.grpd = asGroupDescriptor(dpb.FieldDescriptorProto_LABEL_REQUIRED, protoDollar[3].str, int32(protoDollar[5].ui), protoDollar[7].msgDecs)
		}
	case 77:
		protoDollar = protoS[protopt-8 : protopt+1]
		//line proto.y:353
		{
			checkTag(protolex, protoDollar[5].ui)
			protoVAL.grpd = asGroupDescriptor(dpb.FieldDescriptorProto_LABEL_OPTIONAL, protoDollar[3].str, int32(protoDollar[5].ui), protoDollar[7].msgDecs)
		}
	case 78:
		protoDollar = protoS[protopt-8 : protopt+1]
		//line proto.y:357
		{
			checkTag(protolex, protoDollar[5].ui)
			protoVAL.grpd = asGroupDescriptor(dpb.FieldDescriptorProto_LABEL_REPEATED, protoDollar[3].str, int32(protoDollar[5].ui), protoDollar[7].msgDecs)
		}
	case 79:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:362
		{
			if len(protoDollar[4].msgDecs) == 0 {
				protolex.Error(fmt.Sprintf("oneof must contain at least one field"))
			}
			protoVAL.ood = &oneofDesc{name: protoDollar[2].str}
			for _, i := range protoDollar[4].msgDecs {
				if i.fld != nil {
					protoVAL.ood.fields = append(protoVAL.ood.fields, i.fld)
				} else if i.option != nil {
					protoVAL.ood.options = append(protoVAL.ood.options, i.option)
				}
			}
		}
	case 80:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:376
		{
			protoVAL.msgDecs = append(protoDollar[1].msgDecs, protoDollar[2].msgDecs...)
		}
	case 82:
		protoDollar = protoS[protopt-0 : protopt+1]
		//line proto.y:380
		{
			protoVAL.msgDecs = nil
		}
	case 83:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:384
		{
			protoVAL.msgDecs = []*msgDecl{{option: protoDollar[1].opts[0]}}
		}
	case 84:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:387
		{
			protoVAL.msgDecs = []*msgDecl{{fld: protoDollar[1].fldd}}
		}
	case 85:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:390
		{
			protoVAL.msgDecs = nil
		}
	case 86:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:394
		{
			checkTag(protolex, protoDollar[4].ui)
			protoVAL.fldd = asFieldDescriptor(nil, protoDollar[1].str, protoDollar[2].str, int32(protoDollar[4].ui), nil)
		}
	case 87:
		protoDollar = protoS[protopt-8 : protopt+1]
		//line proto.y:398
		{
			checkTag(protolex, protoDollar[4].ui)
			protoVAL.fldd = asFieldDescriptor(nil, protoDollar[1].str, protoDollar[2].str, int32(protoDollar[4].ui), protoDollar[6].opts)
		}
	case 88:
		protoDollar = protoS[protopt-10 : protopt+1]
		//line proto.y:403
		{
			checkTag(protolex, protoDollar[9].ui)
			protoVAL.grpd = asMapField(protoDollar[3].str, protoDollar[5].str, protoDollar[7].str, int32(protoDollar[9].ui), nil)
		}
	case 89:
		protoDollar = protoS[protopt-13 : protopt+1]
		//line proto.y:407
		{
			checkTag(protolex, protoDollar[9].ui)
			protoVAL.grpd = asMapField(protoDollar[3].str, protoDollar[5].str, protoDollar[7].str, int32(protoDollar[9].ui), protoDollar[11].opts)
		}
	case 102:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:425
		{
			protoVAL.extRngs = asExtensionRanges(protoDollar[2].rngs, nil)
		}
	case 103:
		protoDollar = protoS[protopt-6 : protopt+1]
		//line proto.y:428
		{
			protoVAL.extRngs = asExtensionRanges(protoDollar[2].rngs, protoDollar[4].opts)
		}
	case 104:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:432
		{
			protoVAL.rngs = append(protoDollar[1].rngs, protoDollar[3].rngs...)
		}
	case 106:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:437
		{
			if protoDollar[1].ui > maxTag {
				protolex.Error(fmt.Sprintf("range includes out-of-range tag: %d (should be between 0 and %d)", protoDollar[1].ui, maxTag))
			}
			protoVAL.rngs = []tagRange{{Start: int32(protoDollar[1].ui), End: int32(protoDollar[1].ui) + 1}}
		}
	case 107:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:443
		{
			if protoDollar[1].ui > maxTag {
				protolex.Error(fmt.Sprintf("range start is out-of-range tag: %d (should be between 0 and %d)", protoDollar[1].ui, maxTag))
			}
			if protoDollar[3].ui > maxTag {
				protolex.Error(fmt.Sprintf("range end is out-of-range tag: %d (should be between 0 and %d)", protoDollar[3].ui, maxTag))
			}
			if protoDollar[1].ui > protoDollar[3].ui {
				protolex.Error(fmt.Sprintf("range, %d to %d, is invalid: start must be <= end", protoDollar[1].ui, protoDollar[3].ui))
			}
			protoVAL.rngs = []tagRange{{Start: int32(protoDollar[1].ui), End: int32(protoDollar[3].ui) + 1}}
		}
	case 108:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:455
		{
			if protoDollar[1].ui > maxTag {
				protolex.Error(fmt.Sprintf("range start is out-of-range tag: %d (should be between 0 and %d)", protoDollar[1].ui, maxTag))
			}
			protoVAL.rngs = []tagRange{{Start: int32(protoDollar[1].ui), End: maxTag + 1}}
		}
	case 109:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:462
		{
			protoVAL.resvd = &reservedFields{tags: protoDollar[2].rngs}
		}
	case 110:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:465
		{
			rsvd := map[string]struct{}{}
			for _, n := range protoDollar[2].names {
				if _, ok := rsvd[n]; ok {
					protolex.Error(fmt.Sprintf("field %q is reserved multiple times", n))
					break
				}
				rsvd[n] = struct{}{}
			}
			protoVAL.resvd = &reservedFields{names: protoDollar[2].names}
		}
	case 111:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:477
		{
			protoVAL.names = append(protoDollar[1].names, protoDollar[3].str)
		}
	case 112:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:480
		{
			protoVAL.names = []string{protoDollar[1].str}
		}
	case 113:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:484
		{
			if len(protoDollar[4].enDecs) == 0 {
				protolex.Error(fmt.Sprintf("enums must define at least one value"))
			}
			protoVAL.end = enumDeclsToProto(protoDollar[2].str, protoDollar[4].enDecs)
		}
	case 114:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:491
		{
			protoVAL.enDecs = append(protoDollar[1].enDecs, protoDollar[2].enDecs...)
		}
	case 116:
		protoDollar = protoS[protopt-0 : protopt+1]
		//line proto.y:495
		{
			protoVAL.enDecs = nil
		}
	case 117:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:499
		{
			protoVAL.enDecs = []*enumDecl{{option: protoDollar[1].opts[0]}}
		}
	case 118:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:502
		{
			protoVAL.enDecs = []*enumDecl{{val: protoDollar[1].envd}}
		}
	case 119:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:505
		{
			protoVAL.enDecs = nil
		}
	case 120:
		protoDollar = protoS[protopt-4 : protopt+1]
		//line proto.y:509
		{
			checkUint64InInt32Range(protolex, protoDollar[3].ui)
			protoVAL.envd = asEnumValue(protoDollar[1].str, int32(protoDollar[3].ui), nil)
		}
	case 121:
		protoDollar = protoS[protopt-7 : protopt+1]
		//line proto.y:513
		{
			checkUint64InInt32Range(protolex, protoDollar[3].ui)
			protoVAL.envd = asEnumValue(protoDollar[1].str, int32(protoDollar[3].ui), protoDollar[5].opts)
		}
	case 122:
		protoDollar = protoS[protopt-4 : protopt+1]
		//line proto.y:517
		{
			checkInt64InInt32Range(protolex, protoDollar[3].i)
			protoVAL.envd = asEnumValue(protoDollar[1].str, int32(protoDollar[3].i), nil)
		}
	case 123:
		protoDollar = protoS[protopt-7 : protopt+1]
		//line proto.y:521
		{
			checkInt64InInt32Range(protolex, protoDollar[3].i)
			protoVAL.envd = asEnumValue(protoDollar[1].str, int32(protoDollar[3].i), protoDollar[5].opts)
		}
	case 124:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:526
		{
			protoVAL.msgd = msgDeclsToProto(protoDollar[2].str, protoDollar[4].msgDecs)
		}
	case 125:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:530
		{
			protoVAL.msgDecs = append(protoDollar[1].msgDecs, protoDollar[2].msgDecs...)
		}
	case 127:
		protoDollar = protoS[protopt-0 : protopt+1]
		//line proto.y:534
		{
			protoVAL.msgDecs = nil
		}
	case 128:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:538
		{
			protoVAL.msgDecs = []*msgDecl{{fld: protoDollar[1].fldd}}
		}
	case 129:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:541
		{
			protoVAL.msgDecs = []*msgDecl{{enum: protoDollar[1].end}}
		}
	case 130:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:544
		{
			protoVAL.msgDecs = []*msgDecl{{msg: protoDollar[1].msgd}}
		}
	case 131:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:547
		{
			protoVAL.msgDecs = []*msgDecl{{extend: protoDollar[1].extend}}
		}
	case 132:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:550
		{
			protoVAL.msgDecs = []*msgDecl{{extensions: protoDollar[1].extRngs}}
		}
	case 133:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:553
		{
			protoVAL.msgDecs = []*msgDecl{{grp: protoDollar[1].grpd}}
		}
	case 134:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:556
		{
			protoVAL.msgDecs = []*msgDecl{{option: protoDollar[1].opts[0]}}
		}
	case 135:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:559
		{
			protoVAL.msgDecs = []*msgDecl{{oneof: protoDollar[1].ood}}
		}
	case 136:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:562
		{
			protoVAL.msgDecs = []*msgDecl{{grp: protoDollar[1].grpd}}
		}
	case 137:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:565
		{
			protoVAL.msgDecs = []*msgDecl{{reserved: protoDollar[1].resvd}}
		}
	case 138:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:568
		{
			protoVAL.msgDecs = nil
		}
	case 139:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:572
		{
			if len(protoDollar[4].msgDecs) == 0 {
				protolex.Error(fmt.Sprintf("extend sections must define at least one extension"))
			}
			protoVAL.extend = &extendBlock{}
			for _, i := range protoDollar[4].msgDecs {
				var fd *dpb.FieldDescriptorProto
				if i.fld != nil {
					fd = i.fld
				} else if i.grp != nil {
					fd = i.grp.field
					protoVAL.extend.msgs = append(protoVAL.extend.msgs, i.grp.msg)
				}
				fd.Extendee = proto.String(protoDollar[2].str)
				protoVAL.extend.fields = append(protoVAL.extend.fields, fd)
			}
		}
	case 140:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:590
		{
			protoVAL.msgDecs = append(protoDollar[1].msgDecs, protoDollar[2].msgDecs...)
		}
	case 142:
		protoDollar = protoS[protopt-0 : protopt+1]
		//line proto.y:594
		{
			protoVAL.msgDecs = nil
		}
	case 143:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:598
		{
			protoVAL.msgDecs = []*msgDecl{{fld: protoDollar[1].fldd}}
		}
	case 144:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:601
		{
			protoVAL.msgDecs = []*msgDecl{{grp: protoDollar[1].grpd}}
		}
	case 145:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:604
		{
			protoVAL.msgDecs = nil
		}
	case 146:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:608
		{
			protoVAL.sd = svcDeclsToProto(protoDollar[2].str, protoDollar[4].svcDecs)
		}
	case 147:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:612
		{
			protoVAL.svcDecs = append(protoDollar[1].svcDecs, protoDollar[2].svcDecs...)
		}
	case 149:
		protoDollar = protoS[protopt-0 : protopt+1]
		//line proto.y:616
		{
			protoVAL.svcDecs = nil
		}
	case 150:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:623
		{
			protoVAL.svcDecs = []*serviceDecl{{option: protoDollar[1].opts[0]}}
		}
	case 151:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:626
		{
			protoVAL.svcDecs = []*serviceDecl{{rpc: protoDollar[1].mtd}}
		}
	case 152:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:629
		{
			protoVAL.svcDecs = nil
		}
	case 153:
		protoDollar = protoS[protopt-10 : protopt+1]
		//line proto.y:633
		{
			protoVAL.mtd = asMethodDescriptor(protoDollar[2].str, protoDollar[4].rpcType, protoDollar[8].rpcType, nil)
		}
	case 154:
		protoDollar = protoS[protopt-12 : protopt+1]
		//line proto.y:636
		{
			protoVAL.mtd = asMethodDescriptor(protoDollar[2].str, protoDollar[4].rpcType, protoDollar[8].rpcType, protoDollar[11].opts)
		}
	case 155:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:640
		{
			protoVAL.rpcType = &rpcType{msgType: protoDollar[2].str, stream: true}
		}
	case 156:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:643
		{
			protoVAL.rpcType = &rpcType{msgType: protoDollar[1].str}
		}
	case 157:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:647
		{
			protoVAL.opts = append(protoDollar[1].opts, protoDollar[2].opts...)
		}
	case 159:
		protoDollar = protoS[protopt-0 : protopt+1]
		//line proto.y:651
		{
			protoVAL.opts = nil
		}
	case 160:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:655
		{
			protoVAL.opts = protoDollar[1].opts
		}
	case 161:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:658
		{
			protoVAL.opts = nil
		}
	}
	goto protostack /* stack new state and value */
}
