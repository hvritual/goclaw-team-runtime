// Package objectstorage adapts the installed storage providers to Space ports.
package objectstorage

import (
	"context"

	legacy "github.com/multica-ai/multica/server/internal/storage"
)

// Adapter translates the legacy storage provider into the Space ObjectStore port.
type Adapter struct {
	storage legacy.Storage
}

// New creates an object-storage adapter; a nil provider remains unavailable.
func New(storage legacy.Storage) *Adapter {
	return &Adapter{storage: storage}
}

// Available reports whether a concrete provider was configured.
func (a *Adapter) Available() bool {
	return a != nil && a.storage != nil
}

// Upload stores one complete object through the configured provider.
func (a *Adapter) Upload(ctx context.Context, key string, data []byte, contentType, filename string) (string, error) {
	return a.storage.Upload(ctx, key, data, contentType, filename)
}
