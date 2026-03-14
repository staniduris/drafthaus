package draft_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/drafthaus/drafthaus/internal/draft"
)

// --- helpers ---

func openTestStore(t *testing.T) *draft.SQLiteStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.draft")
	store, err := draft.Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func createTestType(t *testing.T, store *draft.SQLiteStore) *draft.EntityType {
	t.Helper()
	et := &draft.EntityType{
		Name: "BlogPost",
		Slug: "blog-posts",
		Fields: []draft.FieldDef{
			{Name: "title", Type: draft.FieldText, Required: true},
			{Name: "body", Type: draft.FieldRichText},
		},
		Routes: &draft.RouteConfig{List: "/blog", Detail: "/blog/{slug}"},
	}
	if err := store.CreateType(et); err != nil {
		t.Fatalf("create type: %v", err)
	}
	return et
}

func createTestEntity(t *testing.T, store *draft.SQLiteStore, typeID string) *draft.Entity {
	t.Helper()
	e := &draft.Entity{
		TypeID: typeID,
		Data:   map[string]any{"title": "Test Post", "body": "content"},
		Slug:   "test-post",
		Status: "published",
	}
	if err := store.CreateEntity(e); err != nil {
		t.Fatalf("create entity: %v", err)
	}
	return e
}

// --- tests ---

func TestOpenAndClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "site.draft")

	store, err := draft.Open(path)
	if err != nil {
		t.Fatalf("should open new store: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("should close store: %v", err)
	}

	// Reopen the same path — schema already exists, should succeed.
	store2, err := draft.Open(path)
	if err != nil {
		t.Fatalf("should reopen existing store: %v", err)
	}
	defer store2.Close()

	// Verify store is functional after reopen.
	types, err := store2.ListTypes()
	if err != nil {
		t.Fatalf("should list types after reopen: %v", err)
	}
	if len(types) != 0 {
		t.Errorf("expected 0 types, got %d", len(types))
	}
}

func TestCreateAndGetType(t *testing.T) {
	store := openTestStore(t)
	et := createTestType(t, store)

	// Get by ID.
	got, err := store.GetType(et.ID)
	if err != nil {
		t.Fatalf("should get type by ID: %v", err)
	}
	if got.Name != et.Name {
		t.Errorf("name: want %q, got %q", et.Name, got.Name)
	}
	if got.Slug != et.Slug {
		t.Errorf("slug: want %q, got %q", et.Slug, got.Slug)
	}
	if len(got.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(got.Fields))
	}
	if got.Fields[0].Name != "title" || !got.Fields[0].Required {
		t.Errorf("field[0] mismatch: %+v", got.Fields[0])
	}
	if got.Routes == nil || got.Routes.List != "/blog" {
		t.Errorf("routes mismatch: %+v", got.Routes)
	}
	if got.ID == "" {
		t.Error("should have auto-generated ID")
	}
	if got.CreatedAt == 0 {
		t.Error("should have auto-set created_at")
	}

	// Get by slug.
	bySlug, err := store.GetTypeBySlug(et.Slug)
	if err != nil {
		t.Fatalf("should get type by slug: %v", err)
	}
	if bySlug.ID != et.ID {
		t.Errorf("slug lookup ID: want %q, got %q", et.ID, bySlug.ID)
	}
}

func TestGetType_notFound(t *testing.T) {
	store := openTestStore(t)

	_, err := store.GetType("nonexistent-id")
	if err == nil {
		t.Fatal("should return error for missing type")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found, got: %v", err)
	}
}

func TestGetTypeBySlug_notFound(t *testing.T) {
	store := openTestStore(t)

	_, err := store.GetTypeBySlug("no-such-slug")
	if err == nil {
		t.Fatal("should return error for missing slug")
	}
}

func TestListTypes(t *testing.T) {
	store := openTestStore(t)

	slugs := []string{"type-a", "type-b", "type-c"}
	for _, slug := range slugs {
		et := &draft.EntityType{
			Name:   "Type " + slug,
			Slug:   slug,
			Fields: []draft.FieldDef{{Name: "f", Type: draft.FieldText}},
		}
		if err := store.CreateType(et); err != nil {
			t.Fatalf("create type %s: %v", slug, err)
		}
	}

	types, err := store.ListTypes()
	if err != nil {
		t.Fatalf("list types: %v", err)
	}
	if len(types) != 3 {
		t.Errorf("expected 3 types, got %d", len(types))
	}
}

func TestUpdateType(t *testing.T) {
	store := openTestStore(t)
	et := createTestType(t, store)

	et.Name = "UpdatedBlog"
	et.Slug = "updated-blog"
	if err := store.UpdateType(et); err != nil {
		t.Fatalf("update type: %v", err)
	}

	got, err := store.GetType(et.ID)
	if err != nil {
		t.Fatalf("get updated type: %v", err)
	}
	if got.Name != "UpdatedBlog" {
		t.Errorf("name not updated: got %q", got.Name)
	}
	if got.Slug != "updated-blog" {
		t.Errorf("slug not updated: got %q", got.Slug)
	}
}

func TestUpdateType_notFound(t *testing.T) {
	store := openTestStore(t)

	et := &draft.EntityType{
		ID:     "ghost-id",
		Name:   "Ghost",
		Slug:   "ghost",
		Fields: []draft.FieldDef{},
	}
	err := store.UpdateType(et)
	if err == nil {
		t.Fatal("should return error when updating non-existent type")
	}
}

func TestDeleteType(t *testing.T) {
	store := openTestStore(t)
	et := createTestType(t, store)

	if err := store.DeleteType(et.ID); err != nil {
		t.Fatalf("delete type: %v", err)
	}

	_, err := store.GetType(et.ID)
	if err == nil {
		t.Fatal("should return error after deletion")
	}
}

func TestDeleteType_notFound(t *testing.T) {
	store := openTestStore(t)

	err := store.DeleteType("ghost-id")
	if err == nil {
		t.Fatal("should return error when deleting non-existent type")
	}
}

func TestCreateAndGetEntity(t *testing.T) {
	store := openTestStore(t)
	et := createTestType(t, store)
	e := createTestEntity(t, store, et.ID)

	// Get by ID.
	got, err := store.GetEntity(e.ID)
	if err != nil {
		t.Fatalf("should get entity by ID: %v", err)
	}
	if got.TypeID != et.ID {
		t.Errorf("type_id: want %q, got %q", et.ID, got.TypeID)
	}
	if got.Slug != "test-post" {
		t.Errorf("slug: want %q, got %q", "test-post", got.Slug)
	}
	if got.Status != "published" {
		t.Errorf("status: want %q, got %q", "published", got.Status)
	}
	if got.Data["title"] != "Test Post" {
		t.Errorf("data.title: want %q, got %v", "Test Post", got.Data["title"])
	}
	if got.ID == "" {
		t.Error("should have auto-generated ID")
	}

	// Get by slug.
	bySlug, err := store.GetEntityBySlug(et.ID, "test-post")
	if err != nil {
		t.Fatalf("should get entity by slug: %v", err)
	}
	if bySlug.ID != e.ID {
		t.Errorf("slug lookup ID: want %q, got %q", e.ID, bySlug.ID)
	}
}

func TestCreateEntity_defaultStatus(t *testing.T) {
	store := openTestStore(t)
	et := createTestType(t, store)

	e := &draft.Entity{
		TypeID: et.ID,
		Data:   map[string]any{"title": "No Status"},
		// Status intentionally left empty.
	}
	if err := store.CreateEntity(e); err != nil {
		t.Fatalf("create entity: %v", err)
	}

	got, err := store.GetEntity(e.ID)
	if err != nil {
		t.Fatalf("get entity: %v", err)
	}
	if got.Status != "draft" {
		t.Errorf("expected default status 'draft', got %q", got.Status)
	}
}

func TestGetEntity_notFound(t *testing.T) {
	store := openTestStore(t)

	_, err := store.GetEntity("nonexistent")
	if err == nil {
		t.Fatal("should return error for missing entity")
	}
}

func TestListEntities(t *testing.T) {
	store := openTestStore(t)
	et := createTestType(t, store)

	// 2 published, 1 draft.
	for i, status := range []string{"published", "published", "draft"} {
		e := &draft.Entity{
			TypeID: et.ID,
			Data:   map[string]any{"title": "Post"},
			Slug:   "post-" + string(rune('a'+i)),
			Status: status,
		}
		if err := store.CreateEntity(e); err != nil {
			t.Fatalf("create entity: %v", err)
		}
	}

	// All entities.
	all, total, err := store.ListEntities(et.ID, draft.ListOpts{})
	if err != nil {
		t.Fatalf("list all entities: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total=3, got %d", total)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 entities, got %d", len(all))
	}

	// Filter by published.
	published, pubTotal, err := store.ListEntities(et.ID, draft.ListOpts{Status: "published"})
	if err != nil {
		t.Fatalf("list published: %v", err)
	}
	if pubTotal != 2 {
		t.Errorf("expected pubTotal=2, got %d", pubTotal)
	}
	if len(published) != 2 {
		t.Errorf("expected 2 published entities, got %d", len(published))
	}

	// Filter by draft.
	drafts, draftTotal, err := store.ListEntities(et.ID, draft.ListOpts{Status: "draft"})
	if err != nil {
		t.Fatalf("list drafts: %v", err)
	}
	if draftTotal != 1 {
		t.Errorf("expected draftTotal=1, got %d", draftTotal)
	}
	if len(drafts) != 1 {
		t.Errorf("expected 1 draft entity, got %d", len(drafts))
	}
}

func TestListEntities_pagination(t *testing.T) {
	store := openTestStore(t)
	et := createTestType(t, store)

	for i := 0; i < 5; i++ {
		e := &draft.Entity{
			TypeID: et.ID,
			Data:   map[string]any{"n": i},
			Slug:   "post-" + string(rune('a'+i)),
			Status: "published",
		}
		if err := store.CreateEntity(e); err != nil {
			t.Fatalf("create entity: %v", err)
		}
	}

	page, total, err := store.ListEntities(et.ID, draft.ListOpts{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("list with limit: %v", err)
	}
	if total != 5 {
		t.Errorf("total should reflect full count, got %d", total)
	}
	if len(page) != 2 {
		t.Errorf("expected 2 results for limit=2, got %d", len(page))
	}
}

func TestUpdateEntity(t *testing.T) {
	store := openTestStore(t)
	et := createTestType(t, store)
	e := createTestEntity(t, store, et.ID)

	e.Data = map[string]any{"title": "Updated Title"}
	e.Status = "archived"
	e.Slug = "updated-post"
	if err := store.UpdateEntity(e); err != nil {
		t.Fatalf("update entity: %v", err)
	}

	got, err := store.GetEntity(e.ID)
	if err != nil {
		t.Fatalf("get entity after update: %v", err)
	}
	if got.Data["title"] != "Updated Title" {
		t.Errorf("data.title not updated: got %v", got.Data["title"])
	}
	if got.Status != "archived" {
		t.Errorf("status not updated: got %q", got.Status)
	}
	if got.Slug != "updated-post" {
		t.Errorf("slug not updated: got %q", got.Slug)
	}
}

func TestUpdateEntity_notFound(t *testing.T) {
	store := openTestStore(t)

	e := &draft.Entity{
		ID:     "ghost-id",
		TypeID: "type-id",
		Data:   map[string]any{},
		Status: "draft",
	}
	err := store.UpdateEntity(e)
	if err == nil {
		t.Fatal("should return error when updating non-existent entity")
	}
}

func TestDeleteEntity(t *testing.T) {
	store := openTestStore(t)
	et := createTestType(t, store)
	e := createTestEntity(t, store, et.ID)

	if err := store.DeleteEntity(e.ID); err != nil {
		t.Fatalf("delete entity: %v", err)
	}

	_, err := store.GetEntity(e.ID)
	if err == nil {
		t.Fatal("should return error after deletion")
	}
}

func TestDeleteEntity_notFound(t *testing.T) {
	store := openTestStore(t)

	err := store.DeleteEntity("ghost-id")
	if err == nil {
		t.Fatal("should return error when deleting non-existent entity")
	}
}

func TestSetAndGetBlocks(t *testing.T) {
	store := openTestStore(t)
	et := createTestType(t, store)
	e := createTestEntity(t, store, et.ID)

	blocks := []*draft.Block{
		{Type: "heading", Data: map[string]any{"text": "Hello"}, Position: 1},
		{Type: "paragraph", Data: map[string]any{"text": "World"}, Position: 2},
		{Type: "image", Data: map[string]any{"src": "/img.png"}, Position: 3},
	}
	if err := store.SetBlocks(e.ID, "body", blocks); err != nil {
		t.Fatalf("set blocks: %v", err)
	}

	got, err := store.GetBlocks(e.ID, "body")
	if err != nil {
		t.Fatalf("get blocks: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(got))
	}

	// Verify ordering by position.
	if got[0].Type != "heading" {
		t.Errorf("block[0] type: want heading, got %q", got[0].Type)
	}
	if got[1].Type != "paragraph" {
		t.Errorf("block[1] type: want paragraph, got %q", got[1].Type)
	}
	if got[2].Type != "image" {
		t.Errorf("block[2] type: want image, got %q", got[2].Type)
	}

	// Verify IDs were auto-generated.
	for i, b := range got {
		if b.ID == "" {
			t.Errorf("block[%d] should have auto-generated ID", i)
		}
		if b.EntityID != e.ID {
			t.Errorf("block[%d] entity_id: want %q, got %q", i, e.ID, b.EntityID)
		}
		if b.Field != "body" {
			t.Errorf("block[%d] field: want body, got %q", i, b.Field)
		}
	}

	// Verify data roundtrip.
	if got[0].Data["text"] != "Hello" {
		t.Errorf("block[0].data.text: want Hello, got %v", got[0].Data["text"])
	}
}

func TestGetBlocks_emptyField(t *testing.T) {
	store := openTestStore(t)
	et := createTestType(t, store)
	e := createTestEntity(t, store, et.ID)

	got, err := store.GetBlocks(e.ID, "nonexistent")
	if err != nil {
		t.Fatalf("get blocks for empty field: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 blocks, got %d", len(got))
	}
}

func TestBlocksReplace(t *testing.T) {
	store := openTestStore(t)
	et := createTestType(t, store)
	e := createTestEntity(t, store, et.ID)

	// Set initial blocks.
	initial := []*draft.Block{
		{Type: "paragraph", Data: map[string]any{"text": "old"}, Position: 1},
		{Type: "image", Data: map[string]any{"src": "/old.png"}, Position: 2},
	}
	if err := store.SetBlocks(e.ID, "body", initial); err != nil {
		t.Fatalf("set initial blocks: %v", err)
	}

	// Replace with different blocks.
	replacement := []*draft.Block{
		{Type: "heading", Data: map[string]any{"text": "new"}, Position: 1},
	}
	if err := store.SetBlocks(e.ID, "body", replacement); err != nil {
		t.Fatalf("set replacement blocks: %v", err)
	}

	got, err := store.GetBlocks(e.ID, "body")
	if err != nil {
		t.Fatalf("get blocks after replace: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 block after replace, got %d", len(got))
	}
	if got[0].Type != "heading" {
		t.Errorf("expected heading block, got %q", got[0].Type)
	}
}

func TestBlocksReplace_differentFieldsIndependent(t *testing.T) {
	store := openTestStore(t)
	et := createTestType(t, store)
	e := createTestEntity(t, store, et.ID)

	if err := store.SetBlocks(e.ID, "intro", []*draft.Block{
		{Type: "paragraph", Data: map[string]any{"text": "intro"}, Position: 1},
	}); err != nil {
		t.Fatalf("set intro blocks: %v", err)
	}
	if err := store.SetBlocks(e.ID, "body", []*draft.Block{
		{Type: "heading", Data: map[string]any{"text": "body"}, Position: 1},
	}); err != nil {
		t.Fatalf("set body blocks: %v", err)
	}

	// Clear only body — intro must be untouched.
	if err := store.SetBlocks(e.ID, "body", []*draft.Block{}); err != nil {
		t.Fatalf("clear body blocks: %v", err)
	}

	intro, _ := store.GetBlocks(e.ID, "intro")
	if len(intro) != 1 {
		t.Errorf("intro blocks should be untouched, got %d", len(intro))
	}
	body, _ := store.GetBlocks(e.ID, "body")
	if len(body) != 0 {
		t.Errorf("body blocks should be cleared, got %d", len(body))
	}
}

func TestRelations(t *testing.T) {
	store := openTestStore(t)
	et := createTestType(t, store)
	src := createTestEntity(t, store, et.ID)

	tgt := &draft.Entity{
		TypeID: et.ID,
		Data:   map[string]any{"title": "Target"},
		Slug:   "target",
		Status: "published",
	}
	if err := store.CreateEntity(tgt); err != nil {
		t.Fatalf("create target entity: %v", err)
	}

	rel := &draft.Relation{
		SourceID:     src.ID,
		TargetID:     tgt.ID,
		RelationType: "related",
		Position:     1,
		Metadata:     map[string]any{"note": "test"},
	}
	if err := store.AddRelation(rel); err != nil {
		t.Fatalf("add relation: %v", err)
	}

	// Outgoing from source.
	out, err := store.GetRelations(src.ID, "related", draft.Outgoing)
	if err != nil {
		t.Fatalf("get outgoing relations: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 outgoing relation, got %d", len(out))
	}
	if out[0].TargetID != tgt.ID {
		t.Errorf("target_id: want %q, got %q", tgt.ID, out[0].TargetID)
	}
	if out[0].Metadata["note"] != "test" {
		t.Errorf("metadata.note: want 'test', got %v", out[0].Metadata["note"])
	}

	// Incoming to target.
	in, err := store.GetRelations(tgt.ID, "related", draft.Incoming)
	if err != nil {
		t.Fatalf("get incoming relations: %v", err)
	}
	if len(in) != 1 {
		t.Fatalf("expected 1 incoming relation, got %d", len(in))
	}
	if in[0].SourceID != src.ID {
		t.Errorf("source_id: want %q, got %q", src.ID, in[0].SourceID)
	}

	// Remove relation.
	if err := store.RemoveRelation(src.ID, tgt.ID, "related"); err != nil {
		t.Fatalf("remove relation: %v", err)
	}

	after, err := store.GetRelations(src.ID, "related", draft.Outgoing)
	if err != nil {
		t.Fatalf("get relations after remove: %v", err)
	}
	if len(after) != 0 {
		t.Errorf("expected 0 relations after remove, got %d", len(after))
	}
}

func TestRelations_typeFilter(t *testing.T) {
	store := openTestStore(t)
	et := createTestType(t, store)
	src := createTestEntity(t, store, et.ID)
	tgt := &draft.Entity{TypeID: et.ID, Data: map[string]any{}, Slug: "tgt", Status: "published"}
	if err := store.CreateEntity(tgt); err != nil {
		t.Fatalf("create target: %v", err)
	}

	_ = store.AddRelation(&draft.Relation{SourceID: src.ID, TargetID: tgt.ID, RelationType: "tag"})
	_ = store.AddRelation(&draft.Relation{SourceID: src.ID, TargetID: tgt.ID, RelationType: "author"})

	tagged, _ := store.GetRelations(src.ID, "tag", draft.Outgoing)
	if len(tagged) != 1 {
		t.Errorf("expected 1 'tag' relation, got %d", len(tagged))
	}

	all, _ := store.GetRelations(src.ID, "", draft.Outgoing)
	if len(all) != 2 {
		t.Errorf("expected 2 total relations, got %d", len(all))
	}
}

func TestRelations_removeNotFound(t *testing.T) {
	store := openTestStore(t)

	err := store.RemoveRelation("a", "b", "x")
	if err == nil {
		t.Fatal("should return error when removing non-existent relation")
	}
}

func TestTokens(t *testing.T) {
	store := openTestStore(t)

	// Get before any set — should return defaults.
	defaults, err := store.GetTokens()
	if err != nil {
		t.Fatalf("get default tokens: %v", err)
	}
	if defaults == nil {
		t.Fatal("should return non-nil default tokens")
	}
	if defaults.ID != "default" {
		t.Errorf("default token ID: want 'default', got %q", defaults.ID)
	}
	if defaults.Data.Colors["primary"] == "" {
		t.Error("default tokens should have primary color")
	}
	if defaults.Data.Scale.Spacing == 0 {
		t.Error("default tokens should have non-zero spacing")
	}

	// Set custom tokens.
	custom := &draft.TokenSet{
		Data: draft.Tokens{
			Colors: map[string]string{
				"primary": "#ff0000",
			},
			Fonts: map[string]string{
				"sans": "Roboto",
			},
			Scale: draft.ScaleTokens{
				Spacing: 1.5,
				Radius:  "lg",
				Density: "compact",
			},
			Mood: "vibrant",
		},
	}
	if err := store.SetTokens(custom); err != nil {
		t.Fatalf("set tokens: %v", err)
	}

	// Get after set.
	got, err := store.GetTokens()
	if err != nil {
		t.Fatalf("get tokens after set: %v", err)
	}
	if got.ID != "default" {
		t.Errorf("token ID after set: want 'default', got %q", got.ID)
	}
	if got.Data.Colors["primary"] != "#ff0000" {
		t.Errorf("primary color: want #ff0000, got %q", got.Data.Colors["primary"])
	}
	if got.Data.Scale.Radius != "lg" {
		t.Errorf("scale.radius: want 'lg', got %q", got.Data.Scale.Radius)
	}
	if got.Data.Mood != "vibrant" {
		t.Errorf("mood: want 'vibrant', got %q", got.Data.Mood)
	}
	if got.UpdatedAt == 0 {
		t.Error("updated_at should be set")
	}
}

func TestTokens_setTwiceReplaces(t *testing.T) {
	store := openTestStore(t)

	_ = store.SetTokens(&draft.TokenSet{Data: draft.Tokens{Colors: map[string]string{"primary": "#111"}, Scale: draft.ScaleTokens{Spacing: 1}}})
	_ = store.SetTokens(&draft.TokenSet{Data: draft.Tokens{Colors: map[string]string{"primary": "#222"}, Scale: draft.ScaleTokens{Spacing: 2}}})

	got, err := store.GetTokens()
	if err != nil {
		t.Fatalf("get tokens: %v", err)
	}
	if got.Data.Colors["primary"] != "#222" {
		t.Errorf("second set should replace: got %q", got.Data.Colors["primary"])
	}
}

func TestVersioning(t *testing.T) {
	store := openTestStore(t)
	et := createTestType(t, store)
	e := createTestEntity(t, store, et.ID)

	// Save first version.
	if err := store.SaveVersion(e.ID); err != nil {
		t.Fatalf("save version 1: %v", err)
	}

	// Update entity and save second version.
	e.Data = map[string]any{"title": "Revised Post"}
	if err := store.UpdateEntity(e); err != nil {
		t.Fatalf("update entity: %v", err)
	}
	if err := store.SaveVersion(e.ID); err != nil {
		t.Fatalf("save version 2: %v", err)
	}

	// List versions — should be 2, newest first.
	versions, err := store.ListVersions(e.ID)
	if err != nil {
		t.Fatalf("list versions: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}
	if versions[0].Version != 2 {
		t.Errorf("first listed should be version 2 (newest), got %d", versions[0].Version)
	}
	if versions[1].Version != 1 {
		t.Errorf("second listed should be version 1, got %d", versions[1].Version)
	}

	// Get specific version.
	v1, err := store.GetVersion(e.ID, 1)
	if err != nil {
		t.Fatalf("get version 1: %v", err)
	}
	if v1.EntityID != e.ID {
		t.Errorf("version entity_id: want %q, got %q", e.ID, v1.EntityID)
	}
	if v1.Version != 1 {
		t.Errorf("version number: want 1, got %d", v1.Version)
	}
	if v1.Data == "" {
		t.Error("version data should not be empty")
	}
	if v1.ChangedAt == 0 {
		t.Error("version changed_at should be set")
	}
}

func TestVersioning_notFound(t *testing.T) {
	store := openTestStore(t)
	et := createTestType(t, store)
	e := createTestEntity(t, store, et.ID)

	_, err := store.GetVersion(e.ID, 999)
	if err == nil {
		t.Fatal("should return error for non-existent version")
	}
}

func TestVersioning_unknownEntity(t *testing.T) {
	store := openTestStore(t)

	err := store.SaveVersion("nonexistent-id")
	if err == nil {
		t.Fatal("should return error when saving version for non-existent entity")
	}
}

func TestVersioning_snapshotsBlocks(t *testing.T) {
	store := openTestStore(t)
	et := createTestType(t, store)
	e := createTestEntity(t, store, et.ID)

	_ = store.SetBlocks(e.ID, "body", []*draft.Block{
		{Type: "paragraph", Data: map[string]any{"text": "snap"}, Position: 1},
	})
	if err := store.SaveVersion(e.ID); err != nil {
		t.Fatalf("save version with blocks: %v", err)
	}

	v, err := store.GetVersion(e.ID, 1)
	if err != nil {
		t.Fatalf("get version: %v", err)
	}
	// Blocks JSON should contain the paragraph block.
	if !strings.Contains(v.Blocks, "paragraph") {
		t.Errorf("version blocks snapshot should include block type, got: %q", v.Blocks)
	}
}

func TestAuth(t *testing.T) {
	store := openTestStore(t)

	// No admin users initially.
	has, err := store.HasAdminUsers()
	if err != nil {
		t.Fatalf("has admin users: %v", err)
	}
	if has {
		t.Error("should have no admin users initially")
	}

	// Create admin user.
	if err := store.CreateAdminUser("admin", "secret123"); err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	// Has admin users now.
	has, err = store.HasAdminUsers()
	if err != nil {
		t.Fatalf("has admin users after create: %v", err)
	}
	if !has {
		t.Error("should have admin users after creation")
	}

	// Correct password returns true.
	ok, err := store.ValidateCredentials("admin", "secret123")
	if err != nil {
		t.Fatalf("validate correct password: %v", err)
	}
	if !ok {
		t.Error("should validate correct password as true")
	}

	// Wrong password returns false.
	ok, err = store.ValidateCredentials("admin", "wrongpass")
	if err != nil {
		t.Fatalf("validate wrong password: %v", err)
	}
	if ok {
		t.Error("should validate wrong password as false")
	}

	// Unknown username returns false.
	ok, err = store.ValidateCredentials("nobody", "secret123")
	if err != nil {
		t.Fatalf("validate unknown user: %v", err)
	}
	if ok {
		t.Error("should return false for unknown username")
	}
}

func TestAuth_duplicateUsername(t *testing.T) {
	store := openTestStore(t)

	if err := store.CreateAdminUser("admin", "pass1"); err != nil {
		t.Fatalf("create first admin: %v", err)
	}
	err := store.CreateAdminUser("admin", "pass2")
	if err == nil {
		t.Fatal("should return error for duplicate username")
	}
}

func TestCascadeDelete_entityCleansUpBlocksAndRelations(t *testing.T) {
	store := openTestStore(t)
	et := createTestType(t, store)
	src := createTestEntity(t, store, et.ID)

	tgt := &draft.Entity{TypeID: et.ID, Data: map[string]any{}, Slug: "tgt", Status: "published"}
	if err := store.CreateEntity(tgt); err != nil {
		t.Fatalf("create target entity: %v", err)
	}

	// Add blocks to src.
	_ = store.SetBlocks(src.ID, "body", []*draft.Block{
		{Type: "paragraph", Data: map[string]any{"text": "hello"}, Position: 1},
	})

	// Add relation from src to tgt.
	_ = store.AddRelation(&draft.Relation{
		SourceID:     src.ID,
		TargetID:     tgt.ID,
		RelationType: "related",
	})

	// Delete src entity.
	if err := store.DeleteEntity(src.ID); err != nil {
		t.Fatalf("delete entity: %v", err)
	}

	// Blocks should be gone (cascade).
	blocks, err := store.GetBlocks(src.ID, "body")
	if err != nil {
		t.Fatalf("get blocks after cascade delete: %v", err)
	}
	if len(blocks) != 0 {
		t.Errorf("expected blocks to be cascade-deleted, got %d", len(blocks))
	}

	// Outgoing relation from deleted src should be gone (cascade).
	rels, err := store.GetRelations(src.ID, "", draft.Outgoing)
	if err != nil {
		t.Fatalf("get relations after cascade delete: %v", err)
	}
	if len(rels) != 0 {
		t.Errorf("expected relations to be cascade-deleted, got %d", len(rels))
	}

	// Incoming relation to tgt should also be gone.
	incoming, err := store.GetRelations(tgt.ID, "", draft.Incoming)
	if err != nil {
		t.Fatalf("get incoming relations after cascade delete: %v", err)
	}
	if len(incoming) != 0 {
		t.Errorf("expected incoming relation to be cascade-deleted, got %d", len(incoming))
	}
}

func TestCascadeDelete_typeCascadesToEntitiesAndBlocks(t *testing.T) {
	store := openTestStore(t)
	et := createTestType(t, store)
	e := createTestEntity(t, store, et.ID)

	_ = store.SetBlocks(e.ID, "body", []*draft.Block{
		{Type: "paragraph", Data: map[string]any{"text": "content"}, Position: 1},
	})

	// Delete type — should cascade to entities, then blocks.
	if err := store.DeleteType(et.ID); err != nil {
		t.Fatalf("delete type: %v", err)
	}

	_, err := store.GetEntity(e.ID)
	if err == nil {
		t.Error("entity should be gone after type deletion")
	}

	blocks, err := store.GetBlocks(e.ID, "body")
	if err != nil {
		t.Fatalf("get blocks: %v", err)
	}
	if len(blocks) != 0 {
		t.Errorf("blocks should be cascade-deleted, got %d", len(blocks))
	}
}
