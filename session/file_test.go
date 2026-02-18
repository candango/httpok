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

		ok, _ := store.Exists(ctx, "del")
		assert.False(t, ok)
	})

	t.Run("should purge expired sessions", func(t *testing.T) {
		store := NewFileStore()
		defer os.RemoveAll(store.Dir)
		assert.NoError(t, store.Start(ctx))

		store.Set(ctx, "old", []byte("expired"))
		store.Set(ctx, "fresh", []byte("valid"))

		// Make "old" file appear old by setting its mtime to 2 hours ago
		oldFile := filepath.Join(store.Dir, "old.sess")
		oldTime := time.Now().Add(-2 * time.Hour)
		os.Chtimes(oldFile, oldTime, oldTime)

		err := store.Purge(ctx, 1*time.Hour)
		assert.NoError(t, err)

		ok, _ := store.Exists(ctx, "old")
		assert.False(t, ok)

		ok, _ = store.Exists(ctx, "fresh")
		assert.True(t, ok)
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
		sessFile := filepath.Join(store.Dir, "session.sess")
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
