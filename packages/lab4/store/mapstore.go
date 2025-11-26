package store

import (
	"context"
	"errors"
	"sync"
)

// MapStore represents a thread-safe in-memory key-value store.
type MapStore struct {
	data map[string]string
	mu   sync.RWMutex
}

// NewMapStore creates and returns a new instance of MapStore.
func NewMapStore() *MapStore {
	return &MapStore{
		data: make(map[string]string),
	}
}

// Set adds or updates a key-value pair in the store.
func (s *MapStore) Set(ctx context.Context, key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	return nil
}

// Get retrieves the value for a given key from the store.
// It returns an error if the key does not exist.
func (s *MapStore) Get(ctx context.Context, key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, exists := s.data[key]
	if !exists {
		return "", errors.New("key not found")
	}
	return value, nil
}

// Delete removes a key-value pair from the store.
// It returns an error if the key does not exist.
func (s *MapStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.data[key]; !exists {
		return errors.New("key not found")
	}
	delete(s.data, key)
	return nil
}

// Exists checks if a key exists in the store.
func (s *MapStore) Exists(ctx context.Context, key string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.data[key]
	return exists, nil
}
