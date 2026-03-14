package draft

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// SaveVersion snapshots the current entity data and all its blocks as a new version.
func (s *SQLiteStore) SaveVersion(entityID string) error {
	// Fetch the current entity.
	entity, err := s.GetEntity(entityID)
	if err != nil {
		return fmt.Errorf("save version: fetch entity: %w", err)
	}

	dataJSON, err := json.Marshal(entity.Data)
	if err != nil {
		return fmt.Errorf("save version: marshal entity data: %w", err)
	}

	// Fetch all blocks for this entity (all fields).
	rows, err := s.db.Query(
		`SELECT id, entity_id, field, type, data, position, parent_id
		 FROM blocks WHERE entity_id = ? ORDER BY field ASC, position ASC`,
		entityID,
	)
	if err != nil {
		return fmt.Errorf("save version: fetch blocks: %w", err)
	}
	defer rows.Close()

	var blocks []*Block
	for rows.Next() {
		b, scanErr := scanBlock(rows)
		if scanErr != nil {
			return fmt.Errorf("save version: scan block: %w", scanErr)
		}
		blocks = append(blocks, b)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("save version: iterate blocks: %w", err)
	}

	blocksJSON, err := json.Marshal(blocks)
	if err != nil {
		return fmt.Errorf("save version: marshal blocks: %w", err)
	}

	// Determine next version number.
	var maxVersion sql.NullInt64
	if err := s.db.QueryRow(
		`SELECT MAX(version) FROM versions WHERE entity_id = ?`, entityID,
	).Scan(&maxVersion); err != nil {
		return fmt.Errorf("save version: query max version: %w", err)
	}

	nextVersion := 1
	if maxVersion.Valid {
		nextVersion = int(maxVersion.Int64) + 1
	}

	_, err = s.db.Exec(
		`INSERT INTO versions (entity_id, version, data, blocks, changed_at)
		 VALUES (?, ?, ?, ?, ?)`,
		entityID, nextVersion, string(dataJSON), string(blocksJSON), time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("save version: insert: %w", err)
	}
	return nil
}

// GetVersion fetches a specific version snapshot for an entity.
func (s *SQLiteStore) GetVersion(entityID string, version int) (*Version, error) {
	row := s.db.QueryRow(
		`SELECT entity_id, version, data, blocks, changed_at
		 FROM versions WHERE entity_id = ? AND version = ?`,
		entityID, version,
	)
	return scanVersion(row)
}

// ListVersions returns all versions for an entity, ordered newest first.
func (s *SQLiteStore) ListVersions(entityID string) ([]*Version, error) {
	rows, err := s.db.Query(
		`SELECT entity_id, version, data, blocks, changed_at
		 FROM versions WHERE entity_id = ? ORDER BY version DESC`,
		entityID,
	)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	defer rows.Close()

	var versions []*Version
	for rows.Next() {
		v, err := scanVersion(rows)
		if err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate versions: %w", err)
	}
	return versions, nil
}

func scanVersion(s scanner) (*Version, error) {
	var v Version
	err := s.Scan(&v.EntityID, &v.Version, &v.Data, &v.Blocks, &v.ChangedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("version not found")
		}
		return nil, fmt.Errorf("scan version: %w", err)
	}
	return &v, nil
}
