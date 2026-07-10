package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	pingPeriod = 30 * time.Second
	pongWait   = 40 * time.Second // must exceed pingPeriod
)

// handleWS streams live fills for ?symbol=... to a WebSocket client.
//
// The client is auth'd via a Bearer header or a ?token= query param (browsers
// can't set headers on the upgrade request). Each connection gets its own hub
// subscription; a shared per-symbol pump feeds the hub. A slow client is
// dropped by the hub rather than backing pressure onto matching data.
func (s *Server) handleWS(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol query param required"})
		return
	}

	token := bearerOrQueryToken(c)
	if _, err := s.verifier.Verify(token); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// Upgrade already wrote an error response.
		s.log.Warn("ws upgrade failed", "err", err)
		return
	}
	defer conn.Close()

	s.metrics.ActiveWS.Inc()
	defer s.metrics.ActiveWS.Dec()

	// Fills arrive via the shared Kafka consumer; the client just subscribes to
	// this symbol's fan-out room.
	client := s.hub.Subscribe(symbol)
	defer s.hub.Unsubscribe(client)

	// A reader goroutine drains control frames and updates the read deadline via
	// pong handler; it also detects client disconnect and closes the conn, which
	// unblocks the writer below.
	closed := make(chan struct{})
	go func() {
		defer close(closed)
		conn.SetReadLimit(512)
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(pongWait))
		})
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-closed:
			return
		case <-s.baseCtx.Done():
			return
		case msg, ok := <-client.Out():
			if !ok {
				return // hub shut down
			}
			_ = conn.SetWriteDeadline(time.Now().Add(s.cfg.WriteTimeout))
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(s.cfg.WriteTimeout))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func bearerOrQueryToken(c *gin.Context) string {
	const prefix = "Bearer "
	if h := c.GetHeader("Authorization"); len(h) > len(prefix) && h[:len(prefix)] == prefix {
		return h[len(prefix):]
	}
	return c.Query("token")
}
