package models

import "time"

type Channel struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	BaseURL     string    `json:"base_url"`
	Description string    `json:"description"`
	ProbeModel  string    `json:"probe_model"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
