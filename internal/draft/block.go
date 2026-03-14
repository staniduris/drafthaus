package draft

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// SetBlocks replaces all blocks for (entityID, field) with the provided slice.
// The operation runs inside a transaction: delete existing, then insert new.
func (s *SQLiteStore) SetBlocks(entityID, field string, blocks []*Block) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(
		`DELETE FROM blocks WHERE entity_id = ? AND field = ?`, entityID, field,
	); err != nil {
		return fmt.Errorf("delete blocks: %w", err)
	}

	for _, b := range blocks {
		if b.ID == "" {
			b.ID = uuid.NewString()
		}
		b.EntityID = entityID
		b.Field = field

		dataJSON, jsonErr := json.Marshal(b.Data)
		if jsonErr != nil {
			err = fmt.Errorf("marshal block data (id=%s): %w", b.ID, jsonErr)
			return err
		}

		var parentID any
		if b.ParentID != "" {
			parentID = b.ParentID
		}

		if _, err = tx.Exec(
			`INSERT INTO blocks (id, entity_id, field, type, data, position, parent_id)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			b.ID, b.EntityID, b.Field, b.Type, string(dataJSON), b.Position, parentID,
		); err != nil {
			err = fmt.Errorf("insert block (id=%s): %w", b.ID, err)
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit set blocks: %w", err)
	}
	return nil
}

// GetBlocks returns all blocks for (entityID, field) ordered by position ASC.
func (s *SQLiteStore) GetBlocks(entityID, field string) ([]*Block, error) {
	rows, err := s.db.Query(
		`SELECT id, entity_id, field, type, data, position, parent_id
		 FROM blocks
		 WHERE entity_id = ? AND field = ?
		 ORDER BY position ASC`,
		entityID, field,
	)
	if err != nil {
		return nil, fmt.Errorf("get blocks: %w", err)
	}
	defer rows.Close()

	var blocks []*Block
	for rows.Next() {
		b, err := scanBlock(rows)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate blocks: %w", err)
	}
	return blocks, nil
}

func scanBlock(s scanner) (*Block, error) {
	var (
		b        Block
		dataJSON string
		parentID sql.NullString
	)
	err := s.Scan(&b.ID, &b.EntityID, &b.Field, &b.Type, &dataJSON, &b.Position, &parentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("block not found")
		}
		return nil, fmt.Errorf("scan block: %w", err)
	}

	if parentID.Valid {
		b.ParentID = parentID.String
	}

	if err := json.Unmarshal([]byte(dataJSON), &b.Data); err != nil {
		return nil, fmt.Errorf("unmarshal block data: %w", err)
	}

	return &b, nil
}
