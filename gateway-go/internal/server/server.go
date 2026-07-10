// Package server wires the gateway's HTTP + WebSocket surface together.
package server

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/shubham/trading-platform/gateway-go/internal/auth"
	"github.com/shubham/trading-platform/gateway-go/internal/cache"
	"github.com/shubham/trading-platform/gateway-go/internal/config"
	"github.com/shubham/trading-platform/gateway-go/internal/engine"
	"github.com/shubham/trading-platform/gateway-go/internal/hub"
	"github.com/shubham/trading-platform/gateway-go/internal/metrics"
	"github.com/shubham/trading-platform/gateway-go/internal/middleware"
)

type Server struct {
	cfg       config.Config
	log       *slog.Logger
	verifier  *auth.Verifier
	engine    *engine.Client
	hub       *hub.Hub
	bookCache *cache.BookCache
	metrics   *metrics.Metrics
	upgrader  websocket.Upgrader
	baseCtx   context.Context
}

// New builds the server. Fills are delivered to the hub by a Kafka consumer
// (started in main); the engine client + Redis-backed book cache serve
// on-demand book/quote reads.
func New(baseCtx context.Context, cfg config.Config, log *slog.Logger,
	eng *engine.Client, h *hub.Hub, bookCache *cache.BookCache, m *metrics.Metrics) *Server {
	return &Server{
		cfg:       cfg,
		log:       log,
		verifier:  auth.NewVerifier(cfg.JWTSecret),
		engine:    eng,
		hub:       h,
		bookCache: bookCache,
		metrics:   m,
		baseCtx:   baseCtx,
		upgrader: websocket.Upgrader{
			// Same-origin is the safe default; loosen deliberately behind config
			// when a browser dashboard on another origin needs access (Phase 6).
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (s *Server) Router() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	rl := middleware.NewRateLimiter(s.cfg.RateLimitPerSec, s.cfg.RateLimitBurst)
	r.Use(gin.Recovery(), middleware.RequestLogger(s.log),
		middleware.CORS(s.cfg.CORSAllowedOrigin), rl.Middleware())

	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	// Prometheus scrape endpoint (unauthenticated, like the health check).
	r.GET("/metrics", gin.WrapH(s.metrics.Handler()))

	// WebSocket does its own auth (browsers can't set Authorization on the
	// upgrade), so it lives outside the bearer-only group.
	r.GET("/ws", s.handleWS)

	// Auth is public (no token yet) — proxied straight to the Java API so the
	// browser only ever talks to this one origin.
	r.POST("/api/auth/register", s.proxyToAPI(http.MethodPost, "/api/auth/register"))
	r.POST("/api/auth/login", s.proxyToAPI(http.MethodPost, "/api/auth/login"))
	r.POST("/api/auth/refresh", s.proxyToAPI(http.MethodPost, "/api/auth/refresh"))

	api := r.Group("/api", middleware.JWTAuth(s.verifier))
	{
		api.GET("/book/:symbol", s.handleBookSnapshot)
		api.GET("/quote/:symbol", s.handleQuote)
		api.GET("/orders", s.proxyToAPI(http.MethodGet, "/api/orders"))
		api.POST("/orders", s.proxyToAPI(http.MethodPost, "/api/orders"))
		api.GET("/account", s.proxyToAPI(http.MethodGet, "/api/account"))
	}

	return r
}
