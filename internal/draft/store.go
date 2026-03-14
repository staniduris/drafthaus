package draft

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Direction indicates which side of a relation to query.
type Direction int

const (
	Outgoing Direction = iota
	Incoming
)

// ListOpts controls pagination and filtering for entity lists.
type ListOpts struct {
	Status  string
	Limit   int
	Offset  int
	OrderBy string
	Order   string // "asc" or "desc"
}

// Store defines all operations on a .draft file.
type Store interface {
	// Entity types
	CreateType(t *EntityType) error
	GetType(id string) (*EntityType, error)
	GetTypeBySlug(slug string) (*EntityType, error)
	ListTypes() ([]*EntityType, error)
	UpdateType(t *EntityType) error
	DeleteType(id string) error

	// Entities
	CreateEntity(e *Entity) error
	GetEntity(id string) (*Entity, error)
	GetEntityBySlug(typeID, slug string) (*Entity, error)
	ListEntities(typeID string, opts ListOpts) ([]*Entity, int, error)
	UpdateEntity(e *Entity) error
	DeleteEntity(id string) error

	// Blocks
	SetBlocks(entityID, field string, blocks []*Block) error
	GetBlocks(entityID, field string) ([]*Block, error)

	// Relations
	AddRelation(r *Relation) error
	RemoveRelation(sourceID, targetID, relType string) error
	GetRelations(entityID string, relType string, dir Direction) ([]*Relation, error)

	// Assets
	StoreAsset(a *Asset) error
	GetAsset(id string) (*Asset, error)
	GetAssetByHash(hash string) (*Asset, error)

	// Views
	SetView(v *View) error
	GetView(name string) (*View, error)
	ListViews() ([]*View, error)

	// Tokens
	SetTokens(t *TokenSet) error
	GetTokens() (*TokenSet, error)

	// Versions
	SaveVersion(entityID string) error
	GetVersion(entityID string, version int) (*Version, error)
	ListVersions(entityID string) ([]*Version, error)

	// Admin auth
	CreateAdminUser(username, password string) error
	ValidateCredentials(username, password string) (bool, error)
	HasAdminUsers() (bool, error)

	Close() error
}

// SQLiteStore implements Store backed by a SQLite .draft file.
type SQLiteStore struct {
	db *sql.DB
}

// Open opens or creates a .draft file and initializes the schema.
func Open(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open draft file: %w", err)
	}

	if err := InitSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

// DB returns the underlying database connection (for testing/advanced use).
func (s *SQLiteStore) DB() *sql.DB {
	return s.db
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
