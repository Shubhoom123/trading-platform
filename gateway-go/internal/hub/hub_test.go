package hub

import (
	"context"
	"testing"
	"time"
)

// recvWithin reads one message or fails the test after a short timeout.
func recvWithin(t *testing.T, c *Client, d time.Duration) []byte {
	t.Helper()
	select {
	case msg, ok := <-c.Out():
		if !ok {
			t.Fatal("channel closed unexpectedly")
		}
		return msg
	case <-time.After(d):
		t.Fatal("timed out waiting for message")
		return nil
	}
}

func startHub(t *testing.T) *Hub {
	t.Helper()
	h := New()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go h.Run(ctx)
	return h
}

func TestBroadcastFansOutToAllSubscribers(t *testing.T) {
	h := startHub(t)

	a := h.Subscribe("AAPL")
	b := h.Subscribe("AAPL")

	h.Broadcast("AAPL", []byte("fill-1"))

	if got := string(recvWithin(t, a, time.Second)); got != "fill-1" {
		t.Fatalf("subscriber a got %q", got)
	}
	if got := string(recvWithin(t, b, time.Second)); got != "fill-1" {
		t.Fatalf("subscriber b got %q", got)
	}
}

func TestBroadcastIsSymbolScoped(t *testing.T) {
	h := startHub(t)

	aapl := h.Subscribe("AAPL")
	msft := h.Subscribe("MSFT")

	h.Broadcast("AAPL", []byte("only-aapl"))

	if got := string(recvWithin(t, aapl, time.Second)); got != "only-aapl" {
		t.Fatalf("AAPL subscriber got %q", got)
	}
	select {
	case msg := <-msft.Out():
		t.Fatalf("MSFT subscriber should not have received %q", msg)
	case <-time.After(100 * time.Millisecond):
		// expected: no cross-talk
	}
}

func TestSlowSubscriberIsDroppedNotBlocking(t *testing.T) {
	h := startHub(t)

	// This subscriber never reads, so its buffer fills and further messages are
	// dropped for it — without blocking the hub or other subscribers.
	slow := h.Subscribe("AAPL")
	_ = slow

	total := clientBuffer + 50
	for i := 0; i < total; i++ {
		h.Broadcast("AAPL", []byte("x")) // must not deadlock
	}

	// Give the hub goroutine time to process the queued broadcasts.
	deadline := time.After(time.Second)
	for {
		if h.Dropped() > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("expected some drops for the slow subscriber, got %d", h.Dropped())
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func TestUnsubscribeClosesChannel(t *testing.T) {
	h := startHub(t)

	c := h.Subscribe("AAPL")
	h.Unsubscribe(c)

	select {
	case _, ok := <-c.Out():
		if ok {
			t.Fatal("expected channel to be closed after unsubscribe")
		}
	case <-time.After(time.Second):
		t.Fatal("channel not closed after unsubscribe")
	}
}
