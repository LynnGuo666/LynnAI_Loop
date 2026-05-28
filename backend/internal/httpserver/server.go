package httpserver

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
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
	proxy := services.NewProxyHandler(rotator, channelRepo, usageRepo, s.cfg)
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

var _ = hashToken
