package cache

import (
	"context"
	"sync"
	"testing"
	"time"
)

// fakeStore is an in-memory Store for testing the cache without Redis.
type fakeStore struct {
	mu   sync.Mutex
	data map[string]string
	sets int
}

func newFakeStore() *fakeStore { return &fakeStore{data: map[string]string{}} }

func (f *fakeStore) Get(_ context.Context, key string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.data[key]
	if !ok {
		return "", ErrMiss
	}
	return v, nil
}

func (f *fakeStore) Set(_ context.Context, key, value string, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data[key] = value
	f.sets++
	return nil
}

func TestSetThenGetRoundTrips(t *testing.T) {
	store := newFakeStore()
	c := NewBookCache(store, time.Second)
	ctx := context.Background()

	snap := BookSnapshot{
		Symbol: "AAPL",
		Bids:   []BookLevel{{PriceTicks: 15000, Quantity: 3}},
		Asks:   []BookLevel{{PriceTicks: 15100, Quantity: 5}},
	}.WithTopOfBook()

	c.Set(ctx, "AAPL", 0, snap)

	got, ok := c.Get(ctx, "AAPL", 0)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.Symbol != "AAPL" || got.BestBid == nil || *got.BestBid != 15000 ||
		got.BestAsk == nil || *got.BestAsk != 15100 {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}

func TestMissOnAbsentKey(t *testing.T) {
	c := NewBookCache(newFakeStore(), time.Second)
	if _, ok := c.Get(context.Background(), "NOPE", 0); ok {
		t.Fatal("expected miss for absent key")
	}
}

func TestDepthIsPartOfKey(t *testing.T) {
	store := newFakeStore()
	c := NewBookCache(store, time.Second)
	ctx := context.Background()

	c.Set(ctx, "AAPL", 5, BookSnapshot{Symbol: "AAPL"})

	// A different depth must not be served from the depth-5 entry.
	if _, ok := c.Get(ctx, "AAPL", 2); ok {
		t.Fatal("depth-2 read should miss a depth-5 cache entry")
	}
	if _, ok := c.Get(ctx, "AAPL", 5); !ok {
		t.Fatal("depth-5 read should hit")
	}
}

func TestNoopStoreAlwaysMisses(t *testing.T) {
	c := NewBookCache(NoopStore{}, time.Second)
	ctx := context.Background()
	c.Set(ctx, "AAPL", 0, BookSnapshot{Symbol: "AAPL"})
	if _, ok := c.Get(ctx, "AAPL", 0); ok {
		t.Fatal("noop store must never hit")
	}
}
