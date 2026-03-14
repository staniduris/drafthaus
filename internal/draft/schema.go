package draft

import (
	"database/sql"
	"fmt"
)

const schemaVersion = 1
const drafthausVersion = "0.1.0"

const schemaSQL = `
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;

CREATE TABLE IF NOT EXISTS dh_meta (
    key   TEXT PRIMARY KEY,
    value TEXT
);

CREATE TABLE IF NOT EXISTS entity_types (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    slug       TEXT NOT NULL UNIQUE,
    fields     TEXT NOT NULL,
    icon       TEXT DEFAULT '',
    routes     TEXT DEFAULT '',
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE IF NOT EXISTS entities (
    id         TEXT PRIMARY KEY,
    type_id    TEXT NOT NULL REFERENCES entity_types(id) ON DELETE CASCADE,
    data       TEXT NOT NULL DEFAULT '{}',
    slug       TEXT DEFAULT '',
    status     TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','published','archived')),
    position   REAL NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_entities_type_slug ON entities(type_id, slug) WHERE slug != '';
CREATE INDEX IF NOT EXISTS idx_entities_type_status ON entities(type_id, status);

CREATE TABLE IF NOT EXISTS blocks (
    id        TEXT PRIMARY KEY,
    entity_id TEXT NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    field     TEXT NOT NULL,
    type      TEXT NOT NULL,
    data      TEXT NOT NULL DEFAULT '{}',
    position  REAL NOT NULL DEFAULT 0,
    parent_id TEXT REFERENCES blocks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_blocks_entity_field ON blocks(entity_id, field);

CREATE TABLE IF NOT EXISTS relations (
    source_id     TEXT NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    target_id     TEXT NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    relation_type TEXT NOT NULL,
    position      REAL NOT NULL DEFAULT 0,
    metadata      TEXT DEFAULT '{}',
    PRIMARY KEY (source_id, target_id, relation_type)
);

CREATE INDEX IF NOT EXISTS idx_relations_target ON relations(target_id, relation_type);

CREATE TABLE IF NOT EXISTS assets (
    id         TEXT PRIMARY KEY,
    hash       TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    mime       TEXT NOT NULL,
    size       INTEGER NOT NULL,
    data       BLOB NOT NULL,
    metadata   TEXT DEFAULT '{}',
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE IF NOT EXISTS views (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    tree       TEXT NOT NULL,
    version    INTEGER NOT NULL DEFAULT 1,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE IF NOT EXISTS tokens (
    id         TEXT PRIMARY KEY DEFAULT 'default',
    data       TEXT NOT NULL,
    updated_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE IF NOT EXISTS versions (
    entity_id  TEXT NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    version    INTEGER NOT NULL,
    data       TEXT NOT NULL,
    blocks     TEXT DEFAULT '',
    changed_at INTEGER NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (entity_id, version)
);

CREATE TABLE IF NOT EXISTS admin_users (
    id         TEXT PRIMARY KEY,
    username   TEXT NOT NULL UNIQUE,
    password   TEXT NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE IF NOT EXISTS page_views (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    path       TEXT NOT NULL,
    referrer   TEXT DEFAULT '',
    user_agent TEXT DEFAULT '',
    visitor_id TEXT DEFAULT '',
    country    TEXT DEFAULT '',
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE INDEX IF NOT EXISTS idx_page_views_path_date ON page_views(path, created_at);
CREATE INDEX IF NOT EXISTS idx_page_views_date ON page_views(created_at);

CREATE TABLE IF NOT EXISTS plugins (
    name       TEXT PRIMARY KEY,
    version    TEXT NOT NULL,
    wasm       BLOB NOT NULL,
    config     TEXT DEFAULT '{}',
    enabled    BOOLEAN NOT NULL DEFAULT 1,
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE IF NOT EXISTS plugin_hooks (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    plugin_name TEXT NOT NULL REFERENCES plugins(name) ON DELETE CASCADE,
    hook        TEXT NOT NULL,
    function    TEXT NOT NULL,
    route       TEXT DEFAULT '',
    position    INTEGER NOT NULL DEFAULT 0
);
`

// InitSchema creates all tables and seeds metadata if needed.
func InitSchema(db *sql.DB) error {
	if _, err := db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("init schema: %w", err)
	}

	// Seed metadata if not present
	_, err := db.Exec(`INSERT OR IGNORE INTO dh_meta (key, value) VALUES ('schema_version', ?), ('drafthaus_version', ?)`,
		fmt.Sprintf("%d", schemaVersion), drafthausVersion)
	if err != nil {
		return fmt.Errorf("seed metadata: %w", err)
	}
	return nil
}
