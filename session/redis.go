package session

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/candango/httpok/security"
	"github.com/redis/go-redis/v9"
)

// RedisEngine is the implementation of session management using Redis.
type RedisEngine struct {
	EngineProperties
	ctx context.Context
	rdb *redis.Client
}

// NewRedisEngine creates and returns a new RedisEngine with default settings.
func NewRedisEngine() (*RedisEngine, error) {

	return &RedisEngine{
		EngineProperties: EngineProperties{
			AgeLimit:      30 * time.Minute,
			enabled:       true,
			Encoder:       &JsonEncoder{},
			name:          DefaultName,
			Prefix:        DefaultPrefix,
			PurgeDuration: 2 * time.Minute,
		},
	}, nil
}

// Enabled returns whether the session management is enabled.
func (e *RedisEngine) Enabled() bool {
	return e.enabled
}

// SetEnabled sets the enabled status of the session management.
func (e *RedisEngine) SetEnabled(enabled bool) {
	e.enabled = enabled
}

// NewId generates and returns a new session ID.
func (e *RedisEngine) NewId() string {
	return security.RandomString(64)
}

// GetSession retrieves a session from Redis based on the given ID.
func (e *RedisEngine) GetSession(id string, ctx context.Context) (Session, error) {
	if e.Enabled() && id != "" {
		var v map[string]any
		key := fmt.Sprintf("%s:%s", e.Prefix, id)
		keys, err := e.rdb.Keys(e.ctx, key).Result()
		if err != nil {
			return Session{}, err
		}

		if len(keys) == 0 {
			err := e.Store(id, map[string]any{})
			if err != nil {
				return Session{}, err
			}
		}

		err = e.Read(id, &v)
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

// SessionNotExists checks if a session with the given ID does not exist in
// Redis.
func (e *RedisEngine) SessionNotExists(id string) (bool, error) {
	key := fmt.Sprintf("%s:%s", e.Prefix, id)
	log.Println(key)
	keys, err := e.rdb.Keys(e.ctx, key).Result()
	if err != nil {
		return true, err
	}

	if len(keys) > 0 {
		return false, nil
	}
	return true, nil
}

// StoreSession saves the session data back to Redis.
func (e *RedisEngine) StoreSession(id string, s Session) error {
	if e.Enabled() && id != "" {
		err := e.Store(id, s.Data)
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

// Name returns the name of the session engine.
func (e *RedisEngine) Name() string {
	return e.name
}

// SetName sets the name of the session engine.
func (e *RedisEngine) SetName(n string) {
	e.name = n
}

// Read retrieves and decodes session data from Redis.
func (e *RedisEngine) Read(id string, v any) error {
	key := fmt.Sprintf("%s:%s", e.Prefix, id)
	val, err := e.rdb.Get(e.ctx, key).Result()
	if err != nil {
		return err
	}

	err = e.rdb.Expire(e.ctx, key, e.AgeLimit).Err()
	if err != nil {
		return err
	}

	err = e.Encoder.Decode([]byte(val), v)
	if err != nil {
		return err
	}

	return nil
}

// Start initializes the connection to Redis.
func (e *RedisEngine) Start(ctx context.Context) error {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   8,
	})
	e.ctx = ctx

	// TODO: think about ways to backoff from redis connection errors
	_, err := rdb.Ping(e.ctx).Result()
	if err != nil {
		return err
	}

	e.rdb = rdb
	return nil
}

// Stop is a placeholder for stopping the session engine, currently does
// nothing.
func (e *RedisEngine) Stop() error {
	return nil
}

// Store saves session data to Redis.
func (e *RedisEngine) Store(id string, v any) error {
	sessKey := fmt.Sprintf("%s:%s", e.Prefix, id)
	data, err := e.Encoder.Encode(v)
	if err != nil {
		return err
	}
	err = e.rdb.Set(e.ctx, sessKey, string(data), e.AgeLimit).Err()
	if err != nil {
		return err
	}
	return nil
}

// Purge removes expired sessions from Redis.
func (e *RedisEngine) Purge() error {
	keys, err := e.rdb.Keys(e.ctx, fmt.Sprintf("%s:*", e.Prefix)).Result()
	if err != nil {
		return err
	}

	for _, k := range keys {
		key := fmt.Sprintf("%s:%s", e.Prefix, k)
		ttl, err := e.rdb.TTL(e.ctx, key).Result()
		if err != nil {
			return err
		}
		if ttl == -2 || ttl == -1 || ttl == 0 {
			err := e.rdb.Del(e.ctx, key).Err()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
