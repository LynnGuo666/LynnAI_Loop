package services

import (
	"encoding/json"
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

	usage := extractUsageFromBody(body)

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

	usage := parseUsage(raw)

	if usage.CacheCreationTokens != 70 {
		t.Fatalf("CacheCreationTokens = %d, want 70", usage.CacheCreationTokens)
	}
	if usage.CacheReadTokens != 50 {
		t.Fatalf("CacheReadTokens = %d, want 50", usage.CacheReadTokens)
	}
}

func TestExtractTokensFromStreamingEvents(t *testing.T) {
	var input, output, cacheCreate, cacheRead int64
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

	extractTokensFromMessageStart(startEvent, &input, &cacheCreate, &cacheRead)
	extractTokensFromMessageDelta(deltaEvent, &input, &output, &cacheCreate, &cacheRead)

	if input != 11 {
		t.Fatalf("input = %d, want 11", input)
	}
	if output != 44 {
		t.Fatalf("output = %d, want 44", output)
	}
	if cacheCreate != 22 {
		t.Fatalf("cacheCreate = %d, want 22", cacheCreate)
	}
	if cacheRead != 33 {
		t.Fatalf("cacheRead = %d, want 33", cacheRead)
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

	if !eventHasTextDelta(textEvent) {
		t.Fatalf("eventHasTextDelta(textEvent) = false, want true")
	}
	if eventHasTextDelta(emptyEvent) {
		t.Fatalf("eventHasTextDelta(emptyEvent) = true, want false")
	}
}
