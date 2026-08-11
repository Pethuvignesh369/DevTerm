package db

import (
	"database/sql"
	"fmt"
	"log"
)

// Migration represents a database schema migration.
type Migration struct {
	Version int
	SQL     string
}

var migrations = []Migration{
	{
		Version: 1,
		SQL: `
-- Organization
CREATE TABLE IF NOT EXISTS groups (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS tags (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL UNIQUE
);

-- Auth material
CREATE TABLE IF NOT EXISTS ssh_keys (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  key_type TEXT NOT NULL CHECK (key_type IN ('rsa','ed25519')),
  public_key TEXT NOT NULL,
  fingerprint TEXT NOT NULL,
  passphrase_protected INTEGER NOT NULL DEFAULT 0,
  vault_ref TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS identities (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  auth_type TEXT NOT NULL CHECK (auth_type IN ('password','key','agent')),
  ssh_key_id TEXT REFERENCES ssh_keys(id) ON DELETE SET NULL,
  vault_ref TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Hosts
CREATE TABLE IF NOT EXISTS hosts (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  hostname TEXT NOT NULL,
  port INTEGER NOT NULL DEFAULT 22,
  username TEXT NOT NULL,
  identity_id TEXT REFERENCES identities(id) ON DELETE SET NULL,
  group_id INTEGER REFERENCES groups(id) ON DELETE SET NULL,
  favorite INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_hosts_group ON hosts(group_id);
CREATE INDEX IF NOT EXISTS idx_hosts_favorite ON hosts(favorite);

CREATE TABLE IF NOT EXISTS host_tags (
  host_id TEXT NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
  tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  PRIMARY KEY (host_id, tag_id)
);

-- Connection tracking
CREATE TABLE IF NOT EXISTS sessions (
  id TEXT PRIMARY KEY,
  host_id TEXT NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
  started_at TEXT NOT NULL DEFAULT (datetime('now')),
  ended_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_sessions_host ON sessions(host_id);

CREATE TABLE IF NOT EXISTS command_history (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  host_id TEXT NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
  command TEXT NOT NULL,
  executed_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_history_host ON command_history(host_id, executed_at DESC);

CREATE TABLE IF NOT EXISTS snippets (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  command TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS snippet_tags (
  snippet_id TEXT NOT NULL REFERENCES snippets(id) ON DELETE CASCADE,
  tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  PRIMARY KEY (snippet_id, tag_id)
);

CREATE TABLE IF NOT EXISTS port_forwards (
  id TEXT PRIMARY KEY,
  host_id TEXT NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
  type TEXT NOT NULL CHECK (type IN ('local','remote','dynamic')),
  local_host TEXT NOT NULL DEFAULT '127.0.0.1',
  local_port INTEGER NOT NULL,
  remote_host TEXT,
  remote_port INTEGER,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);
`,
	},
	{
		Version: 2,
		SQL:     `ALTER TABLE ssh_keys ADD COLUMN passphrase_vault_ref TEXT;`,
	},
}

// Migrate runs all pending migrations.
func Migrate(db *sql.DB) error {
	// Ensure schema_migrations table exists
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
	if err != nil {
		return fmt.Errorf("creating schema_migrations table: %w", err)
	}

	// Get current version
	var currentVersion int
	err = db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("querying current version: %w", err)
	}

	// Apply pending migrations
	for _, m := range migrations {
		if m.Version <= currentVersion {
			continue
		}
		log.Printf("applying migration %d", m.Version)
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("beginning transaction for migration %d: %w", m.Version, err)
		}
		if _, err := tx.Exec(m.SQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("applying migration %d: %w", m.Version, err)
		}
		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", m.Version); err != nil {
			tx.Rollback()
			return fmt.Errorf("recording migration %d: %w", m.Version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("committing migration %d: %w", m.Version, err)
		}
	}

	return nil
}
