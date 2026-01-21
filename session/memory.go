package session

import (
	"context"
	"errors"
	"sync"
	"time"
)

// memoryEntry stores session value and its last updated time.
type memoryEntry struct {
	Value       []byte
	LastTouched time.Time
}

// Expired returns true if the session is older than maxAge
func (e memoryEntry) Expired(maxAge time.Duration) bool {
	return time.Since(e.LastTouched) > maxAge
}

// MemoryStore is a threadsafe in-memory key-value store, suitable for testing
// or single-instance use.
type MemoryStore struct {
	data map[string]memoryEntry
	mu   sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: map[string]memoryEntry{},
	}
}

// Start is a no-op for MemoryStore.
func (s *MemoryStore) Start(ctx context.Context) error { return nil }

// Stop is a no-op for MemoryStore.
func (s *MemoryStore) Stop(ctx context.Context) error { return nil }

// Delete removes any entry for the given id
func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, id)
	return nil
}

// Exists returns true if the id is present in the store
func (s *MemoryStore) Exists(ctx context.Context, id string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.data[id]
	return ok, nil
}

// Get retrieves the value for the given id. Returns an error if not found.
func (s *MemoryStore) Get(ctx context.Context, id string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return e.Value, nil
}

// GetString retrieves the string value for the given id.
func (s *MemoryStore) GetString(ctx context.Context, id string) (string,
	error) {
	val, err := s.Get(ctx, id)
	if err != nil {
		return "", err
	}
	return string(val), nil
}

// Set saves or updates a value for the given id, updating the LastUpdate time.
func (s *MemoryStore) Set(ctx context.Context, id string, val []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[id] = memoryEntry{
		Value:       val,
		LastTouched: time.Now(),
	}
	return nil
}

// SetString stores a string value as bytes.
func (s *MemoryStore) SetString(ctx context.Context, id string,
	val string) error {
	return s.Set(ctx, id, []byte(val))
}

// Purge deletes all entries older than maxAge
func (s *MemoryStore) Purge(ctx context.Context, maxAge time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, entry := range s.data {
		if entry.Expired(maxAge) {
			delete(s.data, id)
		}
	}
	return nil
}

// Touch updates the LastTouched timestamp for the session entry identified by
// id, effectively renewing its expiration for sliding expiration policies.
// Only the session's last access time is updated; the session value itself is
// not changed.
// Returns an error if the entry does not exist.
func (s *MemoryStore) Touch(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.data[id]
	if !ok {
		return errors.New("not found")
	}
	entry.LastTouched = time.Now()
	s.data[id] = entry
	return nil
}
