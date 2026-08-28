package contract

import "errors"

var (
	ErrInvalidArgument = errors.New("invalid engineering request")
	ErrForbidden       = errors.New("engineering access denied")
	ErrNotFound        = errors.New("engineering record not found")
	ErrConflict        = errors.New("engineering record conflict")
	ErrUnavailable     = errors.New("engineering dependency unavailable")
)
