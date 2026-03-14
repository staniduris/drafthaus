package draft

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SetView inserts or replaces a View. Generates an ID and timestamps if unset.
func (s *SQLiteStore) SetView(v *View) error {
	if v.ID == "" {
		v.ID = uuid.NewString()
	}
	now := time.Now().Unix()
	if v.CreatedAt == 0 {
		v.CreatedAt = now
	}
	v.UpdatedAt = now

	treeJSON, err := json.Marshal(v.Tree)
	if err != nil {
		return fmt.Errorf("marshal view tree: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT OR REPLACE INTO views (id, name, tree, version, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		v.ID, v.Name, string(treeJSON), v.Version, v.CreatedAt, v.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("set view: %w", err)
	}
	return nil
}

// GetView fetches a View by name.
func (s *SQLiteStore) GetView(name string) (*View, error) {
	row := s.db.QueryRow(
		`SELECT id, name, tree, version, created_at, updated_at
		 FROM views WHERE name = ?`, name,
	)
	return scanView(row)
}

// ListViews returns all Views ordered by name ascending.
func (s *SQLiteStore) ListViews() ([]*View, error) {
	rows, err := s.db.Query(
		`SELECT id, name, tree, version, created_at, updated_at
		 FROM views ORDER BY name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list views: %w", err)
	}
	defer rows.Close()

	var views []*View
	for rows.Next() {
		v, err := scanView(rows)
		if err != nil {
			return nil, err
		}
		views = append(views, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate views: %w", err)
	}
	return views, nil
}

func scanView(s scanner) (*View, error) {
	var (
		v        View
		treeJSON string
	)
	err := s.Scan(&v.ID, &v.Name, &treeJSON, &v.Version, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("view not found")
		}
		return nil, fmt.Errorf("scan view: %w", err)
	}

	if err := json.Unmarshal([]byte(treeJSON), &v.Tree); err != nil {
		return nil, fmt.Errorf("unmarshal view tree: %w", err)
	}

	return &v, nil
}
