package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"loop/internal/config"
	"loop/internal/models"
	"loop/internal/repo"
)

type RecoveryProbe struct {
	keyRepo      *repo.APIKeyRepo
	probeRepo    *repo.KeyProbeRepo
	settingsRepo *repo.SettingsRepo
	channelRepo  *repo.ChannelRepo
	cfg          config.Config
	client       *http.Client
}

func NewRecoveryProbe(keyRepo *repo.APIKeyRepo, probeRepo *repo.KeyProbeRepo, settingsRepo *repo.SettingsRepo, channelRepo *repo.ChannelRepo, cfg config.Config) *RecoveryProbe {
	return &RecoveryProbe{
		keyRepo:      keyRepo,
		probeRepo:    probeRepo,
		settingsRepo: settingsRepo,
		channelRepo:  channelRepo,
		cfg:          cfg,
		client:       &http.Client{Timeout: 30 * time.Second},
	}
}

func (rp *RecoveryProbe) Start(ctx context.Context) {
	interval := time.Duration(rp.cfg.ProbeCheckIntervalSec) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("recovery probe started, interval=%v", interval)
	for {
		select {
		case <-ctx.Done():
			log.Println("recovery probe stopped")
			return
		case <-ticker.C:
			rp.runProbeCycle(ctx)
		}
	}
}

func (rp *RecoveryProbe) runProbeCycle(ctx context.Context) {
	enabled, _ := rp.settingsRepo.Get("recovery_probe_enabled")
	if enabled == "false" {
		return
	}

	keys, err := rp.keyRepo.ListDisabledForProbe()
	if err != nil {
		log.Printf("probe: list disabled keys error: %v", err)
		return
	}

	for _, key := range keys {
		select {
		case <-ctx.Done():
			return
		default:
		}
		rp.probeOneKey(ctx, key)
	}
}

func (rp *RecoveryProbe) probeOneKey(ctx context.Context, key models.APIKey) {
	log.Printf("probing key %d (alias=%s)", key.ID, key.Alias)

	probe := models.KeyProbe{
		APIKeyID:  key.ID,
		CreatedAt: time.Now(),
	}

	ch, err := rp.channelRepo.GetByID(key.ChannelID)
	if err != nil {
		log.Printf("probe: get channel for key %d error: %v", key.ID, err)
		return
	}

	body, _ := json.Marshal(map[string]interface{}{
		"model":      "claude-haiku-4-5-20251001",
		"max_tokens": 1,
		"messages":   []map[string]string{{"role": "user", "content": "hi"}},
	})

	start := time.Now()
	upstreamURL := ch.BaseURL + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, "POST", upstreamURL, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", key.KeyValue)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := rp.client.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		probe.LatencyMs = latency
		probe.ErrorMsg = err.Error()
		rp.recordProbeFailure(key, probe)
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	probe.LatencyMs = latency
	probe.StatusCode = resp.StatusCode

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		probe.ErrorMsg = fmt.Sprintf("auth failed: %d", resp.StatusCode)
		rp.recordProbeFailure(key, probe)
		return
	}

	probe.Success = true
	rp.probeRepo.Create(&probe)

	key.IsActive = true
	key.ConsecutiveFailures = 0
	key.DisabledAt = nil
	key.NextProbeAt = nil
	key.ProbeBackoffMin = rp.cfg.ProbeBackoffBaseMin
	rp.keyRepo.Update(&key)
	log.Printf("key %d re-enabled after probe success", key.ID)
}

func (rp *RecoveryProbe) recordProbeFailure(key models.APIKey, probe models.KeyProbe) {
	probe.Success = false
	rp.probeRepo.Create(&probe)

	nextBackoff := key.ProbeBackoffMin * 2
	if nextBackoff > rp.cfg.ProbeBackoffMaxMin {
		nextBackoff = rp.cfg.ProbeBackoffMaxMin
	}
	key.ProbeBackoffMin = nextBackoff
	nextProbe := time.Now().Add(time.Duration(nextBackoff) * time.Minute)
	key.NextProbeAt = &nextProbe
	rp.keyRepo.Update(&key)
	log.Printf("key %d probe failed, next probe in %d min", key.ID, nextBackoff)
}

func (rp *RecoveryProbe) ProbeSingleKey(ctx context.Context, keyID int64) (*models.KeyProbe, error) {
	key, err := rp.keyRepo.GetByID(keyID)
	if err != nil {
		return nil, fmt.Errorf("key not found: %w", err)
	}

	probe := models.KeyProbe{
		APIKeyID:  key.ID,
		CreatedAt: time.Now(),
	}

	ch, err := rp.channelRepo.GetByID(key.ChannelID)
	if err != nil {
		return nil, err
	}

	body, _ := json.Marshal(map[string]interface{}{
		"model":      "claude-haiku-4-5-20251001",
		"max_tokens": 1,
		"messages":   []map[string]string{{"role": "user", "content": "hi"}},
	})

	start := time.Now()
	upstreamURL := ch.BaseURL + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, "POST", upstreamURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", key.KeyValue)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := rp.client.Do(req)
	latency := time.Since(start).Milliseconds()
	probe.LatencyMs = latency

	if err != nil {
		probe.ErrorMsg = err.Error()
		rp.probeRepo.Create(&probe)
		return &probe, nil
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	probe.StatusCode = resp.StatusCode

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		probe.ErrorMsg = fmt.Sprintf("auth failed: %d", resp.StatusCode)
		rp.recordProbeFailure(*key, probe)
		return &probe, nil
	}

	probe.Success = true
	rp.probeRepo.Create(&probe)

	if !key.IsActive {
		key.IsActive = true
		key.ConsecutiveFailures = 0
		key.DisabledAt = nil
		key.NextProbeAt = nil
		key.ProbeBackoffMin = rp.cfg.ProbeBackoffBaseMin
		rp.keyRepo.Update(key)
	}

	return &probe, nil
}
