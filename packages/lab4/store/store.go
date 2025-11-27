// Package store provides a simple in-memory key-value store with basic operations.
package store

import (
	"context"
	"time"
)

type Store interface {
	// Set adds or updates a key-value pair in the store.
	Set(ctx context.Context, key, value string) error

	// Get retrieves the value for a given key from the store.
	// It returns an error if the key does not exist.
	Get(ctx context.Context, key string) (string, error)

	// Delete removes a key-value pair from the store.
	// It returns an error if the key does not exist.
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists in the store.
	Exists(ctx context.Context, key string) (bool, error)
}

// Metadata represents basic version metadata for a key
type Metadata struct {
	Version int       `json:"version"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}

// MetadataStore is an optional interface that stores can implement
// to provide version metadata for keys
type MetadataStore interface {
	Store
	// GetMeta returns the metadata for a key, including version information
	GetMeta(key string) (Metadata, bool)
}
