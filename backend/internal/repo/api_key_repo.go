package repo

import (
	"database/sql"
	"errors"
	"loop/internal/db"
	"loop/internal/models"
	"strings"
	"sync"
	"time"
)

var ErrDuplicateAPIKey = errors.New("duplicate_api_key")

type APIKeyRepo struct {
	db *db.DB
	mu sync.Mutex
}

func NewAPIKeyRepo(db *db.DB) *APIKeyRepo {
	return &APIKeyRepo{db: db}
}

func (r *APIKeyRepo) Create(k *models.APIKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	k.KeyValue = strings.TrimSpace(k.KeyValue)
	if exists, err := r.ExistsKeyValue(k.KeyValue, 0); err != nil {
		return err
	} else if exists {
		return ErrDuplicateAPIKey
	}

	now := time.Now()
	result, err := r.db.Exec(
		`INSERT INTO api_keys (channel_id, key_value, alias, is_active, probe_backoff_min, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		k.ChannelID, k.KeyValue, k.Alias, boolToInt(k.IsActive), k.ProbeBackoffMin, fmtTime(now), fmtTime(now),
	)
	if err != nil {
		return err
	}
	k.ID, _ = result.LastInsertId()
	k.CreatedAt = now
	k.UpdatedAt = now
	return nil
}

// ImportItem is a key queued for batch import along with the caller's index,
// used to report per-item outcomes back in order.
type ImportItem struct {
	Index int
	Key   models.APIKey
}

// ImportError reports a failed item by its caller index.
type ImportError struct {
	Index   int
	Message string
}

// ImportResult is the outcome of CreateBatch.
type ImportResult struct {
	Created []models.APIKey
	Skipped int
	Errors  []ImportError
}

// CreateBatch inserts the given keys in a single transaction. Duplicates — both
// against existing rows and against earlier items in the same batch — are
// skipped; per-item insert errors are recorded and the remaining items still
// commit together. The dedup check runs inside the transaction so it sees the
// batch's own uncommitted inserts.
func (r *APIKeyRepo) CreateBatch(items []ImportItem) (ImportResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := ImportResult{
		Created: make([]models.APIKey, 0, len(items)),
		Errors:  make([]ImportError, 0),
	}

	tx, err := r.db.Begin()
	if err != nil {
		return result, err
	}
	defer tx.Rollback()

	now := time.Now()
	for _, item := range items {
		k := item.Key
		k.KeyValue = strings.TrimSpace(k.KeyValue)

		var one int
		err := tx.QueryRow(`SELECT 1 FROM api_keys WHERE key_value = ? LIMIT 1`, k.KeyValue).Scan(&one)
		if err == nil {
			result.Skipped++
			continue
		}
		if err != sql.ErrNoRows {
			result.Errors = append(result.Errors, ImportError{Index: item.Index, Message: err.Error()})
			continue
		}

		res, err := tx.Exec(
			`INSERT INTO api_keys (channel_id, key_value, alias, is_active, probe_backoff_min, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			k.ChannelID, k.KeyValue, k.Alias, boolToInt(k.IsActive), k.ProbeBackoffMin, fmtTime(now), fmtTime(now),
		)
		if err != nil {
			result.Errors = append(result.Errors, ImportError{Index: item.Index, Message: err.Error()})
			continue
		}
		k.ID, _ = res.LastInsertId()
		k.CreatedAt = now
		k.UpdatedAt = now
		result.Created = append(result.Created, k)
	}

	if err := tx.Commit(); err != nil {
		return ImportResult{Created: make([]models.APIKey, 0), Errors: make([]ImportError, 0)}, err
	}
	return result, nil
}

func (r *APIKeyRepo) GetByID(id int64) (*models.APIKey, error) {
	k := &models.APIKey{}
	var isActive int
	var lastUsed, lastFailure, disabledAt, nextProbe sql.NullTime
	err := r.db.QueryRow(
		`SELECT id, channel_id, key_value, alias, is_active, consecutive_failures, total_failures, total_successes,
		        last_used_at, last_failure_at, disabled_at, next_probe_at, probe_backoff_min, created_at, updated_at
		 FROM api_keys WHERE id = ?`, id,
	).Scan(&k.ID, &k.ChannelID, &k.KeyValue, &k.Alias, &isActive, &k.ConsecutiveFailures,
		&k.TotalFailures, &k.TotalSuccesses, &lastUsed, &lastFailure, &disabledAt, &nextProbe,
		&k.ProbeBackoffMin, &k.CreatedAt, &k.UpdatedAt)
	if err != nil {
		return nil, err
	}
	k.IsActive = isActive == 1
	k.LastUsedAt = nullTimeToPtr(lastUsed)
	k.LastFailureAt = nullTimeToPtr(lastFailure)
	k.DisabledAt = nullTimeToPtr(disabledAt)
	k.NextProbeAt = nullTimeToPtr(nextProbe)
	return k, nil
}

func (r *APIKeyRepo) ListByChannel(channelID int64) ([]models.APIKey, error) {
	rows, err := r.db.Query(
		`SELECT id, channel_id, key_value, alias, is_active, consecutive_failures, total_failures, total_successes,
		        last_used_at, last_failure_at, disabled_at, next_probe_at, probe_backoff_min, created_at, updated_at
		 FROM api_keys WHERE channel_id = ? ORDER BY id`, channelID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKeys(rows)
}

func (r *APIKeyRepo) ListActiveByChannel(channelID int64) ([]models.APIKey, error) {
	rows, err := r.db.Query(
		`SELECT id, channel_id, key_value, alias, is_active, consecutive_failures, total_failures, total_successes,
		        last_used_at, last_failure_at, disabled_at, next_probe_at, probe_backoff_min, created_at, updated_at
		 FROM api_keys WHERE channel_id = ? AND is_active = 1 ORDER BY id`, channelID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKeys(rows)
}

func (r *APIKeyRepo) ListAll() ([]models.APIKey, error) {
	rows, err := r.db.Query(
		`SELECT id, channel_id, key_value, alias, is_active, consecutive_failures, total_failures, total_successes,
		        last_used_at, last_failure_at, disabled_at, next_probe_at, probe_backoff_min, created_at, updated_at
		 FROM api_keys ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKeys(rows)
}

func (r *APIKeyRepo) Update(k *models.APIKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	k.KeyValue = strings.TrimSpace(k.KeyValue)
	if exists, err := r.ExistsKeyValue(k.KeyValue, k.ID); err != nil {
		return err
	} else if exists {
		return ErrDuplicateAPIKey
	}

	_, err := r.db.Exec(
		`UPDATE api_keys SET alias=?, is_active=?, consecutive_failures=?, total_failures=?, total_successes=?,
		        last_used_at=?, last_failure_at=?, disabled_at=?, next_probe_at=?, probe_backoff_min=?, updated_at=?
		 WHERE id=?`,
		k.Alias, boolToInt(k.IsActive), k.ConsecutiveFailures, k.TotalFailures, k.TotalSuccesses,
		k.LastUsedAt, k.LastFailureAt, k.DisabledAt, k.NextProbeAt, k.ProbeBackoffMin, fmtTime(time.Now()), k.ID,
	)
	return err
}

func (r *APIKeyRepo) ExistsKeyValue(keyValue string, excludeID int64) (bool, error) {
	keyValue = strings.TrimSpace(keyValue)
	if keyValue == "" {
		return false, nil
	}

	query := `SELECT 1 FROM api_keys WHERE key_value = ?`
	args := []interface{}{keyValue}
	if excludeID > 0 {
		query += ` AND id != ?`
		args = append(args, excludeID)
	}
	query += ` LIMIT 1`

	var one int
	err := r.db.QueryRow(query, args...).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (r *APIKeyRepo) RecordSuccess(id int64) error {
	now := time.Now()
	_, err := r.db.Exec(
		`UPDATE api_keys
		 SET consecutive_failures = 0,
		     total_successes = total_successes + 1,
		     last_used_at = ?,
		     updated_at = ?
		 WHERE id = ?`,
		fmtTime(now), fmtTime(now), id,
	)
	return err
}

func (r *APIKeyRepo) RecordFailure(id int64, disableThreshold int) error {
	now := time.Now()
	nowStr := fmtTime(now)
	_, err := r.db.Exec(
		`UPDATE api_keys
		 SET consecutive_failures = consecutive_failures + 1,
		     total_failures = total_failures + 1,
		     last_failure_at = ?,
		     last_used_at = ?,
		     is_active = CASE
		         WHEN consecutive_failures + 1 >= ? THEN 0
		         ELSE is_active
		     END,
		     disabled_at = CASE
		         WHEN consecutive_failures + 1 >= ? THEN ?
		         ELSE disabled_at
		     END,
		     next_probe_at = CASE
		         WHEN consecutive_failures + 1 >= ? THEN datetime('now', '+' || probe_backoff_min || ' minutes')
		         ELSE next_probe_at
		     END,
		     updated_at = ?
		 WHERE id = ?`,
		nowStr, nowStr, disableThreshold, disableThreshold, nowStr, disableThreshold, nowStr, id,
	)
	return err
}

func (r *APIKeyRepo) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM api_keys WHERE id=?`, id)
	return err
}

func (r *APIKeyRepo) ListDisabledForProbe() ([]models.APIKey, error) {
	rows, err := r.db.Query(
		`SELECT id, channel_id, key_value, alias, is_active, consecutive_failures, total_failures, total_successes,
		        last_used_at, last_failure_at, disabled_at, next_probe_at, probe_backoff_min, created_at, updated_at
		 FROM api_keys WHERE is_active = 0 AND next_probe_at <= datetime('now') ORDER BY next_probe_at`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKeys(rows)
}

func scanKeys(rows *sql.Rows) ([]models.APIKey, error) {
	keys := make([]models.APIKey, 0)
	for rows.Next() {
		var k models.APIKey
		var isActive int
		var lastUsed, lastFailure, disabledAt, nextProbe sql.NullTime
		if err := rows.Scan(&k.ID, &k.ChannelID, &k.KeyValue, &k.Alias, &isActive, &k.ConsecutiveFailures,
			&k.TotalFailures, &k.TotalSuccesses, &lastUsed, &lastFailure, &disabledAt, &nextProbe,
			&k.ProbeBackoffMin, &k.CreatedAt, &k.UpdatedAt); err != nil {
			return nil, err
		}
		k.IsActive = isActive == 1
		k.LastUsedAt = nullTimeToPtr(lastUsed)
		k.LastFailureAt = nullTimeToPtr(lastFailure)
		k.DisabledAt = nullTimeToPtr(disabledAt)
		k.NextProbeAt = nullTimeToPtr(nextProbe)
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func nullTimeToPtr(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}
