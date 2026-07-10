// Command gateway is the Go edge service: WebSocket fan-out of live fills,
// lightweight REST reads, JWT validation, and rate limiting.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shubham/trading-platform/gateway-go/internal/cache"
	"github.com/shubham/trading-platform/gateway-go/internal/config"
	"github.com/shubham/trading-platform/gateway-go/internal/engine"
	"github.com/shubham/trading-platform/gateway-go/internal/hub"
	"github.com/shubham/trading-platform/gateway-go/internal/kafka"
	"github.com/shubham/trading-platform/gateway-go/internal/metrics"
	"github.com/shubham/trading-platform/gateway-go/internal/server"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		log.Error("config error", "err", err)
		os.Exit(1)
	}

	// Root context cancelled on SIGINT/SIGTERM; drives graceful shutdown of the
	// hub, the per-symbol pumps, and in-flight WebSocket connections.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	eng, err := engine.Dial(cfg.EngineTarget)
	if err != nil {
		log.Error("engine dial failed", "target", cfg.EngineTarget, "err", err)
		os.Exit(1)
	}
	defer eng.Close()

	m := metrics.New()

	h := hub.New()
	go h.Run(ctx)

	// Single consumer-group reader drains the fills topic and fans out to the hub.
	fills := kafka.NewFillConsumer(cfg.KafkaBrokers, cfg.FillsTopic, cfg.KafkaGroupID, h, m, log)
	go fills.Run(ctx)

	// Redis-backed read-through cache for book/quote reads. Falls back to a
	// no-op store (always-miss) when REDIS_ADDR is unset, so the gateway runs
	// fine without Redis — just without the cache layer.
	var store cache.Store = cache.NoopStore{}
	if cfg.RedisAddr != "" {
		if rs, err := cache.NewRedisStore(cfg.RedisAddr); err != nil {
			log.Warn("invalid REDIS_ADDR, caching disabled", "err", err)
		} else {
			store = rs
			defer rs.Close()
			log.Info("book cache enabled", "redis", cfg.RedisAddr, "ttl", cfg.BookCacheTTL)
		}
	}
	bookCache := cache.NewBookCache(store, cfg.BookCacheTTL)

	srv := server.New(ctx, cfg, log, eng, h, bookCache, m)

	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           srv.Router(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("gateway listening", "addr", cfg.ListenAddr, "engine", cfg.EngineTarget)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server error", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "err", err)
	}
}
