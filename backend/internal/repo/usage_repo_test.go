package repo

import (
	"path/filepath"
	"testing"
	"time"

	"loop/internal/db"
	"loop/internal/models"
)

func openTestUsageRepo(t *testing.T) *UsageRepo {
	t.Helper()

	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	if _, err := database.Exec(
		`INSERT INTO channels (id, name, base_url, description, probe_model, is_active)
		 VALUES (1, 'test', 'https://example.com', '', '', 1)`,
	); err != nil {
		t.Fatalf("insert channel: %v", err)
	}
	if _, err := database.Exec(
		`INSERT INTO api_keys (id, channel_id, key_value, alias, is_active, probe_backoff_min)
		 VALUES (1, 1, 'sk-test', 'test', 1, 60)`,
	); err != nil {
		t.Fatalf("insert key: %v", err)
	}

	return NewUsageRepo(database)
}

func TestUsageRepoCreateListAndStatsIncludePerformanceMetrics(t *testing.T) {
	usageRepo := openTestUsageRepo(t)
	now := time.Now()

	if err := usageRepo.Create(&models.UsageLog{
		ChannelID:           1,
		APIKeyID:            1,
		Model:               "claude-test",
		Endpoint:            "/v1/messages",
		InputTokens:         10,
		OutputTokens:        30,
		CacheCreationTokens: 2,
		CacheReadTokens:     3,
		IsStream:            true,
		StatusCode:          200,
		LatencyMs:           2000,
		FirstTokenMs:        500,
		OutputTokensPerSec:  20,
		Success:             true,
		CreatedAt:           now,
	}); err != nil {
		t.Fatalf("create usage: %v", err)
	}
	if err := usageRepo.Create(&models.UsageLog{
		ChannelID:          1,
		APIKeyID:           1,
		Model:              "claude-test",
		Endpoint:           "/v1/messages",
		InputTokens:        5,
		OutputTokens:       0,
		IsStream:           false,
		StatusCode:         200,
		LatencyMs:          100,
		FirstTokenMs:       0,
		OutputTokensPerSec: 0,
		Success:            true,
		CreatedAt:          now,
	}); err != nil {
		t.Fatalf("create zero metric usage: %v", err)
	}

	logs, total, err := usageRepo.List(UsageFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list usage: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	if logs[1].FirstTokenMs != 500 {
		t.Fatalf("FirstTokenMs = %d, want 500", logs[1].FirstTokenMs)
	}
	if logs[1].OutputTokensPerSec != 20 {
		t.Fatalf("OutputTokensPerSec = %f, want 20", logs[1].OutputTokensPerSec)
	}

	stats, err := usageRepo.Stats("", "")
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.AvgFirstTokenMs != 500 {
		t.Fatalf("AvgFirstTokenMs = %f, want 500", stats.AvgFirstTokenMs)
	}
	if stats.AvgOutputTokensPerSec != 20 {
		t.Fatalf("AvgOutputTokensPerSec = %f, want 20", stats.AvgOutputTokensPerSec)
	}
}
