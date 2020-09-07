package ast

import "fmt"

// SourcePos identifies a location in a proto source file.
type SourcePos struct {
	Filename  string
	Line, Col int
	Offset    int
}

func (pos SourcePos) String() string {
	if pos.Line <= 0 || pos.Col <= 0 {
		return pos.Filename
	}
	return fmt.Sprintf("%s:%d:%d", pos.Filename, pos.Line, pos.Col)
}

func UnknownPos(filename string) *SourcePos {
	return &SourcePos{Filename: filename}
}

type PosRange struct {
	Start, End SourcePos
}

type Comment struct {
	PosRange
	Text string
}
