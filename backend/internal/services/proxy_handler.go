package services

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
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
	rotator      *KeyRotator
	channelRepo  *repo.ChannelRepo
	usageRepo    *repo.UsageRepo
	settingsRepo *repo.SettingsRepo
	cfg          config.Config
	client       *http.Client
}

func NewProxyHandler(rotator *KeyRotator, channelRepo *repo.ChannelRepo, usageRepo *repo.UsageRepo, settingsRepo *repo.SettingsRepo, cfg config.Config) *ProxyHandler {
	transport := newSharedTransport()
	// Wait this long for the upstream to send response headers (first byte).
	// The overall request has no client-level timeout so that long streaming
	// responses are not cut off; non-stream requests get a per-request context
	// timeout in HandleMessages instead.
	transport.ResponseHeaderTimeout = time.Duration(cfg.ResponseHeaderTimeoutSec) * time.Second
	return &ProxyHandler{
		rotator:      rotator,
		channelRepo:  channelRepo,
		usageRepo:    usageRepo,
		settingsRepo: settingsRepo,
		cfg:          cfg,
		client:       &http.Client{Transport: transport},
	}
}

func (ph *ProxyHandler) SetChannelRepo(cr *repo.ChannelRepo) {
	ph.channelRepo = cr
}

func (ph *ProxyHandler) HandleProxySingleChannel(w http.ResponseWriter, r *http.Request) {
	channels, err := ph.channelRepo.List()
	if err != nil || len(channels) != 1 {
		http.Error(w, `{"error":{"type":"not_found","message":"single-channel auto-route requires exactly one channel"}}`, http.StatusBadRequest)
		return
	}
	r.SetPathValue("channelID", strconv.FormatInt(channels[0].ID, 10))
	ph.HandleProxy(w, r)
}

func (ph *ProxyHandler) HandleMessagesSingleChannel(w http.ResponseWriter, r *http.Request) {
	ph.HandleProxySingleChannel(w, r)
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
	ph.HandleProxy(w, r)
}

func (ph *ProxyHandler) HandleProxy(w http.ResponseWriter, r *http.Request) {
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
	adapter := adapterForChannel(ch)
	incomingPath := normalizeIncomingPath(r.URL.Path)
	if !adapter.MatchProxyPath(incomingPath) {
		http.Error(w, fmt.Sprintf(`{"error":{"type":"not_found","message":"channel protocol %s does not support %s"}}`, adapter.Name(), incomingPath), http.StatusNotFound)
		return
	}

	// Bound request body size to protect memory under load / abuse.
	if ph.cfg.MaxRequestBodyMB > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, int64(ph.cfg.MaxRequestBodyMB)*1024*1024)
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			http.Error(w, `{"error":{"type":"invalid_request","message":"request body too large"}}`, http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, `{"error":{"type":"read_error","message":"failed to read body"}}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	meta := adapter.RequestMeta(r, body)

	// Snapshot the active keys once and rotate over the slice across attempts,
	// instead of re-querying the DB on every attempt.
	keys, err := ph.rotator.ActiveKeys(channelID)
	if err != nil || len(keys) == 0 {
		http.Error(w, `{"error":{"type":"proxy_error","message":"no_active_keys"}}`, http.StatusBadGateway)
		return
	}
	maxAttempts := ph.cfg.MaxProxyAttempts
	if len(keys) < maxAttempts {
		maxAttempts = len(keys)
	}
	start := ph.rotator.NextIndex(channelID)

	startTime := time.Now()
	clientIP := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		clientIP = strings.Split(fwd, ",")[0]
	}

	var usageLogID int64
	var lastKey *models.APIKey
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		key := &keys[int(start+int64(attempt))%len(keys)]
		lastKey = key

		if ph.usageRepo != nil && usageLogID == 0 {
			pendingLog := &models.UsageLog{
				ChannelID: ch.ID,
				APIKeyID:  key.ID,
				Model:     meta.Model,
				Endpoint:  meta.Endpoint,
				ClientIP:  clientIP,
				Status:    "pending",
				CreatedAt: time.Now().UTC(),
			}
			if err := ph.usageRepo.CreatePending(pendingLog); err != nil {
				log.Printf("usage log create pending error: channel=%d key=%d endpoint=%s err=%v", ch.ID, key.ID, meta.Endpoint, err)
			} else {
				usageLogID = pendingLog.ID
			}
		}

		if err := ph.sendUpstream(w, r, ch, adapter, key, body, meta, clientIP, startTime, usageLogID); err != nil {
			lastErr = err
			continue
		}
		return
	}

	msg := "all_keys_exhausted"
	if lastErr != nil && lastErr == ErrNoActiveKeys {
		msg = "no_active_keys"
	}
	if usageLogID > 0 {
		apiKeyID := int64(0)
		if lastKey != nil {
			apiKeyID = lastKey.ID
		}
		if err := ph.usageRepo.UpdateCompleted(usageLogID, &models.UsageLog{
			ChannelID:    ch.ID,
			APIKeyID:     apiKeyID,
			Model:        meta.Model,
			Endpoint:     meta.Endpoint,
			ClientIP:     clientIP,
			Status:       "failed",
			Success:      false,
			ErrorMessage: msg,
		}); err != nil {
			log.Printf("usage log update failed error: id=%d err=%v", usageLogID, err)
		}
	}
	http.Error(w, fmt.Sprintf(`{"error":{"type":"proxy_error","message":"%s"}}`, msg), http.StatusBadGateway)
}

// sendUpstream performs a single upstream attempt for one key. A nil return means
// the response was fully handled and written to w; a non-nil error means the
// caller should try the next key. Non-stream requests get a per-request context
// timeout (UpstreamTimeoutSec); stream requests use the client's request context
// directly so long-lived SSE streams are not cut off (header wait is bounded by
// the transport's ResponseHeaderTimeout).
func (ph *ProxyHandler) sendUpstream(w http.ResponseWriter, r *http.Request, ch *models.Channel, adapter protocolAdapter, key *models.APIKey, body []byte, meta requestMeta, clientIP string, startTime time.Time, usageLogID int64) error {
	reqCtx := r.Context()
	if !meta.Stream {
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(r.Context(), time.Duration(ph.cfg.UpstreamTimeoutSec)*time.Second)
		defer cancel()
	}

	path, err := adapter.UpstreamPath(meta)
	if err != nil {
		return err
	}
	upstreamURL := strings.TrimRight(ch.BaseURL, "/") + path
	req, err := http.NewRequestWithContext(reqCtx, "POST", upstreamURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	adapter.ApplyHeaders(req, r.Header, key.KeyValue)

	resp, err := ph.client.Do(req)
	if err != nil {
		ph.rotator.ReportFailure(key.ID)
		return err
	}

	if meta.Stream {
		return ph.handleStreamResponse(w, resp, adapter, key, ch, meta.Model, meta.Endpoint, clientIP, startTime, usageLogID)
	}
	return ph.handleNonStreamResponse(w, resp, adapter, key, ch, meta.Model, meta.Endpoint, clientIP, startTime, usageLogID)
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

	adapter := adapterForChannel(ch)
	req, err := http.NewRequestWithContext(r.Context(), "GET", strings.TrimRight(ch.BaseURL, "/")+adapter.ProbeModelsPath(), nil)
	if err != nil {
		http.Error(w, `{"error":{"type":"proxy_error","message":"internal error"}}`, http.StatusInternalServerError)
		return
	}
	adapter.ApplyHeaders(req, r.Header, key.KeyValue)

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

func (ph *ProxyHandler) handleStreamResponse(w http.ResponseWriter, resp *http.Response, adapter protocolAdapter, key *models.APIKey, ch *models.Channel, model, endpoint, clientIP string, startTime time.Time, usageLogID int64) error {
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if shouldRetry(resp.StatusCode) {
			ph.handleRetryableFailure(resp.StatusCode, key.ID)
			return fmt.Errorf("upstream %d", resp.StatusCode)
		}
		latency := time.Since(startTime).Milliseconds()
		ph.updateUsage(usageLogID, key, ch, model, endpoint, 0, 0, 0, 0,
			true, resp.StatusCode, latency, 0, 0, fmt.Sprintf("upstream %d", resp.StatusCode), clientIP)
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

	var metrics streamMetrics
	bodyReader, err := decodedBodyReader(resp)
	if err != nil {
		ph.rotator.ReportFailure(key.ID)
		return err
	}

	scanner := bufio.NewScanner(bodyReader)
	// Raise the per-line limit (default 64KB) so a large `data:` event line does
	// not trigger bufio.ErrTooLong and abort the stream mid-response.
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data != "[DONE]" {
				adapter.ObserveStreamData([]byte(data), &metrics, time.Since(startTime).Milliseconds())
			}
		}
		fmt.Fprintf(w, "%s\n", line)
		if canFlush {
			flusher.Flush()
		}
	}
	if err := scanner.Err(); err != nil {
		ph.rotator.ReportFailure(key.ID)
		return err
	}

	latency := time.Since(startTime).Milliseconds()
	if metrics.FirstTokenMs == 0 {
		metrics.FirstTokenMs = latency
	}
	outputSpeed := outputTokensPerSec(metrics.Usage.OutputTokens, latency-metrics.FirstTokenMs)
	ph.updateUsage(usageLogID, key, ch, model, endpoint, metrics.Usage.InputTokens, metrics.Usage.OutputTokens, metrics.Usage.CacheCreationTokens, metrics.Usage.CacheReadTokens,
		true, resp.StatusCode, latency, metrics.FirstTokenMs, outputSpeed, "", clientIP)
	return nil
}

func (ph *ProxyHandler) handleNonStreamResponse(w http.ResponseWriter, resp *http.Response, adapter protocolAdapter, key *models.APIKey, ch *models.Channel, model, endpoint, clientIP string, startTime time.Time, usageLogID int64) error {
	body, err := readDecodedBody(resp)
	resp.Body.Close()
	if err != nil {
		ph.rotator.ReportFailure(key.ID)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		if shouldRetry(resp.StatusCode) {
			ph.handleRetryableFailure(resp.StatusCode, key.ID)
			return fmt.Errorf("upstream %d", resp.StatusCode)
		}
		latency := time.Since(startTime).Milliseconds()
		ph.updateUsage(usageLogID, key, ch, model, endpoint, 0, 0, 0, 0,
			false, resp.StatusCode, latency, 0, 0, fmt.Sprintf("upstream %d", resp.StatusCode), clientIP)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return nil
	}

	ph.rotator.ReportSuccess(key.ID)

	usage := adapter.ParseNonStreamUsage(body)

	latency := time.Since(startTime).Milliseconds()
	outputSpeed := outputTokensPerSec(usage.OutputTokens, latency)
	ph.updateUsage(usageLogID, key, ch, model, endpoint, usage.InputTokens, usage.OutputTokens,
		usage.CacheCreationTokens, usage.CacheReadTokens, false, resp.StatusCode, latency, latency, outputSpeed, "", clientIP)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
	return nil
}

func (ph *ProxyHandler) updateUsage(usageLogID int64, key *models.APIKey, ch *models.Channel, model, endpoint string, input, output, cacheCreate, cacheRead int64, isStream bool, statusCode int, latencyMs, firstTokenMs int64, outputTokensPerSec float64, errMsg, clientIP string) {
	if ph.usageRepo == nil || usageLogID <= 0 {
		return
	}
	status := "success"
	if errMsg != "" {
		status = "failed"
	}
	if err := ph.usageRepo.UpdateCompleted(usageLogID, &models.UsageLog{
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
		FirstTokenMs:        firstTokenMs,
		OutputTokensPerSec:  outputTokensPerSec,
		Success:             errMsg == "",
		ErrorMessage:        errMsg,
		Status:              status,
	}); err != nil {
		log.Printf("usage log update completed error: id=%d key=%d endpoint=%s err=%v", usageLogID, key.ID, endpoint, err)
	}
}

func (ph *ProxyHandler) handleRetryableFailure(statusCode int, keyID int64) {
	if statusCode == http.StatusUnauthorized && ph.autoDeleteUnauthorizedKeys() {
		ph.rotator.DeleteKey(keyID)
		return
	}
	ph.rotator.ReportFailure(keyID)
}

func (ph *ProxyHandler) autoDeleteUnauthorizedKeys() bool {
	if ph.settingsRepo == nil {
		return false
	}
	value, err := ph.settingsRepo.Get("auto_delete_401_keys_enabled")
	return err == nil && value == "true"
}

func shouldRetry(statusCode int) bool {
	return statusCode == 401 || statusCode == 403 || statusCode == 429 || statusCode >= 500
}

func outputTokensPerSec(outputTokens, durationMs int64) float64 {
	if outputTokens <= 0 || durationMs <= 0 {
		return 0
	}
	return float64(outputTokens) / (float64(durationMs) / 1000)
}

func anthropicEventHasTextDelta(event map[string]json.RawMessage) bool {
	if delta, ok := event["delta"]; ok {
		var d struct {
			Text string `json:"text"`
		}
		if json.Unmarshal(delta, &d) == nil && strings.TrimSpace(d.Text) != "" {
			return true
		}
	}
	if block, ok := event["content_block"]; ok {
		var b struct {
			Text string `json:"text"`
		}
		if json.Unmarshal(block, &b) == nil && strings.TrimSpace(b.Text) != "" {
			return true
		}
	}
	return false
}

type parsedUsage struct {
	InputTokens         int64
	OutputTokens        int64
	CacheCreationTokens int64
	CacheReadTokens     int64
}

type anthropicUsage struct {
	InputTokens               int64              `json:"input_tokens"`
	OutputTokens              int64              `json:"output_tokens"`
	CacheCreationInputTokens  int64              `json:"cache_creation_input_tokens"`
	CacheReadInputTokens      int64              `json:"cache_read_input_tokens"`
	CacheCreation             cacheCreationUsage `json:"cache_creation"`
	CacheCreationInputTokens2 int64              `json:"cache_write_input_tokens"`
	CacheReadInputTokens2     int64              `json:"cache_read_tokens"`
}

type cacheCreationUsage struct {
	Ephemeral5mInputTokens int64 `json:"ephemeral_5m_input_tokens"`
	Ephemeral1hInputTokens int64 `json:"ephemeral_1h_input_tokens"`
}

func readDecodedBody(resp *http.Response) ([]byte, error) {
	reader, err := decodedBodyReader(resp)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(reader)
}

func decodedBodyReader(resp *http.Response) (io.Reader, error) {
	switch strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Encoding"))) {
	case "", "identity":
		return resp.Body, nil
	case "gzip":
		return gzip.NewReader(resp.Body)
	case "deflate":
		return zlib.NewReader(resp.Body)
	default:
		return nil, fmt.Errorf("unsupported content encoding: %s", resp.Header.Get("Content-Encoding"))
	}
}

func anthropicUsageParser(raw json.RawMessage) parsedUsage {
	var u anthropicUsage
	if json.Unmarshal(raw, &u) != nil {
		return parsedUsage{}
	}
	cacheCreation := u.CacheCreationInputTokens
	if cacheCreation == 0 {
		cacheCreation = u.CacheCreationInputTokens2
	}
	cacheCreation += u.CacheCreation.Ephemeral5mInputTokens + u.CacheCreation.Ephemeral1hInputTokens

	cacheRead := u.CacheReadInputTokens
	if cacheRead == 0 {
		cacheRead = u.CacheReadInputTokens2
	}

	return parsedUsage{
		InputTokens:         u.InputTokens,
		OutputTokens:        u.OutputTokens,
		CacheCreationTokens: cacheCreation,
		CacheReadTokens:     cacheRead,
	}
}

func mergeUsage(target *parsedUsage, next parsedUsage) {
	if next.InputTokens != 0 {
		target.InputTokens = next.InputTokens
	}
	if next.OutputTokens != 0 {
		target.OutputTokens = next.OutputTokens
	}
	if next.CacheCreationTokens != 0 {
		target.CacheCreationTokens = next.CacheCreationTokens
	}
	if next.CacheReadTokens != 0 {
		target.CacheReadTokens = next.CacheReadTokens
	}
}

func extractTokensFromMessageStart(event map[string]json.RawMessage, input, cacheCreate, cacheRead *int64) {
	if msg, ok := event["message"]; ok {
		var m struct {
			Usage json.RawMessage `json:"usage"`
		}
		if json.Unmarshal(msg, &m) == nil && len(m.Usage) > 0 {
			usage := anthropicUsageParser(m.Usage)
			if usage.InputTokens != 0 {
				*input = usage.InputTokens
			}
			if usage.CacheCreationTokens != 0 {
				*cacheCreate = usage.CacheCreationTokens
			}
			if usage.CacheReadTokens != 0 {
				*cacheRead = usage.CacheReadTokens
			}
		}
	}
}

func extractTokensFromMessageDelta(event map[string]json.RawMessage, parsed *parsedUsage) {
	if usage, ok := event["usage"]; ok {
		mergeUsage(parsed, anthropicUsageParser(usage))
	}
}
