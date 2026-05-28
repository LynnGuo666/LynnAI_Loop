package httpserver

import (
	"net/http"
	"strconv"

	"loop/internal/repo"
)

func (h *Handlers) ListUsage(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := repo.UsageFilter{
		StartDate: q.Get("start_date"),
		EndDate:   q.Get("end_date"),
		Model:     q.Get("model"),
		Status:    q.Get("status"),
	}
	if v := q.Get("channel_id"); v != "" {
		f.ChannelID, _ = strconv.ParseInt(v, 10, 64)
	}
	if v := q.Get("api_key_id"); v != "" {
		f.APIKeyID, _ = strconv.ParseInt(v, 10, 64)
	}
	if v := q.Get("success"); v != "" {
		b := v == "true" || v == "1"
		f.Success = &b
	}
	f.Page, _ = strconv.Atoi(q.Get("page"))
	f.PageSize, _ = strconv.Atoi(q.Get("page_size"))

	logs, total, err := h.usageRepo.List(f)
	if err != nil {
		writeError(w, 500, "failed to list usage: "+err.Error())
		return
	}
	writeJSON(w, 200, map[string]interface{}{
		"data":  logs,
		"total": total,
		"page":  f.Page,
	})
}

func (h *Handlers) UsageStats(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	stats, err := h.usageRepo.Stats(q.Get("start_date"), q.Get("end_date"))
	if err != nil {
		writeError(w, 500, "failed to get stats")
		return
	}
	writeJSON(w, 200, stats)
}

func (h *Handlers) UsageTimeseries(w http.ResponseWriter, r *http.Request) {
	days := 7
	if v := r.URL.Query().Get("days"); v != "" {
		if d, err := strconv.Atoi(v); err == nil && d > 0 {
			days = d
		}
	}
	points, err := h.usageRepo.Timeseries(days)
	if err != nil {
		writeError(w, 500, "failed to get timeseries")
		return
	}
	writeJSON(w, 200, points)
}

func (h *Handlers) UsageModels(w http.ResponseWriter, r *http.Request) {
	models, err := h.usageRepo.DistinctModels()
	if err != nil {
		writeError(w, 500, "failed to get models")
		return
	}
	writeJSON(w, 200, models)
}
