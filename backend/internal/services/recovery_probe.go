package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
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

type upstreamModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
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

	probe, err := rp.runProbe(ctx, &key)
	if err != nil {
		log.Printf("probe: key %d error: %v", key.ID, err)
		return
	}

	if !probe.Success {
		rp.recordProbeFailure(key, probe)
		return
	}

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

	probe, err := rp.runProbe(ctx, key)
	if err != nil {
		return nil, err
	}

	if !probe.Success {
		if isAuthFailure(probe.StatusCode) {
			rp.recordProbeFailure(*key, probe)
		} else {
			rp.probeRepo.Create(&probe)
		}
		return &probe, nil
	}

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

func (rp *RecoveryProbe) runProbe(ctx context.Context, key *models.APIKey) (models.KeyProbe, error) {
	probe := models.KeyProbe{
		APIKeyID:  key.ID,
		CreatedAt: time.Now(),
	}

	ch, err := rp.channelRepo.GetByID(key.ChannelID)
	if err != nil {
		return probe, err
	}

	modelID := strings.TrimSpace(ch.ProbeModel)
	if modelID == "" {
		var models []string
		models, probe.StatusCode, probe.LatencyMs, err = rp.fetchProbeModels(ctx, ch, key)
		if err != nil {
			probe.ErrorMsg = err.Error()
			return probe, nil
		}
		modelID = chooseProbeModel(models)
		if modelID == "" {
			probe.ErrorMsg = "未配置探测模型，且 /v1/models 没有返回可用模型"
			return probe, nil
		}
	}

	probe.StatusCode, probe.LatencyMs, err = rp.probeMessages(ctx, ch, key, modelID)
	if err != nil {
		probe.ErrorMsg = err.Error()
		return probe, nil
	}

	probe.Success = true
	return probe, nil
}

func (rp *RecoveryProbe) fetchProbeModels(ctx context.Context, ch *models.Channel, key *models.APIKey) ([]string, int, int64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", joinUpstreamURL(ch.BaseURL, "/v1/models"), nil)
	if err != nil {
		return nil, 0, 0, err
	}
	req.Header.Set("x-api-key", key.KeyValue)
	req.Header.Set("anthropic-version", "2023-06-01")

	start := time.Now()
	resp, err := rp.client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return nil, 0, latency, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if isAuthFailure(resp.StatusCode) {
			return nil, resp.StatusCode, latency, fmt.Errorf("模型列表认证失败: %d", resp.StatusCode)
		}
		return nil, resp.StatusCode, latency, fmt.Errorf("/v1/models 请求失败: %d", resp.StatusCode)
	}

	var parsed upstreamModelsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, resp.StatusCode, latency, fmt.Errorf("解析 /v1/models 响应失败: %w", err)
	}

	modelIDs := make([]string, 0, len(parsed.Data))
	for _, item := range parsed.Data {
		if id := strings.TrimSpace(item.ID); id != "" {
			modelIDs = append(modelIDs, id)
		}
	}
	return modelIDs, resp.StatusCode, latency, nil
}

func (rp *RecoveryProbe) probeMessages(ctx context.Context, ch *models.Channel, key *models.APIKey, modelID string) (int, int64, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"model":      modelID,
		"max_tokens": 1,
		"messages":   []map[string]string{{"role": "user", "content": "hi"}},
	})

	req, err := http.NewRequestWithContext(ctx, "POST", joinUpstreamURL(ch.BaseURL, "/v1/messages"), bytes.NewReader(body))
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", key.KeyValue)
	req.Header.Set("anthropic-version", "2023-06-01")

	start := time.Now()
	resp, err := rp.client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return 0, latency, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if isAuthFailure(resp.StatusCode) {
			return resp.StatusCode, latency, fmt.Errorf("认证失败: %d", resp.StatusCode)
		}
		return resp.StatusCode, latency, fmt.Errorf("/v1/messages 探测失败: %d", resp.StatusCode)
	}

	return resp.StatusCode, latency, nil
}

func chooseProbeModel(modelIDs []string) string {
	preferred := []string{"haiku", "sonnet", "claude"}
	for _, keyword := range preferred {
		for _, id := range modelIDs {
			if strings.Contains(strings.ToLower(id), keyword) {
				return id
			}
		}
	}
	if len(modelIDs) > 0 {
		return modelIDs[0]
	}
	return ""
}

func joinUpstreamURL(baseURL, path string) string {
	return strings.TrimRight(baseURL, "/") + path
}

func isAuthFailure(statusCode int) bool {
	return statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden
}
