package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Engine interface {
	Read(string, any) error
	Start() error
	Stop() error
	Store(string, any) error
	Purge() error
}

type FileEngine struct {
	AgeLimit time.Duration
	Dir      string
	Encoder
	PurgeDuration time.Duration
}

func NewFileEngine() *FileEngine {
	dir := filepath.Join(os.TempDir(), "httpok", "sess")
	return &FileEngine{
		AgeLimit:      30 * time.Minute,
		Dir:           dir,
		Encoder:       &JsonEncoder{},
		PurgeDuration: 2 * time.Minute,
	}
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
