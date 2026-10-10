package protoprint

import (
	"bytes"
	"fmt"
	"unicode"
	"unicode/utf8"
)

// quotedBytes implements the text format for string literals for protocol
// buffers. Since the underlying data is a bytes field, this encodes all
// bytes outside the 7-bit ASCII printable range. To preserve unicode strings
// without byte escapes, use quotedString.
func quotedBytes(s string) string {
	var b bytes.Buffer
	b.WriteByte('"')
	// Loop over the bytes, not the runes.
	for i := 0; i < len(s); i++ {
		// Divergence from C++: we don't escape apostrophes.
		// There's no need to escape them, and the C++ parser
		// copes with a naked apostrophe.
		switch c := s[i]; c {
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		case '"':
			b.WriteString("\\\"")
		case '\\':
			b.WriteString("\\\\")
		default:
			if c >= 0x20 && c < 0x7f {
				b.WriteByte(c)
			} else {
				_, _ = fmt.Fprintf(&b, "\\%03o", c)
			}
		}
	}
	b.WriteByte('"')

	return b.String()
}

// quotedString implements the text format for string literals for protocol
// buffers. This form is also acceptable for string literals in option values
// by the protocol buffer compiler, protoc.
func quotedString(s string) string {
	var b bytes.Buffer
	b.WriteByte('"')
	// Loop over the bytes, not the runes.
	for {
		r, n := utf8.DecodeRuneInString(s)
		if n == 0 {
			break // end of string
		}
		if r == utf8.RuneError && n == 1 {
			// Invalid UTF8! Use an octal byte escape to encode the bad byte.
			_, _ = fmt.Fprintf(&b, "\\%03o", s[0])
			s = s[1:]
			continue
		}

		// Divergence from C++: we don't escape apostrophes.
		// There's no need to escape them, and the C++ parser
		// copes with a naked apostrophe.
		switch r {
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		case '"':
			b.WriteString("\\\"")
		case '\\':
			b.WriteString("\\\\")
		default:
			if unicode.IsPrint(r) {
				b.WriteRune(r)
			} else {
				// if it's not printable, use a unicode escape
				if r > 0xffff {
					_, _ = fmt.Fprintf(&b, "\\U%08X", r)
				} else if r > 0x7F {
					_, _ = fmt.Fprintf(&b, "\\u%04X", r)
				} else {
					_, _ = fmt.Fprintf(&b, "\\%03o", byte(r))
				}
			}
		}

		s = s[n:]
	}

	b.WriteByte('"')

	return b.String()
}
