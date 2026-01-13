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

// Package cases provides functions for inter-converting between different
// case styles.
package cases

import (
	"iter"
	"strings"
	"unicode"
)

// Case is a target case style to convert to.
type Case int

const (
	Snake  Case = iota // snake_case
	Enum               // ENUM_CASE
	Camel              // camelCase
	Pascal             // PascalCase
)

// Convert converts str to the given case.
func (c Case) Convert(str string) string {
	return Converter{Case: c}.Convert(str)
}

// Converter contains specific options for converting to a given case.
type Converter struct {
	Case Case

	// If set, word boundaries are only underscores, which is the naive
	// word splitting algorithm used by protoc.
	NaiveSplit bool

	// If set, runes will not be converted to lowercase as part of the
	// conversion.
	NoLowercase bool
}

// Convert convert str according to the options set in this converter.
func (c Converter) Convert(str string) string {
	buf := new(strings.Builder)
	c.Append(buf, str)
	return buf.String()
}

// Append is like [Converter.Convert], but it appends to the given buffer
// instead.
func (c Converter) Append(buf *strings.Builder, str string) {
	var iter iter.Seq[string]
	if c.NaiveSplit {
		iter = strings.SplitSeq(str, "_")
	} else {
		iter = Words(str)
	}
	c.Case.convert(buf, !c.NoLowercase, iter)
}

func (c Case) convert(buf *strings.Builder, lowercase bool, words iter.Seq[string]) {
	switch c {
	case Snake, Enum:
		uppercase := c == Enum
		first := true
		for word := range words {
			if !first {
				buf.WriteRune('_')
			}
			for _, r := range word {
				if uppercase || lowercase {
					buf.WriteRune(setCase(r, uppercase))
				}
			}
			first = false
		}
	case Camel, Pascal:
		uppercase := c == Pascal
		firstWord := true
		for word := range words {
			firstRune := true
			for _, r := range word {
				uppercase := (uppercase || !firstWord) && firstRune
				if uppercase || lowercase {
					r = setCase(r, uppercase)
				}
				buf.WriteRune(r)
				firstRune = false
			}
			firstWord = false
		}
	}
}

func setCase(r rune, upper bool) rune {
	if upper {
		return unicode.ToUpper(r)
	}
	return unicode.ToLower(r)
}
