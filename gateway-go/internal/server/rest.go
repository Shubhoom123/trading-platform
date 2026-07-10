package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shubham/trading-platform/gateway-go/internal/cache"
	"github.com/shubham/trading-platform/gateway-go/internal/pb"
)

// handleBookSnapshot serves an order book read. It tries Redis first and only
// falls back to a gRPC call to the engine on a miss, caching the result under a
// short TTL. Redis being down degrades to the engine path, never an error.
func (s *Server) handleBookSnapshot(c *gin.Context) {
	symbol := c.Param("symbol")

	var depth uint32
	if d := c.Query("depth"); d != "" {
		if _, err := fmt.Sscanf(d, "%d", &depth); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "depth must be a non-negative integer"})
			return
		}
	}

	snap, ok := s.fetchBook(c.Request.Context(), symbol, depth)
	if !ok {
		c.JSON(http.StatusBadGateway, gin.H{"error": "matching engine unavailable"})
		return
	}
	c.JSON(http.StatusOK, snap)
}

// handleQuote returns just the top of book (best bid/ask), served from the same
// cache (depth 1).
func (s *Server) handleQuote(c *gin.Context) {
	symbol := c.Param("symbol")
	snap, ok := s.fetchBook(c.Request.Context(), symbol, 1)
	if !ok {
		c.JSON(http.StatusBadGateway, gin.H{"error": "matching engine unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"symbol":  snap.Symbol,
		"bestBid": snap.BestBid,
		"bestAsk": snap.BestAsk,
	})
}

// fetchBook is the read-through: cache -> engine -> cache.
func (s *Server) fetchBook(ctx context.Context, symbol string, depth uint32) (cache.BookSnapshot, bool) {
	if snap, hit := s.bookCache.Get(ctx, symbol, depth); hit {
		s.metrics.CacheHits.Inc()
		return snap, true
	}
	s.metrics.CacheMisses.Inc()

	rpcCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	pbSnap, err := s.engine.GetBookSnapshot(rpcCtx, symbol, depth)
	if err != nil {
		s.log.Warn("book snapshot failed", "symbol", symbol, "err", err)
		return cache.BookSnapshot{}, false
	}

	snap := toCacheSnapshot(pbSnap).WithTopOfBook()
	s.bookCache.Set(ctx, symbol, depth, snap)
	return snap, true
}

func toCacheSnapshot(snap *pb.BookSnapshot) cache.BookSnapshot {
	out := cache.BookSnapshot{Symbol: snap.GetSymbol()}
	for _, l := range snap.GetBids() {
		out.Bids = append(out.Bids, cache.BookLevel{PriceTicks: l.GetPriceTicks(), Quantity: l.GetQuantity()})
	}
	for _, l := range snap.GetAsks() {
		out.Asks = append(out.Asks, cache.BookLevel{PriceTicks: l.GetPriceTicks(), Quantity: l.GetQuantity()})
	}
	return out
}

// proxyToAPI forwards a request to the Java API, passing the caller's
// Authorization header and request body through and streaming the response
// back. The Java service stays the owner of auth/account/order data; the
// gateway is the single front door clients talk to.
func (s *Server) proxyToAPI(method, path string) gin.HandlerFunc {
	client := &http.Client{Timeout: 5 * time.Second}
	return func(c *gin.Context) {
		var body io.Reader
		if c.Request.Body != nil {
			body = c.Request.Body
		}
		req, err := http.NewRequestWithContext(c.Request.Context(),
			method, s.cfg.APIBaseURL+path, body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "proxy build failed"})
			return
		}
		if auth := c.GetHeader("Authorization"); auth != "" {
			req.Header.Set("Authorization", auth)
		}
		if ct := c.GetHeader("Content-Type"); ct != "" {
			req.Header.Set("Content-Type", ct)
		}

		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "upstream api unavailable"})
			return
		}
		defer resp.Body.Close()

		c.Status(resp.StatusCode)
		if ct := resp.Header.Get("Content-Type"); ct != "" {
			c.Header("Content-Type", ct)
		}
		_, _ = io.Copy(c.Writer, resp.Body)
	}
}
