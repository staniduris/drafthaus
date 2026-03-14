# Drafthaus

Your entire digital presence in one file.

Drafthaus is a single-binary CMS that stores your complete website in a portable `.draft` file (SQLite). No database server, no runtime dependencies, no Docker — just one 16MB binary and one file.

## Quick Start

```bash
# Build
go build -o drafthaus ./cmd/drafthaus/

# Create a site
./drafthaus init mysite --template blog

# Serve it
./drafthaus serve mysite.draft
# Open http://localhost:3000
```

## Features

**Content**
- Typed entity system — define any content type with fields (text, richtext, currency, enum, boolean, datetime, etc.)
- Block-based rich content — paragraphs, headings, code blocks, callouts (not HTML blobs)
- Relational content graph — entities connect to each other with typed relations
- Full version history on every entity

**Rendering**
- 25+ built-in primitives (Stack, Grid, Columns, Card, Image, Price, Badge, etc.)
- Design token system — colors, fonts, spacing, radius generate all CSS
- Google Fonts auto-loading
- Auto-generated meta tags, OpenGraph, JSON-LD, sitemap, RSS

**Admin**
- Built-in admin panel at `/_admin` — entity editor, block editor, token manager
- Session-based authentication with bcrypt passwords
- Analytics dashboard with cookieless visitor tracking

**API**
- Admin REST API at `/_api/*` — full CRUD for everything
- Public read-only API at `/api/v1/content/:type/:slug`
- RSS feed at `/feed.xml`

**AI**
- Generate sites from natural language descriptions
- Supports OpenAI, Anthropic, and Ollama (local) providers
- Or provide a JSON spec file directly

**CLI**
```
drafthaus init <name> [--template blog|cafe|portfolio|business|blank]
drafthaus serve <file.draft> [--port 3000]
drafthaus export <file.draft> <output-dir>
drafthaus generate <name> "<description>" [--provider anthropic --api-key KEY]
drafthaus generate <name> --spec <spec.json>
drafthaus passwd <file.draft> <username> <new-password>
```

## Templates

Five built-in templates with sample content:

- **blog** — posts, authors, tags, pages
- **cafe** — menu items with categories and prices, pages
- **portfolio** — projects with descriptions, pages
- **business** — services, team members, pages
- **blank** — empty site with default tokens

## Architecture

```
.draft file (SQLite)
    |
    +-- Entity Types (schema definitions)
    +-- Entities (content instances)
    +-- Blocks (rich content units)
    +-- Relations (typed edges)
    +-- Assets (binary files)
    +-- Views (component trees as JSON)
    +-- Tokens (design system values)
    +-- Versions (entity history)
    +-- Analytics (page views)
```

The binary reads the `.draft` file and renders HTML on each request. Views are component trees (not templates) — data structures that map entity fields to primitives. All CSS is generated from design tokens at runtime.

## The .draft File

A `.draft` file is a standard SQLite database. You can:
- Copy it to another machine and serve it immediately
- Back it up by copying one file
- Inspect it with any SQLite client
- Version control it in git (binary, but portable)

## Development

```bash
make build    # Build binary
make test     # Run tests
make demo     # Create and serve a demo site
make clean    # Remove build artifacts
```

## Dependencies

Three Go module dependencies:
- `modernc.org/sqlite` — pure Go SQLite (no CGO, cross-compiles everywhere)
- `github.com/google/uuid` — entity IDs
- `golang.org/x/crypto` — bcrypt for admin passwords

Zero runtime dependencies.

## License

MIT
