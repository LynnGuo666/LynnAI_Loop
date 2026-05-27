package services

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"loop/internal/config"
	"loop/internal/models"
	"loop/internal/repo"
)

var ErrNoActiveKeys = errors.New("no_active_keys")

type ProxyHandler struct {
	rotator     *KeyRotator
	channelRepo *repo.ChannelRepo
	usageRepo   *repo.UsageRepo
	cfg         config.Config
	client      *http.Client
}

func NewProxyHandler(rotator *KeyRotator, channelRepo *repo.ChannelRepo, usageRepo *repo.UsageRepo, cfg config.Config) *ProxyHandler {
	return &ProxyHandler{
		rotator:     rotator,
		channelRepo: channelRepo,
		usageRepo:   usageRepo,
		cfg:         cfg,
		client:      &http.Client{Timeout: time.Duration(cfg.UpstreamTimeoutSec) * time.Second},
	}
}

func (ph *ProxyHandler) SetChannelRepo(cr *repo.ChannelRepo) {
	ph.channelRepo = cr
}

func (ph *ProxyHandler) HandleMessagesSingleChannel(w http.ResponseWriter, r *http.Request) {
	channels, err := ph.channelRepo.List()
	if err != nil || len(channels) != 1 {
		http.Error(w, `{"error":{"type":"not_found","message":"single-channel auto-route requires exactly one channel"}}`, http.StatusBadRequest)
		return
	}
	r.SetPathValue("channelID", strconv.FormatInt(channels[0].ID, 10))
	ph.HandleMessages(w, r)
}

func (ph *ProxyHandler) HandleModelsSingleChannel(w http.ResponseWriter, r *http.Request) {
	channels, err := ph.channelRepo.List()
	if err != nil || len(channels) != 1 {
		http.Error(w, `{"error":{"type":"not_found","message":"single-channel auto-route requires exactly one channel"}}`, http.StatusBadRequest)
		return
	}
	r.SetPathValue("channelID", strconv.FormatInt(channels[0].ID, 10))
	ph.HandleModels(w, r)
}

func (ph *ProxyHandler) HandleMessages(w http.ResponseWriter, r *http.Request) {
	channelID, err := strconv.ParseInt(r.PathValue("channelID"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":{"type":"invalid_request","message":"invalid channel id"}}`, http.StatusBadRequest)
		return
	}

	ch, err := ph.channelRepo.GetByID(channelID)
	if err != nil {
		http.Error(w, `{"error":{"type":"not_found","message":"channel not found"}}`, http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":{"type":"read_error","message":"failed to read body"}}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var reqBody struct {
		Stream bool   `json:"stream"`
		Model  string `json:"model"`
	}
	json.Unmarshal(body, &reqBody)

	maxAttempts := ph.cfg.MaxProxyAttempts
	activeCount, _ := ph.rotator.ActiveKeyCount(channelID)
	if activeCount < maxAttempts {
		maxAttempts = activeCount
	}
	if maxAttempts == 0 {
		http.Error(w, `{"error":{"type":"proxy_error","message":"no_active_keys"}}`, http.StatusBadGateway)
		return
	}

	startTime := time.Now()
	clientIP := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		clientIP = strings.Split(fwd, ",")[0]
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		key, err := ph.rotator.Select(channelID)
		if err != nil {
			lastErr = err
			break
		}

		upstreamURL := strings.TrimRight(ch.BaseURL, "/") + "/v1/messages"
		req, err := http.NewRequestWithContext(r.Context(), "POST", upstreamURL, bytes.NewReader(body))
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", key.KeyValue)
		if v := r.Header.Get("anthropic-version"); v != "" {
			req.Header.Set("anthropic-version", v)
		} else {
			req.Header.Set("anthropic-version", "2023-06-01")
		}
		if beta := r.Header.Get("anthropic-beta"); beta != "" {
			req.Header.Set("anthropic-beta", beta)
		}

		resp, err := ph.client.Do(req)
		if err != nil {
			ph.rotator.ReportFailure(key.ID)
			lastErr = err
			continue
		}

		if reqBody.Stream {
			err = ph.handleStreamResponse(w, resp, key, ch, reqBody.Model, "/v1/messages", clientIP, startTime)
		} else {
			err = ph.handleNonStreamResponse(w, resp, key, ch, reqBody.Model, "/v1/messages", clientIP, startTime)
		}

		if err == nil {
			return
		}
		lastErr = err
	}

	msg := "all_keys_exhausted"
	if lastErr != nil && lastErr == ErrNoActiveKeys {
		msg = "no_active_keys"
	}
	http.Error(w, fmt.Sprintf(`{"error":{"type":"proxy_error","message":"%s"}}`, msg), http.StatusBadGateway)
}

func (ph *ProxyHandler) HandleModels(w http.ResponseWriter, r *http.Request) {
	channelID, err := strconv.ParseInt(r.PathValue("channelID"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":{"type":"invalid_request","message":"invalid channel id"}}`, http.StatusBadRequest)
		return
	}

	ch, err := ph.channelRepo.GetByID(channelID)
	if err != nil {
		http.Error(w, `{"error":{"type":"not_found","message":"channel not found"}}`, http.StatusNotFound)
		return
	}

	key, err := ph.rotator.Select(channelID)
	if err != nil {
		http.Error(w, `{"error":{"type":"proxy_error","message":"no_active_keys"}}`, http.StatusBadGateway)
		return
	}

	upstreamURL := strings.TrimRight(ch.BaseURL, "/") + "/v1/models"
	req, err := http.NewRequestWithContext(r.Context(), "GET", upstreamURL, nil)
	if err != nil {
		http.Error(w, `{"error":{"type":"proxy_error","message":"internal error"}}`, http.StatusInternalServerError)
		return
	}
	req.Header.Set("x-api-key", key.KeyValue)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := ph.client.Do(req)
	if err != nil {
		ph.rotator.ReportFailure(key.ID)
		http.Error(w, `{"error":{"type":"proxy_error","message":"upstream error"}}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (ph *ProxyHandler) handleStreamResponse(w http.ResponseWriter, resp *http.Response, key *models.APIKey, ch *models.Channel, model, endpoint, clientIP string, startTime time.Time) error {
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if shouldRetry(resp.StatusCode) {
			ph.rotator.ReportFailure(key.ID)
			return fmt.Errorf("upstream %d", resp.StatusCode)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return nil
	}

	ph.rotator.ReportSuccess(key.ID)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, canFlush := w.(http.Flusher)

	var inputTokens, outputTokens, cacheCreate, cacheRead int64
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data != "[DONE]" {
				var event map[string]json.RawMessage
				if json.Unmarshal([]byte(data), &event) == nil {
					if t, ok := event["type"]; ok {
						var eventType string
						json.Unmarshal(t, &eventType)
						switch eventType {
						case "message_start":
							extractTokensFromMessageStart(event, &inputTokens, &cacheCreate, &cacheRead)
						case "message_delta":
							extractTokensFromMessageDelta(event, &outputTokens)
						}
					}
				}
			}
		}
		fmt.Fprintf(w, "%s\n", line)
		if canFlush {
			flusher.Flush()
		}
	}

	latency := time.Since(startTime).Milliseconds()
	ph.logUsage(key, ch, model, endpoint, inputTokens, outputTokens, cacheCreate, cacheRead, true, resp.StatusCode, latency, "", clientIP)
	return nil
}

func (ph *ProxyHandler) handleNonStreamResponse(w http.ResponseWriter, resp *http.Response, key *models.APIKey, ch *models.Channel, model, endpoint, clientIP string, startTime time.Time) error {
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		ph.rotator.ReportFailure(key.ID)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		if shouldRetry(resp.StatusCode) {
			ph.rotator.ReportFailure(key.ID)
			return fmt.Errorf("upstream %d", resp.StatusCode)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return nil
	}

	ph.rotator.ReportSuccess(key.ID)

	var usage struct {
		Usage struct {
			InputTokens         int64 `json:"input_tokens"`
			OutputTokens        int64 `json:"output_tokens"`
			CacheCreationTokens int64 `json:"cache_creation_input_tokens"`
			CacheReadTokens     int64 `json:"cache_read_input_tokens"`
		} `json:"usage"`
	}
	json.Unmarshal(body, &usage)

	latency := time.Since(startTime).Milliseconds()
	ph.logUsage(key, ch, model, endpoint, usage.Usage.InputTokens, usage.Usage.OutputTokens,
		usage.Usage.CacheCreationTokens, usage.Usage.CacheReadTokens, false, resp.StatusCode, latency, "", clientIP)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
	return nil
}

func (ph *ProxyHandler) logUsage(key *models.APIKey, ch *models.Channel, model, endpoint string, input, output, cacheCreate, cacheRead int64, isStream bool, statusCode int, latencyMs int64, errMsg, clientIP string) {
	if ph.usageRepo == nil {
		return
	}
	ph.usageRepo.Create(&models.UsageLog{
		ChannelID:           ch.ID,
		APIKeyID:            key.ID,
		Model:               model,
		Endpoint:            endpoint,
		InputTokens:         input,
		OutputTokens:        output,
		CacheCreationTokens: cacheCreate,
		CacheReadTokens:     cacheRead,
		IsStream:            isStream,
		StatusCode:          statusCode,
		LatencyMs:           latencyMs,
		Success:             errMsg == "",
		ErrorMessage:        errMsg,
		ClientIP:            clientIP,
		CreatedAt:           time.Now(),
	})
}

func shouldRetry(statusCode int) bool {
	return statusCode == 401 || statusCode == 403 || statusCode == 429 || statusCode >= 500
}

func extractTokensFromMessageStart(event map[string]json.RawMessage, input, cacheCreate, cacheRead *int64) {
	if msg, ok := event["message"]; ok {
		var m struct {
			Usage struct {
				InputTokens         int64 `json:"input_tokens"`
				CacheCreationTokens int64 `json:"cache_creation_input_tokens"`
				CacheReadTokens     int64 `json:"cache_read_input_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal(msg, &m) == nil {
			*input = m.Usage.InputTokens
			*cacheCreate = m.Usage.CacheCreationTokens
			*cacheRead = m.Usage.CacheReadTokens
		}
	}
}

func extractTokensFromMessageDelta(event map[string]json.RawMessage, output *int64) {
	if usage, ok := event["usage"]; ok {
		var u struct {
			OutputTokens int64 `json:"output_tokens"`
		}
		if json.Unmarshal(usage, &u) == nil {
			*output = u.OutputTokens
		}
	}
}
