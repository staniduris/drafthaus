# Drafthaus

Single-binary CMS in Go. One `.draft` file (SQLite) = one website.

## Quick Start

```bash
go build -o drafthaus ./cmd/drafthaus/
./drafthaus init mysite --template blog
./drafthaus serve mysite.draft
# Open http://localhost:3000 (site) and http://localhost:3000/_admin (admin panel)
```

## Architecture

```
cmd/drafthaus/main.go     CLI entry (init, serve, export)
internal/draft/            Storage layer — SQLite CRUD for all entities
internal/graph/            Graph query resolver (entity + relations + blocks)
internal/render/           Component tree renderer (25 primitives + CSS + meta)
internal/server/           HTTP server, router, API handlers, auth
embed/admin/               Admin UI (single HTML file, vanilla JS)
```

## Key Design Rules

- Content is NEVER stored as HTML. Blocks are typed JSON. Views are component trees.
- The .draft file must be portable — copy it anywhere, serve it, get the same site.
- Zero external runtime dependencies. Two Go module deps: modernc.org/sqlite, google/uuid.
- Primitives emit `dh-` prefixed CSS classes. All styling comes from design tokens.
- The Store interface is the only way to access SQLite. No raw SQL outside internal/draft/.

## Testing

```bash
go test ./internal/draft/ -v     # Storage layer
go test ./...                     # All tests
```

## Templates

Five built-in: `blank`, `blog`, `cafe`, `portfolio`, `business`.
Default admin credentials after init: `admin` / `admin`.
