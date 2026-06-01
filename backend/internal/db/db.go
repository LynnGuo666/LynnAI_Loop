package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// DB wraps two connection pools over the same SQLite file:
//
//   - write: pinned to a single connection so writes are serialized in-process,
//     eliminating contention for SQLite's single writer lock.
//   - read:  multiple connections that serve concurrent reads under WAL without
//     blocking (or being blocked by) the writer.
//
// It exposes the subset of *sql.DB methods the repos use, so repo code needs no
// changes beyond the field/constructor type.
type DB struct {
	write *sql.DB
	read  *sql.DB
}

// pragmas applied to every connection. synchronous(NORMAL) is safe under WAL and
// noticeably faster than the FULL default.
const pragmas = "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_pragma=synchronous(NORMAL)"

// Open opens the read and write pools against dbPath. readPoolSize bounds the
// read pool (clamped to >= 1). Schema creation and migrations run once on the
// write pool.
func Open(dbPath string, readPoolSize int) (*DB, error) {
	if readPoolSize < 1 {
		readPoolSize = 1
	}
	dsn := dbPath + pragmas

	write, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open write db: %w", err)
	}
	write.SetMaxOpenConns(1)
	write.SetMaxIdleConns(1)
	write.SetConnMaxLifetime(0)
	if err := write.Ping(); err != nil {
		write.Close()
		return nil, fmt.Errorf("ping write db: %w", err)
	}

	read, err := sql.Open("sqlite", dsn)
	if err != nil {
		write.Close()
		return nil, fmt.Errorf("open read db: %w", err)
	}
	read.SetMaxOpenConns(readPoolSize)
	read.SetMaxIdleConns(readPoolSize)
	read.SetConnMaxLifetime(0)
	if err := read.Ping(); err != nil {
		write.Close()
		read.Close()
		return nil, fmt.Errorf("ping read db: %w", err)
	}

	if _, err := write.Exec(createTablesSQL); err != nil {
		write.Close()
		read.Close()
		return nil, fmt.Errorf("create tables: %w", err)
	}
	if err := runMigrations(write); err != nil {
		write.Close()
		read.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &DB{write: write, read: read}, nil
}

// Exec runs a write statement on the single-connection write pool.
func (d *DB) Exec(query string, args ...any) (sql.Result, error) {
	return d.write.Exec(query, args...)
}

// Query runs a read query on the read pool.
func (d *DB) Query(query string, args ...any) (*sql.Rows, error) {
	return d.read.Query(query, args...)
}

// QueryRow runs a single-row read query on the read pool.
func (d *DB) QueryRow(query string, args ...any) *sql.Row {
	return d.read.QueryRow(query, args...)
}

// Begin starts a transaction on the write pool (used by batch writes).
func (d *DB) Begin() (*sql.Tx, error) {
	return d.write.Begin()
}

// Close closes both pools, returning the first error encountered.
func (d *DB) Close() error {
	werr := d.write.Close()
	rerr := d.read.Close()
	if werr != nil {
		return werr
	}
	return rerr
}
