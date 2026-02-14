package api

import (
	"net/http"

	"github.com/annismckenzie/x-article-exporter/internal/config"
	"github.com/annismckenzie/x-article-exporter/internal/jobs"
)

// Server is the HTTP API server for article exports.
type Server struct {
	manager *jobs.Manager
	apiKeys map[string]string // key -> name
	limiter *rateLimiter
}

// NewServer creates an API server from the given config and manager.
func NewServer(cfg *config.ServerConfig, manager *jobs.Manager) *Server {
	keys := make(map[string]string, len(cfg.APIKeys))
	limiter := newRateLimiter(cfg.DefaultRateLimitPerHour)

	for _, k := range cfg.APIKeys {
		keys[k.Key] = k.Name
		if k.RateLimitPerHour > 0 {
			limiter.setKeyLimit(k.Key, k.RateLimitPerHour)
		}
	}

	return &Server{
		manager: manager,
		apiKeys: keys,
		limiter: limiter,
	}
}

// Handler returns the configured HTTP handler with all routes registered.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /export", s.requireAuth(s.requireRateLimit(s.handleExport)))
	mux.HandleFunc("GET /export/{id}", s.requireAuth(s.handleExportStatus))
	mux.HandleFunc("GET /export/{id}/pdf", s.requireAuth(s.handleExportPDF))
	return mux
}
