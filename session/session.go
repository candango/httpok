package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/candango/httpok/logger"
)

type Engine interface {
	Enabled() bool
	SetEnabled(enabled bool)
	GetSession(id string) *Session
	// TODO: PurgeSession(id string)
	Read(string, any) error
	Start() error
	Stop() error
	Store(string, any) error
	Purge() error
}

type FileEngine struct {
	AgeLimit time.Duration
	Dir      string
	enabled  bool
	Encoder
	logger.Logger
	PurgeDuration time.Duration
}

func NewFileEngine() *FileEngine {
	dir := filepath.Join(os.TempDir(), "httpok", "sess")
	return &FileEngine{
		AgeLimit:      30 * time.Minute,
		Dir:           dir,
		enabled:       true,
		Encoder:       &JsonEncoder{},
		PurgeDuration: 2 * time.Minute,
	}
}

func (e *FileEngine) Enabled() bool {
	return e.enabled
}

func (e *FileEngine) SetEnabled(enabled bool) {
	e.enabled = enabled
}

func (e *FileEngine) GetSession(id string) *Session {
	if e.Enabled() {
		return nil
	}
	return nil
}

func (e *FileEngine) Read(sess string, v any) error {
	sessFile := filepath.Join(e.Dir, fmt.Sprintf("%s.sess", sess))
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

func (e *FileEngine) Start() error {
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

func (e *FileEngine) Store(sess string, v any) error {
	sessFile := filepath.Join(e.Dir, fmt.Sprintf("%s.sess", sess))
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

func (s *Session) Destroy() {
	s.Clear()
	s.Destroyed = true
	// TODO: after destroyed we need to store the session
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
