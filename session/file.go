package session

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/candango/iook/pathx"
)

// FileStore implements the Engine interface for file-based session storage.
type FileStore struct {
	Dir string
	mu  sync.RWMutex
}

// NewFileStore creates and returns a new FileStore with default settings.
func NewFileStore() *FileStore {
	dir := filepath.Join(os.TempDir(), "httpok", "sess")
	return &FileStore{
		Dir: dir,
	}
}

// Start initializes the directory for session storage.
func (s *FileStore) Start(_ context.Context) error {
	fileInfo, err := os.Stat(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(s.Dir, 0o774)
			if err != nil {
				return fmt.Errorf("error creating session dir %s: %v", s.Dir,
					err)
			}
			return nil
		}
		return fmt.Errorf("error stating session dir %s: %v", s.Dir, err)
	}

	if fileInfo.Mode().IsRegular() {
		return fmt.Errorf("there is a file named as %s it is not possible to "+
			"create the session dir", s.Dir)
	}
	return nil
}

// Stop is a placeholder for stopping the session engine, currently does
// nothing.
func (s *FileStore) Stop(ctx context.Context) error {
	return nil
}

// Delete removes any entry for the given id
func (s *FileStore) Delete(ctx context.Context, id string) error {
	sessFile := filepath.Join(s.Dir, fmt.Sprintf("%s.sess", id))
	if !pathx.Exists(sessFile) {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	err := os.Remove(sessFile)
	if err != nil {
		return err
	}
	return nil
}

// Exists checks if a session with the given ID exists in file storage.
func (s *FileStore) Exists(ctx context.Context, id string) (bool, error) {
	sessFile := filepath.Join(s.Dir, fmt.Sprintf("%s.sess", id))
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !pathx.Exists(sessFile) {
		return false, nil
	}
	return true, nil
}

// Get retrieves a session from file storage based on the given ID.
func (s *FileStore) Get(ctx context.Context, id string) ([]byte, error) {
	sessFile := filepath.Join(s.Dir, fmt.Sprintf("%s.sess", id))
	if !pathx.Exists(sessFile) {
		return nil, errors.New("session not found")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.ReadFile(sessFile)
}

// GetString retrieves the string value for the given id.
func (s *FileStore) GetString(ctx context.Context, id string) (string,
	error) {
	val, err := s.Get(ctx, id)
	if err != nil {
		return "", err
	}
	return string(val), nil
}

// Set saves or updates a value for the given id, updating the LastUpdate time.
func (s *FileStore) Set(ctx context.Context, id string, val []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessFile := filepath.Join(s.Dir, fmt.Sprintf("%s.sess", id))
	file, err := os.OpenFile(sessFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(val)
	if err != nil {
		return err
	}

	return nil
}

// SetString stores a string value as bytes.
func (s *FileStore) SetString(ctx context.Context, id string,
	val string) error {
	return s.Set(ctx, id, []byte(val))
}

// Read retrieves and decodes session data from file storage.
func (s *FileStore) Read(id string, v any) ([]byte, error) {
	sessFile := filepath.Join(s.Dir, fmt.Sprintf("%s.sess", id))
	s.mu.RLock()
	defer s.mu.RUnlock()
	return os.ReadFile(sessFile)
}

// Purge removes expired sessions from file storage.
func (s *FileStore) Purge(ctx context.Context, maxAge time.Duration) error {
	files, err := os.ReadDir(s.Dir)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			return err
		}
		filePath := filepath.Join(s.Dir, file.Name())
		age := time.Since(info.ModTime())
		if age > maxAge {
			err := os.Remove(filePath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *FileStore) RequiresPurge() bool {
	return true
}

// Touch updates the LastTouched timestamp for the session entry identified by
// id, effectively renewing its expiration for sliding expiration policies.
// Only the session's last access time is updated; the session value itself is
// not changed.
// Returns an error if the entry does not exist.
func (s *FileStore) Touch(ctx context.Context, id string) error {
	sessFile := filepath.Join(s.Dir, fmt.Sprintf("%s.sess", id))
	if !pathx.Exists(sessFile) {
		return errors.New("session not found")
	}
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.Chtimes(sessFile, now, now)
}
