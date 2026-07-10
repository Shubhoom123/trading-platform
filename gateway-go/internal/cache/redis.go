package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore adapts go-redis to the Store interface.
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore builds a store from a redis:// URL (or host:port). Returns an
// error only if the URL is unparseable; connectivity is checked lazily.
func NewRedisStore(addr string) (*RedisStore, error) {
	opts, err := redis.ParseURL(normalize(addr))
	if err != nil {
		return nil, err
	}
	return &RedisStore{client: redis.NewClient(opts)}, nil
}

func (r *RedisStore) Get(ctx context.Context, key string) (string, error) {
	v, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", ErrMiss
	}
	return v, err
}

func (r *RedisStore) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisStore) Close() error { return r.client.Close() }

// normalize accepts either a full redis:// URL or a bare host:port.
func normalize(addr string) string {
	if len(addr) >= 8 && addr[:8] == "redis://" {
		return addr
	}
	if len(addr) >= 9 && addr[:9] == "rediss://" {
		return addr
	}
	return "redis://" + addr
}

// NoopStore is used when Redis is not configured: every read misses and every
// write is discarded, so the gateway transparently falls back to the engine.
type NoopStore struct{}

func (NoopStore) Get(context.Context, string) (string, error) { return "", ErrMiss }

func (NoopStore) Set(context.Context, string, string, time.Duration) error { return nil }
