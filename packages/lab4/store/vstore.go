package store

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/andyp1xe1/network_programming_labs/packages/lab4/common"
)

var (
	ErrorVersionConflict        = errors.New("version conflict")
	ErrorGreaterVersionRequired = errors.New("greater version required")
)

// VStore wraps any Store implementation with versioning capabilities
type VStore struct {
	Store
	metadata map[string]Metadata
	mu       sync.RWMutex
}

// NewVStore creates a versioned wrapper around the provided store
func NewVStore(store Store) *VStore {
	return &VStore{
		Store:    store,
		metadata: make(map[string]Metadata),
	}
}

// Set implements Store interface with version checking from context
func (vs *VStore) Set(ctx context.Context, key, value string) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	expectedVersion, hasExpected := common.ExpectedVersionFromCtx(ctx)
	ctxVersion, hasCtxVersion := common.VersionFromCtx(ctx)
	meta, metaExists := vs.metadata[key]

	if hasExpected {
		if metaExists && meta.Version != expectedVersion {
			err := errors.Join(
				ErrorVersionConflict,
				fmt.Errorf("expected %d, got %d", expectedVersion, meta.Version),
			)
			return err
		}
	}

	if hasCtxVersion {
		if metaExists && ctxVersion <= meta.Version {
			err := errors.Join(
				ErrorGreaterVersionRequired,
				fmt.Errorf("have %d, got %d", meta.Version, ctxVersion),
			)
			return err
		}
	}

	if err := vs.Store.Set(ctx, key, value); err != nil {
		return err
	}

	now := time.Now()
	newVersion := 1
	if existing, exists := vs.metadata[key]; exists {
		if hasCtxVersion {
			newVersion = ctxVersion
		} else {
			newVersion = existing.Version + 1
		}
		vs.metadata[key] = Metadata{
			Version: newVersion,
			Created: existing.Created,
			Updated: now,
		}
	} else {
		vs.metadata[key] = Metadata{
			Version: newVersion,
			Created: now,
			Updated: now,
		}
	}

	return nil
}

// Delete implements Store interface with version checking from context
func (vs *VStore) Delete(ctx context.Context, key string) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if expectedVersion, hasExpected := common.ExpectedVersionFromCtx(ctx); hasExpected {
		if existing, exists := vs.metadata[key]; exists {
			if existing.Version != expectedVersion {
				return fmt.Errorf("version conflict: expected %d, got %d", expectedVersion, existing.Version)
			}
		} else {
			return fmt.Errorf("version conflict: key does not exist")
		}
	}

	if err := vs.Store.Delete(ctx, key); err != nil {
		return err
	}

	delete(vs.metadata, key)
	return nil
}

// GetMeta returns the metadata for a key, including version information
func (vs *VStore) GetMeta(key string) (Metadata, bool) {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	meta, exists := vs.metadata[key]
	if !exists {
		return Metadata{}, false
	}

	return meta, true
}
