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
