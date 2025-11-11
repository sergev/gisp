package parser

import (
	"errors"
	"fmt"
)

// Error represents a parser error with optional metadata.
type Error struct {
	Err        error
	Pos        Position
	Incomplete bool
}

func (e *Error) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	if e.Pos.Line > 0 {
		col := e.Pos.Column
		if col <= 0 {
			col = 1
		}
		return fmt.Sprintf("line %d:%d: %s", e.Pos.Line, col, e.Err.Error())
	}
	return e.Err.Error()
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func newError(err error) error {
	if err == nil {
		return nil
	}
	return &Error{Err: err}
}

func newErrorAt(pos Position, err error) error {
	if err == nil {
		return nil
	}
	var perr *Error
	if errors.As(err, &perr) {
		if perr.Pos.Line == 0 && pos.Line > 0 {
			perr.Pos = pos
		}
		return err
	}
	return &Error{
		Err: err,
		Pos: pos,
	}
}

func newIncompleteError(err error) error {
	if err == nil {
		return nil
	}
	return &Error{
		Err:        err,
		Incomplete: true,
	}
}

func newIncompleteErrorAt(pos Position, err error) error {
	if err == nil {
		return nil
	}
	var perr *Error
	if errors.As(err, &perr) {
		if !perr.Incomplete {
			perr.Incomplete = true
		}
		if perr.Pos.Line == 0 && pos.Line > 0 {
			perr.Pos = pos
		}
		return err
	}
	return &Error{
		Err:        err,
		Pos:        pos,
		Incomplete: true,
	}
}

func wrapError(err error) error {
	if err == nil {
		return nil
	}
	var perr *Error
	if errors.As(err, &perr) {
		return err
	}
	return newError(err)
}

// IsIncomplete reports whether the supplied error represents incomplete input.
func IsIncomplete(err error) bool {
	var perr *Error
	if errors.As(err, &perr) {
		return perr.Incomplete
	}
	return false
}
