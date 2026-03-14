package draft

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// AddRelation inserts or replaces a relation between two entities.
func (s *SQLiteStore) AddRelation(r *Relation) error {
	metaJSON, err := json.Marshal(r.Metadata)
	if err != nil {
		return fmt.Errorf("marshal relation metadata: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT OR REPLACE INTO relations (source_id, target_id, relation_type, position, metadata)
		 VALUES (?, ?, ?, ?, ?)`,
		r.SourceID, r.TargetID, r.RelationType, r.Position, string(metaJSON),
	)
	if err != nil {
		return fmt.Errorf("add relation: %w", err)
	}
	return nil
}

// RemoveRelation deletes a specific relation identified by its three-part key.
func (s *SQLiteStore) RemoveRelation(sourceID, targetID, relType string) error {
	res, err := s.db.Exec(
		`DELETE FROM relations WHERE source_id = ? AND target_id = ? AND relation_type = ?`,
		sourceID, targetID, relType,
	)
	if err != nil {
		return fmt.Errorf("remove relation: %w", err)
	}
	return requireOneRow(res, "relation", sourceID+"/"+targetID+"/"+relType)
}

// GetRelations returns all relations for an entity in the given direction,
// optionally filtered by relation type. Results are ordered by position ASC.
func (s *SQLiteStore) GetRelations(entityID string, relType string, dir Direction) ([]*Relation, error) {
	var (
		query string
		args  []any
	)

	switch dir {
	case Outgoing:
		query = `SELECT source_id, target_id, relation_type, position, metadata
		          FROM relations WHERE source_id = ?`
		args = append(args, entityID)
	case Incoming:
		query = `SELECT source_id, target_id, relation_type, position, metadata
		          FROM relations WHERE target_id = ?`
		args = append(args, entityID)
	default:
		return nil, fmt.Errorf("unknown direction: %d", dir)
	}

	if relType != "" {
		query += ` AND relation_type = ?`
		args = append(args, relType)
	}

	query += ` ORDER BY position ASC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("get relations: %w", err)
	}
	defer rows.Close()

	var relations []*Relation
	for rows.Next() {
		r, err := scanRelation(rows)
		if err != nil {
			return nil, err
		}
		relations = append(relations, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate relations: %w", err)
	}
	return relations, nil
}

func scanRelation(s scanner) (*Relation, error) {
	var (
		r        Relation
		metaJSON string
	)
	err := s.Scan(&r.SourceID, &r.TargetID, &r.RelationType, &r.Position, &metaJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("relation not found")
		}
		return nil, fmt.Errorf("scan relation: %w", err)
	}

	if metaJSON != "" && metaJSON != "null" {
		if err := json.Unmarshal([]byte(metaJSON), &r.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal relation metadata: %w", err)
		}
	}

	return &r, nil
}
