// Package id provides Asset identity adapters.
package id

import "github.com/google/uuid"

// UUIDv7 generates time-ordered UUIDv7 Asset identities.
type UUIDv7 struct{}

// NewID returns a new UUIDv7 string.
func (UUIDv7) NewID() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}
