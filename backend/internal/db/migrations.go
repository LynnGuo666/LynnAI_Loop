package db

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

const createTablesSQL = `
CREATE TABLE IF NOT EXISTS channels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    base_url TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    probe_model TEXT NOT NULL DEFAULT '',
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS api_keys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id INTEGER NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    key_value TEXT NOT NULL,
    alias TEXT NOT NULL DEFAULT '',
    is_active INTEGER NOT NULL DEFAULT 1,
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    total_failures INTEGER NOT NULL DEFAULT 0,
    total_successes INTEGER NOT NULL DEFAULT 0,
    last_used_at DATETIME,
    last_failure_at DATETIME,
    disabled_at DATETIME,
    next_probe_at DATETIME,
    probe_backoff_min INTEGER NOT NULL DEFAULT 60,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS usage_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id INTEGER NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    api_key_id INTEGER NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    model TEXT NOT NULL DEFAULT '',
    endpoint TEXT NOT NULL DEFAULT '',
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
    cache_read_tokens INTEGER NOT NULL DEFAULT 0,
    is_stream INTEGER NOT NULL DEFAULT 0,
    status_code INTEGER NOT NULL DEFAULT 0,
    latency_ms INTEGER NOT NULL DEFAULT 0,
    first_token_ms INTEGER NOT NULL DEFAULT 0,
    output_tokens_per_sec REAL NOT NULL DEFAULT 0,
    success INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    client_ip TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'success',
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS key_probes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    api_key_id INTEGER NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    success INTEGER NOT NULL DEFAULT 0,
    latency_ms INTEGER NOT NULL DEFAULT 0,
    status_code INTEGER NOT NULL DEFAULT 0,
    error_msg TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT '',
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_api_keys_channel ON api_keys(channel_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_active ON api_keys(is_active);
CREATE INDEX IF NOT EXISTS idx_usage_logs_channel ON usage_logs(channel_id);
CREATE INDEX IF NOT EXISTS idx_usage_logs_created ON usage_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_key_probes_key ON key_probes(api_key_id);
`

type migration struct {
	Version int
	SQL     string
	Fn      func(*sql.DB) error
}

var migrations = []migration{
	{Version: 1, SQL: "ALTER TABLE channels ADD COLUMN probe_model TEXT NOT NULL DEFAULT ''"},
	{Version: 2, SQL: "ALTER TABLE usage_logs ADD COLUMN first_token_ms INTEGER NOT NULL DEFAULT 0"},
	{Version: 3, SQL: "ALTER TABLE usage_logs ADD COLUMN output_tokens_per_sec REAL NOT NULL DEFAULT 0"},
	{Version: 4, SQL: "ALTER TABLE usage_logs ADD COLUMN status TEXT NOT NULL DEFAULT 'success'"},
	{Version: 5, SQL: "CREATE INDEX IF NOT EXISTS idx_usage_logs_status ON usage_logs(status)"},
	{Version: 6, SQL: "CREATE INDEX IF NOT EXISTS idx_api_keys_channel_active ON api_keys(channel_id, is_active)"},
	{Version: 7, Fn: fixTimeFormats},
}

func runMigrations(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	var currentVersion int
	err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("read migration version: %w", err)
	}

	for _, m := range migrations {
		if m.Version <= currentVersion {
			continue
		}
		log.Printf("applying migration %d...", m.Version)
		if m.Fn != nil {
			if err := m.Fn(db); err != nil {
				return fmt.Errorf("migration %d: %w", m.Version, err)
			}
		} else if _, err := db.Exec(m.SQL); err != nil {
			// On a fresh database, createTablesSQL already includes these columns/indexes.
			// "duplicate column" errors are expected and can be safely ignored.
			if isDuplicateError(err) {
				log.Printf("migration %d: already applied (skipped)", m.Version)
			} else {
				return fmt.Errorf("migration %d: %w", m.Version, err)
			}
		}
		if _, err := db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", m.Version); err != nil {
			return fmt.Errorf("record migration %d: %w", m.Version, err)
		}
	}

	return nil
}

func isDuplicateError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "duplicate column") ||
		strings.Contains(msg, "already exists")
}

// fixTimeFormats migrates time values stored in Go's time.Time.String() format
// (e.g. "2026-05-27 13:49:23.341308 +0800 CST m=+105.895498543") to
// SQLite-compatible UTC format ("2026-05-27 05:49:23"). Without this,
// SQLite's date()/datetime() functions cannot parse the stored values.
func fixTimeFormats(db *sql.DB) error {
	// Go reference time: Mon Jan 2 15:04:05 MST 2006
	// Layout without monotonic clock reading (Go strips it in format strings).
	layouts := []string{
		"2006-01-02 15:04:05.000000 -0700 MST",
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05.000000000Z07:00",
	}

	fixColumns := []struct {
		table  string
		column string
	}{
		{"usage_logs", "created_at"},
		{"key_probes", "created_at"},
		{"channels", "created_at"},
		{"channels", "updated_at"},
		{"api_keys", "created_at"},
		{"api_keys", "updated_at"},
		{"api_keys", "last_used_at"},
		{"api_keys", "last_failure_at"},
		{"api_keys", "disabled_at"},
		{"settings", "updated_at"},
	}

	for _, fc := range fixColumns {
		rows, err := db.Query(fmt.Sprintf("SELECT rowid, %s FROM %s WHERE %s != ''", fc.column, fc.table, fc.column))
		if err != nil {
			log.Printf("migration 7: skip %s.%s: %v", fc.table, fc.column, err)
			continue
		}

		type fixup struct {
			rowid    int64
			rawValue string
		}
		var fixups []fixup
		for rows.Next() {
			var f fixup
			if err := rows.Scan(&f.rowid, &f.rawValue); err != nil {
				continue
			}
			fixups = append(fixups, f)
		}
		rows.Close()

		fixed := 0
		for _, f := range fixups {
			clean := strings.Split(f.rawValue, " m=")[0]
			var t time.Time
			var parsed bool
			for _, layout := range layouts {
				if pt, err := time.Parse(layout, clean); err == nil {
					t = pt
					parsed = true
					break
				}
			}
			if !parsed {
				continue
			}
			newVal := t.UTC().Format("2006-01-02 15:04:05")
			if _, err := db.Exec(
				fmt.Sprintf("UPDATE %s SET %s = ? WHERE rowid = ?", fc.table, fc.column),
				newVal, f.rowid,
			); err != nil {
				log.Printf("migration 7: update %s.%s rowid=%d: %v", fc.table, fc.column, f.rowid, err)
				continue
			}
			fixed++
		}
		if fixed > 0 {
			log.Printf("migration 7: fixed %d rows in %s.%s", fixed, fc.table, fc.column)
		}
	}
	return nil
}
