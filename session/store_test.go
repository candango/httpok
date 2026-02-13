package session

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStoreEngine(t *testing.T) {
	ctx := context.Background()

	t.Run("should create new session", func(t *testing.T) {
		store := NewMemoryStore()
		engine := NewStoreEngine(store)
		engine.Start(ctx)
		defer engine.Stop(ctx)

		id := "new-session"
		session := Session{Id: id, Data: map[string]any{"key": "val"}}
		err := engine.SaveSession(ctx, id, session)

		assert.NoError(t, err)
		assert.NotNil(t, store.Data[id])
		savedData := map[string]any{}
		err = engine.properties.Encoder.Decode(store.Data[id].Value, &savedData)
		if err != nil {
			t.Fatal(err)
		}
		assert.NoError(t, err)
		assert.Equal(t, session.Data, savedData)
	})

	t.Run("should update existing session and implicitly touch", func(t *testing.T) {
		store := NewMemoryStore()
		engine := NewStoreEngine(store)
		engine.Start(ctx)
		defer engine.Stop(ctx)

		id := "update-test"
		session := Session{Id: id, Data: map[string]any{"key": "val1"}}
		engine.SaveSession(ctx, id, session)
		created := store.Data[id].LastTouched
		time.Sleep(300 * time.Microsecond)

		// Update with new data - this should touch
		session.Data["key"] = "val2"
		err := engine.SaveSession(ctx, id, session)
		savedData := map[string]any{}
		err = engine.properties.Encoder.Decode(store.Data[id].Value, &savedData)
		if err != nil {
			t.Fatal(err)
		}

		assert.NoError(t, err)
		assert.Less(t, created, store.Data[id].LastTouched)
		assert.Equal(t, "val2", savedData["key"])
	})

	t.Run("should retrieve existing session", func(t *testing.T) {
		store := NewMemoryStore()
		engine := NewStoreEngine(store)
		engine.Start(ctx)
		defer engine.Stop(ctx)

		id := "get-test"
		session := Session{Id: id, Data: map[string]any{"key": "val"}}
		engine.SaveSession(ctx, id, session)

		retrieved, err := engine.GetSession(ctx, id)

		assert.NoError(t, err)
		assert.Equal(t, session.Data, retrieved.Data)
	})

	t.Run("should return empty session if not found", func(t *testing.T) {
		store := NewMemoryStore()
		engine := NewStoreEngine(store)
		engine.Start(ctx)
		defer engine.Stop(ctx)

		session, err := engine.GetSession(ctx, "nonexistent")

		assert.NoError(t, err)
		assert.NotNil(t, session)
		assert.NotNil(t, store.Data["nonexistent"]) // proves it was created
	})

	t.Run("should implicitly touch via sliding expiration", func(t *testing.T) {
		store := NewMemoryStore()
		engine := NewStoreEngine(store)
		engine.Start(ctx)
		defer engine.Stop(ctx)

		id := "sliding-test"
		session := Session{Id: id, Data: map[string]any{"key": "val"}}
		engine.SaveSession(ctx, id, session)
		initial := store.Data[id].LastTouched
		time.Sleep(300 * time.Microsecond)

		// GetSession should touch the session
		_, err := engine.GetSession(ctx, id)
		assert.NoError(t, err)
		assert.Less(t, initial, store.Data[id].LastTouched)

		// Now sleep again (without touching)
		time.Sleep(300 * time.Microsecond)

		// Purge should NOT delete because touch refreshed TTL
		engine.Purge(ctx)

		assert.NotNil(t, store.Data[id], "session should still exist (proves GetSession touched it)")
	})
}
