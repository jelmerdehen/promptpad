#!/bin/bash
# promptpad db helpers — sourced by bin/promptpad.

SQLITE3=${SQLITE3:-/usr/bin/sqlite3}
PROMPTPAD_DATA=${PROMPTPAD_DATA:-${XDG_DATA_HOME:-$HOME/.local/share}/promptpad}
PROMPTPAD_DB=$PROMPTPAD_DATA/usage.db

db_init() {
    mkdir -p "$PROMPTPAD_DATA"
    "$SQLITE3" "$PROMPTPAD_DB" <<'SQL'
CREATE TABLE IF NOT EXISTS uses (
  id    INTEGER PRIMARY KEY AUTOINCREMENT,
  slot  INTEGER NOT NULL,
  ts    TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  hash  TEXT
);
CREATE INDEX IF NOT EXISTS idx_uses_slot ON uses(slot);
SQL
}

db_log_use() {
    local slot=$1 hash=$2
    db_init
    "$SQLITE3" "$PROMPTPAD_DB" \
        "INSERT INTO uses(slot, hash) VALUES($slot, '$hash');"
}

db_count() {
    local slot=$1
    "$SQLITE3" "$PROMPTPAD_DB" \
        "SELECT COUNT(*) FROM uses WHERE slot=$slot;" 2>/dev/null || echo 0
}

db_last() {
    local slot=$1
    "$SQLITE3" "$PROMPTPAD_DB" \
        "SELECT COALESCE(MAX(ts),'never') FROM uses WHERE slot=$slot;" 2>/dev/null || echo never
}

db_reset() {
    db_init
    if [ -n "${1:-}" ]; then
        "$SQLITE3" "$PROMPTPAD_DB" "DELETE FROM uses WHERE slot=$1;"
    else
        "$SQLITE3" "$PROMPTPAD_DB" "DELETE FROM uses;"
    fi
}
