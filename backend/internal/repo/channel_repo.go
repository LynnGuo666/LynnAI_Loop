package repo

import (
	"database/sql"
	"loop/internal/models"
	"time"
)

type ChannelRepo struct {
	db *sql.DB
}

func NewChannelRepo(db *sql.DB) *ChannelRepo {
	return &ChannelRepo{db: db}
}

func (r *ChannelRepo) Create(ch *models.Channel) error {
	now := time.Now()
	result, err := r.db.Exec(
		`INSERT INTO channels (name, base_url, description, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		ch.Name, ch.BaseURL, ch.Description, boolToInt(ch.IsActive), now, now,
	)
	if err != nil {
		return err
	}
	ch.ID, _ = result.LastInsertId()
	ch.CreatedAt = now
	ch.UpdatedAt = now
	return nil
}

func (r *ChannelRepo) GetByID(id int64) (*models.Channel, error) {
	ch := &models.Channel{}
	var isActive int
	err := r.db.QueryRow(
		`SELECT id, name, base_url, description, is_active, created_at, updated_at FROM channels WHERE id = ?`, id,
	).Scan(&ch.ID, &ch.Name, &ch.BaseURL, &ch.Description, &isActive, &ch.CreatedAt, &ch.UpdatedAt)
	if err != nil {
		return nil, err
	}
	ch.IsActive = isActive == 1
	return ch, nil
}

func (r *ChannelRepo) List() ([]models.Channel, error) {
	rows, err := r.db.Query(`SELECT id, name, base_url, description, is_active, created_at, updated_at FROM channels ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	channels := make([]models.Channel, 0)
	for rows.Next() {
		var ch models.Channel
		var isActive int
		if err := rows.Scan(&ch.ID, &ch.Name, &ch.BaseURL, &ch.Description, &isActive, &ch.CreatedAt, &ch.UpdatedAt); err != nil {
			return nil, err
		}
		ch.IsActive = isActive == 1
		channels = append(channels, ch)
	}
	return channels, rows.Err()
}

func (r *ChannelRepo) Update(ch *models.Channel) error {
	_, err := r.db.Exec(
		`UPDATE channels SET name=?, base_url=?, description=?, is_active=?, updated_at=? WHERE id=?`,
		ch.Name, ch.BaseURL, ch.Description, boolToInt(ch.IsActive), time.Now(), ch.ID,
	)
	return err
}

func (r *ChannelRepo) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM channels WHERE id=?`, id)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
