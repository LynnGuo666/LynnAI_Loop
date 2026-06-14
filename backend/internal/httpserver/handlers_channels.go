package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"loop/internal/models"
	"loop/internal/repo"
)

type Handlers struct {
	channelRepo  *repo.ChannelRepo
	keyRepo      *repo.APIKeyRepo
	usageRepo    *repo.UsageRepo
	probeRepo    *repo.KeyProbeRepo
	settingsRepo *repo.SettingsRepo
}

func NewHandlers(cr *repo.ChannelRepo, kr *repo.APIKeyRepo, ur *repo.UsageRepo, pr *repo.KeyProbeRepo, sr *repo.SettingsRepo) *Handlers {
	return &Handlers{
		channelRepo:  cr,
		keyRepo:      kr,
		usageRepo:    ur,
		probeRepo:    pr,
		settingsRepo: sr,
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]interface{}{
		"error": map[string]string{"type": "error", "message": msg},
	})
}

// Channels

func (h *Handlers) ListChannels(w http.ResponseWriter, r *http.Request) {
	channels, err := h.channelRepo.List()
	if err != nil {
		writeError(w, 500, "failed to list channels")
		return
	}
	writeJSON(w, 200, channels)
}

func (h *Handlers) CreateChannel(w http.ResponseWriter, r *http.Request) {
	var ch models.Channel
	if err := json.NewDecoder(r.Body).Decode(&ch); err != nil {
		writeError(w, 400, "invalid request body")
		return
	}
	ch.Name = strings.TrimSpace(ch.Name)
	ch.BaseURL = strings.TrimSpace(ch.BaseURL)
	ch.Protocol = strings.TrimSpace(ch.Protocol)
	ch.ProbeModel = strings.TrimSpace(ch.ProbeModel)
	ch.IsActive = true
	if ch.Protocol == "" {
		ch.Protocol = models.ProtocolAnthropicMessages
	}
	if !validChannelProtocol(ch.Protocol) {
		writeError(w, 400, "invalid protocol")
		return
	}
	if ch.Name == "" || ch.BaseURL == "" {
		writeError(w, 400, "name and base_url are required")
		return
	}
	if err := h.channelRepo.Create(&ch); err != nil {
		writeError(w, 500, "failed to create channel: "+err.Error())
		return
	}
	writeJSON(w, 201, ch)
}

func (h *Handlers) GetChannel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, 400, "invalid id")
		return
	}
	ch, err := h.channelRepo.GetByID(id)
	if err != nil {
		writeError(w, 404, "channel not found")
		return
	}
	writeJSON(w, 200, ch)
}

func (h *Handlers) UpdateChannel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, 400, "invalid id")
		return
	}
	ch, err := h.channelRepo.GetByID(id)
	if err != nil {
		writeError(w, 404, "channel not found")
		return
	}
	var input struct {
		Name        *string `json:"name"`
		BaseURL     *string `json:"base_url"`
		Protocol    *string `json:"protocol"`
		Description *string `json:"description"`
		ProbeModel  *string `json:"probe_model"`
		IsActive    *bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, 400, "invalid request body")
		return
	}
	if input.Name != nil {
		ch.Name = strings.TrimSpace(*input.Name)
	}
	if input.BaseURL != nil {
		ch.BaseURL = strings.TrimSpace(*input.BaseURL)
	}
	if input.Protocol != nil {
		ch.Protocol = strings.TrimSpace(*input.Protocol)
		if ch.Protocol == "" {
			ch.Protocol = models.ProtocolAnthropicMessages
		}
		if !validChannelProtocol(ch.Protocol) {
			writeError(w, 400, "invalid protocol")
			return
		}
	}
	if input.Description != nil {
		ch.Description = *input.Description
	}
	if input.ProbeModel != nil {
		ch.ProbeModel = strings.TrimSpace(*input.ProbeModel)
	}
	if input.IsActive != nil {
		ch.IsActive = *input.IsActive
	}
	if err := h.channelRepo.Update(ch); err != nil {
		writeError(w, 500, "failed to update channel")
		return
	}
	writeJSON(w, 200, ch)
}

func (h *Handlers) DeleteChannel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, 400, "invalid id")
		return
	}
	if err := h.channelRepo.Delete(id); err != nil {
		writeError(w, 500, "failed to delete channel")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func validChannelProtocol(protocol string) bool {
	switch protocol {
	case models.ProtocolAnthropicMessages,
		models.ProtocolOpenAIChatCompletions,
		models.ProtocolOpenAIResponses,
		models.ProtocolGeminiGenerateContent:
		return true
	default:
		return false
	}
}
