package repo

import (
	"loop/internal/db"
	"loop/internal/models"
)

type KeyProbeRepo struct {
	db *db.DB
}

func NewKeyProbeRepo(db *db.DB) *KeyProbeRepo {
	return &KeyProbeRepo{db: db}
}

func (r *KeyProbeRepo) Create(p *models.KeyProbe) error {
	result, err := r.db.Exec(
		`INSERT INTO key_probes (api_key_id, success, latency_ms, status_code, error_msg, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		p.APIKeyID, boolToInt(p.Success), p.LatencyMs, p.StatusCode, p.ErrorMsg, p.CreatedAt,
	)
	if err != nil {
		return err
	}
	p.ID, _ = result.LastInsertId()
	return nil
}

func (r *KeyProbeRepo) ListByKey(keyID int64, limit int) ([]models.KeyProbe, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.Query(
		`SELECT id, api_key_id, success, latency_ms, status_code, error_msg, created_at
		 FROM key_probes WHERE api_key_id = ? ORDER BY id DESC LIMIT ?`, keyID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var probes []models.KeyProbe
	for rows.Next() {
		var p models.KeyProbe
		var success int
		if err := rows.Scan(&p.ID, &p.APIKeyID, &success, &p.LatencyMs, &p.StatusCode, &p.ErrorMsg, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.Success = success == 1
		probes = append(probes, p)
	}
	return probes, rows.Err()
}
