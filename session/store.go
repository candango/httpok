package session

import (
	"context"
	"errors"
	"time"

	"github.com/candango/httpok/logger"
	"github.com/candango/httpok/security"
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

	// Purge removes expired or invalid sessions and returns an error.
	Purge(ctx context.Context, maxAge time.Duration) error

	// Set stores a value and MUST implicitly refresh its expiration/TTL.
	// How TTL is tracked is implementation-specific:
	// - MemoryStore: updates LastTouched timestamp
	// - FileStore: updates file modification time
	// - Redis: uses SET key value EX ttl
	// Implementations that do not refresh TTL when setting a value violate the
	// Store contract.
	Set(ctx context.Context, id string, val []byte) error

	// SetString stores a string value as bytes.
	SetString(ctx context.Context, id string, val string) error

	// Start initializes the store.
	Start(ctx context.Context) error

	// Stop tears down resources.
	Stop(ctx context.Context) error

	// RequiresPurge returns whether this Store implementation requires manual
	// session expiration cleanup. Stores with automatic TTL/expiration (e.g.,
	// Redis) return false. Stores tracking expiration manually (e.g.,
	// MemoryStore, FileStore) return true, signaling that StoreEngine must
	// periodically call Purge to remove expired sessions.
	//
	// Example implementations:
	//   - MemoryStore.RequiresPurge() -> true  (manual LastTouched tracking)
	//   - FileStore.RequiresPurge() -> true    (file mtime tracking)
	//   - RedisStore.RequiresPurge() -> false  (Redis TTL automatic)
	RequiresPurge() bool

	// Touch updates the session's ttl, typically to implement sliding
	// expiration. It does not modify the session data.
	// Returns an error if the id does not exist.
	Touch(ctx context.Context, id string) error
}

type storeEngineOptions func(*StoreEngine)

// StoreEngine implements the Engine interface by delegating session operations
// to a pluggable Store backend. It holds engine properties and a Store
// instance, allowing flexible session storage strategies (e.g., in-memory,
// file, etc.).
type StoreEngine struct {
	properties *EngineProperties
	Store
	logger    logger.Logger
	purgeDone chan struct{}
	started   bool
}

// NewStoreEngine creates and returns a new StoreEngine.
// If custom properties are provided, they are used; otherwise, default
// settings are applied.
func NewStoreEngine(store Store, opts ...storeEngineOptions) *StoreEngine {
	e := &StoreEngine{
		properties: &EngineProperties{
			AgeLimit:      30 * time.Minute,
			Enabled:       true,
			Encoder:       &JsonEncoder{},
			Name:          DefaultName,
			Prefix:        DefaultPrefix,
			PurgeDuration: 2 * time.Minute,
		},
		Store:  store,
		logger: &logger.StandardLogger{},
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func WithLogger(l logger.Logger) storeEngineOptions {
	return func(e *StoreEngine) {
		e.logger = l
	}
}

func WithProperties(p *EngineProperties) storeEngineOptions {
	return func(e *StoreEngine) {
		e.properties = p
	}
}

// NewId generates a new unique session ID.
func (e *StoreEngine) NewId(ctx context.Context) string {
	// TODO: use the id generator here
	return security.RandomString(60)
}

// Start initializes the engine with the given context.
func (e *StoreEngine) Start(ctx context.Context) error {
	if e.started {
		return errors.New("store engine already started")
	}
	e.started = true
	if e.RequiresPurge() {
		e.purgeDone = make(chan struct{})
		go e.periodicPurge(ctx)
	}

	return e.Store.Start(ctx)
}

// Stop releases any resources held by the engine and performs cleanup using
// the provided context.
func (e *StoreEngine) Stop(ctx context.Context) error {
	return e.Store.Stop(ctx)
}

// Properties returns engine configuration and metadata.
func (e *StoreEngine) Properties() *EngineProperties {
	return e.properties
}

func (e *StoreEngine) periodicPurge(ctx context.Context) error {
	ticker := time.NewTicker(e.properties.PurgeDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ticker.Stop()

			if err := e.Purge(ctx); err != nil {
				e.logger.Errorf("periodic purge failed: %v", err)
			}
			ticker = time.NewTicker(e.properties.PurgeDuration)
		case <-e.purgeDone:
			return nil
		case <-ctx.Done():
			return nil
		}
	}
}

// Purge removes expired or invalid sessions.
func (e *StoreEngine) Purge(ctx context.Context) error {
	if !e.properties.Enabled {
		return errors.New("engine is disabled")
	}
	return e.Store.Purge(ctx, e.properties.AgeLimit)
}

// GetSession retrieves a session by ID and context.
func (e *StoreEngine) GetSession(ctx context.Context, id string) (Session, error) {
	s := Session{}
	if !e.properties.Enabled {
		return s, errors.New("engine is disabled")
	}
	if id == "" {
		return s, errors.New("session id is empty")
	}
	var v map[string]any
	ok, err := e.Store.Exists(ctx, id)
	if err != nil {
		return s, err
	}
	if !ok {
		data, err := e.properties.Encoder.Encode(map[string]any{})
		if err != nil {
			return s, err
		}
		e.Store.Set(ctx, id, data)
	}
	data, err := e.Store.Get(ctx, id)
	if err != nil {
		return s, err
	}
	err = e.Store.Touch(ctx, id)
	if err != nil {
		return s, err
	}
	err = e.properties.Encoder.Decode(data, &v)
	if err != nil {
		return s, err
	}
	// TODO: I think we should only set Id and Data here
	return Session{
		Id:        id,
		Changed:   false,
		Ctx:       ctx, // <=== THIS GUY SHOULD GO!!!
		Data:      v,
		Destroyed: false,
	}, nil
}

// SessionExists checks if a session with the given ID exists.
func (e *StoreEngine) SessionExists(ctx context.Context, id string) (bool, error) {
	if !e.properties.Enabled {
		return false, errors.New("engine is disabled")
	}
	return e.Store.Exists(ctx, id)
}

// SaveSession persists the session data for the given ID.
func (e *StoreEngine) SaveSession(ctx context.Context, id string, session Session) error {
	if !e.properties.Enabled {
		return errors.New("engine is disabled")
	}
	if id == "" {
		return errors.New("session id is empty")
	}

	data, err := e.properties.Encoder.Encode(session.Data)
	if err != nil {
		return err
	}
	return e.Store.Set(ctx, id, data)
}
