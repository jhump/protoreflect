package protoprint

import (
	"fmt"
	"io"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/jhump/protoreflect/v2/sourceloc"
)

func inline(indent int) int {
	if indent < 0 {
		// already inlined
		return indent
	}
	// negative indent means inline; indent 2 stops further in case value wraps
	return -indent - 2
}

func (p *Printer) printBlockElement(
	isDecriptor bool,
	si protoreflect.SourceLocation,
	w *writer,
	indent int,
	el func(w *writer, trailer func(indent int, wantTrailingNewline bool)),
) {
	includeComments := isDecriptor || p.includeCommentType(CommentsTokens)

	if includeComments && si.Path != nil {
		p.printLeadingComments(si, w, indent)
	}
	el(w, func(indent int, wantTrailingNewline bool) {
		if includeComments && !sourceloc.IsZero(si) {
			if p.printTrailingComments(si, w, indent) && wantTrailingNewline && !p.Compact {
				// separator line between trailing comment and next element
				_, _ = fmt.Fprintln(w)
			}
		}
	})
	if indent >= 0 && !w.newline {
		// if we're not printing inline but element did not have trailing newline, add one now
		_, _ = fmt.Fprintln(w)
	}
}

func (p *Printer) printElement(isDecriptor bool, si protoreflect.SourceLocation, w *writer, indent int, el func(*writer)) {
	includeComments := isDecriptor || p.includeCommentType(CommentsTokens)

	if includeComments && !sourceloc.IsZero(si) {
		p.printLeadingComments(si, w, indent)
	}
	el(w)
	if includeComments && !sourceloc.IsZero(si) {
		p.printTrailingComments(si, w, indent)
	}
	if indent >= 0 && !w.newline {
		// if we're not printing inline but element did not have trailing newline, add one now
		_, _ = fmt.Fprintln(w)
	}
}

func (p *Printer) printElementString(si protoreflect.SourceLocation, w *writer, indent int, str string) {
	p.printElement(false, si, w, inline(indent), func(w *writer) {
		_, _ = fmt.Fprintf(w, "%s ", str)
	})
}

func (p *Printer) includeCommentType(c CommentType) bool {
	return (p.OmitComments & c) == 0
}

func (p *Printer) printLeadingComments(si protoreflect.SourceLocation, w *writer, indent int) bool {
	endsInNewLine := false

	if p.includeCommentType(CommentsDetached) {
		for _, c := range si.LeadingDetachedComments {
			if p.printComment(c, w, indent, true) {
				// if comment ended in newline, add another newline to separate
				// this comment from the next
				p.newLine(w)
				endsInNewLine = true
			} else if indent < 0 {
				// comment did not end in newline and we are trying to inline?
				// just add a space to separate this comment from what follows
				_, _ = fmt.Fprint(w, " ")
				endsInNewLine = false
			} else {
				// comment did not end in newline and we are *not* trying to inline?
				// add newline to end of comment and add another to separate this
				// comment from what follows
				_, _ = fmt.Fprintln(w) // needed to end comment, regardless of p.Compact
				p.newLine(w)
				endsInNewLine = true
			}
		}
	}

	if p.includeCommentType(CommentsLeading) && si.LeadingComments != "" {
		endsInNewLine = p.printComment(si.LeadingComments, w, indent, true)
		if !endsInNewLine {
			if indent >= 0 {
				// leading comment didn't end with newline but needs one
				// (because we're *not* inlining)
				_, _ = fmt.Fprintln(w) // needed to end comment, regardless of p.Compact
				endsInNewLine = true
			} else {
				// space between comment and following element when inlined
				_, _ = fmt.Fprint(w, " ")
			}
		}
	}

	return endsInNewLine
}

func (p *Printer) printTrailingComments(si protoreflect.SourceLocation, w *writer, indent int) bool {
	if p.includeCommentType(CommentsTrailing) && si.TrailingComments != "" {
		if !p.printComment(si.TrailingComments, w, indent, p.TrailingCommentsOnSeparateLine) && indent >= 0 {
			// trailing comment didn't end with newline but needs one
			// (because we're *not* inlining)
			_, _ = fmt.Fprintln(w) // needed to end comment, regardless of p.Compact
		} else if indent < 0 {
			_, _ = fmt.Fprint(w, " ")
		}
		return true
	}

	return false
}

func (p *Printer) printComment(comments string, w *writer, indent int, forceNextLine bool) bool {
	if comments == "" {
		return false
	}

	var multiLine bool
	if indent < 0 {
		// use multi-line style when inlining
		multiLine = true
	} else {
		multiLine = p.PreferMultiLineStyleComments
	}
	if multiLine && strings.Contains(comments, "*/") {
		// can't emit '*/' in a multi-line style comment
		multiLine = false
	}

	lines := strings.Split(comments, "\n")

	// first, remove leading and trailing blank lines
	if lines[0] == "" {
		lines = lines[1:]
	}
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return false
	}

	if indent >= 0 && !w.newline {
		// last element did not have trailing newline, so we
		// either need to tack on newline or, if comment is
		// just one line, inline it on the end
		if forceNextLine || len(lines) > 1 {
			_, _ = fmt.Fprintln(w)
		} else {
			if !w.space {
				_, _ = fmt.Fprint(w, " ")
			}
			indent = inline(indent)
		}
	}

	if len(lines) == 1 && multiLine {
		p.indent(w, indent)
		line := lines[0]
		if line != "" && line[0] == ' ' && line[len(line)-1] != ' ' {
			// add trailing space for symmetry
			line += " "
		}
		_, _ = fmt.Fprintf(w, "/*%s*/", line)
		if indent >= 0 {
			_, _ = fmt.Fprintln(w)
			return true
		}
		return false
	}

	if multiLine {
		// multi-line style comments that actually span multiple lines
		// get a blank line before and after so that comment renders nicely
		lines = append(lines, "", "")
		copy(lines[1:], lines)
		lines[0] = ""
	}

	for i, l := range lines {
		if l != "" && !strings.HasPrefix(l, " ") {
			l = " " + l
		}
		p.maybeIndent(w, indent, i > 0)
		if multiLine {
			if i == 0 {
				// first line
				_, _ = fmt.Fprintf(w, "/*%s\n", strings.TrimRight(l, " \t"))
			} else if i == len(lines)-1 {
				// last line
				if strings.TrimSpace(l) == "" {
					_, _ = fmt.Fprint(w, " */")
				} else {
					_, _ = fmt.Fprintf(w, " *%s*/", l)
				}
				if indent >= 0 {
					_, _ = fmt.Fprintln(w)
				}
			} else {
				_, _ = fmt.Fprintf(w, " *%s\n", strings.TrimRight(l, " \t"))
			}
		} else {
			_, _ = fmt.Fprintf(w, "//%s\n", strings.TrimRight(l, " \t"))
		}
	}

	// single-line comments always end in newline; multi-line comments only
	// end in newline for non-negative (e.g. non-inlined) indentation
	return !multiLine || indent >= 0
}

func (p *Printer) indent(w io.Writer, indent int) {
	for range indent {
		_, _ = fmt.Fprint(w, p.Indent)
	}
}

func (p *Printer) maybeIndent(w io.Writer, indent int, requireIndent bool) {
	if indent < 0 && requireIndent {
		p.indent(w, -indent)
	} else {
		p.indent(w, indent)
	}
}

type writer struct {
	io.Writer
	err     error
	space   bool
	newline bool
}

func newWriter(w io.Writer) *writer {
	return &writer{Writer: w, newline: true}
}

func (w *writer) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	w.newline = false

	if w.space {
		// skip any trailing space if the following
		// character is semicolon, comma, or close bracket
		if p[0] != ';' && p[0] != ',' {
			_, err := w.Writer.Write([]byte{' '})
			if err != nil {
				w.err = err
				return 0, err
			}
		}
		w.space = false
	}

	if p[len(p)-1] == ' ' {
		w.space = true
		p = p[:len(p)-1]
	}
	if len(p) > 0 && p[len(p)-1] == '\n' {
		w.newline = true
	}

	num, err := w.Writer.Write(p)
	if err != nil {
		w.err = err
	} else if w.space {
		// pretend space was written
		num++
	}
	return num, err
}
