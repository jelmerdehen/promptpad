package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type DB struct{ sql *sql.DB }

func Open() (*DB, error) {
	path, err := defaultPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	s, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := s.Exec(`
CREATE TABLE IF NOT EXISTS uses (
  id    INTEGER PRIMARY KEY AUTOINCREMENT,
  slot  INTEGER NOT NULL,
  ts    TEXT    NOT NULL DEFAULT (datetime('now')),
  hash  TEXT
);
CREATE INDEX IF NOT EXISTS idx_uses_slot ON uses(slot);
`); err != nil {
		return nil, err
	}
	return &DB{s}, nil
}

func (d *DB) Close() error { return d.sql.Close() }

func (d *DB) Path() string { p, _ := defaultPath(); return p }

func (d *DB) Log(slot int, hash string) error {
	_, err := d.sql.Exec("INSERT INTO uses(slot, hash) VALUES(?, ?)", slot, hash)
	return err
}

type Stat struct {
	Slot  int
	Count int
	Last  string
}

func (d *DB) Stat(slot int) (Stat, error) {
	st := Stat{Slot: slot, Last: "never"}
	row := d.sql.QueryRow(
		"SELECT COUNT(*), COALESCE(MAX(ts),'never') FROM uses WHERE slot=?", slot)
	if err := row.Scan(&st.Count, &st.Last); err != nil {
		return st, err
	}
	return st, nil
}

func (d *DB) Reset(slot *int) error {
	if slot == nil {
		_, err := d.sql.Exec("DELETE FROM uses")
		return err
	}
	_, err := d.sql.Exec("DELETE FROM uses WHERE slot=?", *slot)
	return err
}

func defaultPath() (string, error) {
	if p := os.Getenv("PROMPTPAD_DB"); p != "" {
		return p, nil
	}
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("home dir: %w", err)
		}
		data = filepath.Join(h, ".local", "share")
	}
	return filepath.Join(data, "promptpad", "usage.db"), nil
}
