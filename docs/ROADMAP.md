# Drafthaus Roadmap

> Drafthaus is a digital presence operating system. Every person or business gets a single `.draft` file — a self-contained SQLite database holding their entire digital identity. A website is just one projection of that file. The binary is a single Go executable with zero dependencies.

This document covers the path from v0.2 through v1.0. Version 0.1 (currently in development) establishes the foundation: the `.draft` file format, a component-tree renderer with 25 primitives, design tokens, entity-type-driven routing, a REST API, and a CLI with `init`, `serve`, and `export` commands.

---

## v0.2 — Admin UI & Editing

**Theme:** Make the `.draft` file editable by humans, not just APIs.

### Features

- Built-in web admin dashboard served directly from the Go binary (embedded assets, no separate build step or Node dependency)
- Block editor for rich content composition — structured blocks similar to Notion, not a WYSIWYG HTML editor
- Visual entity type editor: define fields, relations, and constraints without writing JSON
- Live preview panel that re-renders the public site as content changes
- Drag-and-drop asset upload with automatic metadata extraction
- Basic authentication (username + bcrypt-hashed password stored in the `.draft` file) for admin access
- WordPress XML (WXR) importer: maps posts/pages/categories/media to Drafthaus entities and blocks

### Key Technical Decisions

- Admin UI built with vanilla JS or a lightweight framework (Preact/Alpine) to keep the embedded bundle small — no React
- All admin state flows through the existing REST API; the UI is a first-party API consumer
- Auth uses HTTP-only cookies with CSRF protection; no JWTs
- WordPress import runs as a CLI command (`drafthaus import wordpress export.xml`) that writes directly to the `.draft` file

### What This Enables

Non-technical users can create and manage content without touching the CLI or API. A single person can download the binary, run `drafthaus init`, open the admin, and build a site entirely in the browser. WordPress migration provides a concrete on-ramp for the largest CMS user base.

---

## v0.3 — AI Layer

**Theme:** Describe what you want; Drafthaus builds it.

### Features

- Natural language site generation: describe your business in plain text and get a populated `.draft` file with entity types, sample content, design tokens, and routing
- AI content generation for individual entities and blocks (write a blog post, generate a product description)
- Design token generation from natural language ("make it warmer," "more brutalist," "use my brand color #2A4B7C")
- AI-powered SEO suggestions: title/meta analysis, keyword density, readability scoring
- Dual model support: local inference via Ollama (Mistral, Llama) with automatic fallback to cloud APIs (OpenAI, Anthropic)
- Content repurposing pipeline: take a blog post and generate social media posts, a newsletter draft, and an RSS summary

### Key Technical Decisions

- AI features are optional — the binary works identically without any model configured
- Model communication via a unified adapter interface (`AIProvider`) so new backends can be added without changing calling code
- Prompts are stored as templates in the binary, not hardcoded in Go functions, to allow iteration without recompilation
- Local model support uses HTTP calls to Ollama's API (not embedded inference) to keep the binary size manageable
- Content repurposing outputs are stored as draft entities (not auto-published) so the user always reviews before projecting

### What This Enables

A bakery owner in Bratislava types three sentences about their business and gets a working site with pages, a menu, opening hours, and a contact form — all populated with real content. Ongoing content creation drops from hours to minutes. Users who cannot afford designers get reasonable aesthetics through natural language token tuning.

---

## v0.4 — Projection Adapters

**Theme:** One source of truth, projected everywhere.

### Features

- Google Business Profile sync: push hours, descriptions, posts, and photos from the `.draft` file; pull reviews and Q&A back in (bidirectional via GBP API)
- RSS/Atom feed generation for any entity type (not just blog posts)
- Email newsletter composition in the admin UI with SMTP sending (no third-party service required)
- PDF export: generate menus, catalogs, price lists, and reports from entity data using a Go PDF library
- Social media post generation: format entity content for Instagram, Facebook, and LinkedIn with platform-specific constraints
- JSON-LD / structured data auto-generation expanded beyond v0.1's partial support (full coverage of Schema.org types relevant to local business, articles, products, events)
- Sitemap.xml generation enhanced with priority hinting and change frequency based on entity update patterns

### Key Technical Decisions

- Each projection adapter implements a `Projector` interface: `Project(entity) -> []byte` + `Sync(draft) -> error`
- Google Business Profile uses OAuth2 with refresh tokens stored in the `.draft` file (encrypted at rest)
- PDF generation uses a pure-Go library (e.g., `go-pdf/fpdf` or `johnfercher/maroto`) — no external dependencies like wkhtmltopdf
- Newsletter SMTP credentials are stored in the `.draft` file's config table; no built-in mail server
- Social media adapters generate content but do not auto-post (v0.4 scope is generation; API posting comes later)

### What This Enables

A restaurant updates their menu once in Drafthaus. The website updates instantly. Google Business Profile syncs within minutes. A PDF menu is downloadable from the site. An Instagram caption is ready to copy. The newsletter goes out with the same content reformatted for email. One edit, every channel.

---

## v0.5 — Real-time & Collaboration

**Theme:** Multiple people, one `.draft` file, no conflicts.

### Features

- WebSocket connection from admin UI for real-time content updates
- CRDT-based conflict resolution so concurrent edits merge deterministically without data loss
- Multi-user editing with live presence indicators (cursors, active entity highlighting)
- Role-based access control: admin (full access), editor (content only), viewer (read-only)
- Commenting system on entities and individual blocks
- Activity log and audit trail stored in the `.draft` file

### Key Technical Decisions

- CRDT implementation via Yjs compiled to WASM and called from Go using wazero, or a native Go CRDT library if one matures sufficiently by this point
- WebSocket messages are operation-based (not full-state sync) to minimize bandwidth
- Roles and permissions are stored in the `.draft` file; user records include bcrypt-hashed passwords and role assignments
- Audit trail is append-only and queryable via the REST API
- Presence is ephemeral (in-memory) and not persisted to SQLite

### What This Enables

A small team — owner, writer, designer — can work on the same site simultaneously without stepping on each other. An agency can give a client viewer access to approve content without risking accidental edits. Every change is traceable.

---

## v0.6 — WASM Plugin System

**Theme:** Extend Drafthaus without forking it.

### Features

- Sandboxed plugin runtime using wazero (pure-Go WebAssembly runtime, no CGO, no system dependencies)
- Custom component primitives: plugins can register new renderable components beyond the built-in 25
- Custom behavior definitions: plugins can implement checkout flows, booking systems, contact forms, or any interactive behavior
- Plugin registry and marketplace (initially a curated GitHub-based registry, later a hosted marketplace)
- Plugin API for reading and writing the content graph (entities, blocks, relations) within sandbox constraints

### Key Technical Decisions

- Plugins are `.wasm` binaries stored in the `.draft` file's asset table or loaded from a local directory
- wazero provides memory isolation and deterministic execution — a plugin crash cannot take down the server
- Plugin API is capability-based: plugins declare required permissions (read entities, write entities, network access) and the user grants them at install time
- Plugins can be authored in any language that compiles to WASM (Rust, Go, C, AssemblyScript)
- Island architecture from v0.1 is extended so plugin components can hydrate independently on the client

### What This Enables

A developer builds a booking widget as a WASM plugin. A yoga studio installs it from the registry. The plugin adds a "booking" entity type, a calendar component, and email notification behavior — all sandboxed, all stored in the `.draft` file. No server configuration, no dependency management.

---

## v0.7 — E-commerce & Transactions

**Theme:** Sell things without bolting on Shopify.

### Features

- Product catalog entity type with variant support (size, color), inventory tracking, and pricing rules
- Stripe integration as a built-in behavior (not a plugin) for payment processing
- Shopping cart implemented as an island component with client-side state and server-side validation
- Order management interface in the admin dashboard
- Invoice generation via the PDF projection adapter from v0.4

### Key Technical Decisions

- Stripe is the only payment provider at launch; the integration uses Stripe Checkout (hosted payment page) to avoid PCI scope
- Inventory is tracked in the `.draft` file with optimistic locking to handle concurrent purchases
- Cart state uses server-side sessions (cookie-referenced, stored in SQLite) rather than client-only state to prevent tampering
- Order data is stored as entities with a dedicated `order` entity type, making them queryable and projectable like any other content
- Tax calculation is left to Stripe Tax or manual configuration — Drafthaus does not implement tax engines

### What This Enables

A ceramics studio sells their work directly from their Drafthaus site. Products are entities with photos, descriptions, and variants. Checkout goes through Stripe. Orders appear in the admin. Invoices are auto-generated PDFs. No WooCommerce, no Shopify, no monthly platform fee beyond Stripe's transaction cut.

---

## v0.8 — Analytics & Optimization

**Theme:** Measure and improve without surveillance.

### Features

- Built-in cookieless analytics: page views, referrers, devices, and geographic regions (country-level via IP geolocation) — all stored in the `.draft` file
- A/B testing for headlines, hero images, and layout variations with statistical significance calculations
- Core Web Vitals monitoring: LCP, FID/INP, CLS reported from the client and stored server-side
- Automatic image optimization: WebP/AVIF conversion, responsive srcset generation, lazy loading
- Performance budgets: define maximum page weight and render time; get warnings in the admin when exceeded

### Key Technical Decisions

- Analytics are privacy-respecting by design: no cookies, no fingerprinting, no cross-site tracking; compliant with GDPR without a consent banner
- Analytics data is stored in dedicated SQLite tables within the `.draft` file, with automatic daily rollup and configurable retention
- Image optimization runs at upload time (not on-the-fly) using pure-Go image libraries; original assets are preserved
- A/B test assignment uses deterministic hashing of the visitor's IP + date (no cookies needed, stable within a day)
- Core Web Vitals are collected via a small inline script (under 1KB) that uses the web-vitals API

### What This Enables

Site owners see how their content performs without installing Google Analytics or any third-party script. They know which headline converts better. They know their site is fast. Images are optimized automatically — no manual Photoshop export. All of this data lives in the `.draft` file, portable and private.

---

## v0.9 — Multi-site & Agency Tools

**Theme:** Manage ten sites as easily as one.

### Features

- Multi-site dashboard: manage multiple `.draft` files from a single admin interface
- White-label admin: customize the admin UI's branding per client (logo, colors, domain)
- Template marketplace: share and sell site archetypes (a "bakery" template, a "portfolio" template) as pre-populated `.draft` files
- Client billing integration: track hours or charge flat fees, generate invoices
- Bulk operations: update design tokens, deploy changes, or run imports across multiple sites simultaneously

### Key Technical Decisions

- Multi-site management is a separate mode of the same binary (`drafthaus agency`), not a different product
- Each client site remains an independent `.draft` file — no shared database, no multi-tenant complexity
- White-labeling is achieved via a config entity in each `.draft` file that overrides admin UI assets
- Template marketplace uses a simple package format: a `.draft` file plus a manifest JSON with metadata, screenshots, and pricing
- Billing integration connects to Stripe (reusing v0.7 infrastructure) for invoicing

### What This Enables

A freelance web developer manages 15 client sites from one dashboard. Each client sees a branded admin with their logo. New sites start from proven templates. The developer bills clients through the same system. All without a SaaS platform taking a cut — just `.draft` files on their own infrastructure.

---

## v1.0 — Production Ready

**Theme:** Ship it. For real.

### Features

- Drafthaus Cloud: managed hosting service with edge CDN, automatic deployments on `.draft` file save
- Automatic backups with Litestream replication to S3-compatible storage
- Custom domain management with automatic TLS certificate provisioning (Let's Encrypt)
- Passkey (WebAuthn) authentication as the default, with password fallback
- SOC2-relevant security hardening: rate limiting, input sanitization audit, dependency review, penetration testing
- Migration tools: import from Strapi, Contentful, Ghost, and Sanity with entity/field mapping
- Comprehensive documentation site (itself built with Drafthaus)
- `.draft` file format stability guarantee: v1.0 files will be readable by all future versions

### Key Technical Decisions

- Drafthaus Cloud runs the same Go binary as self-hosted — no feature gating between self-hosted and cloud
- Litestream provides continuous SQLite replication without any application-level backup logic
- TLS uses autocert (Go's `golang.org/x/crypto/acme/autocert`) for self-hosted and a managed certificate pipeline for Cloud
- Passkey implementation uses the `go-webauthn` library; credentials are stored in the `.draft` file
- The `.draft` file format is versioned with a schema version integer; migrations run automatically on open
- Migration tools are CLI commands (`drafthaus import strapi`, `drafthaus import ghost`, etc.) that map source schemas to Drafthaus entity types via configurable mapping files

### What This Enables

Drafthaus is ready for production workloads. A business can self-host with confidence (backups, TLS, auth) or use Drafthaus Cloud for zero-ops hosting. Agencies can migrate existing client sites from legacy CMSes. The file format is stable — data is safe for the long term. One binary, one file, every channel.

---

## Timeline Note

This roadmap is sequenced by dependency, not by calendar. Each version builds on the previous one. Timelines will be published per-version as development progresses. Community feedback will influence priority within versions but not the overall sequencing, which reflects genuine technical dependencies (e.g., the plugin system must exist before third-party e-commerce extensions make sense).
