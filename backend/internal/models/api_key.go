package models

import "time"

type APIKey struct {
	ID                  int64      `json:"id"`
	ChannelID           int64      `json:"channel_id"`
	KeyValue            string     `json:"key_value"`
	Alias               string     `json:"alias"`
	IsActive            bool       `json:"is_active"`
	ConsecutiveFailures int        `json:"consecutive_failures"`
	TotalFailures       int64      `json:"total_failures"`
	TotalSuccesses      int64      `json:"total_successes"`
	LastUsedAt          *time.Time `json:"last_used_at"`
	LastFailureAt       *time.Time `json:"last_failure_at"`
	DisabledAt          *time.Time `json:"disabled_at"`
	NextProbeAt         *time.Time `json:"next_probe_at"`
	ProbeBackoffMin     int        `json:"probe_backoff_min"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}
