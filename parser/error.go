package parser

import "errors"

// Error represents a parser error with optional metadata.
type Error struct {
	Err        error
	Incomplete bool
}

func (e *Error) Error() string {
	if e == nil || e.Err == nil {
		return ""
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

func newIncompleteError(err error) error {
	if err == nil {
		return nil
	}
	return &Error{
		Err:        err,
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
