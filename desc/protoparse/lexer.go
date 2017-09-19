package protoparse

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
)

type runeReader struct {
	rr     *bufio.Reader
	unread []rune
	err    error
}

func (rr *runeReader) ReadRune() (r rune, size int, err error) {
	if rr.err != nil {
		return 0, 0, rr.err
	}
	if len(rr.unread) > 0 {
		r := rr.unread[len(rr.unread)-1]
		rr.unread = rr.unread[:len(rr.unread)-1]
		return r, utf8.RuneLen(r), nil
	}
	r, sz, err := rr.rr.ReadRune()
	if err != nil {
		rr.err = err
	}
	return r, sz, err
}

func (rr *runeReader) UnreadRune(r rune) {
	rr.unread = append(rr.unread, r)
}

type protoLex struct {
	input *runeReader
	err   error
	res   *dpb.FileDescriptorProto

	aggregates map[string][]*aggregate

	lineNo int
	colNo  int

	prevLineNo int
	prevColNo  int
}

func newLexer(in io.Reader) *protoLex {
	return &protoLex{input: &runeReader{rr: bufio.NewReader(in)}}
}

var keywords = map[string]int{
	"syntax":     _SYNTAX,
	"import":     _IMPORT,
	"weak":       _WEAK,
	"public":     _PUBLIC,
	"package":    _PACKAGE,
	"option":     _OPTION,
	"true":       _TRUE,
	"false":      _FALSE,
	"inf":        _INF,
	"nan":        _NAN,
	"repeated":   _REPEATED,
	"optional":   _OPTIONAL,
	"required":   _REQUIRED,
	"double":     _DOUBLE,
	"float":      _FLOAT,
	"int32":      _INT32,
	"int64":      _INT64,
	"uint32":     _UINT32,
	"uint64":     _UINT64,
	"sint32":     _SINT32,
	"sint64":     _SINT64,
	"fixed32":    _FIXED32,
	"fixed64":    _FIXED64,
	"sfixed32":   _SFIXED32,
	"sfixed64":   _SFIXED64,
	"bool":       _BOOL,
	"string":     _STRING,
	"bytes":      _BYTES,
	"group":      _GROUP,
	"oneof":      _ONEOF,
	"map":        _MAP,
	"extensions": _EXTENSIONS,
	"to":         _TO,
	"max":        _MAX,
	"reserved":   _RESERVED,
	"enum":       _ENUM,
	"message":    _MESSAGE,
	"extend":     _EXTEND,
	"service":    _SERVICE,
	"rpc":        _RPC,
	"stream":     _STREAM,
	"returns":    _RETURNS,
}

func (l *protoLex) Lex(lval *protoSymType) (code int) {
	// TODO: substantial work but also subtantial improvement: include location
	// for every token and build source_code_info for resulting file descriptor
	// (allows locations to be shown in post-parse validation errors and enables
	// more accurate location information in errors encountered during parsing)

	if l.err != nil {
		lval.u = l.err
		return _ERROR
	}

	defer func() {
		if code == _ERROR && l.err == nil {
			l.err = lval.u.(error)
		}
	}()

	l.prevLineNo = l.lineNo
	l.prevColNo = l.colNo

	for {
		c, _, err := l.input.ReadRune()
		if err == io.EOF {
			return 0
		} else if err != nil {
			lval.u = err
			return _ERROR
		}

		if c == '\n' || c == '\r' {
			l.colNo = 0
			l.lineNo++
			continue
		}
		l.colNo++
		if c == ' ' || c == '\t' {
			continue
		}

		l.prevLineNo = l.lineNo
		l.prevColNo = l.colNo

		if c == '.' {
			// tokens that start with a dot include type names and decimal literals
			cn, _, err := l.input.ReadRune()
			if err != nil {
				return int(c)
			}
			if cn == '_' || (cn >= 'a' && cn <= 'z') || (cn >= 'A' && cn <= 'Z') {
				l.colNo++
				token := []rune{c, cn}
				token = l.readIdentifier(token)
				lval.str = string(token)
				return _TYPENAME
			}
			if cn >= '0' && cn <= '9' {
				l.colNo++
				token := []rune{c, cn}
				token = l.readNumber(token, false, true)
				lval.f, err = strconv.ParseFloat(string(token), 64)
				if err != nil {
					lval.u = err
					return _ERROR
				}
				return _FLOAT_LIT
			}
			l.input.UnreadRune(cn)
			return int(c)
		}

		if c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			// identifier
			token := []rune{c}
			token = l.readIdentifier(token)
			lval.str = string(token)
			if strings.Contains(lval.str, ".") {
				return _FQNAME
			}
			if t, ok := keywords[lval.str]; ok {
				return t
			}
			return _NAME
		}

		if c >= '0' && c <= '9' {
			// integer or float literal
			if c == '0' {
				cn, _, err := l.input.ReadRune()
				if err != nil {
					lval.ui = 0
					return _INT_LIT
				}
				if cn == 'x' || cn == 'X' {
					cnn, _, err := l.input.ReadRune()
					if err != nil {
						l.input.UnreadRune(cn)
						lval.ui = 0
						return _INT_LIT
					}
					if (cnn >= '0' && cnn <= '9') || (cnn >= 'a' && cnn <= 'f') || (cnn >= 'A' && cnn <= 'F') {
						// hexadecimal!
						l.colNo += 2
						token := []rune{cnn}
						token = l.readHexNumber(token)
						lval.ui, err = strconv.ParseUint(string(token), 16, 64)
						if err != nil {
							lval.u = err
							return _ERROR
						}
						return _INT_LIT
					}
					l.input.UnreadRune(cnn)
					l.input.UnreadRune(cn)
					lval.ui = 0
					return _INT_LIT
				} else {
					l.input.UnreadRune(cn)
				}
			}
			token := []rune{c}
			token = l.readNumber(token, true, true)
			numstr := string(token)
			if strings.Contains(numstr, ".") || strings.Contains(numstr, "e") || strings.Contains(numstr, "E") {
				// floating point!
				lval.f, err = strconv.ParseFloat(numstr, 64)
				if err != nil {
					lval.u = err
					return _ERROR
				}
				return _FLOAT_LIT
			}
			// integer! (decimal or octal)
			lval.ui, err = strconv.ParseUint(numstr, 0, 64)
			if err != nil {
				lval.u = err
				return _ERROR
			}
			return _INT_LIT
		}

		if c == '\'' || c == '"' {
			// string literal
			lval.str, err = l.readStringLiteral(c)
			if err != nil {
				lval.u = err
				return _ERROR
			}
			return _STRING_LIT
		}

		if c == '/' {
			// comment
			cn, _, err := l.input.ReadRune()
			if err != nil {
				return int(c)
			}
			if cn == '/' {
				l.skipToEndOfLine()
				continue
			}
			if cn == '*' {
				if !l.skipToEndOfBlockComment() {
					lval.u = errors.New("Block comment never terminates, unexpected EOF")
					return _ERROR
				}
				continue
			}
			l.input.UnreadRune(cn)
		}

		return int(c)
	}
}

func (l *protoLex) readNumber(sofar []rune, allowDot bool, allowExp bool) []rune {
	token := sofar
	for {
		c, _, err := l.input.ReadRune()
		if err != nil {
			break
		}
		if c == '.' {
			if !allowDot {
				l.input.UnreadRune(c)
				break
			}
			allowDot = false
			cn, _, err := l.input.ReadRune()
			if err != nil {
				l.input.UnreadRune(c)
				break
			}
			if cn < '0' || cn > '9' {
				l.input.UnreadRune(cn)
				l.input.UnreadRune(c)
				break
			}
			l.colNo++
			token = append(token, c)
			c = cn
		} else if c == 'e' || c == 'E' {
			if !allowExp {
				l.input.UnreadRune(c)
				break
			}
			allowExp = false
			cn, _, err := l.input.ReadRune()
			if err != nil {
				l.input.UnreadRune(c)
				break
			}
			if cn == '-' || cn == '+' {
				cnn, _, err := l.input.ReadRune()
				if err != nil {
					l.input.UnreadRune(cn)
					l.input.UnreadRune(c)
					break
				}
				if cnn < '0' || cnn > '9' {
					l.input.UnreadRune(cnn)
					l.input.UnreadRune(cn)
					l.input.UnreadRune(c)
					break
				}
				l.colNo++
				token = append(token, c)
				c = cn
				cn = cnn
			} else if cn < '0' || cn > '9' {
				l.input.UnreadRune(cn)
				l.input.UnreadRune(c)
				break
			}
			l.colNo++
			token = append(token, c)
			c = cn
		} else if c < '0' || c > '9' {
			l.input.UnreadRune(c)
			break
		}
		l.colNo++
		token = append(token, c)
	}
	return token
}

func (l *protoLex) readHexNumber(sofar []rune) []rune {
	token := sofar
	for {
		c, _, err := l.input.ReadRune()
		if err != nil {
			break
		}
		if (c < 'a' || c > 'f') && (c < 'A' || c > 'F') && (c < '0' || c > '9') {
			l.input.UnreadRune(c)
			break
		}
		l.colNo++
		token = append(token, c)
	}
	return token
}

func (l *protoLex) readIdentifier(sofar []rune) []rune {
	token := sofar
	for {
		c, _, err := l.input.ReadRune()
		if err != nil {
			break
		}
		if c == '.' {
			cn, _, err := l.input.ReadRune()
			if err != nil {
				l.input.UnreadRune(c)
				break
			}
			if cn != '_' && (cn < 'a' || cn > 'z') && (cn < 'A' || cn > 'Z') {
				l.input.UnreadRune(cn)
				l.input.UnreadRune(c)
				break
			}
			l.colNo++
			token = append(token, c)
			c = cn
		} else if c != '_' && (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
			l.input.UnreadRune(c)
			break
		}
		l.colNo++
		token = append(token, c)
	}
	return token
}

func (l *protoLex) readStringLiteral(quote rune) (string, error) {
	var buf bytes.Buffer
	for {
		c, _, err := l.input.ReadRune()
		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return "", err
		}
		if c == '\n' {
			l.colNo = 0
			l.lineNo++
			return "", errors.New("encountered end-of-line before end of string literal")
		}
		l.colNo++
		if c == quote {
			break
		}
		if c == 0 {
			return "", errors.New("null character ('\\0') not allowed in string literal")
		}
		if c == '\\' {
			// escape sequence
			c, _, err = l.input.ReadRune()
			if err != nil {
				return "", err
			}
			l.colNo++
			if c == 'x' || c == 'X' {
				// hex escape
				c, _, err := l.input.ReadRune()
				if err != nil {
					return "", err
				}
				l.colNo++
				c2, _, err := l.input.ReadRune()
				if err != nil {
					return "", err
				}
				var hex string
				if (c2 < '0' || c2 > '9') && (c2 < 'a' || c2 > 'f') && (c2 < 'A' || c2 > 'F') {
					l.input.UnreadRune(c2)
					hex = string(c)
				} else {
					l.colNo++
					hex = string([]rune{c, c2})
				}
				i, err := strconv.ParseInt(hex, 16, 32)
				if err != nil {
					return "", fmt.Errorf("invalid hex escape: \\x%q", hex)
				}
				buf.WriteByte(byte(i))

			} else if c >= '0' && c <= '7' {
				// octal escape
				c2, _, err := l.input.ReadRune()
				if err != nil {
					return "", err
				}
				var octal string
				if c2 < '0' || c2 > '7' {
					l.input.UnreadRune(c2)
					octal = string(c)
				} else {
					l.colNo++
					c3, _, err := l.input.ReadRune()
					if err != nil {
						return "", err
					}
					if c3 < '0' || c3 > '7' {
						l.input.UnreadRune(c3)
						octal = string([]rune{c, c2})
					} else {
						l.colNo++
						octal = string([]rune{c, c2, c3})
					}
				}
				i, err := strconv.ParseInt(octal, 8, 32)
				if err != nil {
					return "", fmt.Errorf("invalid octal escape: \\%q", octal)
				}
				if i > 0xff {
					return "", fmt.Errorf("octal escape is out range, must be between 0 and 377: \\%q", octal)
				}
				buf.WriteByte(byte(i))

			} else if c == 'u' {
				// short unicode escape
				u := make([]rune, 4)
				for i := range u {
					c, _, err := l.input.ReadRune()
					if err != nil {
						return "", err
					}
					l.colNo++
					u[i] = c
				}
				i, err := strconv.ParseInt(string(u), 16, 32)
				if err != nil {
					return "", fmt.Errorf("invalid unicode escape: \\u%q", string(u))
				}
				buf.WriteRune(rune(i))

			} else if c == 'U' {
				// long unicode escape
				u := make([]rune, 8)
				for i := range u {
					c, _, err := l.input.ReadRune()
					if err != nil {
						return "", err
					}
					l.colNo++
					u[i] = c
				}
				i, err := strconv.ParseInt(string(u), 16, 32)
				if err != nil {
					return "", fmt.Errorf("invalid unicode escape: \\U%q", string(u))
				}
				if i > 0x10ffff || i < 0 {
					return "", fmt.Errorf("unicode escape is out of range, must be between 0 and 0x10ffff: \\U%q", string(u))
				}
				buf.WriteRune(rune(i))

			} else if c == 'a' {
				buf.WriteByte('\a')
			} else if c == 'b' {
				buf.WriteByte('\b')
			} else if c == 'f' {
				buf.WriteByte('\f')
			} else if c == 'n' {
				buf.WriteByte('\n')
			} else if c == 'r' {
				buf.WriteByte('\r')
			} else if c == 't' {
				buf.WriteByte('\t')
			} else if c == 'v' {
				buf.WriteByte('\v')
			} else if c == '\\' {
				buf.WriteByte('\\')
			} else if c == '\'' {
				buf.WriteByte('\'')
			} else if c == '"' {
				buf.WriteByte('"')
			} else if c == '?' {
				buf.WriteByte('?')
			} else {
				return "", fmt.Errorf("invalid escape sequence: %q", "\\"+string(c))
			}
		} else {
			buf.WriteRune(c)
		}
	}
	return buf.String(), nil
}

func (l *protoLex) skipToEndOfLine() {
	for {
		c, _, err := l.input.ReadRune()
		if err != nil {
			return
		}
		if c == '\n' {
			l.colNo = 0
			l.lineNo++
			return
		}
		l.colNo++
	}
}

func (l *protoLex) skipToEndOfBlockComment() bool {
	for {
		c, _, err := l.input.ReadRune()
		if err != nil {
			return false
		}
		if c == '\n' {
			l.colNo = 0
			l.lineNo++
		} else {
			l.colNo++
		}
		if c == '*' {
			c, _, err := l.input.ReadRune()
			if err != nil {
				return false
			}
			if c == '/' {
				l.colNo++
				return true
			}
			l.input.UnreadRune(c)
		}
	}
}

func (l *protoLex) Error(s string) {
	if l.err == nil {
		l.err = errors.New(s)
	}
}
