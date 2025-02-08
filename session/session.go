package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/candango/httpok/logger"
	"github.com/candango/httpok/security"
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
type Engine interface {
	Enabled() bool
	NewId() string
	SetEnabled(bool)
	// TODO: PurgeSession(id string)
	Name() string
	SetName(string)
	Read(string, any) error
	Start(context.Context) error
	Stop() error
	Store(string, any) error
	Purge() error
	GetSession(string, context.Context) (Session, error)
	SessionNotExists(string) (bool, error)
	StoreSession(string, Session) error
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
	log.Println(test.Id)

	return test, nil
}

// EngineProperties contains common properties for session engines.
type EngineProperties struct {
	AgeLimit time.Duration
	enabled  bool
	Encoder
	logger.Logger
	name string
	// TODO: Add this to the interface
	Prefix        string
	PurgeDuration time.Duration
}

// FileEngine implements the Engine interface for file-based session storage.
type FileEngine struct {
	EngineProperties
	Dir string
}

// NewFileEngine creates and returns a new FileEngine with default settings.
func NewFileEngine() *FileEngine {
	dir := filepath.Join(os.TempDir(), "httpok", "sess")
	return &FileEngine{
		EngineProperties: EngineProperties{
			AgeLimit:      30 * time.Minute,
			enabled:       true,
			Encoder:       &JsonEncoder{},
			name:          DefaultName,
			Prefix:        DefaultPrefix,
			PurgeDuration: 2 * time.Minute,
		},
		Dir: dir,
	}
}

// Enabled returns whether the session management is enabled.
func (e *FileEngine) Enabled() bool {
	return e.enabled
}

// SetEnabled sets the enabled status of the session management.
func (e *FileEngine) SetEnabled(enabled bool) {
	e.enabled = enabled
}

// NewId generates and returns a new session ID.
func (e *FileEngine) NewId() string {
	return security.RandomString(64)
}

// GetSession retrieves a session from file storage based on the given ID.
func (e *FileEngine) GetSession(id string, ctx context.Context) (Session, error) {
	if e.Enabled() && id != "" {
		var v map[string]any
		sessFile := filepath.Join(e.Dir, fmt.Sprintf("%s.sess", id))
		if !fileExists(sessFile) {
			err := e.Store(id, map[string]any{})
			if err != nil {
				return Session{}, err
			}
		}
		err := e.Read(id, &v)
		if err != nil {
			return Session{}, err
		}
		s := Session{
			Id:        id,
			Changed:   false,
			Ctx:       ctx,
			Data:      v,
			Destroyed: false,
		}
		return s, nil
	}
	return Session{}, nil
}

// SessionNotExists checks if a session with the given ID does not exist in
// file storage.
func (e *FileEngine) SessionNotExists(id string) (bool, error) {
	sessFile := filepath.Join(e.Dir, fmt.Sprintf("%s.sess", id))
	log.Println(sessFile)
	if fileExists(sessFile) {
		return false, nil
	}
	return true, nil
}

// StoreSession saves the session data back to file storage.
func (e *FileEngine) StoreSession(id string, s Session) error {
	if e.Enabled() && id != "" {
		err := e.Store(id, s.Data)
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

// Name returns the name of the session engine.
func (e *FileEngine) Name() string {
	return e.name
}

// SetName sets the name of the session engine.
func (e *FileEngine) SetName(n string) {
	e.name = n
}

// fileExists checks if a file exists at the given path.
func fileExists(name string) bool {
	_, err := os.Stat(name)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

// Read retrieves and decodes session data from file storage.
func (e *FileEngine) Read(id string, v any) error {
	sessFile := filepath.Join(e.Dir, fmt.Sprintf("%s.sess", id))
	file, err := os.Open(sessFile)
	if err != nil {
		return err
	}

	defer file.Close()

	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	if err != nil {
		return err
	}

	err = e.Encoder.Decode(buffer[:n], v)
	if err != nil {
		return err
	}

	return nil
}

// Start initializes the directory for session storage.
func (e *FileEngine) Start(_ context.Context) error {
	fileInfo, err := os.Stat(e.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(e.Dir, 0o774)
			if err != nil {
				return errors.New(
					fmt.Sprintf("error creating session dir %s: %v", e.Dir,
						err),
				)
			}
			return nil
		}
		return errors.New(
			fmt.Sprintf("error stating session dir %s: %v", e.Dir, err),
		)
	}

	if fileInfo.Mode().IsRegular() {
		return errors.New(
			fmt.Sprintf("there is a file named as %s it is not possible to "+
				"create the sesssion dir", e.Dir),
		)
	}
	// TODO: start purge rotine

	return nil
}

// Stop is a placeholder for stopping the session engine, currently does
// nothing.
func (e *FileEngine) Stop() error {
	return nil
}

// Store saves session data to file storage.
func (e *FileEngine) Store(id string, v any) error {
	sessFile := filepath.Join(e.Dir, fmt.Sprintf("%s.sess", id))
	file, err := os.OpenFile(sessFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := e.Encoder.Encode(v)

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return nil
}

// Purge removes expired sessions from file storage.
func (e *FileEngine) Purge() error {

	files, err := os.ReadDir(e.Dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			return err
		}
		filePath := filepath.Join(e.Dir, file.Name())
		age := time.Now().Sub(info.ModTime())
		if age > e.AgeLimit {
			err := os.Remove(filePath)
			if err != nil {
				return err
			}
		}
	}

	return nil
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
	e, err := EngineFromContext(s.Ctx)
	if err != nil {
		return err
	}
	e.Store(s.Id, s.Data)
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
