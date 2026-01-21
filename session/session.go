package session

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/candango/httpok/logger"
)

const (
	// ContextEngValue is the context key for storing the session Engine.
	ContextEngValue = "HTTPOKSESSCTXENGVALUE"
	// ContextSessValue is the context key for storing the session data.
	ContextSessValue = "HTTPOKSESSCTXSESSVALUE"
	// DefaultName is the default name for session cookies.
	DefaultName = "HTTPOKSESSID"
	// DefaultPrefix is the default prefix used for session keys.
	DefaultPrefix = "httpok:session"
)

// Engine defines the interface for session management.
// It abstracts session lifecycle operations and storage, allowing different backends.
// Implementations should handle session creation, retrieval, persistence, and cleanup.
type Engine interface {
	// NewId generates a new unique session ID.
	NewId(ctx context.Context) string

	// Start initializes the engine with the given context.
	Start(context.Context) error

	// Stop releases any resources held by the engine and performs cleanup
	// using the provided context.
	Stop(context.Context) error

	// Properties returns engine configuration and metadata.
	Properties() *EngineProperties

	// Purge removes expired or invalid sessions.
	Purge(ctx context.Context) error

	// GetSession retrieves a session by ID and context.
	GetSession(ctx context.Context, id string) (Session, error)

	// SessionExists checks if a session with the given ID exists.
	SessionExists(ctx context.Context, id string) (bool, error)

	// SaveSession persists the session data for the given ID.
	SaveSession(ctx context.Context, id string, s Session) error
}

// IdGenerator defines an interface for generating unique session IDs.
// Implementations can provide different algorithms (e.g., UUID, random strings).
type IdGenerator interface {

	// NewId returns a new unique session ID as a string.
}

// EngineFromContext retrieves the session Engine from the context.
func EngineFromContext(ctx context.Context) (Engine, error) {
	s := ctx.Value(ContextEngValue)
	if s == nil {
		return nil, errors.New("engine value not found into the conext")
	}
	return s.(Engine), nil
}

// SessionFromContext retrieves the session from the context.
func SessionFromContext(ctx context.Context) (*Session, error) {
	s := ctx.Value(ContextSessValue)
	if s == nil {
		return nil, errors.New("session value not found into the conext")
	}
	test := s.(*Session)
	log.Printf("\n\nThe session id we should get: %s\n\n", test.Id)

	return test, nil
}

// EngineProperties contains common properties for session engines.
type EngineProperties struct {
	AgeLimit time.Duration
	Enabled  bool
	Encoder
	logger.Logger
	Name string
	// TODO: Add this to the interface
	Prefix        string
	PurgeDuration time.Duration
}

// Encoder is an interface for encoding and decoding session data.
type Encoder interface {
	Encode(any) ([]byte, error)
	Decode([]byte, any) error
}

// JsonEncoder implements the Encoder interface using JSON serialization.
type JsonEncoder struct {
}

// Encode serializes the given value into JSON.
func (e *JsonEncoder) Encode(v any) ([]byte, error) {
	return json.Marshal(v)
}

// Decode deserializes the JSON data into the provided value.
func (e *JsonEncoder) Decode(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// sessionDestroyedError is an error returned when trying to use a destroyed
// session.
var sessionDestroyedError error = errors.New("the session is already " +
	"destroyed, renew the session before using it")

// Session represents a user session with data storage capabilities.
type Session struct {
	Id        string
	Changed   bool
	Ctx       context.Context
	Data      map[string]any
	Destroyed bool
	Params    any
}

// Clear removes all data from the session and marks it as changed.
func (s *Session) Clear() {
	s.Data = map[string]any{}
	s.Changed = true
}

// Delete removes a key-value pair from the session data.
func (s *Session) Delete(key string) error {
	if s.Destroyed {
		return sessionDestroyedError
	}
	_, ok := s.Data[key]
	if ok {
		delete(s.Data, key)
	}
	s.Changed = true
	return nil
}

// Destroy clears the session data and marks it as destroyed.
func (s *Session) Destroy() error {
	s.Clear()
	s.Destroyed = true
	// e, err := EngineFromContext(s.Ctx)
	// if err != nil {
	// 	return err
	// }
	// e.Store(s.Id, s.Data)
	return nil

}

// Get retrieves a value from the session by key.
func (s *Session) Get(key string) (any, error) {
	if s.Destroyed {
		return nil, sessionDestroyedError
	}
	data, ok := s.Data[key]
	if !ok {
		return nil, nil
	}
	return data, nil
}

// Has checks if a key exists in the session data.
func (s *Session) Has(key string) (bool, error) {
	if s.Destroyed {
		return false, sessionDestroyedError
	}
	_, ok := s.Data[key]
	return ok, nil
}

// Set adds or updates a key-value pair in the session data.
func (s *Session) Set(key string, value any) error {
	if s.Destroyed {
		return sessionDestroyedError
	}
	s.Data[key] = value
	s.Changed = true
	return nil
}
