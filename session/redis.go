package session

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/candango/httpok/util"
	"github.com/redis/go-redis/v9"
)

type RedisEngine struct {
	EngineProperties
	ctx context.Context
	rdb *redis.Client
}

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

func (e *RedisEngine) Enabled() bool {
	return e.enabled
}

func (e *RedisEngine) SetEnabled(enabled bool) {
	e.enabled = enabled
}

func (e *RedisEngine) NewId() string {
	return util.RandomString(64)
}

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

func (e *RedisEngine) SessionNotExists(id string) (bool, error) {
	key := fmt.Sprintf("%s:%s", e.Prefix, id)
	log.Println(key)
	keys, err := e.rdb.Keys(e.ctx, key).Result()
	// FIXME: SessionNotExists MUST return an error also
	if err != nil {
		return true, err //  <---- Bhruuuu
	}

	if len(keys) > 0 {
		return false, nil
	}
	return true, nil
}

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

func (e *RedisEngine) Name() string {
	return e.name
}

func (e *RedisEngine) SetName(n string) {
	e.name = n
}

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

func (e *RedisEngine) Stop() error {
	return nil
}

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
