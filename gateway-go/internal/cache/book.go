// Package cache is the gateway's read-through cache for order book data.
//
// Book snapshots and top-of-book quotes are read far more often than they
// change, and recomputing them means a gRPC round trip to the engine on every
// request. Caching them in Redis with a short TTL lets the gateway answer the
// common case from memory-speed storage and only fall back to the engine on a
// miss — the "fast reads without hitting the source every time" goal of Phase 5.
//
// The cache is best-effort: every method degrades to a miss/no-op on error so a
// Redis outage slows reads down to the engine path but never fails them.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// ErrMiss is returned by a Store when the key is absent.
var ErrMiss = errors.New("cache miss")

// Store is the minimal key/value contract the cache needs. Keeping it an
// interface lets the book cache be unit-tested with an in-memory fake and keeps
// the Redis dependency at the edge.
type Store interface {
	Get(ctx context.Context, key string) (string, error) // ErrMiss when absent
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
}

type BookLevel struct {
	PriceTicks int64  `json:"priceTicks"`
	Quantity   uint64 `json:"quantity"`
}

type BookSnapshot struct {
	Symbol  string      `json:"symbol"`
	Bids    []BookLevel `json:"bids"`
	Asks    []BookLevel `json:"asks"`
	BestBid *int64      `json:"bestBid,omitempty"` // top bid price in ticks, if any
	BestAsk *int64      `json:"bestAsk,omitempty"` // top ask price in ticks, if any
}

// WithTopOfBook fills BestBid/BestAsk from the first level of each side.
func (s BookSnapshot) WithTopOfBook() BookSnapshot {
	if len(s.Bids) > 0 {
		p := s.Bids[0].PriceTicks
		s.BestBid = &p
	}
	if len(s.Asks) > 0 {
		p := s.Asks[0].PriceTicks
		s.BestAsk = &p
	}
	return s
}

type BookCache struct {
	store Store
	ttl   time.Duration
}

func NewBookCache(store Store, ttl time.Duration) *BookCache {
	return &BookCache{store: store, ttl: ttl}
}

func key(symbol string, depth uint32) string {
	// depth is part of the key: a depth-5 read must not serve a depth-2 cache.
	return "book:" + symbol + ":" + itoa(depth)
}

// Get returns a cached snapshot, or ok=false on miss or any error.
func (c *BookCache) Get(ctx context.Context, symbol string, depth uint32) (BookSnapshot, bool) {
	raw, err := c.store.Get(ctx, key(symbol, depth))
	if err != nil {
		return BookSnapshot{}, false
	}
	var snap BookSnapshot
	if err := json.Unmarshal([]byte(raw), &snap); err != nil {
		return BookSnapshot{}, false
	}
	return snap, true
}

// Set stores a snapshot under the short TTL. Errors are swallowed (best-effort).
func (c *BookCache) Set(ctx context.Context, symbol string, depth uint32, snap BookSnapshot) {
	raw, err := json.Marshal(snap)
	if err != nil {
		return
	}
	_ = c.store.Set(ctx, key(symbol, depth), string(raw), c.ttl)
}

func itoa(v uint32) string {
	if v == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
