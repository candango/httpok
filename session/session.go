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
	ContextEngValue  = "HTTPOKSESSCTXENGVALUE"
	ContextSessValue = "HTTPOKSESSCTXSESSVALUE"
	DefaultName      = "HTTPOKSESSID"
	DefaultPrefix    = "httpok:session"
)

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

func EngineFromContext(ctx context.Context) (Engine, error) {
	s := ctx.Value(ContextEngValue)
	if s == nil {
		return nil, errors.New("engine value not found into the conext")
	}
	return s.(Engine), nil
}

func SessionFromContext(ctx context.Context) (*Session, error) {
	s := ctx.Value(ContextSessValue)
	if s == nil {
		return nil, errors.New("session value not found into the conext")
	}
	test := s.(*Session)
	log.Println(test.Id)

	return test, nil
}

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

type FileEngine struct {
	EngineProperties
	Dir string
}

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

func (e *FileEngine) Enabled() bool {
	return e.enabled
}

func (e *FileEngine) SetEnabled(enabled bool) {
	e.enabled = enabled
}

func (e *FileEngine) NewId() string {
	return security.RandomString(64)
}

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

func (e *FileEngine) SessionNotExists(id string) (bool, error) {
	sessFile := filepath.Join(e.Dir, fmt.Sprintf("%s.sess", id))
	log.Println(sessFile)
	if fileExists(sessFile) {
		return false, nil
	}
	return true, nil
}

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

func (e *FileEngine) Name() string {
	return e.name
}

func (e *FileEngine) SetName(n string) {
	e.name = n
}

func fileExists(name string) bool {
	_, err := os.Stat(name)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

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

func (e *FileEngine) Stop() error {
	return nil
}

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

type Encoder interface {
	Encode(any) ([]byte, error)
	Decode([]byte, any) error
}

type JsonEncoder struct {
}

func (e *JsonEncoder) Encode(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (e *JsonEncoder) Decode(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

var sessionDestroyedError error = errors.New("the session is already " +
	"destroyed, renew the session before using it")

type Session struct {
	Id        string
	Changed   bool
	Ctx       context.Context
	Data      map[string]any
	Destroyed bool
	Params    any
}

func (s *Session) Clear() {
	s.Data = map[string]any{}
	s.Changed = true
}

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

func (s *Session) Has(key string) (bool, error) {
	if s.Destroyed {
		return false, sessionDestroyedError
	}
	_, ok := s.Data[key]
	return ok, nil
}

func (s *Session) Set(key string, value any) error {
	if s.Destroyed {
		return sessionDestroyedError
	}
	s.Data[key] = value
	s.Changed = true
	return nil
}
