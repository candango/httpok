package session

import (
	"context"
)

// Store is a generic key-value engine where the key is a string (such as a
// session ID, filename, etc.) and the value can be either opaque binary
// ([]byte) or text (string).
type Store interface {
	// Delete removes the specific entry (by id) from the store.
	Delete(ctx context.Context, id string) error

	// Exists returns true if the key exist.
	Exists(ctx context.Context, id string) (bool, error)

	// Get retrieves raw data as []byte for the given id (key).
	Get(ctx context.Context, id string) ([]byte, error)

	// GetString retrieves the string value for the given id.
	GetString(ctx context.Context, id string) (string, error)

	// Set saves or updates a value for the given id, updating the LastUpdate
	// time.
	Set(ctx context.Context, id string, val []byte) error

	// SetString stores a string value as bytes.
	SetString(ctx context.Context, id string, val string) error

	// Start initializes the store.
	Start(ctx context.Context) error

	// Stop tears down resources.
	Stop(ctx context.Context) error

	// Touch updates the session's ttl, typically to implement sliding
	// expiration. It does not modify the session data.
	// Returns an error if the id does not exist.
	Touch(ctx context.Context, id string) error
}
