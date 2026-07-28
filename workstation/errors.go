package workstation

import "errors"

var (
	ErrNotFound            = errors.New("workstation record not found")
	ErrConflict            = errors.New("workstation state conflict")
	ErrUnauthorized        = errors.New("runner is not authorized")
	ErrNoTaskAvailable     = errors.New("no compatible queued task")
	ErrLeaseExpired        = errors.New("task lease expired")
	ErrInvalidEvidence     = errors.New("invalid evidence bundle")
	ErrInvalidSignature    = errors.New("invalid evidence signature")
	ErrIdempotencyConflict = errors.New("idempotency key reused with different input")
)
