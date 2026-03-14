# Drafthaus — Vision Document

## What is Drafthaus?

Drafthaus is not a CMS. It is the operating system for digital presence.

Every person and business has a digital identity scattered across a dozen platforms — WordPress, Google Business, Instagram, Yelp, Mailchimp, Booking.com, WhatsApp Business. None of these talk to each other. None are owned by the user. Any can be taken away overnight.

Drafthaus unifies this into a single portable file — the `.draft` file — that contains everything a person or business is digitally. A website is just one projection of that file. An Instagram post is another. A voice answer is another. An AI citation is another.

**Tagline: "One file. Every screen. Yours forever."**

## The iPod Analogy

Steve Jobs didn't build a better MP3 player. He unified the fragmented music experience (CDs, Napster, radio) into iTunes + iPod + Music Store.

Drafthaus doesn't build a better CMS. It unifies the fragmented digital presence (WordPress, Google, Instagram, Yelp, email) into one file + one runtime + projection adapters.

| iPod Era | Drafthaus Era |
|-|-|
| Music scattered across CDs, Napster, radio | Presence scattered across WP, Google, Instagram, Yelp |
| iTunes unified library | .draft file unified identity |
| iPod was the playback device | Website is just one playback device |
| Music Store was the ecosystem | Projections/integrations are the ecosystem |
| "1,000 songs in your pocket" | "Your entire digital presence in one file" |

## Core Principles

### 1. The .draft File is Everything

A single SQLite file contains: brand identity, structured data (products, people, posts, locations), content blocks, customer touchpoints (booking, ordering, forms), audience data (subscribers). It is portable — email it, copy it, back it up by copying one file. It is self-contained — no external database, no config files, no asset directories.

### 2. Content is a Knowledge Graph, Not Documents

There are no "pages" or "posts." There are typed entities in a relational graph. A bakery's menu item connects to ingredients, allergens, images, categories, prices. A blog post connects to authors, tags, related posts. The graph is pure semantics — what things ARE and how they relate. Never how they look.

### 3. Rendering is a Projection

The content graph projects into whatever medium asks for it:
- A browser gets a website (HTML + CSS)
- An API consumer gets JSON
- A voice assistant gets spoken answers
- An AI crawler gets structured data (JSON-LD)
- A print request gets a PDF
- A social platform gets formatted posts
- Future media get future formats

One source of truth, infinite outputs. The view layer is data (component trees stored as JSON), not code (templates). This makes it AI-generatable, output-agnostic, and future-proof.

### 4. Zero Dependencies, Single Binary

`./drafthaus serve mysite.draft` — that's it. No Docker, no database server, no Redis, no PHP, no Node runtime. Runs on a $4 VPS, a Raspberry Pi, or any machine with an OS. The binary embeds everything: the HTTP server, the renderer, the asset pipeline, the admin UI, the JS islands.

### 5. 50-Year Architecture

The .draft file must be readable in 50 years. SQLite's file format is a Library of Congress recommended preservation format. The content is typed JSON in relational tables — no proprietary binary formats, no HTML blobs, no framework-specific markup. The runtime (Go binary) is disposable and meant to be rewritten. The knowledge layer (the .draft file) is permanent.

## The Content Architecture

### Entity Types
User-defined schemas. A "Product" type has fields: name (text), price (currency), description (richtext), roast_level (enum). Types are stored as JSON field definitions — no code generation, no migration files.

### Entities
Instances of types. Each has typed data validated against its type definition, a slug for URL routing, a status (draft/published/archived), and a position for ordering.

### Blocks
Rich content stored as ordered, typed blocks — not HTML strings. A blog post body is: heading block, paragraph block, image block, code block, callout block. Each block is pure semantics. The renderer decides presentation.

### Relations
Typed edges between entities. A blog post is "authored_by" a person, "tagged_with" tags, "has_image" an asset. Relations are first-class, queryable, and traversable.

### Views
Component trees stored as JSON. A view maps entity data to a tree of primitives (Stack, Columns, Heading, Image, RichText, etc.) via bind expressions. The renderer walks the tree and emits HTML. Views are data, not code — AI can generate and modify them.

### Design Tokens
Colors, fonts, spacing, radius, density, mood — stored as structured values. The binary generates all CSS from tokens. No Tailwind, no SCSS. Change tokens, entire site changes.

## Technology Decisions

| Layer | Choice | Why |
|-|-|-|
| Language | Go | Single binary cross-compilation, large stdlib, fast compile, modernc.org/sqlite (pure Go, no CGO) |
| Storage | SQLite (embedded) | Single file, ACID, fast, portable, Library of Congress format, 2050+ support commitment |
| Rendering | Server-side HTML + JS islands | Fast by default, progressive enhancement, no hydration overhead |
| Interactivity | Vanilla JS islands (~3KB each) | No React, no framework, embedded in binary |
| Extension (future) | WASM via wazero | Sandboxed, portable, language-agnostic |
| AI (future) | Local models (Ollama) + cloud API fallback | Works offline, privacy-first |

## Business Model

The .draft file portability is the strategic moat. People trust Drafthaus because they can leave at any time. Trust builds adoption.

| Tier | Price | What |
|-|-|-|
| Free | $0 | Open-source binary, self-host forever, no limits |
| Cloud | $5-15/mo | We host it, edge CDN, backups, custom domains |
| Pro | $25-50/mo | Cloud + AI features, content generation, auto-optimization |
| Business | $49/mo | Projection adapters: Google sync, social media, email newsletters |
| Teams | Per seat | Collaboration, roles, approval workflows, audit log |
| Agency | $50-200/site/mo | Multi-site management, white-label, client billing |

The core insight: businesses already pay $200-500/mo spread across Mailchimp + Squarespace + Hootsuite + Yelp + booking software. Drafthaus collapses all of that.

## What We're NOT Building

- Another React/Next.js app with a Postgres backend
- A headless CMS that requires a separate frontend
- A website builder with drag-and-drop
- A WordPress clone with better UX

## What We ARE Building

A knowledge representation system that happens to output websites today. When the web evolves — and it will, radically — Drafthaus sites evolve with it because the knowledge layer never cared about the output format.

## Key Insight from Research

WordPress is bleeding: governance crisis (WP Engine lawsuit), security nightmare (plugins are the #1 attack vector), developer exodus to modern stacks. But no one has unified the best innovations into a radically simpler package. Everyone is building sprawling TypeScript monorepos. We build a single binary with two Go module dependencies.

## Competition Analysis

| Product | What They Do | Our Advantage |
|-|-|-|
| WordPress | PHP CMS, 43% market share | We're 20 years of architecture ahead |
| Payload CMS | TypeScript, code-first, headless | We're zero-dependency, self-contained |
| Strapi | Open-source headless CMS | We don't require Node.js + database server |
| Ghost | Clean publishing platform | We have structured content graph, not just posts |
| Webflow/Framer | Visual builders | We're open-source, self-hostable, no lock-in |
| Sanity/Contentful | Cloud headless CMS | We're local-first, portable, no API dependency |
| Lovable/v0.dev | AI code generators | We're a living system, not a generated codebase |

The critical distinction from Lovable/v0: they generate a codebase and leave. We ARE the runtime. There's no generated code to maintain because there is no code — it's a runtime that interprets content and intent directly.
