package middleware

import "testing"

func TestRateLimiterAllowsBurstThenBlocks(t *testing.T) {
	rl := NewRateLimiter(1, 2) // 1 req/s, burst of 2
	lim := rl.limiterFor("10.0.0.1")

	if !lim.Allow() || !lim.Allow() {
		t.Fatal("the first two requests (within burst) should be allowed")
	}
	if lim.Allow() {
		t.Fatal("the third immediate request should be rate limited")
	}
}

func TestRateLimiterIsPerClient(t *testing.T) {
	rl := NewRateLimiter(1, 1)

	if !rl.limiterFor("10.0.0.1").Allow() {
		t.Fatal("first client's first request should be allowed")
	}
	// A different client has its own bucket and is unaffected.
	if !rl.limiterFor("10.0.0.2").Allow() {
		t.Fatal("second client should have an independent bucket")
	}
}

func TestSameClientReusesBucket(t *testing.T) {
	rl := NewRateLimiter(1, 1)
	a := rl.limiterFor("10.0.0.1")
	b := rl.limiterFor("10.0.0.1")
	if a != b {
		t.Fatal("the same client IP must map to the same limiter instance")
	}
}
