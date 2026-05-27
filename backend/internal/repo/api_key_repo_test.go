package repo

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"loop/internal/db"
	"loop/internal/models"
)

func openTestKeyRepo(t *testing.T) *APIKeyRepo {
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

	return NewAPIKeyRepo(database)
}

func TestAPIKeyRepoRecordFailureIsAtomic(t *testing.T) {
	keyRepo := openTestKeyRepo(t)
	key := &models.APIKey{
		ChannelID:       1,
		KeyValue:        "sk-test",
		Alias:           "test",
		IsActive:        true,
		ProbeBackoffMin: 60,
	}
	if err := keyRepo.Create(key); err != nil {
		t.Fatalf("create key: %v", err)
	}

	const failures = 20
	var wg sync.WaitGroup
	for i := 0; i < failures; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := keyRepo.RecordFailure(key.ID, 5); err != nil {
				t.Errorf("record failure: %v", err)
			}
		}()
	}
	wg.Wait()

	got, err := keyRepo.GetByID(key.ID)
	if err != nil {
		t.Fatalf("get key: %v", err)
	}
	if got.TotalFailures != failures {
		t.Fatalf("TotalFailures = %d, want %d", got.TotalFailures, failures)
	}
	if got.ConsecutiveFailures != failures {
		t.Fatalf("ConsecutiveFailures = %d, want %d", got.ConsecutiveFailures, failures)
	}
	if got.IsActive {
		t.Fatalf("IsActive = true, want false")
	}
	if got.DisabledAt == nil {
		t.Fatalf("DisabledAt is nil")
	}
	if got.NextProbeAt == nil {
		t.Fatalf("NextProbeAt is nil")
	}
}

func TestAPIKeyRepoRecordSuccessClearsFailures(t *testing.T) {
	keyRepo := openTestKeyRepo(t)
	key := &models.APIKey{
		ChannelID:           1,
		KeyValue:            "sk-test",
		Alias:               "test",
		IsActive:            true,
		ConsecutiveFailures: 3,
		TotalFailures:       7,
		ProbeBackoffMin:     60,
	}
	if err := keyRepo.Create(key); err != nil {
		t.Fatalf("create key: %v", err)
	}
	key.ConsecutiveFailures = 3
	key.TotalFailures = 7
	if err := keyRepo.Update(key); err != nil {
		t.Fatalf("seed failures: %v", err)
	}

	if err := keyRepo.RecordSuccess(key.ID); err != nil {
		t.Fatalf("record success: %v", err)
	}

	got, err := keyRepo.GetByID(key.ID)
	if err != nil {
		t.Fatalf("get key: %v", err)
	}
	if got.TotalSuccesses != 1 {
		t.Fatalf("TotalSuccesses = %d, want 1", got.TotalSuccesses)
	}
	if got.ConsecutiveFailures != 0 {
		t.Fatalf("ConsecutiveFailures = %d, want 0", got.ConsecutiveFailures)
	}
	if got.LastUsedAt == nil {
		t.Fatalf("LastUsedAt is nil")
	}
}

func TestAPIKeyRepoRejectsDuplicateKeyOnCreateAndUpdate(t *testing.T) {
	keyRepo := openTestKeyRepo(t)
	first := &models.APIKey{
		ChannelID:       1,
		KeyValue:        "sk-duplicate",
		Alias:           "first",
		IsActive:        true,
		ProbeBackoffMin: 60,
	}
	if err := keyRepo.Create(first); err != nil {
		t.Fatalf("create first key: %v", err)
	}

	duplicate := &models.APIKey{
		ChannelID:       1,
		KeyValue:        " sk-duplicate ",
		Alias:           "duplicate",
		IsActive:        true,
		ProbeBackoffMin: 60,
	}
	if err := keyRepo.Create(duplicate); !errors.Is(err, ErrDuplicateAPIKey) {
		t.Fatalf("Create duplicate error = %v, want ErrDuplicateAPIKey", err)
	}

	second := &models.APIKey{
		ChannelID:       1,
		KeyValue:        "sk-second",
		Alias:           "second",
		IsActive:        true,
		ProbeBackoffMin: 60,
	}
	if err := keyRepo.Create(second); err != nil {
		t.Fatalf("create second key: %v", err)
	}
	second.KeyValue = "sk-duplicate"
	if err := keyRepo.Update(second); !errors.Is(err, ErrDuplicateAPIKey) {
		t.Fatalf("Update duplicate error = %v, want ErrDuplicateAPIKey", err)
	}
}
