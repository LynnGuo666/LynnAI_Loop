package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"loop/internal/models"
)

func (h *Handlers) ListKeys(w http.ResponseWriter, r *http.Request) {
	channelID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, 400, "invalid channel id")
		return
	}
	keys, err := h.keyRepo.ListByChannel(channelID)
	if err != nil {
		writeError(w, 500, "failed to list keys")
		return
	}
	writeJSON(w, 200, keys)
}

func (h *Handlers) ListAllKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := h.keyRepo.ListAll()
	if err != nil {
		writeError(w, 500, "failed to list keys")
		return
	}
	writeJSON(w, 200, keys)
}

func (h *Handlers) CreateKey(w http.ResponseWriter, r *http.Request) {
	channelID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, 400, "invalid channel id")
		return
	}
	var k models.APIKey
	if err := json.NewDecoder(r.Body).Decode(&k); err != nil {
		writeError(w, 400, "invalid request body")
		return
	}
	if k.KeyValue == "" {
		writeError(w, 400, "key_value is required")
		return
	}
	k.ChannelID = channelID
	k.IsActive = true
	k.ProbeBackoffMin = 60
	if err := h.keyRepo.Create(&k); err != nil {
		writeError(w, 500, "failed to create key: "+err.Error())
		return
	}
	writeJSON(w, 201, k)
}

func (h *Handlers) GetKey(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, 400, "invalid id")
		return
	}
	k, err := h.keyRepo.GetByID(id)
	if err != nil {
		writeError(w, 404, "key not found")
		return
	}
	writeJSON(w, 200, k)
}

func (h *Handlers) UpdateKey(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, 400, "invalid id")
		return
	}
	k, err := h.keyRepo.GetByID(id)
	if err != nil {
		writeError(w, 404, "key not found")
		return
	}
	var input models.APIKey
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, 400, "invalid request body")
		return
	}
	if input.Alias != "" {
		k.Alias = input.Alias
	}
	if input.KeyValue != "" {
		k.KeyValue = input.KeyValue
	}
	if err := h.keyRepo.Update(k); err != nil {
		writeError(w, 500, "failed to update key")
		return
	}
	writeJSON(w, 200, k)
}

func (h *Handlers) DeleteKey(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, 400, "invalid id")
		return
	}
	if err := h.keyRepo.Delete(id); err != nil {
		writeError(w, 500, "failed to delete key")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func (h *Handlers) EnableKey(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, 400, "invalid id")
		return
	}
	k, err := h.keyRepo.GetByID(id)
	if err != nil {
		writeError(w, 404, "key not found")
		return
	}
	k.IsActive = true
	k.ConsecutiveFailures = 0
	k.DisabledAt = nil
	k.NextProbeAt = nil
	if err := h.keyRepo.Update(k); err != nil {
		writeError(w, 500, "failed to enable key")
		return
	}
	writeJSON(w, 200, k)
}
