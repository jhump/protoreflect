// Copyright 2020-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fastscan

import (
	"bytes"
	"io"
	"strings"

	"github.com/bufbuild/protocompile/ast"
	"github.com/bufbuild/protocompile/reporter"
)

var closeSymbol = map[tokenType]tokenType{
	openParenToken:   closeParenToken,
	openBraceToken:   closeBraceToken,
	openBracketToken: closeBracketToken,
	openAngleToken:   closeAngleToken,
}

// Result is the result of scanning a Protobuf source file. It contains the
// information extracted from the file.
type Result struct {
	PackageName string
	Imports     []Import
}

// Import represents an import in a Protobuf source file.
type Import struct {
	// Path of the imported file.
	Path string
	// Indicate if public or weak keyword was used in import statement.
	IsPublic, IsWeak bool
}

// SyntaxError is returned from Scan when one or more syntax errors are observed.
// Scan does not fully parse the source, so there are many kinds of syntax errors
// that will not be recognized. A full parser should be used to reliably detect
// errors in the source. But if the scanner happens to see things that are clearly
// wrong while scanning for the package and imports, it will return them. The
// slice contains one error for each location where a syntax issue is found.
type SyntaxError []reporter.ErrorWithPos

// Error implements the error interface, returning an error message with the
// details of the syntax error issues.
func (e SyntaxError) Error() string {
	var buf bytes.Buffer
	for i := range e {
		if i > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(e[i].Error())
	}
	return buf.String()
}

// Unwrap returns an error for each location where a syntax error was
// identified.
func (e SyntaxError) Unwrap() []error {
	slice := make([]error, len(e))
	for i := range e {
		slice[i] = e[i]
	}
	return slice
}

func newSyntaxError(errs []reporter.ErrorWithPos) error {
	if len(errs) == 0 {
		return nil
	}
	return SyntaxError(errs)
}

// Scan scans the given reader, which should contain Protobuf source, and
// returns the set of imports declared in the file. The result also contains the
// value of any package declaration in the file. It returns an error if there is
// an I/O error reading from r or if syntax errors are recognized while scanning.
// In the event of such an error, it will still return a result that contains as
// much information as was found (either before the I/O error occurred, or all
// that could be parsed despite syntax errors). The results are not necessarily
// valid, in that the parsed package name might not be a legal package name in
// protobuf or the imports may not refer to valid paths. Full validation of the
// source should be done using a full parser.
func Scan(filename string, r io.Reader) (Result, error) {
	var res Result

	var currentImport []string     // if non-nil, parsing an import statement
	var isPublic, isWeak bool      // if public or weak keyword observed in current import statement
	var packageComponents []string // if non-nil, parsing a package statement
	var syntaxErrs []reporter.ErrorWithPos

	// current stack of open blocks -- those starting with {, [, (, or < for
	// which we haven't yet encountered the closing }, ], ), or >
	var contextStack []tokenType
	declarationStart := true

	lexer := newLexer(r)

	if filename == "" {
		filename = "<input>"
	}
	getSpan := func(line, col int) ast.SourceSpan {
		pos := ast.SourcePos{
			Filename: filename,
			Line:     line,
			Col:      col,
		}
		return ast.NewSourceSpan(pos, pos)
	}
	getLatestSpan := func() ast.SourceSpan {
		return getSpan(lexer.prevTokenLine+1, lexer.prevTokenCol+1)
	}

	var prevLine, prevCol int
	for {
		token, text, err := lexer.Lex()
		if err != nil {
			return res, err
		}
		if token == eofToken {
			return res, newSyntaxError(syntaxErrs)
		}

		if currentImport != nil {
			switch token {
			case stringToken:
				currentImport = append(currentImport, text.(string)) //nolint:errcheck
			case identifierToken:
				ident := text.(string) //nolint:errcheck
				if len(currentImport) == 0 && (ident == "public" || ident == "weak") {
					isPublic = ident == "public"
					isWeak = ident == "weak"
					break
				}
				fallthrough
			default:
				if len(currentImport) > 0 {
					if token != semicolonToken {
						syntaxErrs = append(syntaxErrs,
							reporter.Errorf(getLatestSpan(),
								"unexpected %s; expecting semicolon", token.describe()),
						)
					}
					res.Imports = append(res.Imports, Import{
						Path:     strings.Join(currentImport, ""),
						IsPublic: isPublic,
						IsWeak:   isWeak,
					})
				} else {
					syntaxErrs = append(syntaxErrs,
						reporter.Errorf(getLatestSpan(),
							"unexpected %s; expecting import path string", token.describe()),
					)
				}
				currentImport = nil
			}
		}

		if packageComponents != nil {
			switch token {
			case identifierToken:
				if len(packageComponents) > 0 && packageComponents[len(packageComponents)-1] != "." {
					syntaxErrs = append(syntaxErrs,
						reporter.Errorf(getLatestSpan(),
							"package name should have a period between name components"),
					)
				}
				packageComponents = append(packageComponents, text.(string)) //nolint:errcheck
			case periodToken:
				if len(packageComponents) == 0 {
					syntaxErrs = append(syntaxErrs,
						reporter.Errorf(getLatestSpan(),
							"package name should not begin with a period"),
					)
				} else if packageComponents[len(packageComponents)-1] == "." {
					syntaxErrs = append(syntaxErrs,
						reporter.Errorf(getLatestSpan(),
							"package name should not have two periods in a row"),
					)
				}
				packageComponents = append(packageComponents, ".")
			default:
				if len(packageComponents) > 0 {
					if token != semicolonToken {
						syntaxErrs = append(syntaxErrs,
							reporter.Errorf(getLatestSpan(),
								"unexpected %s; expecting semicolon", token.describe()),
						)
					}
					if packageComponents[len(packageComponents)-1] == "." {
						syntaxErrs = append(syntaxErrs,
							reporter.Errorf(getSpan(prevLine+1, prevCol+1),
								"package name should not end with a period"),
						)
					}
					res.PackageName = strings.Join(packageComponents, "")
				} else {
					syntaxErrs = append(syntaxErrs,
						reporter.Errorf(getLatestSpan(),
							"unexpected %s; expecting package name", token.describe()),
					)
				}
				packageComponents = nil
			}
		}

		switch token {
		case openParenToken, openBraceToken, openBracketToken, openAngleToken:
			contextStack = append(contextStack, closeSymbol[token])
		case closeParenToken, closeBraceToken, closeBracketToken, closeAngleToken:
			if len(contextStack) > 0 && contextStack[len(contextStack)-1] == token {
				contextStack = contextStack[:len(contextStack)-1]
			}
		case identifierToken:
			if declarationStart && len(contextStack) == 0 {
				if text == "import" {
					currentImport = []string{}
					isPublic, isWeak = false, false
				} else if text == "package" {
					packageComponents = []string{}
				}
			}
		}

		declarationStart = token == closeBraceToken || token == semicolonToken
		prevLine, prevCol = lexer.prevTokenLine, lexer.prevTokenCol
	}
}
