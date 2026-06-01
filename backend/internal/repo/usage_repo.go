package repo

import (
	"fmt"
	"loop/internal/db"
	"loop/internal/models"
	"strings"
	"time"
)

type UsageRepo struct {
	db *db.DB
}

func NewUsageRepo(db *db.DB) *UsageRepo {
	return &UsageRepo{db: db}
}

// fmtTime formats a time.Time as a SQLite-compatible datetime string in UTC.
// Go's modernc sqlite driver stores time.Time using time.String() which includes
// timezone names and monotonic clock readings that SQLite's date/time functions
// cannot parse. Formatting as "2006-01-02 15:04:05" avoids this issue.
func fmtTime(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04:05")
}

func (r *UsageRepo) Create(log *models.UsageLog) error {
	result, err := r.db.Exec(
		`INSERT INTO usage_logs (channel_id, api_key_id, model, endpoint, input_tokens, output_tokens,
		        cache_creation_tokens, cache_read_tokens, is_stream, status_code, latency_ms,
		        first_token_ms, output_tokens_per_sec, success, error_message, client_ip, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.ChannelID, log.APIKeyID, log.Model, log.Endpoint, log.InputTokens, log.OutputTokens,
		log.CacheCreationTokens, log.CacheReadTokens, boolToInt(log.IsStream), log.StatusCode,
		log.LatencyMs, log.FirstTokenMs, log.OutputTokensPerSec, boolToInt(log.Success),
		log.ErrorMessage, log.ClientIP, log.Status, fmtTime(log.CreatedAt),
	)
	if err != nil {
		return err
	}
	log.ID, _ = result.LastInsertId()
	return nil
}

func (r *UsageRepo) CreatePending(log *models.UsageLog) error {
	log.Status = "pending"
	result, err := r.db.Exec(
		`INSERT INTO usage_logs (channel_id, api_key_id, model, endpoint, client_ip, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		log.ChannelID, log.APIKeyID, log.Model, log.Endpoint, log.ClientIP, log.Status, fmtTime(log.CreatedAt),
	)
	if err != nil {
		return err
	}
	log.ID, _ = result.LastInsertId()
	return nil
}

func (r *UsageRepo) UpdateCompleted(id int64, log *models.UsageLog) error {
	_, err := r.db.Exec(
		`UPDATE usage_logs SET channel_id = ?, api_key_id = ?, model = ?, endpoint = ?, client_ip = ?,
		        input_tokens = ?, output_tokens = ?, cache_creation_tokens = ?,
		        cache_read_tokens = ?, is_stream = ?, status_code = ?, latency_ms = ?,
		        first_token_ms = ?, output_tokens_per_sec = ?, success = ?, error_message = ?,
		        status = ?
		 WHERE id = ?`,
		log.ChannelID, log.APIKeyID, log.Model, log.Endpoint, log.ClientIP,
		log.InputTokens, log.OutputTokens, log.CacheCreationTokens, log.CacheReadTokens,
		boolToInt(log.IsStream), log.StatusCode, log.LatencyMs, log.FirstTokenMs,
		log.OutputTokensPerSec, boolToInt(log.Success), log.ErrorMessage, log.Status, id,
	)
	return err
}

type UsageFilter struct {
	ChannelID int64
	APIKeyID  int64
	Success   *bool
	Status    string
	StartDate string
	EndDate   string
	Model     string
	Page      int
	PageSize  int
}

func (r *UsageRepo) List(f UsageFilter) ([]models.UsageLog, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 100 {
		f.PageSize = 20
	}

	where, args := buildUsageWhere(f)

	var total int
	countQuery := "SELECT COUNT(*) FROM usage_logs " + where
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (f.Page - 1) * f.PageSize
	query := fmt.Sprintf(
		`SELECT id, channel_id, api_key_id, model, endpoint, input_tokens, output_tokens,
		        cache_creation_tokens, cache_read_tokens, is_stream, status_code, latency_ms,
		        first_token_ms, output_tokens_per_sec, success, error_message, client_ip, status, created_at
		 FROM usage_logs %s ORDER BY id DESC LIMIT ? OFFSET ?`, where,
	)
	args = append(args, f.PageSize, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	logs := make([]models.UsageLog, 0)
	for rows.Next() {
		var l models.UsageLog
		var isStream, success int
		if err := rows.Scan(&l.ID, &l.ChannelID, &l.APIKeyID, &l.Model, &l.Endpoint,
			&l.InputTokens, &l.OutputTokens, &l.CacheCreationTokens, &l.CacheReadTokens,
			&isStream, &l.StatusCode, &l.LatencyMs, &l.FirstTokenMs, &l.OutputTokensPerSec,
			&success, &l.ErrorMessage, &l.ClientIP, &l.Status, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		l.IsStream = isStream == 1
		l.Success = success == 1
		logs = append(logs, l)
	}
	return logs, total, rows.Err()
}

type UsageStats struct {
	TotalRequests         int64   `json:"total_requests"`
	TotalInput            int64   `json:"total_input_tokens"`
	TotalOutput           int64   `json:"total_output_tokens"`
	TotalCache            int64   `json:"total_cache_tokens"`
	SuccessCount          int64   `json:"success_count"`
	FailureCount          int64   `json:"failure_count"`
	AvgFirstTokenMs       float64 `json:"avg_first_token_ms"`
	AvgOutputTokensPerSec float64 `json:"avg_output_tokens_per_sec"`
}

func (r *UsageRepo) Stats(startDate, endDate string) (*UsageStats, error) {
	where, args := buildStatsWhere(startDate, endDate)
	s := &UsageStats{}
	query := fmt.Sprintf(
		`SELECT COUNT(*), COALESCE(SUM(input_tokens),0), COALESCE(SUM(output_tokens),0),
		        COALESCE(SUM(cache_creation_tokens+cache_read_tokens),0),
		        COALESCE(SUM(CASE WHEN success=1 THEN 1 ELSE 0 END),0),
		        COALESCE(SUM(CASE WHEN success=0 THEN 1 ELSE 0 END),0),
		        COALESCE(AVG(NULLIF(first_token_ms, 0)),0),
		        COALESCE(AVG(NULLIF(output_tokens_per_sec, 0)),0)
	 FROM usage_logs %s`, where,
	)
	err := r.db.QueryRow(query, args...).Scan(
		&s.TotalRequests, &s.TotalInput, &s.TotalOutput, &s.TotalCache, &s.SuccessCount, &s.FailureCount,
		&s.AvgFirstTokenMs, &s.AvgOutputTokensPerSec,
	)
	return s, err
}

type TimeseriesPoint struct {
	Date         string `json:"date"`
	Requests     int64  `json:"requests"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
}

func (r *UsageRepo) Timeseries(days int) ([]TimeseriesPoint, error) {
	if days <= 0 {
		days = 7
	}
	rows, err := r.db.Query(
		`SELECT date(created_at) as d, COUNT(*), COALESCE(SUM(input_tokens),0), COALESCE(SUM(output_tokens),0)
		 FROM usage_logs WHERE created_at >= datetime('now', ?)
		 GROUP BY d ORDER BY d`,
		fmt.Sprintf("-%d days", days),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]TimeseriesPoint, 0)
	for rows.Next() {
		var p TimeseriesPoint
		if err := rows.Scan(&p.Date, &p.Requests, &p.InputTokens, &p.OutputTokens); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func buildUsageWhere(f UsageFilter) (string, []interface{}) {
	var conds []string
	var args []interface{}
	if f.ChannelID > 0 {
		conds = append(conds, "channel_id = ?")
		args = append(args, f.ChannelID)
	}
	if f.APIKeyID > 0 {
		conds = append(conds, "api_key_id = ?")
		args = append(args, f.APIKeyID)
	}
	if f.Success != nil {
		conds = append(conds, "success = ?")
		if *f.Success {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}
	if f.StartDate != "" {
		conds = append(conds, "created_at >= ?")
		args = append(args, f.StartDate)
	}
	if f.EndDate != "" {
		conds = append(conds, "created_at <= ?")
		args = append(args, f.EndDate+" 23:59:59")
	}
	if f.Model != "" {
		conds = append(conds, "model = ?")
		args = append(args, f.Model)
	}
	if f.Status != "" {
		conds = append(conds, "status = ?")
		args = append(args, f.Status)
	}
	if len(conds) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conds, " AND "), args
}

func buildStatsWhere(startDate, endDate string) (string, []interface{}) {
	var conds []string
	var args []interface{}
	if startDate != "" {
		conds = append(conds, "created_at >= ?")
		args = append(args, startDate)
	}
	if endDate != "" {
		conds = append(conds, "created_at <= ?")
		args = append(args, endDate+" 23:59:59")
	}
	if len(conds) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conds, " AND "), args
}

type ModelStats struct {
	Model        string  `json:"model"`
	Requests     int64   `json:"requests"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
}

func (r *UsageRepo) ModelStatsByDate(startDate, endDate string) ([]ModelStats, error) {
	var conds []string
	var args []interface{}
	conds = append(conds, "model != ''")
	if startDate != "" {
		conds = append(conds, "created_at >= ?")
		args = append(args, startDate)
	}
	if endDate != "" {
		conds = append(conds, "created_at <= ?")
		args = append(args, endDate+" 23:59:59")
	}
	where := "WHERE " + strings.Join(conds, " AND ")

	rows, err := r.db.Query(
		`SELECT model, COUNT(*), COALESCE(SUM(input_tokens),0), COALESCE(SUM(output_tokens),0),
		        COALESCE(AVG(NULLIF(latency_ms, 0)),0)
		 FROM usage_logs `+where+`
		 GROUP BY model ORDER BY COUNT(*) DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make([]ModelStats, 0)
	for rows.Next() {
		var s ModelStats
		if err := rows.Scan(&s.Model, &s.Requests, &s.InputTokens, &s.OutputTokens, &s.AvgLatencyMs); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

type ChannelStats struct {
	ChannelID    int64   `json:"channel_id"`
	ChannelName  string  `json:"channel_name"`
	Requests     int64   `json:"requests"`
	SuccessCount int64   `json:"success_count"`
	FailureCount int64   `json:"failure_count"`
	SuccessRate  float64 `json:"success_rate"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
}

func (r *UsageRepo) ChannelStatsByDate(startDate, endDate string) ([]ChannelStats, error) {
	var conds []string
	var args []interface{}
	if startDate != "" {
		conds = append(conds, "u.created_at >= ?")
		args = append(args, startDate)
	}
	if endDate != "" {
		conds = append(conds, "u.created_at <= ?")
		args = append(args, endDate+" 23:59:59")
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	rows, err := r.db.Query(
		`SELECT u.channel_id, c.name,
		        COUNT(*),
		        COALESCE(SUM(CASE WHEN u.success=1 THEN 1 ELSE 0 END),0),
		        COALESCE(SUM(CASE WHEN u.success=0 THEN 1 ELSE 0 END),0),
		        COALESCE(SUM(u.input_tokens),0), COALESCE(SUM(u.output_tokens),0),
		        COALESCE(AVG(NULLIF(u.latency_ms, 0)),0)
		 FROM usage_logs u JOIN channels c ON c.id = u.channel_id
		 `+where+`
		 GROUP BY u.channel_id ORDER BY COUNT(*) DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make([]ChannelStats, 0)
	for rows.Next() {
		var s ChannelStats
		if err := rows.Scan(&s.ChannelID, &s.ChannelName, &s.Requests, &s.SuccessCount, &s.FailureCount,
			&s.InputTokens, &s.OutputTokens, &s.AvgLatencyMs); err != nil {
			return nil, err
		}
		if s.Requests > 0 {
			s.SuccessRate = float64(s.SuccessCount) / float64(s.Requests) * 100
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

func (r *UsageRepo) TodayRequestCount() (int64, error) {
	var count int64
	err := r.db.QueryRow(`SELECT COUNT(*) FROM usage_logs WHERE date(created_at) = date('now')`).Scan(&count)
	return count, err
}

func (r *UsageRepo) TodayTokenSum() (int64, error) {
	var total int64
	err := r.db.QueryRow(
		`SELECT COALESCE(SUM(input_tokens + output_tokens), 0) FROM usage_logs WHERE date(created_at) = date('now')`,
	).Scan(&total)
	return total, err
}

func (r *UsageRepo) DistinctModels() ([]string, error) {
	rows, err := r.db.Query(`SELECT DISTINCT model FROM usage_logs WHERE model != '' ORDER BY model`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	models := make([]string, 0)
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	return models, rows.Err()
}

var _ = time.Now // ensure import
