package session

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMemorySessionStore(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	t.Run("should set and get", func(t *testing.T) {
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
		err := store.Set(ctx, "del", []byte("gone"))
		assert.NoError(t, err)

		err = store.Delete(ctx, "gone")
		assert.NoError(t, err)

		ok, _ := store.Exists(ctx, "gone")
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("should purge expired sessions", func(t *testing.T) {
		store := NewMemoryStore()
		store.Set(ctx, "old", []byte("expired"))
		store.mu.Lock()
		store.data["old"] = memoryEntry{
			Value:       []byte("expired"),
			LastTouched: time.Now().Add(-2 * time.Hour),
		}
		store.mu.Unlock()
		store.Set(ctx, "fresh", []byte("valid"))

		err := store.Purge(1 * time.Hour)
		assert.NoError(t, err)

		ok, _ := store.Exists(ctx, "old")
		assert.False(t, ok)

		ok, _ = store.Exists(ctx, "fresh")
		assert.True(t, ok)
	})

	t.Run("should update LastTouched on Touch and error for missing session", func(t *testing.T) {
		store := NewMemoryStore()
		store.Set(ctx, "session", []byte("data"))

		store.mu.Lock()
		oldTime := time.Now().Add(-2 * time.Hour)
		entry := store.data["session"]
		entry.LastTouched = oldTime
		store.data["session"] = entry
		store.mu.Unlock()

		err := store.Touch(ctx, "session")
		assert.NoError(t, err)

		err = store.Purge(1 * time.Hour)
		assert.NoError(t, err)

		ok, _ := store.Exists(ctx, "session")
		assert.True(t, ok)

		store.mu.RLock()
		newEntry := store.data["session"]
		store.mu.RUnlock()
		assert.False(t, newEntry.Expired(1*time.Hour))
		assert.True(t, newEntry.LastTouched.After(oldTime))

		err = store.Touch(ctx, "nope")
		assert.Error(t, err)
	})

	t.Run("should handle concurrent access safely", func(t *testing.T) {
		store := NewMemoryStore()
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
