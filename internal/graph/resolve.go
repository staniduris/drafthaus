package graph

import (
	"fmt"

	"github.com/drafthaus/drafthaus/internal/draft"
)

// ResolvedEntity is a fully hydrated entity with its type, blocks, and related entities.
type ResolvedEntity struct {
	Entity    *draft.Entity               `json:"entity"`
	Type      *draft.EntityType           `json:"type"`
	Blocks    map[string][]*draft.Block   `json:"blocks"`    // field name → blocks
	Relations map[string][]*ResolvedEntity `json:"relations"` // relation type → resolved entities
}

// Resolver resolves entities with their full context from a Store.
type Resolver struct {
	store draft.Store
}

// NewResolver creates a Resolver backed by the given Store.
func NewResolver(s draft.Store) *Resolver {
	return &Resolver{store: s}
}

// Resolve loads an entity with its type, blocks, and one level of related entities.
func (r *Resolver) Resolve(entityID string) (*ResolvedEntity, error) {
	entity, err := r.store.GetEntity(entityID)
	if err != nil {
		return nil, fmt.Errorf("get entity %s: %w", entityID, err)
	}

	entityType, err := r.store.GetType(entity.TypeID)
	if err != nil {
		return nil, fmt.Errorf("get type %s: %w", entity.TypeID, err)
	}

	blocks, err := r.fetchBlocks(entity.ID, entityType)
	if err != nil {
		return nil, err
	}

	relations, err := r.fetchRelations(entity.ID)
	if err != nil {
		return nil, err
	}

	return &ResolvedEntity{
		Entity:    entity,
		Type:      entityType,
		Blocks:    blocks,
		Relations: relations,
	}, nil
}

// ResolveBySlug loads an entity by type slug and entity slug.
func (r *Resolver) ResolveBySlug(typeSlug, entitySlug string) (*ResolvedEntity, error) {
	entityType, err := r.store.GetTypeBySlug(typeSlug)
	if err != nil {
		return nil, fmt.Errorf("get type by slug %q: %w", typeSlug, err)
	}

	entity, err := r.store.GetEntityBySlug(entityType.ID, entitySlug)
	if err != nil {
		return nil, fmt.Errorf("get entity by slug %q (type %s): %w", entitySlug, entityType.ID, err)
	}

	return r.Resolve(entity.ID)
}

// ResolveList loads all entities of a given type with full resolution.
func (r *Resolver) ResolveList(typeSlug string, opts ListOpts) ([]*ResolvedEntity, int, error) {
	entityType, err := r.store.GetTypeBySlug(typeSlug)
	if err != nil {
		return nil, 0, fmt.Errorf("get type by slug %q: %w", typeSlug, err)
	}

	storeOpts := draft.ListOpts{
		Status:  opts.Status,
		Limit:   opts.Limit,
		Offset:  opts.Offset,
		OrderBy: opts.OrderBy,
		Order:   opts.Order,
	}

	entities, total, err := r.store.ListEntities(entityType.ID, storeOpts)
	if err != nil {
		return nil, 0, fmt.Errorf("list entities for type %s: %w", entityType.ID, err)
	}

	resolved := make([]*ResolvedEntity, 0, len(entities))
	for _, e := range entities {
		re, err := r.Resolve(e.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("resolve entity %s: %w", e.ID, err)
		}
		resolved = append(resolved, re)
	}

	return resolved, total, nil
}

// resolveShallow loads an entity with its type and blocks, but without relations.
// Used to resolve related entities without infinite recursion.
func (r *Resolver) resolveShallow(entityID string) (*ResolvedEntity, error) {
	entity, err := r.store.GetEntity(entityID)
	if err != nil {
		return nil, fmt.Errorf("get entity %s: %w", entityID, err)
	}

	entityType, err := r.store.GetType(entity.TypeID)
	if err != nil {
		return nil, fmt.Errorf("get type %s: %w", entity.TypeID, err)
	}

	blocks, err := r.fetchBlocks(entity.ID, entityType)
	if err != nil {
		return nil, err
	}

	return &ResolvedEntity{
		Entity:    entity,
		Type:      entityType,
		Blocks:    blocks,
		Relations: map[string][]*ResolvedEntity{},
	}, nil
}

// fetchBlocks retrieves blocks for all richtext fields of the given entity type.
func (r *Resolver) fetchBlocks(entityID string, entityType *draft.EntityType) (map[string][]*draft.Block, error) {
	blocks := make(map[string][]*draft.Block)
	for _, field := range entityType.Fields {
		if field.Type != draft.FieldRichText {
			continue
		}
		fieldBlocks, err := r.store.GetBlocks(entityID, field.Name)
		if err != nil {
			return nil, fmt.Errorf("get blocks for entity %s field %q: %w", entityID, field.Name, err)
		}
		if len(fieldBlocks) > 0 {
			blocks[field.Name] = fieldBlocks
		}
	}
	return blocks, nil
}

// fetchRelations loads all outgoing relations and resolves each target entity shallowly,
// grouping results by relation type.
func (r *Resolver) fetchRelations(entityID string) (map[string][]*ResolvedEntity, error) {
	relations, err := r.store.GetRelations(entityID, "", draft.Outgoing)
	if err != nil {
		return nil, fmt.Errorf("get relations for entity %s: %w", entityID, err)
	}

	grouped := make(map[string][]*ResolvedEntity)
	for _, rel := range relations {
		re, err := r.resolveShallow(rel.TargetID)
		if err != nil {
			return nil, fmt.Errorf("resolve related entity %s: %w", rel.TargetID, err)
		}
		grouped[rel.RelationType] = append(grouped[rel.RelationType], re)
	}

	return grouped, nil
}
