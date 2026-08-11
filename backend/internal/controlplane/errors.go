package controlplane

import (
	"errors"
	"fmt"
)

var (
	ErrInvalid   = errors.New("invalid argument")
	ErrNotFound  = errors.New("not found")
	ErrConflict  = errors.New("conflict")
	ErrDenied    = errors.New("permission denied")
	ErrInvariant = errors.New("invariant violation")
)

// OpError preserves a stable machine code while retaining operation context.
type OpError struct {
	Op    string
	Code  string
	Field string
	Err   error
}

func (e *OpError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s: %s: %v", e.Op, e.Code, e.Field, e.Err)
	}
	return fmt.Sprintf("%s: %s: %v", e.Op, e.Code, e.Err)
}

func (e *OpError) Unwrap() error { return e.Err }

func invalid(op, field, message string) error {
	return &OpError{Op: op, Code: "invalid_argument", Field: field, Err: fmt.Errorf("%w: %s", ErrInvalid, message)}
}

func notFound(op, resource string) error {
	return &OpError{Op: op, Code: "not_found", Err: fmt.Errorf("%w: %s", ErrNotFound, resource)}
}

func conflict(op, message string) error {
	return &OpError{Op: op, Code: "conflict", Err: fmt.Errorf("%w: %s", ErrConflict, message)}
}

func denied(op, message string) error {
	return &OpError{Op: op, Code: "denied", Err: fmt.Errorf("%w: %s", ErrDenied, message)}
}

func invariant(op, message string) error {
	return &OpError{Op: op, Code: "invariant", Err: fmt.Errorf("%w: %s", ErrInvariant, message)}
}
