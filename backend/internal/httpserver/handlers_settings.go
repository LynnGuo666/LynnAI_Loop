package httpserver

import (
	"encoding/json"
	"net/http"

	"loop/internal/config"
	"loop/internal/version"
)

type SettingsHandlers struct {
	settingsRepo interface {
		Get(string) (string, error)
		Set(string, string) error
		GetAll() (map[string]string, error)
	}
	cfg config.Config
}

func NewSettingsHandlers(sr interface {
	Get(string) (string, error)
	Set(string, string) error
	GetAll() (map[string]string, error)
}, cfg config.Config) *SettingsHandlers {
	return &SettingsHandlers{settingsRepo: sr, cfg: cfg}
}

func (sh *SettingsHandlers) GetSettings(w http.ResponseWriter, r *http.Request) {
	all, err := sh.settingsRepo.GetAll()
	if err != nil {
		writeError(w, 500, "failed to get settings")
		return
	}
	writeJSON(w, 200, all)
}

func (sh *SettingsHandlers) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var input map[string]string
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, 400, "invalid request body")
		return
	}
	allowed := map[string]bool{
		"recovery_probe_enabled":       true,
		"auto_disable_threshold":       true,
		"auto_delete_401_keys_enabled": true,
		"admin_token":                  true,
	}
	for k, v := range input {
		if !allowed[k] {
			continue
		}
		if err := sh.settingsRepo.Set(k, v); err != nil {
			writeError(w, 500, "failed to update setting: "+k)
			return
		}
	}
	all, _ := sh.settingsRepo.GetAll()
	writeJSON(w, 200, all)
}

func (sh *SettingsHandlers) Healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok", "version": version.Version})
}
