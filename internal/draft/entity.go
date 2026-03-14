package draft

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CreateType inserts a new EntityType. Generates an ID and timestamps if unset.
func (s *SQLiteStore) CreateType(t *EntityType) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	now := time.Now().Unix()
	if t.CreatedAt == 0 {
		t.CreatedAt = now
	}
	if t.UpdatedAt == 0 {
		t.UpdatedAt = now
	}

	fieldsJSON, err := json.Marshal(t.Fields)
	if err != nil {
		return fmt.Errorf("marshal fields: %w", err)
	}

	routesJSON, err := marshalRoutes(t.Routes)
	if err != nil {
		return fmt.Errorf("marshal routes: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT INTO entity_types (id, name, slug, fields, icon, routes, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Name, t.Slug, string(fieldsJSON), t.Icon, routesJSON, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create entity type: %w", err)
	}
	return nil
}

// GetType fetches an EntityType by ID.
func (s *SQLiteStore) GetType(id string) (*EntityType, error) {
	row := s.db.QueryRow(
		`SELECT id, name, slug, fields, icon, routes, created_at, updated_at
		 FROM entity_types WHERE id = ?`, id,
	)
	return scanEntityType(row)
}

// GetTypeBySlug fetches an EntityType by its slug.
func (s *SQLiteStore) GetTypeBySlug(slug string) (*EntityType, error) {
	row := s.db.QueryRow(
		`SELECT id, name, slug, fields, icon, routes, created_at, updated_at
		 FROM entity_types WHERE slug = ?`, slug,
	)
	return scanEntityType(row)
}

// ListTypes returns all EntityTypes ordered by name.
func (s *SQLiteStore) ListTypes() ([]*EntityType, error) {
	rows, err := s.db.Query(
		`SELECT id, name, slug, fields, icon, routes, created_at, updated_at
		 FROM entity_types ORDER BY rowid ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list entity types: %w", err)
	}
	defer rows.Close()

	var types []*EntityType
	for rows.Next() {
		t, err := scanEntityType(rows)
		if err != nil {
			return nil, err
		}
		types = append(types, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate entity types: %w", err)
	}
	return types, nil
}

// UpdateType updates name, slug, fields, icon, routes, and updated_at.
func (s *SQLiteStore) UpdateType(t *EntityType) error {
	t.UpdatedAt = time.Now().Unix()

	fieldsJSON, err := json.Marshal(t.Fields)
	if err != nil {
		return fmt.Errorf("marshal fields: %w", err)
	}

	routesJSON, err := marshalRoutes(t.Routes)
	if err != nil {
		return fmt.Errorf("marshal routes: %w", err)
	}

	res, err := s.db.Exec(
		`UPDATE entity_types SET name = ?, slug = ?, fields = ?, icon = ?, routes = ?, updated_at = ?
		 WHERE id = ?`,
		t.Name, t.Slug, string(fieldsJSON), t.Icon, routesJSON, t.UpdatedAt, t.ID,
	)
	if err != nil {
		return fmt.Errorf("update entity type: %w", err)
	}
	return requireOneRow(res, "entity type", t.ID)
}

// DeleteType deletes an EntityType by ID. Cascades to entities and blocks.
func (s *SQLiteStore) DeleteType(id string) error {
	res, err := s.db.Exec(`DELETE FROM entity_types WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete entity type: %w", err)
	}
	return requireOneRow(res, "entity type", id)
}

// CreateEntity inserts a new Entity. Generates an ID and timestamps if unset.
func (s *SQLiteStore) CreateEntity(e *Entity) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	now := time.Now().Unix()
	if e.CreatedAt == 0 {
		e.CreatedAt = now
	}
	if e.UpdatedAt == 0 {
		e.UpdatedAt = now
	}
	if e.Status == "" {
		e.Status = "draft"
	}

	dataJSON, err := json.Marshal(e.Data)
	if err != nil {
		return fmt.Errorf("marshal entity data: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT INTO entities (id, type_id, data, slug, status, position, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.TypeID, string(dataJSON), e.Slug, e.Status, e.Position, e.CreatedAt, e.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create entity: %w", err)
	}
	return nil
}

// GetEntity fetches an Entity by ID.
func (s *SQLiteStore) GetEntity(id string) (*Entity, error) {
	row := s.db.QueryRow(
		`SELECT id, type_id, data, slug, status, position, created_at, updated_at
		 FROM entities WHERE id = ?`, id,
	)
	return scanEntity(row)
}

// GetEntityBySlug fetches an Entity by type ID and slug.
func (s *SQLiteStore) GetEntityBySlug(typeID, slug string) (*Entity, error) {
	row := s.db.QueryRow(
		`SELECT id, type_id, data, slug, status, position, created_at, updated_at
		 FROM entities WHERE type_id = ? AND slug = ?`, typeID, slug,
	)
	return scanEntity(row)
}

// ListEntities returns a paginated, filtered list of entities for a type plus
// the total count matching the filter (before pagination).
func (s *SQLiteStore) ListEntities(typeID string, opts ListOpts) ([]*Entity, int, error) {
	var (
		whereClauses []string
		args         []any
	)

	whereClauses = append(whereClauses, "type_id = ?")
	args = append(args, typeID)

	if opts.Status != "" {
		whereClauses = append(whereClauses, "status = ?")
		args = append(args, opts.Status)
	}

	where := "WHERE " + strings.Join(whereClauses, " AND ")

	// Count query
	var total int
	countQuery := "SELECT COUNT(*) FROM entities " + where
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count entities: %w", err)
	}

	// Build ORDER BY
	orderBy := safeColumn(opts.OrderBy, "position")
	order := "ASC"
	if strings.ToUpper(opts.Order) == "DESC" {
		order = "DESC"
	}

	// Build LIMIT / OFFSET
	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	listQuery := fmt.Sprintf(
		`SELECT id, type_id, data, slug, status, position, created_at, updated_at
		 FROM entities %s ORDER BY %s %s LIMIT ? OFFSET ?`,
		where, orderBy, order,
	)
	listArgs := append(args, limit, offset)

	rows, err := s.db.Query(listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list entities: %w", err)
	}
	defer rows.Close()

	var entities []*Entity
	for rows.Next() {
		e, err := scanEntity(rows)
		if err != nil {
			return nil, 0, err
		}
		entities = append(entities, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate entities: %w", err)
	}
	return entities, total, nil
}

// UpdateEntity updates data, slug, status, position, and updated_at.
func (s *SQLiteStore) UpdateEntity(e *Entity) error {
	e.UpdatedAt = time.Now().Unix()

	dataJSON, err := json.Marshal(e.Data)
	if err != nil {
		return fmt.Errorf("marshal entity data: %w", err)
	}

	res, err := s.db.Exec(
		`UPDATE entities SET data = ?, slug = ?, status = ?, position = ?, updated_at = ?
		 WHERE id = ?`,
		string(dataJSON), e.Slug, e.Status, e.Position, e.UpdatedAt, e.ID,
	)
	if err != nil {
		return fmt.Errorf("update entity: %w", err)
	}
	return requireOneRow(res, "entity", e.ID)
}

// DeleteEntity deletes an entity by ID. Cascades to blocks and relations.
func (s *SQLiteStore) DeleteEntity(id string) error {
	res, err := s.db.Exec(`DELETE FROM entities WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete entity: %w", err)
	}
	return requireOneRow(res, "entity", id)
}

// --- scanner helpers ---

// scanner is the common interface shared by *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanEntityType(s scanner) (*EntityType, error) {
	var (
		t          EntityType
		fieldsJSON string
		routesJSON string
	)
	err := s.Scan(&t.ID, &t.Name, &t.Slug, &fieldsJSON, &t.Icon, &routesJSON, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("entity type not found")
		}
		return nil, fmt.Errorf("scan entity type: %w", err)
	}

	if err := json.Unmarshal([]byte(fieldsJSON), &t.Fields); err != nil {
		return nil, fmt.Errorf("unmarshal fields: %w", err)
	}

	if routesJSON != "" && routesJSON != "null" {
		var rc RouteConfig
		if err := json.Unmarshal([]byte(routesJSON), &rc); err != nil {
			return nil, fmt.Errorf("unmarshal routes: %w", err)
		}
		t.Routes = &rc
	}

	return &t, nil
}

func scanEntity(s scanner) (*Entity, error) {
	var (
		e        Entity
		dataJSON string
	)
	err := s.Scan(&e.ID, &e.TypeID, &dataJSON, &e.Slug, &e.Status, &e.Position, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("entity not found")
		}
		return nil, fmt.Errorf("scan entity: %w", err)
	}

	if err := json.Unmarshal([]byte(dataJSON), &e.Data); err != nil {
		return nil, fmt.Errorf("unmarshal entity data: %w", err)
	}

	return &e, nil
}

// marshalRoutes encodes a *RouteConfig to a JSON string, or "" if nil.
func marshalRoutes(rc *RouteConfig) (string, error) {
	if rc == nil {
		return "", nil
	}
	b, err := json.Marshal(rc)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// requireOneRow returns an error if the result set does not contain exactly one affected row.
func requireOneRow(res sql.Result, kind, id string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("%s not found: %s", kind, id)
	}
	return nil
}

// safeColumn returns the provided column name if it is in the allowed set,
// otherwise returns the fallback. This prevents SQL injection in ORDER BY.
func safeColumn(col, fallback string) string {
	allowed := map[string]bool{
		"position":   true,
		"created_at": true,
		"updated_at": true,
		"slug":       true,
		"status":     true,
	}
	if allowed[col] {
		return col
	}
	return fallback
}
