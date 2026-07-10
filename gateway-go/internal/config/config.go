// Package config loads gateway settings from the environment. No secrets are
// ever hard-coded; the JWT secret must match the one the Java API signs with.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	// Address the HTTP/WebSocket server listens on.
	ListenAddr string
	// gRPC target of the C++ matching engine, used for on-demand book snapshots.
	EngineTarget string
	// Base URL of the Java API, used to proxy account/order reads.
	APIBaseURL string
	// HS256 secret shared with the Java API. Required — no default.
	JWTSecret []byte
	// Kafka: the fills topic feeds the live WebSocket stream (Phase 4).
	KafkaBrokers []string
	FillsTopic   string
	KafkaGroupID string
	// Redis: read-through cache for book/quote reads (Phase 5). Empty = disabled.
	RedisAddr    string
	BookCacheTTL time.Duration
	// CORS: origin allowed to call the API from a browser ("*" for any).
	CORSAllowedOrigin string
	// Per-client request rate limit (token bucket).
	RateLimitPerSec float64
	RateLimitBurst  int
	// How long a WebSocket write may block before the client is considered slow.
	WriteTimeout time.Duration
}

func Load() (Config, error) {
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		// Must match the Java service's HS256 key and be >= 256 bits.
		return Config{}, fmt.Errorf("JWT_SECRET must be set and at least 32 bytes")
	}

	return Config{
		ListenAddr:      getenv("GATEWAY_LISTEN_ADDR", ":8090"),
		EngineTarget:    getenv("ENGINE_GRPC_TARGET", "localhost:50051"),
		APIBaseURL:      getenv("API_BASE_URL", "http://localhost:8080"),
		JWTSecret:       []byte(secret),
		KafkaBrokers:    splitCSV(getenv("KAFKA_BOOTSTRAP_SERVERS", "localhost:19092")),
		FillsTopic:      getenv("KAFKA_FILLS_TOPIC", "fills"),
		KafkaGroupID:    getenv("KAFKA_GROUP_ID", "gateway-fills"),
		RedisAddr:         os.Getenv("REDIS_ADDR"), // empty disables caching
		BookCacheTTL:      getenvDuration("BOOK_CACHE_TTL", 750*time.Millisecond),
		CORSAllowedOrigin: getenv("CORS_ALLOWED_ORIGIN", "*"),
		RateLimitPerSec: getenvFloat("RATE_LIMIT_PER_SEC", 20),
		RateLimitBurst:  getenvInt("RATE_LIMIT_BURST", 40),
		WriteTimeout:    getenvDuration("WS_WRITE_TIMEOUT", 5*time.Second),
	}, nil
}

// splitCSV parses a comma-separated broker list, trimming spaces and empties.
func splitCSV(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return def
}

func getenvFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		var f float64
		if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
			return f
		}
	}
	return def
}

func getenvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
