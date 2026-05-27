package httpserver

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"loop/internal/middleware"
	"loop/internal/services"
)

func NewRouter(h *Handlers, sh *SettingsHandlers, proxy *services.ProxyHandler, adminToken string, recoverProbe *services.RecoveryProbe) http.Handler {
	r := chi.NewRouter()
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(corsMiddleware)

	// Admin API and proxy endpoints require the admin token.
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(adminToken))

		// Proxy endpoints
		r.Post("/channel/{channelID}/v1/messages", proxy.HandleMessages)
		r.Get("/channel/{channelID}/v1/models", proxy.HandleModels)

		// Auto-route for single channel: /v1/messages and /v1/models
		r.Post("/v1/messages", proxy.HandleMessagesSingleChannel)
		r.Get("/v1/models", proxy.HandleModelsSingleChannel)

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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-api-key, anthropic-version, anthropic-beta")
		w.Header().Set("Access-Control-Expose-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
