package httpserver

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"loop/internal/config"
	"loop/internal/db"
	"loop/internal/models"
	"loop/internal/repo"
	"loop/internal/services"
)

type Server struct {
	cfg config.Config
}

func NewServer(cfg config.Config) *Server {
	return &Server{cfg: cfg}
}

func (s *Server) Run() error {
	database, err := db.Open(s.cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	channelRepo := repo.NewChannelRepo(database)
	keyRepo := repo.NewAPIKeyRepo(database)
	usageRepo := repo.NewUsageRepo(database)
	probeRepo := repo.NewKeyProbeRepo(database)
	settingsRepo := repo.NewSettingsRepo(database)

	// Initialize default settings
	if v, _ := settingsRepo.Get("recovery_probe_enabled"); v == "" {
		settingsRepo.Set("recovery_probe_enabled", "true")
	}
	if v, _ := settingsRepo.Get("auto_disable_threshold"); v == "" {
		settingsRepo.Set("auto_disable_threshold", fmt.Sprintf("%d", s.cfg.DisableThreshold))
	}
	if v, _ := settingsRepo.Get("auto_delete_401_keys_enabled"); v == "" {
		settingsRepo.Set("auto_delete_401_keys_enabled", "false")
	}

	// Generate admin token if not set
	adminToken := s.cfg.AdminToken
	if adminToken == "" {
		adminToken = generateToken()
		log.Printf("========================================")
		log.Printf("  Admin Token: %s", adminToken)
		log.Printf("========================================")
		log.Printf("Save this token! It won't be shown again.")
	}

	rotator := services.NewKeyRotator(keyRepo, s.cfg)
	proxy := services.NewProxyHandler(rotator, channelRepo, usageRepo, settingsRepo, s.cfg)
	recoveryProbe := services.NewRecoveryProbe(keyRepo, probeRepo, settingsRepo, channelRepo, usageRepo, s.cfg)

	// Start recovery probe
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go recoveryProbe.Start(ctx)

	handlers := NewHandlers(channelRepo, keyRepo, usageRepo, probeRepo, settingsRepo)
	settingsHandlers := NewSettingsHandlers(settingsRepo, s.cfg)

	router := NewRouter(handlers, settingsHandlers, proxy, adminToken, recoveryProbe)

	srv := &http.Server{
		Addr:    ":" + s.cfg.Port,
		Handler: router,
	}

	go func() {
		log.Printf("server starting on :%s", s.cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	return srv.Shutdown(shutdownCtx)
}

func generateToken() string {
	b := make([]byte, 24)
	rand.Read(b)
	return "loop_" + hex.EncodeToString(b)
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// probeKeyHandler handles POST /api/keys/:id/probe
func probeKeyHandler(w http.ResponseWriter, r *http.Request, rp *services.RecoveryProbe) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, 400, "invalid id")
		return
	}
	probe, err := rp.ProbeSingleKey(r.Context(), id)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, probe)
}

type probeKeysRequest struct {
	IDs         []int64 `json:"ids"`
	DeleteOn401 bool    `json:"delete_on_401"`
}

type probeKeysResult struct {
	ID      int64            `json:"id"`
	Probe   *models.KeyProbe `json:"probe,omitempty"`
	Error   string           `json:"error,omitempty"`
	Deleted bool             `json:"deleted"`
}

type probeKeysResponse struct {
	Total   int               `json:"total"`
	Success int               `json:"success"`
	Failed  int               `json:"failed"`
	Results []probeKeysResult `json:"results"`
}

func probeKeysHandler(w http.ResponseWriter, r *http.Request, rp *services.RecoveryProbe) {
	var input probeKeysRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, 400, "invalid request body")
		return
	}
	if len(input.IDs) == 0 {
		writeError(w, 400, "ids are required")
		return
	}

	seen := make(map[int64]bool, len(input.IDs))
	results := make([]probeKeysResult, 0, len(input.IDs))
	for _, id := range input.IDs {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true

		probe, deleted, err := rp.ProbeSingleKeyWithOptions(r.Context(), id, services.ProbeSingleKeyOptions{
			DeleteOn401: input.DeleteOn401,
		})
		result := probeKeysResult{ID: id, Probe: probe, Deleted: deleted}
		if err != nil {
			result.Error = err.Error()
		}
		results = append(results, result)
	}

	resp := probeKeysResponse{Total: len(results), Results: results}
	for _, result := range results {
		if result.Probe != nil && result.Probe.Success {
			resp.Success++
			continue
		}
		resp.Failed++
	}
	writeJSON(w, 200, resp)
}

var _ = hashToken
