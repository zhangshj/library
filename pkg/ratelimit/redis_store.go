package ratelimit

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore is the production Store backed by Redis. It uses a Lua script so
// that INCR + (conditional) EXPIRE runs atomically on the server, preventing
// the lost-update race that a naive GET-then-SET would have under concurrency.
type RedisStore struct {
	cli redis.UniversalClient
}

// NewRedisStore wraps a go-redis client (or any UniversalClient, including the
// one returned by gofiber's storage.Conn()) as a Store.
func NewRedisStore(cli redis.UniversalClient) *RedisStore {
	return &RedisStore{cli: cli}
}

// incrScript: atomic increment with TTL set only on first creation.
// KEYS[1]=key  ARGV[1]=ttlSeconds
var incrScript = redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
if n == 1 then
  redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return n
`)

// Incr atomically increments key, applying ttl only when the key is new.
func (s *RedisStore) Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	n, err := incrScript.Run(ctx, s.cli, []string{key}, int64(ttl.Seconds())).Int64()
	if err != nil {
		return 0, err
	}
	return n, nil
}

// Get returns the current counter (0 if missing).
func (s *RedisStore) Get(ctx context.Context, key string) (int64, error) {
	n, err := s.cli.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return n, nil
}

// Del removes key.
func (s *RedisStore) Del(ctx context.Context, key string) error {
	return s.cli.Del(ctx, key).Err()
}
