package models

import "time"

type UsageLog struct {
	ID                  int64     `json:"id"`
	ChannelID           int64     `json:"channel_id"`
	APIKeyID            int64     `json:"api_key_id"`
	Model               string    `json:"model"`
	Endpoint            string    `json:"endpoint"`
	InputTokens         int64     `json:"input_tokens"`
	OutputTokens        int64     `json:"output_tokens"`
	CacheCreationTokens int64     `json:"cache_creation_tokens"`
	CacheReadTokens     int64     `json:"cache_read_tokens"`
	IsStream            bool      `json:"is_stream"`
	StatusCode          int       `json:"status_code"`
	LatencyMs           int64     `json:"latency_ms"`
	Success             bool      `json:"success"`
	ErrorMessage        string    `json:"error_message"`
	ClientIP            string    `json:"client_ip"`
	CreatedAt           time.Time `json:"created_at"`
}
