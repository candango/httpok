package session

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FileStore implements the Engine interface for file-based session storage.
type FileStore struct {
	Dir string
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
			"create the sesssion dir", s.Dir)
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
	if !fileExists(sessFile) {
		// TODO: I need to think if we should return an error here
		return nil
	}

	err := os.Remove(sessFile)
	if err != nil {
		return err
	}
	return nil
}

// Exists checks if a session with the given ID exists in file storage.
func (s *FileStore) Exists(ctx context.Context, id string) (bool, error) {
	sessFile := filepath.Join(s.Dir, fmt.Sprintf("%s.sess", id))
	if !fileExists(sessFile) {
		return false, nil
	}
	return true, nil
}

// Get retrieves a session from file storage based on the given ID.
func (s *FileStore) Get(ctx context.Context, id string) ([]byte, error) {
	var v []byte
	sessFile := filepath.Join(s.Dir, fmt.Sprintf("%s.sess", id))
	if !fileExists(sessFile) {
		err := s.Store(id, v)
		if err != nil {
			return nil, err
		}
	}
	return s.Read(id, &v)
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
	return s.Store(id, val)
}

// SetString stores a string value as bytes.
func (s *FileStore) SetString(ctx context.Context, id string,
	val string) error {
	return s.Set(ctx, id, []byte(val))
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
func (s *FileStore) Read(id string, v any) ([]byte, error) {
	sessFile := filepath.Join(s.Dir, fmt.Sprintf("%s.sess", id))
	file, err := os.Open(sessFile)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	if err != nil {
		return nil, err
	}

	return buffer[:n], nil
}

// Store saves session data to file storage.
func (s *FileStore) Store(id string, v []byte) error {
	sessFile := filepath.Join(s.Dir, fmt.Sprintf("%s.sess", id))
	file, err := os.OpenFile(sessFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(v)
	if err != nil {
		return err
	}

	return nil
}

// Purge removes expired sessions from file storage.
func (s *FileStore) Purge(ctx context.Context, maxAge time.Duration) error {
	files, err := os.ReadDir(s.Dir)
	if err != nil {
		return err
	}

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
	// TODO: We need to implement this later
	return nil
}
