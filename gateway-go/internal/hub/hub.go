// Package hub fans out per-symbol market messages to many subscribers.
//
// The design goal from the build plan: a slow WebSocket client must never
// block matching-data delivery to everyone else. Each subscriber has its own
// buffered channel, and broadcast uses a non-blocking send — if a client's
// buffer is full it drops the message for that client only (and counts it),
// rather than stalling the hub.
package hub

import (
	"context"
	"sync/atomic"
)

// clientBuffer is how many messages may queue for one subscriber before the
// hub starts dropping for that (slow) subscriber.
const clientBuffer = 64

type message struct {
	symbol string
	data   []byte
}

// Client is one subscription to a symbol's stream.
type Client struct {
	symbol string
	send   chan []byte
}

// Out is the read side the consumer (a WebSocket writer) ranges over. It is
// closed when the client is unsubscribed or the hub stops.
func (c *Client) Out() <-chan []byte { return c.send }

// Symbol returns the symbol this client is subscribed to.
func (c *Client) Symbol() string { return c.symbol }

// Hub is a goroutine-safe fan-out. Construct with New, then run Run in its own
// goroutine; all mutation of the subscriber map happens on that goroutine so no
// lock is needed on the hot path.
type Hub struct {
	register   chan *Client
	unregister chan *Client
	broadcast  chan message
	rooms      map[string]map[*Client]struct{}

	dropped atomic.Uint64
}

func New() *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan message, 256),
		rooms:      make(map[string]map[*Client]struct{}),
	}
}

// Run owns the subscriber map until ctx is cancelled, at which point every
// subscriber channel is closed so readers unblock and exit.
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			for _, room := range h.rooms {
				for c := range room {
					close(c.send)
				}
			}
			h.rooms = make(map[string]map[*Client]struct{})
			return

		case c := <-h.register:
			room := h.rooms[c.symbol]
			if room == nil {
				room = make(map[*Client]struct{})
				h.rooms[c.symbol] = room
			}
			room[c] = struct{}{}

		case c := <-h.unregister:
			if room, ok := h.rooms[c.symbol]; ok {
				if _, present := room[c]; present {
					delete(room, c)
					close(c.send)
				}
				if len(room) == 0 {
					delete(h.rooms, c.symbol)
				}
			}

		case m := <-h.broadcast:
			for c := range h.rooms[m.symbol] {
				select {
				case c.send <- m.data:
				default:
					// Subscriber is not keeping up; drop for it alone.
					h.dropped.Add(1)
				}
			}
		}
	}
}

// Subscribe registers and returns a new client for the given symbol.
func (h *Hub) Subscribe(symbol string) *Client {
	c := &Client{symbol: symbol, send: make(chan []byte, clientBuffer)}
	h.register <- c
	return c
}

// Unsubscribe removes a client. Safe to call once per client.
func (h *Hub) Unsubscribe(c *Client) {
	h.unregister <- c
}

// Broadcast delivers data to every current subscriber of symbol. Never blocks
// on a slow subscriber.
func (h *Hub) Broadcast(symbol string, data []byte) {
	h.broadcast <- message{symbol: symbol, data: data}
}

// Dropped reports the total number of messages dropped due to slow subscribers.
// Exposed for the /metrics endpoint (Phase 6).
func (h *Hub) Dropped() uint64 { return h.dropped.Load() }
