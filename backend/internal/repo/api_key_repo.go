package repo

import (
	"database/sql"
	"loop/internal/models"
	"time"
)

type APIKeyRepo struct {
	db *sql.DB
}

func NewAPIKeyRepo(db *sql.DB) *APIKeyRepo {
	return &APIKeyRepo{db: db}
}

func (r *APIKeyRepo) Create(k *models.APIKey) error {
	now := time.Now()
	result, err := r.db.Exec(
		`INSERT INTO api_keys (channel_id, key_value, alias, is_active, probe_backoff_min, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		k.ChannelID, k.KeyValue, k.Alias, boolToInt(k.IsActive), k.ProbeBackoffMin, now, now,
	)
	if err != nil {
		return err
	}
	k.ID, _ = result.LastInsertId()
	k.CreatedAt = now
	k.UpdatedAt = now
	return nil
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
	_, err := r.db.Exec(
		`UPDATE api_keys SET alias=?, is_active=?, consecutive_failures=?, total_failures=?, total_successes=?,
		        last_used_at=?, last_failure_at=?, disabled_at=?, next_probe_at=?, probe_backoff_min=?, updated_at=?
		 WHERE id=?`,
		k.Alias, boolToInt(k.IsActive), k.ConsecutiveFailures, k.TotalFailures, k.TotalSuccesses,
		k.LastUsedAt, k.LastFailureAt, k.DisabledAt, k.NextProbeAt, k.ProbeBackoffMin, time.Now(), k.ID,
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
	var keys []models.APIKey
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
