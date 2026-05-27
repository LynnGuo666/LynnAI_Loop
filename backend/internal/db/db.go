package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func Open(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	if _, err := db.Exec(createTablesSQL); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	if err := ensureColumn(db, "channels", "probe_model", "ALTER TABLE channels ADD COLUMN probe_model TEXT NOT NULL DEFAULT ''"); err != nil {
		return nil, fmt.Errorf("migrate channels.probe_model: %w", err)
	}
	if err := ensureColumn(db, "usage_logs", "first_token_ms", "ALTER TABLE usage_logs ADD COLUMN first_token_ms INTEGER NOT NULL DEFAULT 0"); err != nil {
		return nil, fmt.Errorf("migrate usage_logs.first_token_ms: %w", err)
	}
	if err := ensureColumn(db, "usage_logs", "output_tokens_per_sec", "ALTER TABLE usage_logs ADD COLUMN output_tokens_per_sec REAL NOT NULL DEFAULT 0"); err != nil {
		return nil, fmt.Errorf("migrate usage_logs.output_tokens_per_sec: %w", err)
	}
	return db, nil
}

func ensureColumn(db *sql.DB, table, column, ddl string) error {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue interface{}
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == column {
			return rows.Err()
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = db.Exec(ddl)
	return err
}
