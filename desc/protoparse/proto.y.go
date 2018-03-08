//line proto.y:2
package protoparse

import __yyfmt__ "fmt"

//line proto.y:3
import (
	"fmt"
	"math"
	"unicode"

	"github.com/jhump/protoreflect/desc/internal"
)

//line proto.y:17
type protoSymType struct {
	yys       int
	file      *fileNode
	fileDecls []*fileElement
	syn       *syntaxNode
	pkg       *packageNode
	imprt     *importNode
	msg       *messageNode
	msgDecls  []*messageElement
	fld       *fieldNode
	mapFld    *mapFieldNode
	grp       *groupNode
	oo        *oneOfNode
	ooDecls   []*oneOfElement
	ext       *extensionRangeNode
	resvd     *reservedNode
	en        *enumNode
	enDecls   []*enumElement
	env       *enumValueNode
	extend    *extendNode
	extDecls  []*extendElement
	svc       *serviceNode
	svcDecls  []*serviceElement
	mtd       *methodNode
	rpcType   *rpcTypeNode
	opts      []*optionNode
	optNm     []*optionNamePartNode
	rngs      []*rangeNode
	names     []*stringLiteralNode
	sl        []valueNode
	agg       []*aggregateEntryNode
	aggName   *aggregateNameNode
	v         valueNode
	str       *stringLiteralNode
	i         *negativeIntLiteralNode
	ui        *intLiteralNode
	f         *floatLiteralNode
	id        *identNode
	b         *basicNode
	err       error
}

const _STRING_LIT = 57346
const _INT_LIT = 57347
const _FLOAT_LIT = 57348
const _NAME = 57349
const _FQNAME = 57350
const _TYPENAME = 57351
const _SYNTAX = 57352
const _IMPORT = 57353
const _WEAK = 57354
const _PUBLIC = 57355
const _PACKAGE = 57356
const _OPTION = 57357
const _TRUE = 57358
const _FALSE = 57359
const _INF = 57360
const _NAN = 57361
const _REPEATED = 57362
const _OPTIONAL = 57363
const _REQUIRED = 57364
const _DOUBLE = 57365
const _FLOAT = 57366
const _INT32 = 57367
const _INT64 = 57368
const _UINT32 = 57369
const _UINT64 = 57370
const _SINT32 = 57371
const _SINT64 = 57372
const _FIXED32 = 57373
const _FIXED64 = 57374
const _SFIXED32 = 57375
const _SFIXED64 = 57376
const _BOOL = 57377
const _STRING = 57378
const _BYTES = 57379
const _GROUP = 57380
const _ONEOF = 57381
const _MAP = 57382
const _EXTENSIONS = 57383
const _TO = 57384
const _MAX = 57385
const _RESERVED = 57386
const _ENUM = 57387
const _MESSAGE = 57388
const _EXTEND = 57389
const _SERVICE = 57390
const _RPC = 57391
const _STREAM = 57392
const _RETURNS = 57393
const _ERROR = 57394

var protoToknames = [...]string{
	"$end",
	"error",
	"$unk",
	"_STRING_LIT",
	"_INT_LIT",
	"_FLOAT_LIT",
	"_NAME",
	"_FQNAME",
	"_TYPENAME",
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
	"_ERROR",
	"'='",
	"';'",
	"':'",
	"'{'",
	"'}'",
	"'\\\\'",
	"'/'",
	"'?'",
	"'.'",
	"','",
	"'>'",
	"'<'",
	"'+'",
	"'-'",
	"'('",
	"')'",
	"'['",
	"']'",
	"'*'",
	"'&'",
	"'^'",
	"'%'",
	"'$'",
	"'#'",
	"'@'",
	"'!'",
	"'~'",
	"'`'",
}
var protoStatenames = [...]string{}

const protoEofCode = 1
const protoErrCode = 2
const protoInitialStackSize = 16

//line proto.y:865

//line yacctab:1
var protoExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
}

const protoPrivate = 57344

const protoLast = 2032

var protoAct = [...]int{

	118, 8, 270, 8, 8, 368, 252, 79, 126, 111,
	154, 253, 259, 110, 180, 98, 97, 100, 101, 155,
	166, 148, 8, 27, 74, 143, 28, 117, 78, 112,
	96, 179, 254, 288, 301, 121, 243, 359, 288, 136,
	288, 371, 300, 288, 76, 77, 360, 81, 345, 73,
	299, 343, 288, 288, 288, 288, 288, 212, 361, 348,
	341, 333, 320, 319, 289, 214, 310, 307, 304, 297,
	286, 268, 213, 362, 349, 266, 279, 109, 338, 183,
	105, 311, 308, 305, 88, 287, 269, 237, 318, 204,
	267, 137, 231, 149, 199, 104, 230, 198, 264, 168,
	232, 312, 309, 201, 214, 197, 306, 216, 4, 14,
	92, 91, 15, 16, 103, 90, 89, 141, 16, 145,
	171, 144, 363, 374, 364, 365, 139, 356, 355, 321,
	354, 158, 172, 174, 176, 347, 137, 16, 78, 74,
	235, 236, 337, 18, 17, 19, 20, 336, 149, 315,
	16, 169, 13, 178, 77, 76, 16, 370, 153, 182,
	372, 95, 141, 94, 73, 93, 184, 202, 191, 193,
	145, 139, 144, 200, 196, 188, 370, 86, 83, 194,
	350, 158, 317, 290, 152, 250, 249, 248, 203, 151,
	152, 247, 192, 246, 256, 151, 245, 211, 189, 205,
	206, 207, 208, 209, 210, 87, 14, 233, 234, 15,
	16, 23, 242, 244, 215, 240, 238, 260, 163, 164,
	358, 74, 334, 5, 285, 263, 103, 22, 158, 255,
	24, 165, 257, 115, 11, 284, 11, 11, 25, 26,
	18, 17, 19, 20, 283, 22, 272, 186, 181, 13,
	113, 10, 277, 10, 10, 11, 282, 114, 9, 260,
	9, 9, 196, 281, 280, 163, 103, 263, 158, 158,
	275, 292, 10, 294, 295, 74, 296, 74, 160, 9,
	298, 181, 251, 265, 150, 160, 161, 302, 85, 84,
	293, 82, 147, 12, 313, 74, 74, 196, 162, 3,
	314, 142, 21, 158, 158, 138, 135, 116, 185, 258,
	120, 119, 327, 74, 261, 329, 74, 103, 331, 74,
	328, 316, 196, 330, 156, 274, 332, 103, 103, 158,
	102, 322, 324, 157, 339, 217, 340, 167, 171, 367,
	171, 351, 171, 7, 6, 335, 2, 272, 1, 0,
	158, 0, 158, 0, 0, 0, 357, 74, 0, 196,
	196, 0, 0, 0, 0, 369, 158, 158, 369, 366,
	74, 0, 0, 373, 99, 105, 108, 30, 0, 0,
	31, 32, 33, 34, 35, 36, 37, 38, 39, 40,
	41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
	51, 52, 53, 54, 55, 56, 57, 58, 59, 60,
	61, 62, 63, 64, 65, 66, 67, 68, 69, 70,
	71, 72, 0, 0, 0, 0, 104, 0, 0, 0,
	0, 0, 0, 0, 276, 106, 107, 0, 0, 0,
	273, 99, 105, 108, 30, 0, 0, 31, 32, 33,
	34, 35, 36, 37, 38, 39, 40, 41, 42, 43,
	44, 45, 46, 47, 48, 49, 50, 51, 52, 53,
	54, 55, 56, 57, 58, 59, 60, 61, 62, 63,
	64, 65, 66, 67, 68, 69, 70, 71, 72, 0,
	0, 0, 0, 104, 0, 0, 0, 0, 0, 0,
	0, 241, 106, 107, 0, 0, 239, 99, 105, 108,
	30, 0, 0, 31, 32, 33, 34, 35, 36, 37,
	38, 39, 40, 41, 42, 43, 44, 45, 46, 47,
	48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
	58, 59, 60, 61, 62, 63, 64, 65, 66, 67,
	68, 69, 70, 71, 72, 0, 0, 0, 0, 104,
	0, 0, 0, 0, 0, 0, 0, 325, 106, 107,
	99, 105, 108, 30, 0, 0, 31, 32, 33, 34,
	35, 36, 37, 38, 39, 40, 41, 42, 43, 44,
	45, 46, 47, 48, 49, 50, 51, 52, 53, 54,
	55, 56, 57, 58, 59, 60, 61, 62, 63, 64,
	65, 66, 67, 68, 69, 70, 71, 72, 0, 0,
	0, 0, 104, 0, 0, 0, 0, 0, 0, 0,
	323, 106, 107, 99, 105, 108, 30, 0, 0, 31,
	32, 33, 34, 35, 36, 37, 38, 39, 40, 41,
	42, 43, 44, 45, 46, 47, 48, 49, 50, 51,
	52, 53, 54, 55, 56, 57, 58, 59, 60, 61,
	62, 63, 64, 65, 66, 67, 68, 69, 70, 71,
	72, 0, 0, 0, 0, 104, 0, 0, 0, 0,
	0, 0, 0, 30, 106, 107, 31, 32, 33, 34,
	35, 36, 37, 38, 39, 40, 41, 42, 43, 44,
	45, 46, 47, 48, 49, 50, 51, 52, 53, 54,
	55, 56, 57, 58, 59, 60, 61, 62, 63, 64,
	65, 66, 67, 68, 69, 70, 71, 72, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 353,
	0, 0, 0, 30, 0, 159, 31, 32, 33, 34,
	35, 36, 37, 38, 39, 40, 41, 42, 43, 44,
	45, 46, 47, 48, 49, 50, 51, 52, 53, 54,
	55, 56, 57, 58, 59, 60, 61, 62, 63, 64,
	65, 66, 67, 68, 69, 70, 71, 72, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 352,
	0, 0, 0, 30, 0, 159, 31, 32, 33, 34,
	35, 36, 37, 38, 39, 40, 41, 42, 43, 44,
	45, 46, 47, 48, 49, 50, 51, 52, 53, 54,
	55, 56, 57, 58, 59, 60, 61, 62, 63, 64,
	65, 66, 67, 68, 69, 70, 71, 72, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 326,
	0, 0, 0, 30, 0, 159, 31, 32, 33, 34,
	35, 36, 37, 38, 39, 40, 41, 42, 43, 44,
	45, 46, 47, 48, 49, 50, 51, 52, 53, 54,
	55, 56, 57, 58, 59, 60, 61, 62, 63, 64,
	65, 66, 67, 68, 69, 70, 71, 72, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 303,
	0, 0, 0, 30, 0, 159, 31, 32, 33, 34,
	35, 36, 37, 38, 39, 40, 41, 42, 43, 44,
	45, 46, 47, 48, 49, 50, 51, 52, 53, 54,
	55, 56, 57, 58, 59, 60, 61, 62, 63, 64,
	65, 66, 67, 68, 69, 70, 71, 72, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 278,
	0, 0, 0, 30, 0, 159, 31, 32, 33, 34,
	35, 36, 37, 38, 39, 40, 41, 42, 43, 44,
	45, 46, 47, 48, 49, 50, 51, 52, 53, 54,
	55, 56, 57, 58, 59, 60, 61, 62, 63, 64,
	65, 66, 67, 68, 69, 70, 71, 72, 0, 0,
	0, 0, 0, 195, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 30, 0, 159, 31, 32, 33, 34,
	35, 36, 37, 38, 39, 40, 41, 42, 43, 44,
	45, 46, 47, 48, 49, 50, 51, 52, 53, 54,
	55, 56, 57, 58, 59, 60, 61, 62, 63, 64,
	65, 66, 67, 68, 69, 70, 71, 72, 218, 219,
	220, 221, 222, 223, 224, 225, 226, 227, 228, 229,
	0, 0, 0, 30, 29, 159, 31, 32, 33, 34,
	35, 36, 37, 38, 39, 40, 41, 42, 43, 44,
	45, 46, 47, 48, 49, 50, 51, 52, 53, 54,
	55, 56, 57, 58, 59, 60, 61, 62, 63, 64,
	65, 66, 67, 68, 69, 70, 71, 72, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 75, 30, 29, 80, 31, 32, 33,
	34, 35, 131, 37, 38, 39, 40, 125, 124, 123,
	44, 45, 46, 47, 48, 49, 50, 51, 52, 53,
	54, 55, 56, 57, 58, 59, 132, 133, 130, 63,
	64, 134, 127, 128, 129, 69, 70, 71, 72, 0,
	0, 122, 0, 0, 346, 30, 29, 80, 31, 32,
	33, 34, 35, 131, 37, 38, 39, 40, 125, 124,
	123, 44, 45, 46, 47, 48, 49, 50, 51, 52,
	53, 54, 55, 56, 57, 58, 59, 132, 133, 130,
	63, 64, 134, 127, 128, 129, 69, 70, 71, 72,
	0, 0, 122, 0, 0, 344, 30, 29, 80, 31,
	32, 33, 34, 35, 131, 37, 38, 39, 40, 125,
	124, 123, 44, 45, 46, 47, 48, 49, 50, 51,
	52, 53, 54, 55, 56, 57, 58, 59, 132, 133,
	130, 63, 64, 134, 127, 128, 129, 69, 70, 71,
	72, 0, 0, 122, 0, 0, 342, 30, 29, 80,
	31, 32, 33, 34, 35, 131, 37, 38, 39, 40,
	41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
	51, 52, 53, 54, 55, 56, 57, 58, 59, 60,
	61, 62, 63, 64, 65, 66, 67, 68, 69, 70,
	71, 72, 0, 0, 262, 0, 0, 291, 30, 29,
	80, 31, 32, 33, 34, 35, 36, 37, 38, 39,
	40, 125, 124, 123, 44, 45, 46, 47, 48, 49,
	50, 51, 52, 53, 54, 55, 56, 57, 58, 59,
	60, 61, 62, 63, 64, 65, 66, 67, 68, 69,
	70, 71, 72, 0, 0, 146, 0, 0, 190, 30,
	29, 80, 31, 32, 33, 34, 35, 131, 37, 38,
	39, 40, 125, 124, 123, 44, 45, 46, 47, 48,
	49, 50, 51, 52, 53, 54, 55, 56, 57, 58,
	59, 132, 133, 130, 63, 64, 134, 127, 128, 129,
	69, 70, 71, 72, 0, 0, 122, 30, 0, 170,
	31, 32, 33, 34, 35, 131, 37, 38, 39, 40,
	41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
	51, 52, 53, 54, 55, 56, 57, 58, 59, 60,
	61, 62, 63, 64, 134, 66, 67, 68, 69, 70,
	71, 72, 0, 0, 140, 0, 0, 187, 30, 29,
	80, 31, 32, 33, 34, 35, 131, 37, 38, 39,
	40, 125, 124, 123, 44, 45, 46, 47, 48, 49,
	50, 51, 52, 53, 54, 55, 56, 57, 58, 59,
	132, 133, 130, 63, 64, 134, 127, 128, 129, 69,
	70, 71, 72, 0, 0, 122, 30, 29, 80, 31,
	32, 33, 34, 35, 131, 37, 38, 39, 40, 41,
	42, 43, 44, 45, 46, 47, 48, 49, 50, 51,
	52, 53, 54, 55, 56, 57, 58, 59, 60, 61,
	62, 63, 64, 65, 66, 67, 68, 69, 70, 71,
	72, 0, 0, 262, 30, 29, 80, 31, 32, 33,
	34, 35, 36, 37, 38, 39, 40, 125, 124, 123,
	44, 45, 46, 47, 48, 49, 50, 51, 52, 53,
	54, 55, 56, 57, 58, 59, 60, 61, 62, 63,
	64, 65, 66, 67, 68, 69, 70, 71, 72, 30,
	0, 146, 31, 32, 33, 34, 35, 131, 37, 38,
	39, 40, 41, 42, 43, 44, 45, 46, 47, 48,
	49, 50, 51, 52, 53, 54, 55, 56, 57, 58,
	59, 60, 61, 62, 63, 64, 134, 66, 67, 68,
	69, 70, 71, 72, 0, 0, 140, 30, 29, 80,
	31, 32, 33, 34, 35, 36, 37, 38, 39, 40,
	41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
	51, 52, 53, 54, 55, 56, 57, 58, 59, 60,
	61, 62, 63, 64, 65, 66, 67, 68, 69, 70,
	271, 72, 30, 29, 80, 31, 32, 33, 34, 35,
	36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
	46, 47, 48, 49, 50, 51, 52, 53, 54, 55,
	56, 57, 58, 59, 60, 61, 62, 63, 64, 65,
	66, 67, 68, 69, 70, 71, 72, 30, 29, 80,
	31, 32, 33, 34, 35, 36, 37, 38, 39, 40,
	41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
	51, 52, 53, 54, 55, 56, 57, 58, 177, 60,
	61, 62, 63, 64, 65, 66, 67, 68, 69, 70,
	71, 72, 30, 29, 80, 31, 32, 33, 34, 35,
	36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
	46, 47, 48, 49, 50, 51, 52, 53, 54, 55,
	56, 57, 58, 175, 60, 61, 62, 63, 64, 65,
	66, 67, 68, 69, 70, 71, 72, 30, 29, 80,
	31, 32, 33, 34, 35, 36, 37, 38, 39, 40,
	41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
	51, 52, 53, 54, 55, 56, 57, 58, 173, 60,
	61, 62, 63, 64, 65, 66, 67, 68, 69, 70,
	71, 72, 30, 29, 0, 31, 32, 33, 34, 35,
	36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
	46, 47, 48, 49, 50, 51, 52, 53, 54, 55,
	56, 57, 58, 59, 60, 61, 62, 63, 64, 65,
	66, 67, 68, 69, 70, 71, 72, 30, 0, 0,
	31, 32, 33, 34, 35, 36, 37, 38, 39, 40,
	41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
	51, 52, 53, 54, 55, 56, 57, 58, 59, 60,
	61, 62, 63, 64, 65, 66, 67, 68, 69, 70,
	71, 72,
}
var protoPact = [...]int{

	98, -1000, 195, 195, 158, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, 226, 1935, 1106, 1980, 1980, 1755,
	1980, 195, -1000, 287, 124, 285, 284, 123, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, 152, -1000, 1755, 60, 59, 55, -1000,
	-1000, 54, 111, -1000, 109, 107, -1000, 629, 9, 1521,
	1662, 1617, 141, -1000, -1000, -1000, 104, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, 1046, -1000, 280, 213, -1000, 90,
	1422, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, 1890, 1845, 1800, 1980, 1980, 1980, 1755,
	276, 1106, 1980, 15, 243, 1470, -1000, -1000, -1000, -1000,
	-1000, 145, 1371, -1000, -1000, -1000, -1000, 135, -1000, -1000,
	-1000, -1000, 1980, -1000, 986, -1000, 43, 39, -1000, 1935,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, 90, -1000, 22,
	-1000, -1000, 1980, 1980, 1980, 1980, 1980, 1980, 144, 3,
	-1000, 172, 51, 1073, 42, 38, -1000, -1000, -1000, 75,
	-1000, -1000, -1000, -1000, 20, -1000, -1000, -1000, -1000, 437,
	-1000, 1046, -34, -1000, 1755, 143, 140, 138, 134, 133,
	132, 277, -1000, 1106, 276, 189, 1569, 36, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, 279, 21, 17, 273, 260, 1710, -1000, 370,
	-1000, 1046, 926, -1000, 8, 259, 258, 251, 239, 230,
	219, 16, -6, -1000, 130, -1000, -1000, -1000, 1320, -1000,
	-1000, -1000, -1000, 1980, 1755, -1000, -1000, 1106, -1000, 1106,
	1, 1755, -1000, -1000, -20, -1000, 1046, 866, -1000, -1000,
	14, 50, 13, 46, 12, 45, -1000, 1106, 1106, 95,
	629, -1000, -1000, 129, 25, -7, -8, 78, -1000, -1000,
	566, 503, 806, -1000, -1000, 1106, 1521, -1000, 1106, 1521,
	-1000, 1106, 1521, -9, -1000, -1000, -1000, 217, 1980, 93,
	88, 11, -1000, 1046, -1000, 1046, -1000, -10, 1269, -19,
	1218, -22, 1167, 81, 5, 127, -1000, -1000, 1710, 746,
	686, 76, -1000, 74, -1000, 73, -1000, -1000, -1000, 1106,
	215, -31, -1000, -1000, -1000, -1000, -1000, -24, 4, 68,
	71, -1000, 1106, -1000, 122, -1000, -29, 103, -1000, -1000,
	-1000, 69, -1000, -1000, -1000,
}
var protoPgo = [...]int{

	0, 348, 346, 223, 299, 344, 343, 0, 11, 6,
	5, 339, 32, 20, 337, 30, 16, 15, 26, 7,
	8, 335, 333, 18, 17, 330, 325, 10, 19, 324,
	29, 314, 311, 27, 310, 257, 9, 13, 12, 309,
	308, 35, 14, 31, 307, 250, 39, 306, 305, 233,
	25, 301, 293, 21, 292, 284, 2,
}
var protoR1 = [...]int{

	0, 1, 1, 1, 1, 4, 4, 3, 3, 3,
	3, 3, 3, 3, 3, 2, 5, 5, 5, 6,
	19, 19, 7, 12, 12, 12, 13, 13, 14, 14,
	15, 15, 16, 16, 16, 16, 16, 24, 24, 23,
	25, 25, 25, 25, 25, 17, 27, 27, 27, 28,
	28, 28, 29, 29, 29, 29, 29, 29, 29, 22,
	22, 26, 26, 26, 26, 26, 26, 20, 20, 30,
	30, 30, 30, 30, 30, 30, 30, 9, 9, 8,
	33, 33, 33, 32, 39, 39, 39, 38, 38, 38,
	31, 31, 34, 34, 21, 21, 21, 21, 21, 21,
	21, 21, 21, 21, 21, 21, 44, 44, 43, 43,
	42, 42, 42, 41, 41, 40, 40, 45, 47, 47,
	47, 46, 46, 46, 46, 48, 48, 48, 48, 35,
	37, 37, 37, 36, 36, 36, 36, 36, 36, 36,
	36, 36, 36, 36, 49, 51, 51, 51, 50, 50,
	50, 52, 54, 54, 54, 53, 53, 53, 55, 55,
	56, 56, 11, 11, 11, 10, 10, 18, 18, 18,
	18, 18, 18, 18, 18, 18, 18, 18, 18, 18,
	18, 18, 18, 18, 18, 18, 18, 18, 18, 18,
	18, 18, 18, 18, 18, 18, 18, 18, 18, 18,
	18, 18, 18, 18, 18, 18, 18, 18, 18, 18,
}
var protoR2 = [...]int{

	0, 1, 1, 2, 0, 2, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 4, 3, 4, 4, 3,
	1, 1, 5, 1, 3, 4, 1, 2, 1, 4,
	1, 1, 1, 1, 1, 1, 1, 1, 2, 2,
	1, 2, 2, 2, 2, 3, 1, 2, 0, 1,
	2, 2, 3, 4, 5, 3, 2, 5, 4, 1,
	3, 1, 3, 3, 3, 5, 5, 1, 1, 6,
	6, 6, 5, 9, 9, 9, 8, 3, 1, 3,
	8, 8, 8, 5, 2, 1, 0, 1, 1, 1,
	5, 8, 10, 13, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 3, 6, 3, 1,
	1, 3, 3, 3, 3, 3, 1, 5, 2, 1,
	0, 1, 1, 1, 1, 4, 7, 4, 7, 5,
	2, 1, 0, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 5, 2, 1, 0, 1, 1,
	1, 5, 2, 1, 0, 1, 1, 1, 10, 12,
	2, 1, 2, 1, 0, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
}
var protoChk = [...]int{

	-1000, -1, -2, -4, 10, -3, -5, -6, -7, -35,
	-45, -49, -52, 54, 11, 14, 15, 46, 45, 47,
	48, -4, -3, 53, 4, 12, 13, -19, -18, 8,
	7, 10, 11, 12, 13, 14, 15, 16, 17, 18,
	19, 20, 21, 22, 23, 24, 25, 26, 27, 28,
	29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
	39, 40, 41, 42, 43, 44, 45, 46, 47, 48,
	49, 50, 51, -12, -19, 67, -18, -18, -20, -19,
	9, -18, 4, 54, 4, 4, 54, 53, -20, 56,
	56, 56, 56, 54, 54, 54, -15, -16, -17, 4,
	-24, -23, -25, -18, 56, 5, 65, 66, 6, 68,
	-37, -36, -30, -45, -35, -49, -44, -33, -7, -32,
	-34, -41, 54, 22, 21, 20, -20, 45, 46, 47,
	41, 15, 39, 40, 44, -47, -46, -7, -48, -41,
	54, -18, -51, -50, -30, -33, 54, -54, -53, -7,
	-55, 54, 49, 54, -27, -28, -29, -22, -18, 69,
	5, 6, 18, 5, 6, 18, -13, -14, 9, 61,
	57, -36, -20, 38, -20, 38, -20, 38, -18, -43,
	-42, 5, -18, 64, -43, -40, 4, 57, -46, 53,
	57, -50, 57, -53, -18, 57, -28, 62, 54, 55,
	-17, 64, -19, -13, 67, -18, -18, -18, -18, -18,
	-18, 53, 54, 69, 62, 42, 56, -21, 25, 26,
	27, 28, 29, 30, 31, 32, 33, 34, 35, 36,
	54, 54, 62, -24, -23, 65, 66, 67, -16, 69,
	-17, 64, -27, 70, -20, 53, 53, 53, 53, 53,
	53, 5, -9, -8, -12, -42, 5, 43, -39, -38,
	-7, -31, 54, -20, 62, 4, 54, 69, 54, 69,
	-56, 50, -20, 70, -26, -15, 64, -27, 63, 68,
	5, 5, 5, 5, 5, 5, 54, 69, 62, 70,
	53, 57, -38, -18, -20, -9, -9, 68, -20, 70,
	62, 54, -27, 63, 54, 69, 56, 54, 69, 56,
	54, 69, 56, -9, -8, 54, -15, 53, 63, 70,
	70, 51, -15, 64, -15, 64, 63, -9, -37, -9,
	-37, -9, -37, 70, 5, -18, 54, 54, 67, -27,
	-27, 70, 57, 70, 57, 70, 57, 54, 54, 69,
	53, -56, 63, 63, 54, 54, 54, -9, 5, 68,
	70, 54, 69, 54, 56, 54, -9, -11, -10, -7,
	54, 70, 57, -10, 54,
}
var protoDef = [...]int{

	4, -2, 1, 2, 0, 6, 7, 8, 9, 10,
	11, 12, 13, 14, 0, 0, 0, 0, 0, 0,
	0, 3, 5, 0, 0, 0, 0, 0, 20, 21,
	167, 168, 169, 170, 171, 172, 173, 174, 175, 176,
	177, 178, 179, 180, 181, 182, 183, 184, 185, 186,
	187, 188, 189, 190, 191, 192, 193, 194, 195, 196,
	197, 198, 199, 200, 201, 202, 203, 204, 205, 206,
	207, 208, 209, 0, 23, 0, 0, 0, 0, 67,
	68, 0, 0, 16, 0, 0, 19, 0, 0, 132,
	120, 147, 154, 15, 17, 18, 0, 30, 31, 32,
	33, 34, 35, 36, 48, 37, 0, 0, 40, 24,
	0, 131, 133, 134, 135, 136, 137, 138, 139, 140,
	141, 142, 143, 0, 0, 0, 0, 0, 0, 0,
	199, 173, 0, 198, 202, 0, 119, 121, 122, 123,
	124, 0, 0, 146, 148, 149, 150, 0, 153, 155,
	156, 157, 0, 22, 0, 46, 49, 0, 59, 0,
	38, 42, 43, 39, 41, 44, 25, 26, 28, 0,
	129, 130, 0, 0, 0, 0, 0, 0, 0, 0,
	109, 110, 0, 0, 0, 0, 116, 117, 118, 0,
	144, 145, 151, 152, 0, 45, 47, 50, 51, 0,
	56, 48, 0, 27, 0, 0, 0, 0, 0, 0,
	0, 0, 106, 0, 0, 0, 86, 0, 94, 95,
	96, 97, 98, 99, 100, 101, 102, 103, 104, 105,
	113, 114, 0, 0, 0, 0, 0, 0, 52, 0,
	55, 48, 0, 60, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 78, 0, 108, 111, 112, 0, 85,
	87, 88, 89, 0, 0, 115, 125, 0, 127, 0,
	0, 208, 161, 53, 0, 61, 48, 0, 58, 29,
	0, 0, 0, 0, 0, 0, 72, 0, 0, 0,
	0, 83, 84, 0, 0, 0, 0, 0, 160, 54,
	0, 0, 0, 57, 69, 0, 132, 70, 0, 132,
	71, 0, 132, 0, 77, 107, 79, 0, 0, 0,
	0, 0, 62, 48, 63, 48, 64, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 126, 128, 0, 0,
	0, 0, 80, 0, 81, 0, 82, 76, 90, 0,
	0, 0, 65, 66, 73, 74, 75, 0, 0, 0,
	0, 92, 0, 158, 164, 91, 0, 0, 163, 165,
	166, 0, 159, 162, 93,
}
var protoTok1 = [...]int{

	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 78, 3, 76, 75, 74, 72, 3,
	67, 68, 71, 65, 62, 66, 61, 59, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 55, 54,
	64, 53, 63, 60, 77, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 69, 58, 70, 73, 3, 80, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 56, 3, 57, 79,
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
		//line proto.y:112
		{
			protoVAL.file = &fileNode{syntax: protoDollar[1].syn}
			protoVAL.file.setRange(protoDollar[1].syn, protoDollar[1].syn)
			protolex.(*protoLex).res = protoVAL.file
		}
	case 2:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:117
		{
			protoVAL.file = &fileNode{decls: protoDollar[1].fileDecls}
			if len(protoDollar[1].fileDecls) > 0 {
				protoVAL.file.setRange(protoDollar[1].fileDecls[0], protoDollar[1].fileDecls[len(protoDollar[1].fileDecls)-1])
			}
			protolex.(*protoLex).res = protoVAL.file
		}
	case 3:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:124
		{
			protoVAL.file = &fileNode{syntax: protoDollar[1].syn, decls: protoDollar[2].fileDecls}
			var end node
			if len(protoDollar[2].fileDecls) > 0 {
				end = protoDollar[2].fileDecls[len(protoDollar[2].fileDecls)-1]
			} else {
				end = protoDollar[1].syn
			}
			protoVAL.file.setRange(protoDollar[1].syn, end)
			protolex.(*protoLex).res = protoVAL.file
		}
	case 4:
		protoDollar = protoS[protopt-0 : protopt+1]
		//line proto.y:135
		{
		}
	case 5:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:138
		{
			protoVAL.fileDecls = append(protoDollar[1].fileDecls, protoDollar[2].fileDecls...)
		}
	case 7:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:143
		{
			protoVAL.fileDecls = []*fileElement{{imp: protoDollar[1].imprt}}
		}
	case 8:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:146
		{
			protoVAL.fileDecls = []*fileElement{{pkg: protoDollar[1].pkg}}
		}
	case 9:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:149
		{
			protoVAL.fileDecls = []*fileElement{{option: protoDollar[1].opts[0]}}
		}
	case 10:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:152
		{
			protoVAL.fileDecls = []*fileElement{{message: protoDollar[1].msg}}
		}
	case 11:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:155
		{
			protoVAL.fileDecls = []*fileElement{{enum: protoDollar[1].en}}
		}
	case 12:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:158
		{
			protoVAL.fileDecls = []*fileElement{{extend: protoDollar[1].extend}}
		}
	case 13:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:161
		{
			protoVAL.fileDecls = []*fileElement{{service: protoDollar[1].svc}}
		}
	case 14:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:164
		{
			protoVAL.fileDecls = []*fileElement{{empty: protoDollar[1].b}}
		}
	case 15:
		protoDollar = protoS[protopt-4 : protopt+1]
		//line proto.y:168
		{
			if protoDollar[3].str.val != "proto2" && protoDollar[3].str.val != "proto3" {
				lexError(protolex, protoDollar[3].str.start(), "syntax value must be 'proto2' or 'proto3'")
			}
			protoVAL.syn = &syntaxNode{syntax: protoDollar[3].str}
			protoVAL.syn.setRange(protoDollar[1].id, protoDollar[4].b)
		}
	case 16:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:176
		{
			protoVAL.imprt = &importNode{name: protoDollar[2].str}
			protoVAL.imprt.setRange(protoDollar[1].id, protoDollar[3].b)
		}
	case 17:
		protoDollar = protoS[protopt-4 : protopt+1]
		//line proto.y:180
		{
			protoVAL.imprt = &importNode{name: protoDollar[3].str, weak: true}
			protoVAL.imprt.setRange(protoDollar[1].id, protoDollar[4].b)
		}
	case 18:
		protoDollar = protoS[protopt-4 : protopt+1]
		//line proto.y:184
		{
			protoVAL.imprt = &importNode{name: protoDollar[3].str, public: true}
			protoVAL.imprt.setRange(protoDollar[1].id, protoDollar[4].b)
		}
	case 19:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:189
		{
			protoVAL.pkg = &packageNode{name: protoDollar[2].id}
			protoVAL.pkg.setRange(protoDollar[1].id, protoDollar[3].b)
		}
	case 22:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:197
		{
			n := &optionNameNode{parts: protoDollar[2].optNm}
			n.setRange(protoDollar[2].optNm[0], protoDollar[2].optNm[len(protoDollar[2].optNm)-1])
			o := &optionNode{name: n, val: protoDollar[4].v}
			o.setRange(protoDollar[1].id, protoDollar[5].b)
			protoVAL.opts = []*optionNode{o}
		}
	case 23:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:205
		{
			protoVAL.optNm = toNameParts(protoDollar[1].id, 0)
		}
	case 24:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:208
		{
			p := &optionNamePartNode{text: protoDollar[2].id, isExtension: true}
			p.setRange(protoDollar[1].b, protoDollar[3].b)
			protoVAL.optNm = []*optionNamePartNode{p}
		}
	case 25:
		protoDollar = protoS[protopt-4 : protopt+1]
		//line proto.y:213
		{
			p := &optionNamePartNode{text: protoDollar[2].id, isExtension: true}
			p.setRange(protoDollar[1].b, protoDollar[3].b)
			ps := make([]*optionNamePartNode, 1, len(protoDollar[4].optNm)+1)
			ps[0] = p
			protoVAL.optNm = append(ps, protoDollar[4].optNm...)
		}
	case 27:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:222
		{
			protoVAL.optNm = append(protoDollar[1].optNm, protoDollar[2].optNm...)
		}
	case 28:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:226
		{
			protoVAL.optNm = toNameParts(protoDollar[1].id, 1 /* exclude leading dot */)
		}
	case 29:
		protoDollar = protoS[protopt-4 : protopt+1]
		//line proto.y:229
		{
			p := &optionNamePartNode{text: protoDollar[3].id, isExtension: true}
			p.setRange(protoDollar[2].b, protoDollar[4].b)
			protoVAL.optNm = []*optionNamePartNode{p}
		}
	case 32:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:238
		{
			protoVAL.v = protoDollar[1].str
		}
	case 33:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:241
		{
			protoVAL.v = protoDollar[1].ui
		}
	case 34:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:244
		{
			protoVAL.v = protoDollar[1].i
		}
	case 35:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:247
		{
			protoVAL.v = protoDollar[1].f
		}
	case 36:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:250
		{
			if protoDollar[1].id.val == "true" {
				protoVAL.v = &boolLiteralNode{basicNode: protoDollar[1].id.basicNode, val: true}
			} else if protoDollar[1].id.val == "false" {
				protoVAL.v = &boolLiteralNode{basicNode: protoDollar[1].id.basicNode, val: false}
			} else if protoDollar[1].id.val == "inf" {
				f := &floatLiteralNode{val: math.Inf(1)}
				f.setRange(protoDollar[1].id, protoDollar[1].id)
				protoVAL.v = f
			} else if protoDollar[1].id.val == "nan" {
				f := &floatLiteralNode{val: math.NaN()}
				f.setRange(protoDollar[1].id, protoDollar[1].id)
				protoVAL.v = f
			} else {
				protoVAL.v = protoDollar[1].id
			}
		}
	case 38:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:269
		{
			protoVAL.ui = protoDollar[2].ui
		}
	case 39:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:273
		{
			if protoDollar[2].ui.val > math.MaxInt64+1 {
				lexError(protolex, protoDollar[2].ui.start(), fmt.Sprintf("numeric constant %d would underflow (allowed range is %d to %d)", protoDollar[2].ui.val, int64(math.MinInt64), int64(math.MaxInt64)))
			}
			protoVAL.i = &negativeIntLiteralNode{val: -int64(protoDollar[2].ui.val)}
			protoVAL.i.setRange(protoDollar[1].b, protoDollar[2].ui)
		}
	case 41:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:282
		{
			protoVAL.f = &floatLiteralNode{val: -protoDollar[2].f.val}
			protoVAL.f.setRange(protoDollar[1].b, protoDollar[2].f)
		}
	case 42:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:286
		{
			protoVAL.f = &floatLiteralNode{val: protoDollar[2].f.val}
			protoVAL.f.setRange(protoDollar[1].b, protoDollar[2].f)
		}
	case 43:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:290
		{
			protoVAL.f = &floatLiteralNode{val: math.Inf(1)}
			protoVAL.f.setRange(protoDollar[1].b, protoDollar[2].id)
		}
	case 44:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:294
		{
			protoVAL.f = &floatLiteralNode{val: math.Inf(-1)}
			protoVAL.f.setRange(protoDollar[1].b, protoDollar[2].id)
		}
	case 45:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:299
		{
			a := &aggregateLiteralNode{elements: protoDollar[2].agg}
			a.setRange(protoDollar[1].b, protoDollar[3].b)
			protoVAL.v = a
		}
	case 47:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:306
		{
			protoVAL.agg = append(protoDollar[1].agg, protoDollar[2].agg...)
		}
	case 48:
		protoDollar = protoS[protopt-0 : protopt+1]
		//line proto.y:309
		{
			protoVAL.agg = nil
		}
	case 50:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:314
		{
			protoVAL.agg = protoDollar[1].agg
		}
	case 51:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:317
		{
			protoVAL.agg = protoDollar[1].agg
		}
	case 52:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:321
		{
			a := &aggregateEntryNode{name: protoDollar[1].aggName, val: protoDollar[3].v}
			a.setRange(protoDollar[1].aggName, protoDollar[3].v)
			protoVAL.agg = []*aggregateEntryNode{a}
		}
	case 53:
		protoDollar = protoS[protopt-4 : protopt+1]
		//line proto.y:326
		{
			s := &sliceLiteralNode{}
			s.setRange(protoDollar[3].b, protoDollar[4].b)
			a := &aggregateEntryNode{name: protoDollar[1].aggName, val: s}
			a.setRange(protoDollar[1].aggName, protoDollar[4].b)
			protoVAL.agg = []*aggregateEntryNode{a}
		}
	case 54:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:333
		{
			s := &sliceLiteralNode{elements: protoDollar[4].sl}
			s.setRange(protoDollar[3].b, protoDollar[5].b)
			a := &aggregateEntryNode{name: protoDollar[1].aggName, val: s}
			a.setRange(protoDollar[1].aggName, protoDollar[5].b)
			protoVAL.agg = []*aggregateEntryNode{a}
		}
	case 55:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:340
		{
			a := &aggregateEntryNode{name: protoDollar[1].aggName, val: protoDollar[3].v}
			a.setRange(protoDollar[1].aggName, protoDollar[3].v)
			protoVAL.agg = []*aggregateEntryNode{a}
		}
	case 56:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:345
		{
			a := &aggregateEntryNode{name: protoDollar[1].aggName, val: protoDollar[2].v}
			a.setRange(protoDollar[1].aggName, protoDollar[2].v)
			protoVAL.agg = []*aggregateEntryNode{a}
		}
	case 57:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:350
		{
			s := &aggregateLiteralNode{elements: protoDollar[4].agg}
			s.setRange(protoDollar[3].b, protoDollar[5].b)
			a := &aggregateEntryNode{name: protoDollar[1].aggName, val: s}
			a.setRange(protoDollar[1].aggName, protoDollar[5].b)
			protoVAL.agg = []*aggregateEntryNode{a}
		}
	case 58:
		protoDollar = protoS[protopt-4 : protopt+1]
		//line proto.y:357
		{
			s := &aggregateLiteralNode{elements: protoDollar[3].agg}
			s.setRange(protoDollar[2].b, protoDollar[4].b)
			a := &aggregateEntryNode{name: protoDollar[1].aggName, val: s}
			a.setRange(protoDollar[1].aggName, protoDollar[4].b)
			protoVAL.agg = []*aggregateEntryNode{a}
		}
	case 59:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:365
		{
			protoVAL.aggName = &aggregateNameNode{name: protoDollar[1].id}
			protoVAL.aggName.setRange(protoDollar[1].id, protoDollar[1].id)
		}
	case 60:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:369
		{
			protoVAL.aggName = &aggregateNameNode{name: protoDollar[2].id, isExtension: true}
			protoVAL.aggName.setRange(protoDollar[1].b, protoDollar[3].b)
		}
	case 61:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:374
		{
			protoVAL.sl = []valueNode{protoDollar[1].v}
		}
	case 62:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:377
		{
			protoVAL.sl = append(protoDollar[1].sl, protoDollar[3].v)
		}
	case 63:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:380
		{
			protoVAL.sl = append(protoDollar[1].sl, protoDollar[3].v)
		}
	case 64:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:383
		{
			s := &aggregateLiteralNode{elements: protoDollar[2].agg}
			s.setRange(protoDollar[1].b, protoDollar[3].b)
			protoVAL.sl = []valueNode{s}
		}
	case 65:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:388
		{
			s := &aggregateLiteralNode{elements: protoDollar[4].agg}
			s.setRange(protoDollar[3].b, protoDollar[5].b)
			protoVAL.sl = append(protoDollar[1].sl, s)
		}
	case 66:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:393
		{
			s := &aggregateLiteralNode{elements: protoDollar[4].agg}
			s.setRange(protoDollar[3].b, protoDollar[5].b)
			protoVAL.sl = append(protoDollar[1].sl, s)
		}
	case 69:
		protoDollar = protoS[protopt-6 : protopt+1]
		//line proto.y:402
		{
			checkTag(protolex, protoDollar[5].ui.start(), protoDollar[5].ui.val)
			lbl := &labelNode{basicNode: protoDollar[1].id.basicNode, required: true}
			protoVAL.fld = &fieldNode{label: lbl, fldType: protoDollar[2].id, name: protoDollar[3].id, tag: protoDollar[5].ui}
			protoVAL.fld.setRange(protoDollar[1].id, protoDollar[6].b)
		}
	case 70:
		protoDollar = protoS[protopt-6 : protopt+1]
		//line proto.y:408
		{
			checkTag(protolex, protoDollar[5].ui.start(), protoDollar[5].ui.val)
			lbl := &labelNode{basicNode: protoDollar[1].id.basicNode}
			protoVAL.fld = &fieldNode{label: lbl, fldType: protoDollar[2].id, name: protoDollar[3].id, tag: protoDollar[5].ui}
			protoVAL.fld.setRange(protoDollar[1].id, protoDollar[6].b)
		}
	case 71:
		protoDollar = protoS[protopt-6 : protopt+1]
		//line proto.y:414
		{
			checkTag(protolex, protoDollar[5].ui.start(), protoDollar[5].ui.val)
			lbl := &labelNode{basicNode: protoDollar[1].id.basicNode, repeated: true}
			protoVAL.fld = &fieldNode{label: lbl, fldType: protoDollar[2].id, name: protoDollar[3].id, tag: protoDollar[5].ui}
			protoVAL.fld.setRange(protoDollar[1].id, protoDollar[6].b)
		}
	case 72:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:420
		{
			checkTag(protolex, protoDollar[4].ui.start(), protoDollar[4].ui.val)
			protoVAL.fld = &fieldNode{fldType: protoDollar[1].id, name: protoDollar[2].id, tag: protoDollar[4].ui}
			protoVAL.fld.setRange(protoDollar[1].id, protoDollar[5].b)
		}
	case 73:
		protoDollar = protoS[protopt-9 : protopt+1]
		//line proto.y:425
		{
			checkTag(protolex, protoDollar[5].ui.start(), protoDollar[5].ui.val)
			lbl := &labelNode{basicNode: protoDollar[1].id.basicNode, required: true}
			protoVAL.fld = &fieldNode{label: lbl, fldType: protoDollar[2].id, name: protoDollar[3].id, tag: protoDollar[5].ui, options: protoDollar[7].opts}
			protoVAL.fld.setRange(protoDollar[1].id, protoDollar[9].b)
		}
	case 74:
		protoDollar = protoS[protopt-9 : protopt+1]
		//line proto.y:431
		{
			checkTag(protolex, protoDollar[5].ui.start(), protoDollar[5].ui.val)
			lbl := &labelNode{basicNode: protoDollar[1].id.basicNode}
			protoVAL.fld = &fieldNode{label: lbl, fldType: protoDollar[2].id, name: protoDollar[3].id, tag: protoDollar[5].ui, options: protoDollar[7].opts}
			protoVAL.fld.setRange(protoDollar[1].id, protoDollar[9].b)
		}
	case 75:
		protoDollar = protoS[protopt-9 : protopt+1]
		//line proto.y:437
		{
			checkTag(protolex, protoDollar[5].ui.start(), protoDollar[5].ui.val)
			lbl := &labelNode{basicNode: protoDollar[1].id.basicNode, repeated: true}
			protoVAL.fld = &fieldNode{label: lbl, fldType: protoDollar[2].id, name: protoDollar[3].id, tag: protoDollar[5].ui, options: protoDollar[7].opts}
			protoVAL.fld.setRange(protoDollar[1].id, protoDollar[9].b)
		}
	case 76:
		protoDollar = protoS[protopt-8 : protopt+1]
		//line proto.y:443
		{
			checkTag(protolex, protoDollar[4].ui.start(), protoDollar[4].ui.val)
			protoVAL.fld = &fieldNode{fldType: protoDollar[1].id, name: protoDollar[2].id, tag: protoDollar[4].ui, options: protoDollar[6].opts}
			protoVAL.fld.setRange(protoDollar[1].id, protoDollar[8].b)
		}
	case 77:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:449
		{
			protoVAL.opts = append(protoDollar[1].opts, protoDollar[3].opts...)
		}
	case 79:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:454
		{
			n := &optionNameNode{parts: protoDollar[1].optNm}
			n.setRange(protoDollar[1].optNm[0], protoDollar[1].optNm[len(protoDollar[1].optNm)-1])
			o := &optionNode{name: n, val: protoDollar[3].v}
			o.setRange(protoDollar[1].optNm[0], protoDollar[3].v)
			protoVAL.opts = []*optionNode{o}
		}
	case 80:
		protoDollar = protoS[protopt-8 : protopt+1]
		//line proto.y:462
		{
			checkTag(protolex, protoDollar[5].ui.start(), protoDollar[5].ui.val)
			if !unicode.IsUpper(rune(protoDollar[3].id.val[0])) {
				lexError(protolex, protoDollar[3].id.start(), fmt.Sprintf("group %s should have a name that starts with a capital letter", protoDollar[3].id.val))
			}
			lbl := &labelNode{basicNode: protoDollar[1].id.basicNode, required: true}
			protoVAL.grp = &groupNode{groupKeyword: protoDollar[2].id, label: lbl, name: protoDollar[3].id, tag: protoDollar[5].ui, decls: protoDollar[7].msgDecls}
			protoVAL.grp.setRange(protoDollar[1].id, protoDollar[8].b)
		}
	case 81:
		protoDollar = protoS[protopt-8 : protopt+1]
		//line proto.y:471
		{
			checkTag(protolex, protoDollar[5].ui.start(), protoDollar[5].ui.val)
			if !unicode.IsUpper(rune(protoDollar[3].id.val[0])) {
				lexError(protolex, protoDollar[3].id.start(), fmt.Sprintf("group %s should have a name that starts with a capital letter", protoDollar[3].id.val))
			}
			lbl := &labelNode{basicNode: protoDollar[1].id.basicNode}
			protoVAL.grp = &groupNode{groupKeyword: protoDollar[2].id, label: lbl, name: protoDollar[3].id, tag: protoDollar[5].ui, decls: protoDollar[7].msgDecls}
			protoVAL.grp.setRange(protoDollar[1].id, protoDollar[8].b)
		}
	case 82:
		protoDollar = protoS[protopt-8 : protopt+1]
		//line proto.y:480
		{
			checkTag(protolex, protoDollar[5].ui.start(), protoDollar[5].ui.val)
			if !unicode.IsUpper(rune(protoDollar[3].id.val[0])) {
				lexError(protolex, protoDollar[3].id.start(), fmt.Sprintf("group %s should have a name that starts with a capital letter", protoDollar[3].id.val))
			}
			lbl := &labelNode{basicNode: protoDollar[1].id.basicNode, repeated: true}
			protoVAL.grp = &groupNode{groupKeyword: protoDollar[2].id, label: lbl, name: protoDollar[3].id, tag: protoDollar[5].ui, decls: protoDollar[7].msgDecls}
			protoVAL.grp.setRange(protoDollar[1].id, protoDollar[8].b)
		}
	case 83:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:490
		{
			c := 0
			for _, el := range protoDollar[4].ooDecls {
				if el.field != nil {
					c++
				}
			}
			if c == 0 {
				lexError(protolex, protoDollar[1].id.start(), "oneof must contain at least one field")
			}
			protoVAL.oo = &oneOfNode{name: protoDollar[2].id, decls: protoDollar[4].ooDecls}
			protoVAL.oo.setRange(protoDollar[1].id, protoDollar[5].b)
		}
	case 84:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:504
		{
			protoVAL.ooDecls = append(protoDollar[1].ooDecls, protoDollar[2].ooDecls...)
		}
	case 86:
		protoDollar = protoS[protopt-0 : protopt+1]
		//line proto.y:508
		{
			protoVAL.ooDecls = nil
		}
	case 87:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:512
		{
			protoVAL.ooDecls = []*oneOfElement{{option: protoDollar[1].opts[0]}}
		}
	case 88:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:515
		{
			protoVAL.ooDecls = []*oneOfElement{{field: protoDollar[1].fld}}
		}
	case 89:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:518
		{
			protoVAL.ooDecls = []*oneOfElement{{empty: protoDollar[1].b}}
		}
	case 90:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:522
		{
			checkTag(protolex, protoDollar[4].ui.start(), protoDollar[4].ui.val)
			protoVAL.fld = &fieldNode{fldType: protoDollar[1].id, name: protoDollar[2].id, tag: protoDollar[4].ui}
			protoVAL.fld.setRange(protoDollar[1].id, protoDollar[5].b)
		}
	case 91:
		protoDollar = protoS[protopt-8 : protopt+1]
		//line proto.y:527
		{
			checkTag(protolex, protoDollar[4].ui.start(), protoDollar[4].ui.val)
			protoVAL.fld = &fieldNode{fldType: protoDollar[1].id, name: protoDollar[2].id, tag: protoDollar[4].ui, options: protoDollar[6].opts}
			protoVAL.fld.setRange(protoDollar[1].id, protoDollar[8].b)
		}
	case 92:
		protoDollar = protoS[protopt-10 : protopt+1]
		//line proto.y:533
		{
			checkTag(protolex, protoDollar[9].ui.start(), protoDollar[9].ui.val)
			protoVAL.mapFld = &mapFieldNode{mapKeyword: protoDollar[1].id, keyType: protoDollar[3].id, valueType: protoDollar[5].id, name: protoDollar[7].id, tag: protoDollar[9].ui}
			protoVAL.mapFld.setRange(protoDollar[1].id, protoDollar[10].b)
		}
	case 93:
		protoDollar = protoS[protopt-13 : protopt+1]
		//line proto.y:538
		{
			checkTag(protolex, protoDollar[9].ui.start(), protoDollar[9].ui.val)
			protoVAL.mapFld = &mapFieldNode{mapKeyword: protoDollar[1].id, keyType: protoDollar[3].id, valueType: protoDollar[5].id, name: protoDollar[7].id, tag: protoDollar[9].ui, options: protoDollar[11].opts}
			protoVAL.mapFld.setRange(protoDollar[1].id, protoDollar[13].b)
		}
	case 106:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:557
		{
			protoVAL.ext = &extensionRangeNode{ranges: protoDollar[2].rngs}
			protoVAL.ext.setRange(protoDollar[1].id, protoDollar[3].b)
		}
	case 107:
		protoDollar = protoS[protopt-6 : protopt+1]
		//line proto.y:561
		{
			protoVAL.ext = &extensionRangeNode{ranges: protoDollar[2].rngs, options: protoDollar[4].opts}
			protoVAL.ext.setRange(protoDollar[1].id, protoDollar[6].b)
		}
	case 108:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:566
		{
			protoVAL.rngs = append(protoDollar[1].rngs, protoDollar[3].rngs...)
		}
	case 110:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:571
		{
			if protoDollar[1].ui.val > internal.MaxTag {
				lexError(protolex, protoDollar[1].ui.start(), fmt.Sprintf("range includes out-of-range tag: %d (should be between 0 and %d)", protoDollar[1].ui.val, internal.MaxTag))
			}
			r := &rangeNode{st: protoDollar[1].ui, en: protoDollar[1].ui}
			r.setRange(protoDollar[1].ui, protoDollar[1].ui)
			protoVAL.rngs = []*rangeNode{r}
		}
	case 111:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:579
		{
			if protoDollar[1].ui.val > internal.MaxTag {
				lexError(protolex, protoDollar[1].ui.start(), fmt.Sprintf("range start is out-of-range tag: %d (should be between 0 and %d)", protoDollar[1].ui.val, internal.MaxTag))
			}
			if protoDollar[3].ui.val > internal.MaxTag {
				lexError(protolex, protoDollar[3].ui.start(), fmt.Sprintf("range end is out-of-range tag: %d (should be between 0 and %d)", protoDollar[3].ui.val, internal.MaxTag))
			}
			if protoDollar[1].ui.val > protoDollar[3].ui.val {
				lexError(protolex, protoDollar[1].ui.start(), fmt.Sprintf("range, %d to %d, is invalid: start must be <= end", protoDollar[1].ui.val, protoDollar[3].ui.val))
			}
			r := &rangeNode{st: protoDollar[1].ui, en: protoDollar[3].ui}
			r.setRange(protoDollar[1].ui, protoDollar[3].ui)
			protoVAL.rngs = []*rangeNode{r}
		}
	case 112:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:593
		{
			if protoDollar[1].ui.val > internal.MaxTag {
				lexError(protolex, protoDollar[1].ui.start(), fmt.Sprintf("range start is out-of-range tag: %d (should be between 0 and %d)", protoDollar[1].ui.val, internal.MaxTag))
			}
			m := &intLiteralNode{basicNode: protoDollar[3].id.basicNode, val: internal.MaxTag}
			r := &rangeNode{st: protoDollar[1].ui, en: m}
			r.setRange(protoDollar[1].ui, protoDollar[3].id)
			protoVAL.rngs = []*rangeNode{r}
		}
	case 113:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:603
		{
			protoVAL.resvd = &reservedNode{ranges: protoDollar[2].rngs}
			protoVAL.resvd.setRange(protoDollar[1].id, protoDollar[3].b)
		}
	case 114:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:607
		{
			rsvd := map[string]struct{}{}
			for _, n := range protoDollar[2].names {
				if _, ok := rsvd[n.val]; ok {
					lexError(protolex, n.start(), fmt.Sprintf("field %q is reserved multiple times", n.val))
					break
				}
				rsvd[n.val] = struct{}{}
			}
			protoVAL.resvd = &reservedNode{names: protoDollar[2].names}
			protoVAL.resvd.setRange(protoDollar[1].id, protoDollar[3].b)
		}
	case 115:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:620
		{
			protoVAL.names = append(protoDollar[1].names, protoDollar[3].str)
		}
	case 116:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:623
		{
			protoVAL.names = []*stringLiteralNode{protoDollar[1].str}
		}
	case 117:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:627
		{
			c := 0
			for _, el := range protoDollar[4].enDecls {
				if el.value != nil {
					c++
				}
			}
			if c == 0 {
				lexError(protolex, protoDollar[1].id.start(), "enums must define at least one value")
			}
			protoVAL.en = &enumNode{name: protoDollar[2].id, decls: protoDollar[4].enDecls}
			protoVAL.en.setRange(protoDollar[1].id, protoDollar[5].b)
		}
	case 118:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:641
		{
			protoVAL.enDecls = append(protoDollar[1].enDecls, protoDollar[2].enDecls...)
		}
	case 120:
		protoDollar = protoS[protopt-0 : protopt+1]
		//line proto.y:645
		{
			protoVAL.enDecls = nil
		}
	case 121:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:649
		{
			protoVAL.enDecls = []*enumElement{{option: protoDollar[1].opts[0]}}
		}
	case 122:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:652
		{
			protoVAL.enDecls = []*enumElement{{value: protoDollar[1].env}}
		}
	case 123:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:655
		{
			protoVAL.enDecls = []*enumElement{{reserved: protoDollar[1].resvd}}
		}
	case 124:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:658
		{
			protoVAL.enDecls = []*enumElement{{empty: protoDollar[1].b}}
		}
	case 125:
		protoDollar = protoS[protopt-4 : protopt+1]
		//line proto.y:662
		{
			checkUint64InInt32Range(protolex, protoDollar[3].ui.start(), protoDollar[3].ui.val)
			protoVAL.env = &enumValueNode{name: protoDollar[1].id, number: protoDollar[3].ui}
			protoVAL.env.setRange(protoDollar[1].id, protoDollar[4].b)
		}
	case 126:
		protoDollar = protoS[protopt-7 : protopt+1]
		//line proto.y:667
		{
			checkUint64InInt32Range(protolex, protoDollar[3].ui.start(), protoDollar[3].ui.val)
			protoVAL.env = &enumValueNode{name: protoDollar[1].id, number: protoDollar[3].ui, options: protoDollar[5].opts}
			protoVAL.env.setRange(protoDollar[1].id, protoDollar[7].b)
		}
	case 127:
		protoDollar = protoS[protopt-4 : protopt+1]
		//line proto.y:672
		{
			checkInt64InInt32Range(protolex, protoDollar[3].i.start(), protoDollar[3].i.val)
			protoVAL.env = &enumValueNode{name: protoDollar[1].id, numberN: protoDollar[3].i}
			protoVAL.env.setRange(protoDollar[1].id, protoDollar[4].b)
		}
	case 128:
		protoDollar = protoS[protopt-7 : protopt+1]
		//line proto.y:677
		{
			checkInt64InInt32Range(protolex, protoDollar[3].i.start(), protoDollar[3].i.val)
			protoVAL.env = &enumValueNode{name: protoDollar[1].id, numberN: protoDollar[3].i, options: protoDollar[5].opts}
			protoVAL.env.setRange(protoDollar[1].id, protoDollar[7].b)
		}
	case 129:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:683
		{
			protoVAL.msg = &messageNode{name: protoDollar[2].id, decls: protoDollar[4].msgDecls}
			protoVAL.msg.setRange(protoDollar[1].id, protoDollar[5].b)
		}
	case 130:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:688
		{
			protoVAL.msgDecls = append(protoDollar[1].msgDecls, protoDollar[2].msgDecls...)
		}
	case 132:
		protoDollar = protoS[protopt-0 : protopt+1]
		//line proto.y:692
		{
			protoVAL.msgDecls = nil
		}
	case 133:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:696
		{
			protoVAL.msgDecls = []*messageElement{{field: protoDollar[1].fld}}
		}
	case 134:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:699
		{
			protoVAL.msgDecls = []*messageElement{{enum: protoDollar[1].en}}
		}
	case 135:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:702
		{
			protoVAL.msgDecls = []*messageElement{{nested: protoDollar[1].msg}}
		}
	case 136:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:705
		{
			protoVAL.msgDecls = []*messageElement{{extend: protoDollar[1].extend}}
		}
	case 137:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:708
		{
			protoVAL.msgDecls = []*messageElement{{extensionRange: protoDollar[1].ext}}
		}
	case 138:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:711
		{
			protoVAL.msgDecls = []*messageElement{{group: protoDollar[1].grp}}
		}
	case 139:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:714
		{
			protoVAL.msgDecls = []*messageElement{{option: protoDollar[1].opts[0]}}
		}
	case 140:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:717
		{
			protoVAL.msgDecls = []*messageElement{{oneOf: protoDollar[1].oo}}
		}
	case 141:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:720
		{
			protoVAL.msgDecls = []*messageElement{{mapField: protoDollar[1].mapFld}}
		}
	case 142:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:723
		{
			protoVAL.msgDecls = []*messageElement{{reserved: protoDollar[1].resvd}}
		}
	case 143:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:726
		{
			protoVAL.msgDecls = []*messageElement{{empty: protoDollar[1].b}}
		}
	case 144:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:730
		{
			c := 0
			for _, el := range protoDollar[4].extDecls {
				if el.field != nil || el.group != nil {
					c++
				}
			}
			if c == 0 {
				lexError(protolex, protoDollar[1].id.start(), "extend sections must define at least one extension")
			}
			protoVAL.extend = &extendNode{extendee: protoDollar[2].id, decls: protoDollar[4].extDecls}
			protoVAL.extend.setRange(protoDollar[1].id, protoDollar[5].b)
		}
	case 145:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:744
		{
			protoVAL.extDecls = append(protoDollar[1].extDecls, protoDollar[2].extDecls...)
		}
	case 147:
		protoDollar = protoS[protopt-0 : protopt+1]
		//line proto.y:748
		{
			protoVAL.extDecls = nil
		}
	case 148:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:752
		{
			protoVAL.extDecls = []*extendElement{{field: protoDollar[1].fld}}
		}
	case 149:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:755
		{
			protoVAL.extDecls = []*extendElement{{group: protoDollar[1].grp}}
		}
	case 150:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:758
		{
			protoVAL.extDecls = []*extendElement{{empty: protoDollar[1].b}}
		}
	case 151:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:762
		{
			protoVAL.svc = &serviceNode{name: protoDollar[2].id, decls: protoDollar[4].svcDecls}
			protoVAL.svc.setRange(protoDollar[1].id, protoDollar[5].b)
		}
	case 152:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:767
		{
			protoVAL.svcDecls = append(protoDollar[1].svcDecls, protoDollar[2].svcDecls...)
		}
	case 154:
		protoDollar = protoS[protopt-0 : protopt+1]
		//line proto.y:771
		{
			protoVAL.svcDecls = nil
		}
	case 155:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:778
		{
			protoVAL.svcDecls = []*serviceElement{{option: protoDollar[1].opts[0]}}
		}
	case 156:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:781
		{
			protoVAL.svcDecls = []*serviceElement{{rpc: protoDollar[1].mtd}}
		}
	case 157:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:784
		{
			protoVAL.svcDecls = []*serviceElement{{empty: protoDollar[1].b}}
		}
	case 158:
		protoDollar = protoS[protopt-10 : protopt+1]
		//line proto.y:788
		{
			protoVAL.mtd = &methodNode{name: protoDollar[2].id, input: protoDollar[4].rpcType, output: protoDollar[8].rpcType}
			protoVAL.mtd.setRange(protoDollar[1].id, protoDollar[10].b)
		}
	case 159:
		protoDollar = protoS[protopt-12 : protopt+1]
		//line proto.y:792
		{
			protoVAL.mtd = &methodNode{name: protoDollar[2].id, input: protoDollar[4].rpcType, output: protoDollar[8].rpcType, options: protoDollar[11].opts}
			protoVAL.mtd.setRange(protoDollar[1].id, protoDollar[12].b)
		}
	case 160:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:797
		{
			protoVAL.rpcType = &rpcTypeNode{msgType: protoDollar[2].id, streamKeyword: protoDollar[1].id}
			protoVAL.rpcType.setRange(protoDollar[1].id, protoDollar[2].id)
		}
	case 161:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:801
		{
			protoVAL.rpcType = &rpcTypeNode{msgType: protoDollar[1].id}
			protoVAL.rpcType.setRange(protoDollar[1].id, protoDollar[1].id)
		}
	case 162:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:806
		{
			protoVAL.opts = append(protoDollar[1].opts, protoDollar[2].opts...)
		}
	case 164:
		protoDollar = protoS[protopt-0 : protopt+1]
		//line proto.y:810
		{
			protoVAL.opts = nil
		}
	case 165:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:814
		{
			protoVAL.opts = protoDollar[1].opts
		}
	case 166:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:817
		{
			protoVAL.opts = nil
		}
	}
	goto protostack /* stack new state and value */
}
