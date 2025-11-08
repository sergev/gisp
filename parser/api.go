package parser

import (
	"io"

	"github.com/sergev/gisp/lang"
)

// ParseString parses Gisp source text and returns compiled Scheme forms.
func ParseString(src string) ([]lang.Value, error) {
	prog, err := Parse(src)
	if err != nil {
		return nil, err
	}
	return CompileProgram(prog)
}

// ParseReader consumes Gisp source from an io.Reader and returns compiled Scheme forms.
func ParseReader(r io.Reader) ([]lang.Value, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return ParseString(string(data))
}
