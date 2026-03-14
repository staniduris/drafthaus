package draft

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// StoreAsset persists an asset. If an asset with the same hash already exists,
// the existing record is loaded into a and no insert is performed (dedup).
func (s *SQLiteStore) StoreAsset(a *Asset) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	if a.CreatedAt == 0 {
		a.CreatedAt = time.Now().Unix()
	}

	metaJSON, err := json.Marshal(a.Metadata)
	if err != nil {
		return fmt.Errorf("marshal asset metadata: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT INTO assets (id, hash, name, mime, size, data, metadata, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.Hash, a.Name, a.Mime, a.Size, a.Data, string(metaJSON), a.CreatedAt,
	)
	if err != nil {
		// Hash unique constraint violation — load the existing asset as dedup.
		existing, lookupErr := s.GetAssetByHash(a.Hash)
		if lookupErr != nil {
			return fmt.Errorf("store asset (dedup lookup): %w", lookupErr)
		}
		*a = *existing
		return nil
	}
	return nil
}

// GetAsset fetches an Asset by ID, including its binary data.
func (s *SQLiteStore) GetAsset(id string) (*Asset, error) {
	row := s.db.QueryRow(
		`SELECT id, hash, name, mime, size, data, metadata, created_at
		 FROM assets WHERE id = ?`, id,
	)
	return scanAsset(row)
}

// GetAssetByHash fetches an Asset by its content hash, including binary data.
func (s *SQLiteStore) GetAssetByHash(hash string) (*Asset, error) {
	row := s.db.QueryRow(
		`SELECT id, hash, name, mime, size, data, metadata, created_at
		 FROM assets WHERE hash = ?`, hash,
	)
	return scanAsset(row)
}

func scanAsset(s scanner) (*Asset, error) {
	var (
		a        Asset
		metaJSON string
	)
	err := s.Scan(&a.ID, &a.Hash, &a.Name, &a.Mime, &a.Size, &a.Data, &metaJSON, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("asset not found")
		}
		return nil, fmt.Errorf("scan asset: %w", err)
	}

	if metaJSON != "" && metaJSON != "null" {
		if err := json.Unmarshal([]byte(metaJSON), &a.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal asset metadata: %w", err)
		}
	}

	return &a, nil
}
