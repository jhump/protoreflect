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

//line proto.y:929

//line yacctab:1
var protoExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
}

const protoPrivate = 57344

const protoLast = 2048

var protoAct = [...]int{

	118, 8, 286, 8, 8, 384, 262, 79, 126, 111,
	157, 158, 263, 269, 101, 194, 183, 110, 98, 97,
	28, 169, 8, 27, 74, 117, 151, 112, 78, 146,
	135, 137, 264, 253, 317, 304, 304, 182, 76, 77,
	304, 81, 316, 387, 376, 304, 304, 304, 361, 73,
	315, 304, 96, 359, 357, 349, 304, 304, 220, 336,
	377, 364, 326, 323, 335, 305, 222, 320, 302, 375,
	278, 276, 354, 221, 284, 378, 365, 327, 324, 313,
	295, 195, 321, 303, 88, 279, 277, 109, 189, 195,
	247, 138, 212, 152, 241, 207, 104, 186, 334, 244,
	274, 239, 238, 390, 209, 206, 328, 243, 103, 240,
	222, 142, 285, 205, 171, 325, 379, 148, 380, 147,
	174, 144, 322, 16, 224, 161, 92, 91, 90, 89,
	337, 381, 175, 177, 179, 197, 372, 138, 78, 74,
	371, 370, 197, 16, 363, 353, 352, 181, 77, 76,
	197, 152, 16, 185, 16, 197, 331, 142, 196, 246,
	156, 95, 386, 94, 73, 388, 172, 144, 191, 204,
	210, 148, 187, 147, 93, 199, 202, 201, 161, 208,
	4, 14, 386, 86, 15, 16, 155, 83, 155, 366,
	245, 154, 211, 154, 200, 333, 213, 214, 215, 216,
	217, 218, 306, 260, 259, 258, 14, 242, 257, 15,
	16, 256, 255, 219, 192, 18, 17, 19, 20, 87,
	252, 254, 23, 223, 13, 270, 250, 248, 103, 74,
	161, 115, 11, 273, 11, 11, 281, 266, 374, 265,
	18, 17, 19, 20, 113, 10, 5, 10, 10, 13,
	22, 189, 184, 11, 166, 167, 288, 350, 196, 280,
	301, 283, 293, 300, 204, 299, 10, 168, 22, 270,
	103, 298, 161, 161, 282, 267, 297, 273, 163, 164,
	24, 275, 308, 310, 311, 74, 312, 74, 25, 26,
	296, 165, 184, 261, 309, 166, 314, 114, 9, 85,
	9, 9, 291, 318, 84, 204, 82, 153, 150, 12,
	329, 74, 74, 161, 161, 3, 145, 330, 21, 9,
	139, 136, 116, 193, 140, 121, 188, 103, 343, 74,
	204, 345, 74, 268, 347, 74, 120, 103, 103, 161,
	344, 119, 271, 346, 159, 290, 348, 102, 100, 160,
	355, 225, 356, 170, 174, 351, 174, 367, 174, 332,
	161, 383, 161, 288, 7, 6, 2, 204, 204, 338,
	340, 1, 373, 74, 0, 0, 161, 161, 0, 0,
	0, 385, 0, 0, 385, 382, 74, 0, 0, 389,
	99, 105, 108, 30, 0, 0, 31, 32, 33, 34,
	35, 36, 37, 38, 39, 40, 41, 42, 43, 44,
	45, 46, 47, 48, 49, 50, 51, 52, 53, 54,
	55, 56, 57, 58, 59, 60, 61, 62, 63, 64,
	65, 66, 67, 68, 69, 70, 71, 72, 0, 0,
	0, 0, 104, 0, 0, 0, 0, 0, 0, 0,
	292, 106, 107, 0, 0, 0, 289, 99, 105, 108,
	30, 0, 0, 31, 32, 33, 34, 35, 36, 37,
	38, 39, 40, 41, 42, 43, 44, 45, 46, 47,
	48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
	58, 59, 60, 61, 62, 63, 64, 65, 66, 67,
	68, 69, 70, 71, 72, 0, 0, 0, 0, 104,
	0, 0, 0, 0, 0, 0, 0, 251, 106, 107,
	0, 0, 249, 99, 105, 108, 30, 0, 0, 31,
	32, 33, 34, 35, 36, 37, 38, 39, 40, 41,
	42, 43, 44, 45, 46, 47, 48, 49, 50, 51,
	52, 53, 54, 55, 56, 57, 58, 59, 60, 61,
	62, 63, 64, 65, 66, 67, 68, 69, 70, 71,
	72, 0, 0, 0, 0, 104, 0, 0, 0, 0,
	0, 0, 0, 341, 106, 107, 99, 105, 108, 30,
	0, 0, 31, 32, 33, 34, 35, 36, 37, 38,
	39, 40, 41, 42, 43, 44, 45, 46, 47, 48,
	49, 50, 51, 52, 53, 54, 55, 56, 57, 58,
	59, 60, 61, 62, 63, 64, 65, 66, 67, 68,
	69, 70, 71, 72, 0, 0, 0, 0, 104, 0,
	0, 0, 0, 0, 0, 0, 339, 106, 107, 99,
	105, 108, 30, 0, 0, 31, 32, 33, 34, 35,
	36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
	46, 47, 48, 49, 50, 51, 52, 53, 54, 55,
	56, 57, 58, 59, 60, 61, 62, 63, 64, 65,
	66, 67, 68, 69, 70, 71, 72, 0, 0, 0,
	0, 104, 0, 0, 0, 0, 0, 0, 0, 30,
	106, 107, 31, 32, 33, 34, 35, 36, 37, 38,
	39, 40, 41, 42, 43, 44, 45, 46, 47, 48,
	49, 50, 51, 52, 53, 54, 55, 56, 57, 58,
	59, 60, 61, 62, 63, 64, 65, 66, 67, 68,
	69, 70, 71, 72, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 369, 0, 0, 0, 30,
	0, 162, 31, 32, 33, 34, 35, 36, 37, 38,
	39, 40, 41, 42, 43, 44, 45, 46, 47, 48,
	49, 50, 51, 52, 53, 54, 55, 56, 57, 58,
	59, 60, 61, 62, 63, 64, 65, 66, 67, 68,
	69, 70, 71, 72, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 368, 0, 0, 0, 30,
	0, 162, 31, 32, 33, 34, 35, 36, 37, 38,
	39, 40, 41, 42, 43, 44, 45, 46, 47, 48,
	49, 50, 51, 52, 53, 54, 55, 56, 57, 58,
	59, 60, 61, 62, 63, 64, 65, 66, 67, 68,
	69, 70, 71, 72, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 342, 0, 0, 0, 30,
	0, 162, 31, 32, 33, 34, 35, 36, 37, 38,
	39, 40, 41, 42, 43, 44, 45, 46, 47, 48,
	49, 50, 51, 52, 53, 54, 55, 56, 57, 58,
	59, 60, 61, 62, 63, 64, 65, 66, 67, 68,
	69, 70, 71, 72, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 319, 0, 0, 0, 30,
	0, 162, 31, 32, 33, 34, 35, 36, 37, 38,
	39, 40, 41, 42, 43, 44, 45, 46, 47, 48,
	49, 50, 51, 52, 53, 54, 55, 56, 57, 58,
	59, 60, 61, 62, 63, 64, 65, 66, 67, 68,
	69, 70, 71, 72, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 294, 0, 0, 0, 30,
	0, 162, 31, 32, 33, 34, 35, 36, 37, 38,
	39, 40, 41, 42, 43, 44, 45, 46, 47, 48,
	49, 50, 51, 52, 53, 54, 55, 56, 57, 58,
	59, 60, 61, 62, 63, 64, 65, 66, 67, 68,
	69, 70, 71, 72, 0, 0, 0, 0, 0, 203,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 30,
	0, 162, 31, 32, 33, 34, 35, 36, 37, 38,
	39, 40, 41, 42, 43, 44, 45, 46, 47, 48,
	49, 50, 51, 52, 53, 54, 55, 56, 57, 58,
	59, 60, 61, 62, 63, 64, 65, 66, 67, 68,
	69, 70, 71, 72, 226, 227, 228, 229, 230, 231,
	232, 233, 234, 235, 236, 237, 0, 0, 0, 30,
	29, 162, 31, 32, 33, 34, 35, 36, 37, 38,
	39, 40, 41, 42, 43, 44, 45, 46, 47, 48,
	49, 50, 51, 52, 53, 54, 55, 56, 57, 58,
	59, 60, 61, 62, 63, 64, 65, 66, 67, 68,
	69, 70, 71, 72, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 75,
	30, 29, 80, 31, 32, 33, 34, 35, 131, 37,
	38, 39, 40, 125, 124, 123, 44, 45, 46, 47,
	48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
	58, 59, 132, 133, 130, 63, 64, 134, 127, 128,
	129, 69, 70, 71, 72, 0, 0, 122, 0, 0,
	362, 30, 29, 80, 31, 32, 33, 34, 35, 131,
	37, 38, 39, 40, 125, 124, 123, 44, 45, 46,
	47, 48, 49, 50, 51, 52, 53, 54, 55, 56,
	57, 58, 59, 132, 133, 130, 63, 64, 134, 127,
	128, 129, 69, 70, 71, 72, 0, 0, 122, 0,
	0, 360, 30, 29, 80, 31, 32, 33, 34, 35,
	131, 37, 38, 39, 40, 125, 124, 123, 44, 45,
	46, 47, 48, 49, 50, 51, 52, 53, 54, 55,
	56, 57, 58, 59, 132, 133, 130, 63, 64, 134,
	127, 128, 129, 69, 70, 71, 72, 0, 0, 122,
	0, 0, 358, 30, 29, 80, 31, 32, 33, 34,
	35, 131, 37, 38, 39, 40, 41, 42, 43, 44,
	45, 46, 47, 48, 49, 50, 51, 52, 53, 54,
	55, 56, 57, 58, 59, 60, 61, 62, 63, 64,
	65, 66, 67, 68, 69, 70, 71, 72, 0, 0,
	272, 0, 0, 307, 30, 29, 80, 31, 32, 33,
	34, 35, 36, 37, 38, 39, 40, 125, 124, 123,
	44, 45, 46, 47, 48, 49, 50, 51, 52, 53,
	54, 55, 56, 57, 58, 59, 60, 61, 62, 63,
	64, 65, 66, 67, 68, 69, 70, 71, 72, 0,
	0, 149, 0, 0, 198, 30, 29, 80, 31, 32,
	33, 34, 35, 131, 37, 38, 39, 40, 125, 124,
	123, 44, 45, 46, 47, 48, 49, 50, 51, 52,
	53, 54, 55, 56, 57, 58, 59, 132, 133, 130,
	63, 64, 134, 127, 128, 129, 69, 70, 71, 72,
	0, 0, 122, 30, 0, 173, 31, 32, 33, 34,
	35, 131, 37, 38, 39, 40, 41, 42, 43, 44,
	45, 46, 47, 48, 49, 50, 51, 52, 53, 54,
	55, 56, 57, 58, 59, 60, 61, 62, 63, 64,
	143, 66, 67, 68, 69, 70, 71, 72, 0, 0,
	141, 0, 0, 190, 30, 29, 80, 31, 32, 33,
	34, 35, 131, 37, 38, 39, 40, 125, 124, 123,
	44, 45, 46, 47, 48, 49, 50, 51, 52, 53,
	54, 55, 56, 57, 58, 59, 132, 133, 130, 63,
	64, 134, 127, 128, 129, 69, 70, 71, 72, 0,
	0, 122, 30, 29, 80, 31, 32, 33, 34, 35,
	131, 37, 38, 39, 40, 41, 42, 43, 44, 45,
	46, 47, 48, 49, 50, 51, 52, 53, 54, 55,
	56, 57, 58, 59, 60, 61, 62, 63, 64, 65,
	66, 67, 68, 69, 70, 71, 72, 0, 0, 272,
	30, 29, 80, 31, 32, 33, 34, 35, 36, 37,
	38, 39, 40, 125, 124, 123, 44, 45, 46, 47,
	48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
	58, 59, 60, 61, 62, 63, 64, 65, 66, 67,
	68, 69, 70, 71, 72, 30, 0, 149, 31, 32,
	33, 34, 35, 131, 37, 38, 39, 40, 41, 42,
	43, 44, 45, 46, 47, 48, 49, 50, 51, 52,
	53, 54, 55, 56, 57, 58, 59, 60, 61, 62,
	63, 64, 143, 66, 67, 68, 69, 70, 71, 72,
	0, 0, 141, 30, 29, 80, 31, 32, 33, 34,
	35, 36, 37, 38, 39, 40, 41, 42, 43, 44,
	45, 46, 47, 48, 49, 50, 51, 52, 53, 54,
	55, 56, 57, 58, 59, 60, 61, 62, 63, 64,
	65, 66, 67, 68, 69, 70, 287, 72, 30, 29,
	80, 31, 32, 33, 34, 35, 36, 37, 38, 39,
	40, 41, 42, 43, 44, 45, 46, 47, 48, 49,
	50, 51, 52, 53, 54, 55, 56, 57, 58, 59,
	60, 61, 62, 63, 64, 65, 66, 67, 68, 69,
	70, 71, 72, 30, 29, 80, 31, 32, 33, 34,
	35, 36, 37, 38, 39, 40, 41, 42, 43, 44,
	45, 46, 47, 48, 49, 50, 51, 52, 53, 54,
	55, 56, 57, 58, 180, 60, 61, 62, 63, 64,
	65, 66, 67, 68, 69, 70, 71, 72, 30, 29,
	80, 31, 32, 33, 34, 35, 36, 37, 38, 39,
	40, 41, 42, 43, 44, 45, 46, 47, 48, 49,
	50, 51, 52, 53, 54, 55, 56, 57, 58, 178,
	60, 61, 62, 63, 64, 65, 66, 67, 68, 69,
	70, 71, 72, 30, 29, 80, 31, 32, 33, 34,
	35, 36, 37, 38, 39, 40, 41, 42, 43, 44,
	45, 46, 47, 48, 49, 50, 51, 52, 53, 54,
	55, 56, 57, 58, 176, 60, 61, 62, 63, 64,
	65, 66, 67, 68, 69, 70, 71, 72, 30, 29,
	0, 31, 32, 33, 34, 35, 36, 37, 38, 39,
	40, 41, 42, 43, 44, 45, 46, 47, 48, 49,
	50, 51, 52, 53, 54, 55, 56, 57, 58, 59,
	60, 61, 62, 63, 64, 65, 66, 67, 68, 69,
	70, 71, 72, 30, 0, 0, 31, 32, 33, 34,
	35, 36, 37, 38, 39, 40, 41, 42, 43, 44,
	45, 46, 47, 48, 49, 50, 51, 52, 53, 54,
	55, 56, 57, 58, 59, 60, 61, 62, 63, 64,
	65, 66, 67, 68, 69, 70, 71, 72,
}
var protoPact = [...]int{

	170, -1000, 195, 195, 169, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, 276, 1951, 1122, 1996, 1996, 1771,
	1996, 195, -1000, 302, 133, 300, 295, 129, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, 166, -1000, 1771, 73, 72, 71, -1000,
	-1000, 70, 120, -1000, 109, 107, -1000, 645, 19, 1537,
	1678, 1633, 139, -1000, -1000, -1000, 106, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, 1062, -1000, 273, 249, -1000, 105,
	1438, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, 1906, 1861, 1816, 1996, 1996, 1996, 1771,
	287, 1122, 1996, 33, 247, -1000, 1486, -1000, -1000, -1000,
	-1000, -1000, 161, 84, -1000, 1387, -1000, -1000, -1000, -1000,
	137, -1000, -1000, -1000, -1000, 1996, -1000, 1002, -1000, 51,
	40, -1000, 1951, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	105, -1000, 25, -1000, -1000, 1996, 1996, 1996, 1996, 1996,
	1996, 160, 4, -1000, 181, 68, 1089, 48, 47, -1000,
	-1000, -1000, 89, 45, -1000, 148, 117, 290, -1000, -1000,
	-1000, -1000, 23, -1000, -1000, -1000, -1000, 453, -1000, 1062,
	-37, -1000, 1771, 159, 158, 155, 152, 151, 150, 288,
	-1000, 1122, 287, 232, 1585, 38, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	277, 17, 16, 76, -1000, 231, 69, 1726, -1000, 386,
	-1000, 1062, 942, -1000, 12, 285, 271, 266, 260, 258,
	255, 14, -5, -1000, 149, -1000, -1000, -1000, 1336, -1000,
	-1000, -1000, -1000, 1996, 1771, -1000, -1000, 1122, -1000, 1122,
	-1000, -1000, -1000, -1000, -1000, -1000, 11, 1771, -1000, -1000,
	-20, -1000, 1062, 882, -1000, -1000, 13, 66, 9, 59,
	8, 50, -1000, 1122, 1122, 102, 645, -1000, -1000, 142,
	35, -6, -11, 79, -1000, -1000, 582, 519, 822, -1000,
	-1000, 1122, 1537, -1000, 1122, 1537, -1000, 1122, 1537, -15,
	-1000, -1000, -1000, 252, 1996, 92, 91, 5, -1000, 1062,
	-1000, 1062, -1000, -16, 1285, -17, 1234, -22, 1183, 90,
	7, 136, -1000, -1000, 1726, 762, 702, 87, -1000, 86,
	-1000, 82, -1000, -1000, -1000, 1122, 233, 1, -1000, -1000,
	-1000, -1000, -1000, -26, 6, 62, 77, -1000, 1122, -1000,
	128, -1000, -27, 108, -1000, -1000, -1000, 49, -1000, -1000,
	-1000,
}
var protoPgo = [...]int{

	0, 371, 366, 246, 315, 365, 364, 0, 12, 6,
	5, 361, 32, 21, 353, 52, 19, 18, 20, 7,
	8, 351, 349, 14, 348, 347, 345, 10, 11, 344,
	27, 342, 341, 25, 336, 297, 9, 17, 13, 333,
	326, 325, 324, 30, 16, 37, 15, 323, 322, 244,
	31, 321, 320, 231, 29, 316, 309, 26, 308, 307,
	2,
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
	21, 21, 21, 21, 21, 21, 48, 48, 45, 45,
	44, 44, 44, 47, 47, 46, 46, 46, 46, 46,
	46, 46, 41, 41, 42, 42, 43, 40, 40, 49,
	51, 51, 51, 50, 50, 50, 50, 52, 52, 52,
	52, 35, 37, 37, 37, 36, 36, 36, 36, 36,
	36, 36, 36, 36, 36, 36, 53, 55, 55, 55,
	54, 54, 54, 56, 58, 58, 58, 57, 57, 57,
	59, 59, 60, 60, 11, 11, 11, 10, 10, 18,
	18, 18, 18, 18, 18, 18, 18, 18, 18, 18,
	18, 18, 18, 18, 18, 18, 18, 18, 18, 18,
	18, 18, 18, 18, 18, 18, 18, 18, 18, 18,
	18, 18, 18, 18, 18, 18, 18, 18, 18, 18,
	18, 18,
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
	1, 3, 3, 3, 1, 1, 1, 3, 3, 3,
	3, 3, 3, 1, 3, 1, 3, 3, 1, 5,
	2, 1, 0, 1, 1, 1, 1, 4, 7, 4,
	7, 5, 2, 1, 0, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 5, 2, 1, 0,
	1, 1, 1, 5, 2, 1, 0, 1, 1, 1,
	10, 12, 2, 1, 2, 1, 0, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1,
}
var protoChk = [...]int{

	-1000, -1, -2, -4, 10, -3, -5, -6, -7, -35,
	-49, -53, -56, 54, 11, 14, 15, 46, 45, 47,
	48, -4, -3, 53, 4, 12, 13, -19, -18, 8,
	7, 10, 11, 12, 13, 14, 15, 16, 17, 18,
	19, 20, 21, 22, 23, 24, 25, 26, 27, 28,
	29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
	39, 40, 41, 42, 43, 44, 45, 46, 47, 48,
	49, 50, 51, -12, -19, 67, -18, -18, -20, -19,
	9, -18, 4, 54, 4, 4, 54, 53, -20, 56,
	56, 56, 56, 54, 54, 54, -15, -16, -17, 4,
	-24, -23, -25, -18, 56, 5, 65, 66, 6, 68,
	-37, -36, -30, -49, -35, -53, -48, -33, -7, -32,
	-34, -41, 54, 22, 21, 20, -20, 45, 46, 47,
	41, 15, 39, 40, 44, -43, -51, -50, -7, -52,
	-42, 54, -18, 44, -43, -55, -54, -30, -33, 54,
	-58, -57, -7, -59, 54, 49, 54, -27, -28, -29,
	-22, -18, 69, 5, 6, 18, 5, 6, 18, -13,
	-14, 9, 61, 57, -36, -20, 38, -20, 38, -20,
	38, -18, -45, -44, 5, -18, 64, -45, -40, 4,
	57, -50, 53, -47, -46, 5, -23, 66, 57, -54,
	57, -57, -18, 57, -28, 62, 54, 55, -17, 64,
	-19, -13, 67, -18, -18, -18, -18, -18, -18, 53,
	54, 69, 62, 42, 56, -21, 25, 26, 27, 28,
	29, 30, 31, 32, 33, 34, 35, 36, 54, 54,
	62, 5, -23, 62, 54, 42, 42, 67, -16, 69,
	-17, 64, -27, 70, -20, 53, 53, 53, 53, 53,
	53, 5, -9, -8, -12, -44, 5, 43, -39, -38,
	-7, -31, 54, -20, 62, 4, 54, 69, 54, 69,
	-46, 5, 43, -23, 5, 43, -60, 50, -20, 70,
	-26, -15, 64, -27, 63, 68, 5, 5, 5, 5,
	5, 5, 54, 69, 62, 70, 53, 57, -38, -18,
	-20, -9, -9, 68, -20, 70, 62, 54, -27, 63,
	54, 69, 56, 54, 69, 56, 54, 69, 56, -9,
	-8, 54, -15, 53, 63, 70, 70, 51, -15, 64,
	-15, 64, 63, -9, -37, -9, -37, -9, -37, 70,
	5, -18, 54, 54, 67, -27, -27, 70, 57, 70,
	57, 70, 57, 54, 54, 69, 53, -60, 63, 63,
	54, 54, 54, -9, 5, 68, 70, 54, 69, 54,
	56, 54, -9, -11, -10, -7, 54, 70, 57, -10,
	54,
}
var protoDef = [...]int{

	4, -2, 1, 2, 0, 6, 7, 8, 9, 10,
	11, 12, 13, 14, 0, 0, 0, 0, 0, 0,
	0, 3, 5, 0, 0, 0, 0, 0, 20, 21,
	179, 180, 181, 182, 183, 184, 185, 186, 187, 188,
	189, 190, 191, 192, 193, 194, 195, 196, 197, 198,
	199, 200, 201, 202, 203, 204, 205, 206, 207, 208,
	209, 210, 211, 212, 213, 214, 215, 216, 217, 218,
	219, 220, 221, 0, 23, 0, 0, 0, 0, 67,
	68, 0, 0, 16, 0, 0, 19, 0, 0, 144,
	132, 159, 166, 15, 17, 18, 0, 30, 31, 32,
	33, 34, 35, 36, 48, 37, 0, 0, 40, 24,
	0, 143, 145, 146, 147, 148, 149, 150, 151, 152,
	153, 154, 155, 0, 0, 0, 0, 0, 0, 0,
	211, 185, 0, 210, 214, 123, 0, 131, 133, 134,
	135, 136, 0, 214, 125, 0, 158, 160, 161, 162,
	0, 165, 167, 168, 169, 0, 22, 0, 46, 49,
	0, 59, 0, 38, 42, 43, 39, 41, 44, 25,
	26, 28, 0, 141, 142, 0, 0, 0, 0, 0,
	0, 0, 0, 109, 110, 0, 0, 0, 0, 128,
	129, 130, 0, 0, 114, 115, 116, 0, 156, 157,
	163, 164, 0, 45, 47, 50, 51, 0, 56, 48,
	0, 27, 0, 0, 0, 0, 0, 0, 0, 0,
	106, 0, 0, 0, 86, 0, 94, 95, 96, 97,
	98, 99, 100, 101, 102, 103, 104, 105, 122, 126,
	0, 0, 0, 0, 124, 0, 0, 0, 52, 0,
	55, 48, 0, 60, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 78, 0, 108, 111, 112, 0, 85,
	87, 88, 89, 0, 0, 127, 137, 0, 139, 0,
	113, 117, 120, 118, 119, 121, 0, 220, 173, 53,
	0, 61, 48, 0, 58, 29, 0, 0, 0, 0,
	0, 0, 72, 0, 0, 0, 0, 83, 84, 0,
	0, 0, 0, 0, 172, 54, 0, 0, 0, 57,
	69, 0, 144, 70, 0, 144, 71, 0, 144, 0,
	77, 107, 79, 0, 0, 0, 0, 0, 62, 48,
	63, 48, 64, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 138, 140, 0, 0, 0, 0, 80, 0,
	81, 0, 82, 76, 90, 0, 0, 0, 65, 66,
	73, 74, 75, 0, 0, 0, 0, 92, 0, 170,
	176, 91, 0, 0, 175, 177, 178, 0, 171, 174,
	93,
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
			r := &rangeNode{stNode: protoDollar[1].ui, enNode: protoDollar[1].ui, st: int32(protoDollar[1].ui.val), en: int32(protoDollar[1].ui.val)}
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
			r := &rangeNode{stNode: protoDollar[1].ui, enNode: protoDollar[3].ui, st: int32(protoDollar[1].ui.val), en: int32(protoDollar[3].ui.val)}
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
			r := &rangeNode{stNode: protoDollar[1].ui, enNode: protoDollar[3].id, st: int32(protoDollar[1].ui.val), en: internal.MaxTag}
			r.setRange(protoDollar[1].ui, protoDollar[3].id)
			protoVAL.rngs = []*rangeNode{r}
		}
	case 113:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:602
		{
			protoVAL.rngs = append(protoDollar[1].rngs, protoDollar[3].rngs...)
		}
	case 115:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:607
		{
			checkUint64InInt32Range(protolex, protoDollar[1].ui.start(), protoDollar[1].ui.val)
			r := &rangeNode{stNode: protoDollar[1].ui, enNode: protoDollar[1].ui, st: int32(protoDollar[1].ui.val), en: int32(protoDollar[1].ui.val)}
			r.setRange(protoDollar[1].ui, protoDollar[1].ui)
			protoVAL.rngs = []*rangeNode{r}
		}
	case 116:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:613
		{
			checkInt64InInt32Range(protolex, protoDollar[1].i.start(), protoDollar[1].i.val)
			r := &rangeNode{stNode: protoDollar[1].i, enNode: protoDollar[1].i, st: int32(protoDollar[1].i.val), en: int32(protoDollar[1].i.val)}
			r.setRange(protoDollar[1].i, protoDollar[1].i)
			protoVAL.rngs = []*rangeNode{r}
		}
	case 117:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:619
		{
			checkUint64InInt32Range(protolex, protoDollar[1].ui.start(), protoDollar[1].ui.val)
			checkUint64InInt32Range(protolex, protoDollar[3].ui.start(), protoDollar[3].ui.val)
			if protoDollar[1].ui.val > protoDollar[3].ui.val {
				lexError(protolex, protoDollar[1].ui.start(), fmt.Sprintf("range, %d to %d, is invalid: start must be <= end", protoDollar[1].ui.val, protoDollar[3].ui.val))
			}
			r := &rangeNode{stNode: protoDollar[1].ui, enNode: protoDollar[3].ui, st: int32(protoDollar[1].ui.val), en: int32(protoDollar[3].ui.val)}
			r.setRange(protoDollar[1].ui, protoDollar[3].ui)
			protoVAL.rngs = []*rangeNode{r}
		}
	case 118:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:629
		{
			checkInt64InInt32Range(protolex, protoDollar[1].i.start(), protoDollar[1].i.val)
			checkInt64InInt32Range(protolex, protoDollar[3].i.start(), protoDollar[3].i.val)
			if protoDollar[1].i.val > protoDollar[3].i.val {
				lexError(protolex, protoDollar[1].i.start(), fmt.Sprintf("range, %d to %d, is invalid: start must be <= end", protoDollar[1].i.val, protoDollar[3].i.val))
			}
			r := &rangeNode{stNode: protoDollar[1].i, enNode: protoDollar[3].i, st: int32(protoDollar[1].i.val), en: int32(protoDollar[3].i.val)}
			r.setRange(protoDollar[1].i, protoDollar[3].i)
			protoVAL.rngs = []*rangeNode{r}
		}
	case 119:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:639
		{
			checkInt64InInt32Range(protolex, protoDollar[1].i.start(), protoDollar[1].i.val)
			checkUint64InInt32Range(protolex, protoDollar[3].ui.start(), protoDollar[3].ui.val)
			r := &rangeNode{stNode: protoDollar[1].i, enNode: protoDollar[3].ui, st: int32(protoDollar[1].i.val), en: int32(protoDollar[3].ui.val)}
			r.setRange(protoDollar[1].i, protoDollar[3].ui)
			protoVAL.rngs = []*rangeNode{r}
		}
	case 120:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:646
		{
			checkUint64InInt32Range(protolex, protoDollar[1].ui.start(), protoDollar[1].ui.val)
			r := &rangeNode{stNode: protoDollar[1].ui, enNode: protoDollar[3].id, st: int32(protoDollar[1].ui.val), en: math.MaxInt32}
			r.setRange(protoDollar[1].ui, protoDollar[3].id)
			protoVAL.rngs = []*rangeNode{r}
		}
	case 121:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:652
		{
			checkInt64InInt32Range(protolex, protoDollar[1].i.start(), protoDollar[1].i.val)
			r := &rangeNode{stNode: protoDollar[1].i, enNode: protoDollar[3].id, st: int32(protoDollar[1].i.val), en: math.MaxInt32}
			r.setRange(protoDollar[1].i, protoDollar[3].id)
			protoVAL.rngs = []*rangeNode{r}
		}
	case 122:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:659
		{
			protoVAL.resvd = &reservedNode{ranges: protoDollar[2].rngs}
			protoVAL.resvd.setRange(protoDollar[1].id, protoDollar[3].b)
		}
	case 124:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:665
		{
			protoVAL.resvd = &reservedNode{ranges: protoDollar[2].rngs}
			protoVAL.resvd.setRange(protoDollar[1].id, protoDollar[3].b)
		}
	case 126:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:671
		{
			rsvd := map[string]struct{}{}
			for _, n := range protoDollar[2].names {
				if _, ok := rsvd[n.val]; ok {
					lexError(protolex, n.start(), fmt.Sprintf("name %q is reserved multiple times", n.val))
					break
				}
				rsvd[n.val] = struct{}{}
			}
			protoVAL.resvd = &reservedNode{names: protoDollar[2].names}
			protoVAL.resvd.setRange(protoDollar[1].id, protoDollar[3].b)
		}
	case 127:
		protoDollar = protoS[protopt-3 : protopt+1]
		//line proto.y:684
		{
			protoVAL.names = append(protoDollar[1].names, protoDollar[3].str)
		}
	case 128:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:687
		{
			protoVAL.names = []*stringLiteralNode{protoDollar[1].str}
		}
	case 129:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:691
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
	case 130:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:705
		{
			protoVAL.enDecls = append(protoDollar[1].enDecls, protoDollar[2].enDecls...)
		}
	case 132:
		protoDollar = protoS[protopt-0 : protopt+1]
		//line proto.y:709
		{
			protoVAL.enDecls = nil
		}
	case 133:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:713
		{
			protoVAL.enDecls = []*enumElement{{option: protoDollar[1].opts[0]}}
		}
	case 134:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:716
		{
			protoVAL.enDecls = []*enumElement{{value: protoDollar[1].env}}
		}
	case 135:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:719
		{
			protoVAL.enDecls = []*enumElement{{reserved: protoDollar[1].resvd}}
		}
	case 136:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:722
		{
			protoVAL.enDecls = []*enumElement{{empty: protoDollar[1].b}}
		}
	case 137:
		protoDollar = protoS[protopt-4 : protopt+1]
		//line proto.y:726
		{
			checkUint64InInt32Range(protolex, protoDollar[3].ui.start(), protoDollar[3].ui.val)
			protoVAL.env = &enumValueNode{name: protoDollar[1].id, numberP: protoDollar[3].ui}
			protoVAL.env.setRange(protoDollar[1].id, protoDollar[4].b)
		}
	case 138:
		protoDollar = protoS[protopt-7 : protopt+1]
		//line proto.y:731
		{
			checkUint64InInt32Range(protolex, protoDollar[3].ui.start(), protoDollar[3].ui.val)
			protoVAL.env = &enumValueNode{name: protoDollar[1].id, numberP: protoDollar[3].ui, options: protoDollar[5].opts}
			protoVAL.env.setRange(protoDollar[1].id, protoDollar[7].b)
		}
	case 139:
		protoDollar = protoS[protopt-4 : protopt+1]
		//line proto.y:736
		{
			checkInt64InInt32Range(protolex, protoDollar[3].i.start(), protoDollar[3].i.val)
			protoVAL.env = &enumValueNode{name: protoDollar[1].id, numberN: protoDollar[3].i}
			protoVAL.env.setRange(protoDollar[1].id, protoDollar[4].b)
		}
	case 140:
		protoDollar = protoS[protopt-7 : protopt+1]
		//line proto.y:741
		{
			checkInt64InInt32Range(protolex, protoDollar[3].i.start(), protoDollar[3].i.val)
			protoVAL.env = &enumValueNode{name: protoDollar[1].id, numberN: protoDollar[3].i, options: protoDollar[5].opts}
			protoVAL.env.setRange(protoDollar[1].id, protoDollar[7].b)
		}
	case 141:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:747
		{
			protoVAL.msg = &messageNode{name: protoDollar[2].id, decls: protoDollar[4].msgDecls}
			protoVAL.msg.setRange(protoDollar[1].id, protoDollar[5].b)
		}
	case 142:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:752
		{
			protoVAL.msgDecls = append(protoDollar[1].msgDecls, protoDollar[2].msgDecls...)
		}
	case 144:
		protoDollar = protoS[protopt-0 : protopt+1]
		//line proto.y:756
		{
			protoVAL.msgDecls = nil
		}
	case 145:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:760
		{
			protoVAL.msgDecls = []*messageElement{{field: protoDollar[1].fld}}
		}
	case 146:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:763
		{
			protoVAL.msgDecls = []*messageElement{{enum: protoDollar[1].en}}
		}
	case 147:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:766
		{
			protoVAL.msgDecls = []*messageElement{{nested: protoDollar[1].msg}}
		}
	case 148:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:769
		{
			protoVAL.msgDecls = []*messageElement{{extend: protoDollar[1].extend}}
		}
	case 149:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:772
		{
			protoVAL.msgDecls = []*messageElement{{extensionRange: protoDollar[1].ext}}
		}
	case 150:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:775
		{
			protoVAL.msgDecls = []*messageElement{{group: protoDollar[1].grp}}
		}
	case 151:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:778
		{
			protoVAL.msgDecls = []*messageElement{{option: protoDollar[1].opts[0]}}
		}
	case 152:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:781
		{
			protoVAL.msgDecls = []*messageElement{{oneOf: protoDollar[1].oo}}
		}
	case 153:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:784
		{
			protoVAL.msgDecls = []*messageElement{{mapField: protoDollar[1].mapFld}}
		}
	case 154:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:787
		{
			protoVAL.msgDecls = []*messageElement{{reserved: protoDollar[1].resvd}}
		}
	case 155:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:790
		{
			protoVAL.msgDecls = []*messageElement{{empty: protoDollar[1].b}}
		}
	case 156:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:794
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
	case 157:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:808
		{
			protoVAL.extDecls = append(protoDollar[1].extDecls, protoDollar[2].extDecls...)
		}
	case 159:
		protoDollar = protoS[protopt-0 : protopt+1]
		//line proto.y:812
		{
			protoVAL.extDecls = nil
		}
	case 160:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:816
		{
			protoVAL.extDecls = []*extendElement{{field: protoDollar[1].fld}}
		}
	case 161:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:819
		{
			protoVAL.extDecls = []*extendElement{{group: protoDollar[1].grp}}
		}
	case 162:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:822
		{
			protoVAL.extDecls = []*extendElement{{empty: protoDollar[1].b}}
		}
	case 163:
		protoDollar = protoS[protopt-5 : protopt+1]
		//line proto.y:826
		{
			protoVAL.svc = &serviceNode{name: protoDollar[2].id, decls: protoDollar[4].svcDecls}
			protoVAL.svc.setRange(protoDollar[1].id, protoDollar[5].b)
		}
	case 164:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:831
		{
			protoVAL.svcDecls = append(protoDollar[1].svcDecls, protoDollar[2].svcDecls...)
		}
	case 166:
		protoDollar = protoS[protopt-0 : protopt+1]
		//line proto.y:835
		{
			protoVAL.svcDecls = nil
		}
	case 167:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:842
		{
			protoVAL.svcDecls = []*serviceElement{{option: protoDollar[1].opts[0]}}
		}
	case 168:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:845
		{
			protoVAL.svcDecls = []*serviceElement{{rpc: protoDollar[1].mtd}}
		}
	case 169:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:848
		{
			protoVAL.svcDecls = []*serviceElement{{empty: protoDollar[1].b}}
		}
	case 170:
		protoDollar = protoS[protopt-10 : protopt+1]
		//line proto.y:852
		{
			protoVAL.mtd = &methodNode{name: protoDollar[2].id, input: protoDollar[4].rpcType, output: protoDollar[8].rpcType}
			protoVAL.mtd.setRange(protoDollar[1].id, protoDollar[10].b)
		}
	case 171:
		protoDollar = protoS[protopt-12 : protopt+1]
		//line proto.y:856
		{
			protoVAL.mtd = &methodNode{name: protoDollar[2].id, input: protoDollar[4].rpcType, output: protoDollar[8].rpcType, options: protoDollar[11].opts}
			protoVAL.mtd.setRange(protoDollar[1].id, protoDollar[12].b)
		}
	case 172:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:861
		{
			protoVAL.rpcType = &rpcTypeNode{msgType: protoDollar[2].id, streamKeyword: protoDollar[1].id}
			protoVAL.rpcType.setRange(protoDollar[1].id, protoDollar[2].id)
		}
	case 173:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:865
		{
			protoVAL.rpcType = &rpcTypeNode{msgType: protoDollar[1].id}
			protoVAL.rpcType.setRange(protoDollar[1].id, protoDollar[1].id)
		}
	case 174:
		protoDollar = protoS[protopt-2 : protopt+1]
		//line proto.y:870
		{
			protoVAL.opts = append(protoDollar[1].opts, protoDollar[2].opts...)
		}
	case 176:
		protoDollar = protoS[protopt-0 : protopt+1]
		//line proto.y:874
		{
			protoVAL.opts = nil
		}
	case 177:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:878
		{
			protoVAL.opts = protoDollar[1].opts
		}
	case 178:
		protoDollar = protoS[protopt-1 : protopt+1]
		//line proto.y:881
		{
			protoVAL.opts = nil
		}
	}
	goto protostack /* stack new state and value */
}
