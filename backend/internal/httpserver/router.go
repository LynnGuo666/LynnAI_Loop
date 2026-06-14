package httpserver

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"loop/internal/config"
	"loop/internal/middleware"
	"loop/internal/services"
)

func NewRouter(h *Handlers, sh *SettingsHandlers, proxy *services.ProxyHandler, adminToken string, recoverProbe *services.RecoveryProbe, cfg config.Config) http.Handler {
	r := chi.NewRouter()
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(corsMiddleware)

	// Admin API and proxy endpoints require the admin token.
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(adminToken))

		// Proxy endpoints — throttled to provide backpressure under load.
		// Requests beyond the concurrency limit queue (up to backlog) and then
		// receive 503 if the queue wait exceeds the timeout, instead of piling
		// up goroutines that all contend for the DB write lock.
		r.Group(func(r chi.Router) {
			r.Use(chiMiddleware.ThrottleBacklog(
				cfg.ProxyMaxConcurrency,
				cfg.ProxyBacklog,
				time.Duration(cfg.ProxyBacklogTimeoutSec)*time.Second,
			))

			r.Post("/channel/{channelID}/v1/messages", proxy.HandleMessages)
			r.Post("/channel/{channelID}/v1/chat/completions", proxy.HandleProxy)
			r.Post("/channel/{channelID}/v1/responses", proxy.HandleProxy)
			r.Post("/channel/{channelID}/v1beta/models/{model}:generateContent", proxy.HandleProxy)
			r.Post("/channel/{channelID}/v1beta/models/{model}:streamGenerateContent", proxy.HandleProxy)
			r.Get("/channel/{channelID}/v1/models", proxy.HandleModels)

			// Auto-route for single channel proxy endpoints.
			r.Post("/v1/messages", proxy.HandleMessagesSingleChannel)
			r.Post("/v1/chat/completions", proxy.HandleProxySingleChannel)
			r.Post("/v1/responses", proxy.HandleProxySingleChannel)
			r.Post("/v1beta/models/{model}:generateContent", proxy.HandleProxySingleChannel)
			r.Post("/v1beta/models/{model}:streamGenerateContent", proxy.HandleProxySingleChannel)
			r.Get("/v1/models", proxy.HandleModelsSingleChannel)
		})

		r.Get("/api/healthz", sh.Healthz)

		// Channels
		r.Get("/api/channels", h.ListChannels)
		r.Post("/api/channels", h.CreateChannel)
		r.Get("/api/channels/{id}", h.GetChannel)
		r.Put("/api/channels/{id}", h.UpdateChannel)
		r.Delete("/api/channels/{id}", h.DeleteChannel)

		// Keys (under channel)
		r.Get("/api/channels/{id}/keys", h.ListKeys)
		r.Post("/api/channels/{id}/keys", h.CreateKey)

		// Keys (standalone)
		r.Get("/api/keys", h.ListAllKeys)
		r.Get("/api/keys/export", h.ExportKeys)
		r.Post("/api/keys/import", h.ImportKeys)
		r.Post("/api/keys/probe", func(w http.ResponseWriter, r *http.Request) {
			probeKeysHandler(w, r, recoverProbe, cfg.ProbeBatchConcurrency)
		})
		r.Get("/api/keys/{id}", h.GetKey)
		r.Put("/api/keys/{id}", h.UpdateKey)
		r.Delete("/api/keys/{id}", h.DeleteKey)
		r.Post("/api/keys/{id}/enable", h.EnableKey)
		r.Post("/api/keys/{id}/probe", func(w http.ResponseWriter, r *http.Request) {
			probeKeyHandler(w, r, recoverProbe)
		})

		// Usage
		r.Get("/api/usage", h.ListUsage)
		r.Get("/api/usage/stats", h.UsageStats)
		r.Get("/api/usage/timeseries", h.UsageTimeseries)
		r.Get("/api/usage/model-stats", h.UsageModelStats)
		r.Get("/api/usage/channel-stats", h.UsageChannelStats)
		r.Get("/api/usage/models", h.UsageModels)

		// Settings
		r.Get("/api/settings", sh.GetSettings)
		r.Put("/api/settings", sh.UpdateSettings)
	})

	// Frontend SPA (must be last - catch-all)
	frontend := FrontendHandler()
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		// Don't serve frontend for API/proxy paths
		if strings.HasPrefix(r.URL.Path, "/api/") ||
			strings.HasPrefix(r.URL.Path, "/channel/") ||
			strings.HasPrefix(r.URL.Path, "/v1/") {
			http.Error(w, `{"error":{"type":"not_found","message":"endpoint not found"}}`, http.StatusNotFound)
			return
		}
		frontend.ServeHTTP(w, r)
	})

	return r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-api-key, x-goog-api-key, anthropic-version, anthropic-beta, OpenAI-Beta")
		w.Header().Set("Access-Control-Expose-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
