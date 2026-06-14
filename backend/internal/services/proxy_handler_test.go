package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractUsageFromBody(t *testing.T) {
	body := []byte(`{
		"id": "msg_1",
		"usage": {
			"input_tokens": 12,
			"output_tokens": 34,
			"cache_creation_input_tokens": 56,
			"cache_read_input_tokens": 78
		}
	}`)

	usage := extractUsageFromBody(body, anthropicUsageParser)

	if usage.InputTokens != 12 {
		t.Fatalf("InputTokens = %d, want 12", usage.InputTokens)
	}
	if usage.OutputTokens != 34 {
		t.Fatalf("OutputTokens = %d, want 34", usage.OutputTokens)
	}
	if usage.CacheCreationTokens != 56 {
		t.Fatalf("CacheCreationTokens = %d, want 56", usage.CacheCreationTokens)
	}
	if usage.CacheReadTokens != 78 {
		t.Fatalf("CacheReadTokens = %d, want 78", usage.CacheReadTokens)
	}
}

func TestParseUsageWithNestedCacheCreation(t *testing.T) {
	raw := json.RawMessage(`{
		"input_tokens": 10,
		"output_tokens": 20,
		"cache_creation": {
			"ephemeral_5m_input_tokens": 30,
			"ephemeral_1h_input_tokens": 40
		},
		"cache_read_input_tokens": 50
	}`)

	usage := anthropicUsageParser(raw)

	if usage.CacheCreationTokens != 70 {
		t.Fatalf("CacheCreationTokens = %d, want 70", usage.CacheCreationTokens)
	}
	if usage.CacheReadTokens != 50 {
		t.Fatalf("CacheReadTokens = %d, want 50", usage.CacheReadTokens)
	}
}

func TestExtractTokensFromStreamingEvents(t *testing.T) {
	var metrics streamMetrics
	startEvent := map[string]json.RawMessage{
		"message": json.RawMessage(`{
			"usage": {
				"input_tokens": 11,
				"cache_creation_input_tokens": 22,
				"cache_read_input_tokens": 33
			}
		}`),
	}
	deltaEvent := map[string]json.RawMessage{
		"usage": json.RawMessage(`{"output_tokens": 44}`),
	}

	extractTokensFromMessageStart(startEvent, &metrics.Usage.InputTokens, &metrics.Usage.CacheCreationTokens, &metrics.Usage.CacheReadTokens)
	extractTokensFromMessageDelta(deltaEvent, &metrics.Usage)

	if metrics.Usage.InputTokens != 11 {
		t.Fatalf("input = %d, want 11", metrics.Usage.InputTokens)
	}
	if metrics.Usage.OutputTokens != 44 {
		t.Fatalf("output = %d, want 44", metrics.Usage.OutputTokens)
	}
	if metrics.Usage.CacheCreationTokens != 22 {
		t.Fatalf("cacheCreate = %d, want 22", metrics.Usage.CacheCreationTokens)
	}
	if metrics.Usage.CacheReadTokens != 33 {
		t.Fatalf("cacheRead = %d, want 33", metrics.Usage.CacheReadTokens)
	}
}

func TestOutputTokensPerSec(t *testing.T) {
	tests := []struct {
		name       string
		tokens     int64
		durationMs int64
		want       float64
	}{
		{name: "normal", tokens: 30, durationMs: 1500, want: 20},
		{name: "zero tokens", tokens: 0, durationMs: 1500, want: 0},
		{name: "zero duration", tokens: 30, durationMs: 0, want: 0},
		{name: "negative duration", tokens: 30, durationMs: -1, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := outputTokensPerSec(tt.tokens, tt.durationMs); got != tt.want {
				t.Fatalf("outputTokensPerSec() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestEventHasTextDelta(t *testing.T) {
	textEvent := map[string]json.RawMessage{
		"type":  json.RawMessage(`"content_block_delta"`),
		"delta": json.RawMessage(`{"type":"text_delta","text":"hi"}`),
	}
	emptyEvent := map[string]json.RawMessage{
		"type":  json.RawMessage(`"message_delta"`),
		"usage": json.RawMessage(`{"output_tokens": 12}`),
	}

	if !anthropicEventHasTextDelta(textEvent) {
		t.Fatalf("eventHasTextDelta(textEvent) = false, want true")
	}
	if anthropicEventHasTextDelta(emptyEvent) {
		t.Fatalf("eventHasTextDelta(emptyEvent) = true, want false")
	}
}

func TestOpenAIUsageParser(t *testing.T) {
	usage := openAIUsageParser(json.RawMessage(`{
		"prompt_tokens": 15,
		"completion_tokens": 25
	}`))

	if usage.InputTokens != 15 {
		t.Fatalf("InputTokens = %d, want 15", usage.InputTokens)
	}
	if usage.OutputTokens != 25 {
		t.Fatalf("OutputTokens = %d, want 25", usage.OutputTokens)
	}
}

func TestGeminiUsageFromBody(t *testing.T) {
	usage := geminiUsageFromBody([]byte(`{
		"usageMetadata": {
			"promptTokenCount": 7,
			"candidatesTokenCount": 9
		}
	}`))

	if usage.InputTokens != 7 {
		t.Fatalf("InputTokens = %d, want 7", usage.InputTokens)
	}
	if usage.OutputTokens != 9 {
		t.Fatalf("OutputTokens = %d, want 9", usage.OutputTokens)
	}
}

func TestOpenAIChatAdapterHeadersAndMeta(t *testing.T) {
	adapter := openAIChatAdapter{}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	upstreamReq := httptest.NewRequest(http.MethodPost, "https://example.com/v1/chat/completions", nil)

	meta := adapter.RequestMeta(req, []byte(`{"model":"gpt-test","stream":true}`))
	adapter.ApplyHeaders(upstreamReq, nil, "sk-test")

	if meta.Model != "gpt-test" {
		t.Fatalf("Model = %q, want gpt-test", meta.Model)
	}
	if !meta.Stream {
		t.Fatalf("Stream = false, want true")
	}
	if upstreamReq.Header.Get("Authorization") != "Bearer sk-test" {
		t.Fatalf("Authorization = %q", upstreamReq.Header.Get("Authorization"))
	}
}

func TestGeminiAdapterPathHeadersAndModels(t *testing.T) {
	adapter := geminiAdapter{}
	req := httptest.NewRequest(http.MethodPost, "/channel/7/v1beta/models/gemini-2.5-flash:streamGenerateContent", nil)
	upstreamReq := httptest.NewRequest(http.MethodPost, "https://example.com/v1beta/models/gemini-2.5-flash:streamGenerateContent", nil)

	meta := adapter.RequestMeta(req, []byte(`{}`))
	adapter.ApplyHeaders(upstreamReq, nil, "gemini-key")
	models, err := adapter.ParseModels([]byte(`{"models":[{"name":"models/gemini-a"},{"name":"models/gemini-b"}]}`))
	if err != nil {
		t.Fatalf("ParseModels error: %v", err)
	}

	if meta.Model != "gemini-2.5-flash" {
		t.Fatalf("Model = %q, want gemini-2.5-flash", meta.Model)
	}
	if !meta.Stream {
		t.Fatalf("Stream = false, want true")
	}
	if upstreamReq.Header.Get("x-goog-api-key") != "gemini-key" {
		t.Fatalf("x-goog-api-key = %q", upstreamReq.Header.Get("x-goog-api-key"))
	}
	if upstreamReq.URL.Query().Get("alt") != "sse" {
		t.Fatalf("alt = %q, want sse", upstreamReq.URL.Query().Get("alt"))
	}
	if len(models) != 2 || models[0] != "gemini-a" || models[1] != "gemini-b" {
		t.Fatalf("models = %#v", models)
	}
}
