package session

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	defaultFileStorePrefix = "httpok_"
	fileStoreSuffix         = ".sess"
)

var errInvalidSessionID = errors.New("invalid session id")

// FileStore implements the Engine interface for file-based session storage.
type FileStore struct {
	Dir string
	// Prefix is the validated physical filename namespace. Empty uses the
	// default httpok_ prefix.
	Prefix string
	mu     sync.RWMutex
}

// NewFileStore creates and returns a new FileStore with default settings.
func NewFileStore() *FileStore {
	dir := filepath.Join(os.TempDir(), "httpok", "sess")
	return &FileStore{
		Dir:    dir,
		Prefix: defaultFileStorePrefix,
	}
}

// Start initializes the directory for session storage.
func (s *FileStore) Start(_ context.Context) error {
	if !validFileStorePrefix(s.filePrefix()) {
		return fmt.Errorf("invalid file store prefix %q", s.filePrefix())
	}

	fileInfo, err := os.Stat(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(s.Dir, 0o700); err != nil {
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
func (s *FileStore) Stop(_ context.Context) error {
	return nil
}

// Delete removes any entry for the given id. Deletion is idempotent.
func (s *FileStore) Delete(_ context.Context, id string) error {
	sessFile, err := s.sessionPath(id)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(sessFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// Exists checks if a session with the given ID exists in file storage.
func (s *FileStore) Exists(_ context.Context, id string) (bool, error) {
	sessFile, err := s.sessionPath(id)
	if err != nil {
		return false, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	info, err := os.Lstat(sessFile)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.Mode().IsRegular(), nil
}

// Get retrieves a session from file storage based on the given ID.
func (s *FileStore) Get(_ context.Context, id string) ([]byte, error) {
	sessFile, err := s.sessionPath(id)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	info, err := os.Lstat(sessFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("session not found")
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("session not found")
	}
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
func (s *FileStore) Set(_ context.Context, id string, val []byte) error {
	sessFile, err := s.sessionPath(id)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if info, err := os.Lstat(sessFile); err == nil && !info.Mode().IsRegular() {
		return errors.New("session path is not a regular file")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	file, err := os.OpenFile(sessFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(val)
	return err
}

// SetString stores a string value as bytes.
func (s *FileStore) SetString(ctx context.Context, id string,
	val string) error {
	return s.Set(ctx, id, []byte(val))
}

// Read retrieves and decodes session data from file storage.
func (s *FileStore) Read(id string, _ any) ([]byte, error) {
	sessFile, err := s.sessionPath(id)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	info, err := os.Lstat(sessFile)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("session not found")
	}
	return os.ReadFile(sessFile)
}

// Purge removes expired namespaced session files from file storage.
func (s *FileStore) Purge(_ context.Context, maxAge time.Duration) error {
	files, err := os.ReadDir(s.Dir)
	if err != nil {
		return err
	}

	prefix := s.filePrefix()
	if !validFileStorePrefix(prefix) {
		return fmt.Errorf("invalid file store prefix %q", prefix)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, file := range files {
		if file.IsDir() || !file.Type().IsRegular() ||
			!strings.HasPrefix(file.Name(), prefix) ||
			!strings.HasSuffix(file.Name(), fileStoreSuffix) {
			continue
		}
		info, err := file.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || time.Since(info.ModTime()) <= maxAge {
			continue
		}
		filePath := filepath.Join(s.Dir, file.Name())
		if err := os.Remove(filePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	return nil
}

// RequiresPurge reports that FileStore uses manual expiration cleanup.
func (s *FileStore) RequiresPurge() bool {
	return true
}

// Touch updates the session's ttl, typically to implement sliding
// expiration. It does not modify the session data.
// Returns an error if the id does not exist.
func (s *FileStore) Touch(_ context.Context, id string) error {
	sessFile, err := s.sessionPath(id)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	info, err := os.Lstat(sessFile)
	if errors.Is(err, os.ErrNotExist) {
		return errors.New("session not found")
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("session not found")
	}
	now := time.Now()
	return os.Chtimes(sessFile, now, now)
}

func (s *FileStore) sessionPath(id string) (string, error) {
	if !validSessionID(id) {
		return "", errInvalidSessionID
	}
	prefix := s.filePrefix()
	if !validFileStorePrefix(prefix) {
		return "", fmt.Errorf("invalid file store prefix %q", prefix)
	}
	return filepath.Join(s.Dir, prefix+id+fileStoreSuffix), nil
}

func (s *FileStore) filePrefix() string {
	if s.Prefix == "" {
		return defaultFileStorePrefix
	}
	return s.Prefix
}

func validFileStorePrefix(prefix string) bool {
	if prefix == "" || len(prefix) > 64 {
		return false
	}
	for _, char := range prefix {
		if (char < 'a' || char > 'z') &&
			(char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') && char != '-' && char != '_' {
			return false
		}
	}
	return true
}

func validSessionID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, char := range id {
		if (char < 'a' || char > 'z') &&
			(char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') && char != '-' && char != '_' {
			return false
		}
	}
	return true
}
