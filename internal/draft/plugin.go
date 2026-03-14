package draft

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// SavePlugin upserts a plugin record in the store.
func (s *SQLiteStore) SavePlugin(name, version string, wasm []byte, config map[string]any) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal plugin config: %w", err)
	}
	_, err = s.db.Exec(
		`INSERT INTO plugins (name, version, wasm, config) VALUES (?, ?, ?, ?)
         ON CONFLICT(name) DO UPDATE SET version=excluded.version, wasm=excluded.wasm, config=excluded.config`,
		name, version, wasm, string(configJSON),
	)
	if err != nil {
		return fmt.Errorf("save plugin: %w", err)
	}
	return nil
}

// GetPlugin retrieves a single plugin record by name.
func (s *SQLiteStore) GetPlugin(name string) (PluginRecord, error) {
	row := s.db.QueryRow(
		`SELECT name, version, wasm, config, enabled, created_at FROM plugins WHERE name = ?`, name,
	)
	return scanPlugin(row)
}

// ListPlugins returns all plugin records.
func (s *SQLiteStore) ListPlugins() ([]PluginRecord, error) {
	rows, err := s.db.Query(
		`SELECT name, version, wasm, config, enabled, created_at FROM plugins ORDER BY created_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("list plugins: %w", err)
	}
	defer rows.Close()

	var out []PluginRecord
	for rows.Next() {
		rec, err := scanPlugin(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// DeletePlugin removes a plugin and its hooks (cascades via FK).
func (s *SQLiteStore) DeletePlugin(name string) error {
	res, err := s.db.Exec(`DELETE FROM plugins WHERE name = ?`, name)
	if err != nil {
		return fmt.Errorf("delete plugin: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("plugin %q not found", name)
	}
	return nil
}

// SaveHook inserts a plugin hook registration.
func (s *SQLiteStore) SaveHook(pluginName, hook, function, route string) error {
	_, err := s.db.Exec(
		`INSERT INTO plugin_hooks (plugin_name, hook, function, route) VALUES (?, ?, ?, ?)`,
		pluginName, hook, function, route,
	)
	if err != nil {
		return fmt.Errorf("save hook: %w", err)
	}
	return nil
}

// ListHooks returns all hook registrations for a given hook type.
func (s *SQLiteStore) ListHooks(hookType string) ([]HookRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, plugin_name, hook, function, route, position FROM plugin_hooks WHERE hook = ? ORDER BY position, id`,
		hookType,
	)
	if err != nil {
		return nil, fmt.Errorf("list hooks: %w", err)
	}
	defer rows.Close()

	var out []HookRecord
	for rows.Next() {
		var h HookRecord
		if err := rows.Scan(&h.ID, &h.PluginName, &h.Hook, &h.Function, &h.Route, &h.Position); err != nil {
			return nil, fmt.Errorf("scan hook: %w", err)
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func scanPlugin(s scanner) (PluginRecord, error) {
	var rec PluginRecord
	var configStr string
	var enabledInt int
	if err := s.Scan(&rec.Name, &rec.Version, &rec.Wasm, &configStr, &enabledInt, &rec.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return rec, fmt.Errorf("plugin not found")
		}
		return rec, fmt.Errorf("scan plugin: %w", err)
	}
	rec.Enabled = enabledInt != 0
	if configStr == "" || configStr == "null" {
		rec.Config = map[string]any{}
	} else {
		if err := json.Unmarshal([]byte(configStr), &rec.Config); err != nil {
			return rec, fmt.Errorf("unmarshal plugin config: %w", err)
		}
	}
	return rec, nil
}
