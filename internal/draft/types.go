package draft

// FieldType represents the type of a field in an entity type definition.
type FieldType string

const (
	FieldText     FieldType = "text"
	FieldRichText FieldType = "richtext"
	FieldNumber   FieldType = "number"
	FieldCurrency FieldType = "currency"
	FieldDate     FieldType = "date"
	FieldDateTime FieldType = "datetime"
	FieldBool     FieldType = "boolean"
	FieldEnum     FieldType = "enum"
	FieldEmail    FieldType = "email"
	FieldURL      FieldType = "url"
	FieldGeo      FieldType = "geo"
	FieldAsset    FieldType = "asset"
	FieldRelation FieldType = "relation"
	FieldJSON     FieldType = "json"
	FieldSlug     FieldType = "slug"
)

// Validation defines constraints on a field value.
type Validation struct {
	MinLength *int    `json:"min_length,omitempty"`
	MaxLength *int    `json:"max_length,omitempty"`
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	Pattern   string  `json:"pattern,omitempty"`
}

// FieldDef defines a field within an entity type.
type FieldDef struct {
	Name       string      `json:"name"`
	Type       FieldType   `json:"type"`
	Required   bool        `json:"required,omitempty"`
	Default    any         `json:"default,omitempty"`
	Validation *Validation `json:"validation,omitempty"`
	Values     []string    `json:"values,omitempty"` // for enum type
}

// RouteConfig maps an entity type to URL patterns.
type RouteConfig struct {
	List   string `json:"list,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// EntityType defines a kind of entity (e.g. BlogPost, Product).
type EntityType struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Slug      string       `json:"slug"`
	Fields    []FieldDef   `json:"fields"`
	Icon      string       `json:"icon,omitempty"`
	Routes    *RouteConfig `json:"routes,omitempty"`
	CreatedAt int64        `json:"created_at"`
	UpdatedAt int64        `json:"updated_at"`
}

// Entity is an instance of an EntityType.
type Entity struct {
	ID        string         `json:"id"`
	TypeID    string         `json:"type_id"`
	Data      map[string]any `json:"data"`
	Slug      string         `json:"slug,omitempty"`
	Status    string         `json:"status"`
	Position  float64        `json:"position"`
	CreatedAt int64          `json:"created_at"`
	UpdatedAt int64          `json:"updated_at"`
}

// Block is a unit of rich content within an entity field.
type Block struct {
	ID       string         `json:"id"`
	EntityID string         `json:"entity_id"`
	Field    string         `json:"field"`
	Type     string         `json:"type"`
	Data     map[string]any `json:"data"`
	Position float64        `json:"position"`
	ParentID string         `json:"parent_id,omitempty"`
}

// Relation connects two entities.
type Relation struct {
	SourceID     string         `json:"source_id"`
	TargetID     string         `json:"target_id"`
	RelationType string         `json:"relation_type"`
	Position     float64        `json:"position"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// Asset is a binary file stored in the .draft file.
type Asset struct {
	ID        string         `json:"id"`
	Hash      string         `json:"hash"`
	Name      string         `json:"name"`
	Mime      string         `json:"mime"`
	Size      int64          `json:"size"`
	Data      []byte         `json:"-"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt int64          `json:"created_at"`
}

// View defines a component tree for rendering an entity type.
type View struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Tree      any    `json:"tree"`
	Version   int    `json:"version"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// TokenSet holds design tokens for a site.
type TokenSet struct {
	ID        string `json:"id"`
	Data      Tokens `json:"data"`
	UpdatedAt int64  `json:"updated_at"`
}

// Tokens defines the design system values.
type Tokens struct {
	Colors   map[string]string `json:"colors"`
	Fonts    map[string]string `json:"fonts"`
	Scale    ScaleTokens       `json:"scale"`
	Mood     string            `json:"mood,omitempty"`
	SiteName string            `json:"site_name,omitempty"`
}

// ScaleTokens controls spacing, radius, and density.
type ScaleTokens struct {
	Spacing float64 `json:"spacing"`
	Radius  string  `json:"radius"`
	Density string  `json:"density"`
}

// Version is a snapshot of an entity at a point in time.
type Version struct {
	EntityID  string `json:"entity_id"`
	Version   int    `json:"version"`
	Data      string `json:"data"`
	Blocks    string `json:"blocks,omitempty"`
	ChangedAt int64  `json:"changed_at"`
}
