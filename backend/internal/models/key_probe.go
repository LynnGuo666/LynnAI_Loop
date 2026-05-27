package models

import "time"

type KeyProbe struct {
	ID         int64     `json:"id"`
	APIKeyID   int64     `json:"api_key_id"`
	Success    bool      `json:"success"`
	LatencyMs  int64     `json:"latency_ms"`
	StatusCode int       `json:"status_code"`
	ErrorMsg   string    `json:"error_msg"`
	CreatedAt  time.Time `json:"created_at"`
}
