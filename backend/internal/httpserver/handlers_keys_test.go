package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"loop/internal/db"
	"loop/internal/models"
	"loop/internal/repo"
)

func openTestHandlers(t *testing.T) *Handlers {
	t.Helper()

	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"), 4)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	channelRepo := repo.NewChannelRepo(database)
	if err := channelRepo.Create(&models.Channel{
		Name:     "test",
		BaseURL:  "https://example.com",
		IsActive: true,
	}); err != nil {
		t.Fatalf("create channel: %v", err)
	}

	return NewHandlers(channelRepo, repo.NewAPIKeyRepo(database), nil, nil, nil)
}

func TestImportAndExportKeys(t *testing.T) {
	handlers := openTestHandlers(t)
	body := bytes.NewBufferString(`{
		"channel_id": 1,
		"keys": [
			{"key_value": "sk-one", "alias": "one"},
			{"key_value": "sk-one", "alias": "duplicate"},
			{"key_value": "sk-two", "alias": "two", "is_active": false}
		]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/keys/import", body)
	rr := httptest.NewRecorder()

	handlers.ImportKeys(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var imported keyImportResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &imported); err != nil {
		t.Fatalf("decode import response: %v", err)
	}
	if imported.Created != 2 {
		t.Fatalf("Created = %d, want 2", imported.Created)
	}
	if imported.Skipped != 1 {
		t.Fatalf("Skipped = %d, want 1", imported.Skipped)
	}
	if imported.Failed != 0 {
		t.Fatalf("Failed = %d, want 0", imported.Failed)
	}

	exportReq := httptest.NewRequest(http.MethodGet, "/api/keys/export?channel_id=1", nil)
	exportRR := httptest.NewRecorder()
	handlers.ExportKeys(exportRR, exportReq)

	if exportRR.Code != http.StatusOK {
		t.Fatalf("export status = %d, body = %s", exportRR.Code, exportRR.Body.String())
	}
	var exported struct {
		Data  []keyImportItem `json:"data"`
		Count int             `json:"count"`
	}
	if err := json.Unmarshal(exportRR.Body.Bytes(), &exported); err != nil {
		t.Fatalf("decode export response: %v", err)
	}
	if exported.Count != 2 {
		t.Fatalf("Count = %d, want 2", exported.Count)
	}
	if len(exported.Data) != 2 {
		t.Fatalf("len(Data) = %d, want 2", len(exported.Data))
	}
}
