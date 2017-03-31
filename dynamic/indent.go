package dynamic

import "bytes"

type indentBuffer struct {
	bytes.Buffer
	indent int
	comma  bool
}

func (b *indentBuffer) start() error {
	if b.indent >= 0 {
		b.indent++
		return b.newLine(false)
	}
	return nil
}

func (b *indentBuffer) sep() error {
	if b.indent >= 0 {
		_, err := b.WriteString(": ")
		return err
	} else {
		return b.WriteByte(':')
	}
}

func (b *indentBuffer) end() error {
	if b.indent >= 0 {
		b.indent--
		return b.newLine(false)
	}
	return nil
}

func (b *indentBuffer) maybeNext(first *bool) error {
	if *first {
		*first = false
		return nil
	} else {
		return b.next()
	}
}

func (b *indentBuffer) next() error {
	if b.indent >= 0 {
		return b.newLine(b.comma)
	} else if b.comma {
		return b.WriteByte(',')
	} else {
		return b.WriteByte(' ')
	}
}

func (b *indentBuffer) newLine(comma bool) error {
	if comma {
		err := b.WriteByte(',')
		if err != nil { return err }
	}

	err := b.WriteByte('\n')
	if err != nil { return err }

	for i := 0; i < b.indent; i++ {
		_, err := b.WriteString("  ")
		if err != nil { return err }
	}
	return nil
}
