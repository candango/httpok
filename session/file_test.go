package session

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFileSessionStore(t *testing.T) {
	ctx := context.Background()

	t.Run("should set and get", func(t *testing.T) {
		store := NewFileStore()
		defer os.RemoveAll(store.Dir)
		assert.NoError(t, store.Start(ctx))

		err := store.Set(ctx, "foo", []byte("bar"))
		assert.NoError(t, err)

		ok, err := store.Exists(ctx, "foo")
		assert.NoError(t, err)
		assert.True(t, ok)

		val, err := store.Get(ctx, "foo")
		assert.NoError(t, err)
		assert.Equal(t, []byte("bar"), val)
	})

	t.Run("should set and get string", func(t *testing.T) {
		store := NewFileStore()
		defer os.RemoveAll(store.Dir)
		assert.NoError(t, store.Start(ctx))

		err := store.SetString(ctx, "a", "b")
		assert.NoError(t, err)

		ok, err := store.Exists(ctx, "a")
		assert.NoError(t, err)
		assert.True(t, ok)

		val, err := store.GetString(ctx, "a")
		assert.NoError(t, err)
		assert.Equal(t, "b", val)
	})

	t.Run("should return true if exists", func(t *testing.T) {
		store := NewFileStore()
		defer os.RemoveAll(store.Dir)
		assert.NoError(t, store.Start(ctx))

		err := store.SetString(ctx, "ping", "pong")
		assert.NoError(t, err)

		ok, err := store.Exists(ctx, "ping")
		assert.NoError(t, err)
		assert.True(t, ok)

		ok, err = store.Exists(ctx, "pah")
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("should delete session", func(t *testing.T) {
		store := NewFileStore()
		defer os.RemoveAll(store.Dir)
		assert.NoError(t, store.Start(ctx))

		err := store.Set(ctx, "del", []byte("gone"))
		assert.NoError(t, err)

		err = store.Delete(ctx, "del")
		assert.NoError(t, err)
		assert.NoError(t, store.Delete(ctx, "del"))

		ok, _ := store.Exists(ctx, "del")
		assert.False(t, ok)
	})

	t.Run("should purge expired sessions", func(t *testing.T) {
		store := NewFileStore()
		defer os.RemoveAll(store.Dir)
		assert.NoError(t, store.Start(ctx))

		store.Set(ctx, "old", []byte("expired"))
		store.Set(ctx, "fresh", []byte("valid"))
		unrelatedFile := filepath.Join(store.Dir, "unrelated.txt")
		assert.NoError(t, os.WriteFile(unrelatedFile, []byte("keep"), 0600))
		legacyFile := filepath.Join(store.Dir, "legacy.sess")
		assert.NoError(t, os.WriteFile(legacyFile, []byte("keep"), 0600))

		// Make "old" file appear old by setting its mtime to 2 hours ago
		oldFile := filepath.Join(store.Dir, defaultFileStorePrefix+"old"+fileStoreSuffix)
		oldTime := time.Now().Add(-2 * time.Hour)
		os.Chtimes(oldFile, oldTime, oldTime)

		err := store.Purge(ctx, 1*time.Hour)
		assert.NoError(t, err)

		ok, _ := store.Exists(ctx, "old")
		assert.False(t, ok)

		ok, _ = store.Exists(ctx, "fresh")
		assert.True(t, ok)
		assert.FileExists(t, unrelatedFile)
		assert.FileExists(t, legacyFile)
	})

	t.Run("should fail to get non-existent session", func(t *testing.T) {
		store := NewFileStore()
		defer os.RemoveAll(store.Dir)
		assert.NoError(t, store.Start(ctx))

		val, err := store.Get(ctx, "nope")
		assert.Error(t, err)
		assert.Nil(t, val)
	})

	t.Run("should update mtime on Touch and error for missing session", func(t *testing.T) {
		store := NewFileStore()
		defer os.RemoveAll(store.Dir)
		assert.NoError(t, store.Start(ctx))

		store.Set(ctx, "session", []byte("data"))

		// Make file appear old
		sessFile := filepath.Join(store.Dir, defaultFileStorePrefix+"session"+fileStoreSuffix)
		oldTime := time.Now().Add(-2 * time.Hour)
		os.Chtimes(sessFile, oldTime, oldTime)

		// Touch should update mtime
		err := store.Touch(ctx, "session")
		assert.NoError(t, err)

		// Purge should not delete it since we touched it
		err = store.Purge(ctx, 1*time.Hour)
		assert.NoError(t, err)

		ok, _ := store.Exists(ctx, "session")
		assert.True(t, ok)

		// Touch non-existent should error
		err = store.Touch(ctx, "nope")
		assert.Error(t, err)
	})

	t.Run("should namespace files and reject unsafe IDs", func(t *testing.T) {
		store := NewFileStore()
		store.Prefix = "tenant_"
		defer os.RemoveAll(store.Dir)
		assert.NoError(t, store.Start(ctx))

		assert.NoError(t, store.Set(ctx, "safe-id", []byte("data")))
		assert.FileExists(t, filepath.Join(store.Dir, "tenant_safe-id.sess"))
		oldFile := filepath.Join(store.Dir, "tenant_old.sess")
		assert.NoError(t, store.Set(ctx, "old", []byte("expired")))
		oldTime := time.Now().Add(-2 * time.Hour)
		assert.NoError(t, os.Chtimes(oldFile, oldTime, oldTime))
		assert.NoError(t, store.Purge(ctx, time.Hour))
		assert.NoFileExists(t, oldFile)
		assert.NoFileExists(t, filepath.Join(store.Dir, "httpok_safe-id.sess"))
		assert.NoFileExists(t, filepath.Join(store.Dir, "safe-id.sess"))

		assert.NoError(t, os.Mkdir(filepath.Join(store.Dir, "httpok_directory.sess"), 0700))
		exists, err := store.Exists(ctx, "directory")
		assert.NoError(t, err)
		assert.False(t, exists)

		for _, id := range []string{
			"../escape",
			"..",
			".",
			"nested/id",
			"absolute/path",
			"contains.dot",
			`contains\\slash`,
		} {
			t.Run(id, func(t *testing.T) {
				assert.Error(t, store.Set(ctx, id, []byte("blocked")))
				assert.Error(t, store.Delete(ctx, id))
				assert.Error(t, store.Touch(ctx, id))
				_, getErr := store.Get(ctx, id)
				assert.Error(t, getErr)
				_, existsErr := store.Exists(ctx, id)
				assert.Error(t, existsErr)
			})
		}

		outside := filepath.Join(store.Dir, "..", "escape.sess")
		assert.NoFileExists(t, outside)
	})

	t.Run("should reject unsafe file prefixes", func(t *testing.T) {
		store := NewFileStore()
		store.Prefix = "../escape"
		defer os.RemoveAll(store.Dir)
		assert.Error(t, store.Start(ctx))
	})

	t.Run("should handle concurrent access safely", func(t *testing.T) {
		store := NewFileStore()
		defer os.RemoveAll(store.Dir)
		assert.NoError(t, store.Start(ctx))

		keys := []string{"a", "b", "c", "d", "e"}

		done := make(chan struct{})
		for _, k := range keys {
			go func(k string) {
				for range 100 {
					_ = store.Set(ctx, k, []byte("val"))
					_ = store.Touch(ctx, k)
					_, _ = store.Get(ctx, k)
					_, _ = store.Exists(ctx, k)
				}
				done <- struct{}{}
			}(k)
		}

		for range keys {
			<-done
		}

		for _, k := range keys {
			ok, err := store.Exists(ctx, k)
			assert.NoError(t, err)
			assert.True(t, ok)
		}
	})
}
