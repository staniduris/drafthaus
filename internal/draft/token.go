package draft

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// SetTokens inserts or replaces the token set (always stored with id='default').
func (s *SQLiteStore) SetTokens(t *TokenSet) error {
	t.ID = "default"
	t.UpdatedAt = time.Now().Unix()

	dataJSON, err := json.Marshal(t.Data)
	if err != nil {
		return fmt.Errorf("marshal tokens data: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT OR REPLACE INTO tokens (id, data, updated_at)
		 VALUES (?, ?, ?)`,
		t.ID, string(dataJSON), t.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("set tokens: %w", err)
	}
	return nil
}

// GetTokens fetches the default TokenSet. Returns sensible defaults if no row exists.
func (s *SQLiteStore) GetTokens() (*TokenSet, error) {
	row := s.db.QueryRow(
		`SELECT id, data, updated_at FROM tokens WHERE id = 'default'`,
	)

	var (
		ts       TokenSet
		dataJSON string
	)
	err := row.Scan(&ts.ID, &dataJSON, &ts.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return defaultTokenSet(), nil
		}
		return nil, fmt.Errorf("scan tokens: %w", err)
	}

	if err := json.Unmarshal([]byte(dataJSON), &ts.Data); err != nil {
		return nil, fmt.Errorf("unmarshal tokens data: %w", err)
	}

	// Migrate site_name from Colors map to SiteName field.
	if ts.Data.SiteName == "" {
		if sn, ok := ts.Data.Colors["site_name"]; ok && sn != "" {
			ts.Data.SiteName = sn
			delete(ts.Data.Colors, "site_name")
		}
	}

	return &ts, nil
}

// defaultTokenSet returns a TokenSet with sensible design defaults.
func defaultTokenSet() *TokenSet {
	return &TokenSet{
		ID:        "default",
		UpdatedAt: time.Now().Unix(),
		Data: Tokens{
			Colors: map[string]string{
				"primary":    "#1a1a1a",
				"secondary":  "#6b7280",
				"background": "#ffffff",
				"surface":    "#f9fafb",
				"border":     "#e5e7eb",
				"text":       "#111827",
				"muted":      "#9ca3af",
			},
			Fonts: map[string]string{
				"sans":  "Inter, system-ui, sans-serif",
				"serif": "Georgia, serif",
				"mono":  "JetBrains Mono, monospace",
			},
			Scale: ScaleTokens{
				Spacing: 1.0,
				Radius:  "md",
				Density: "comfortable",
			},
		},
	}
}
