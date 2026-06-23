# pkg — Shared Packages

Importable packages that are independent of any specific binary. Binaries under
`cmd/` depend on these; packages here must not import from `cmd/`.

## Package Overview

### `pkg/notes`

Core notes domain: types, repository interface, PostgreSQL implementation,
service layer, Connect RPC handler, proto mappers, and domain errors.

The RPC contract is defined in `proto/notes/v1/notes.proto` (generates code under `gen/`).
The persistence layer (schema migrations, raw SQL, internal model) is maintained by hand
but designed for low-friction evolution:

- `pkg/notes/storage_postgres.go` uses `pgx/v5` **named struct scanning** (`RowTo*ByNameLax`, `Collect*`) driven by `db:"..."` tags on `Note`.
- `noteColumns` / `searchNoteColumns` in `sql.go` + the `Note` struct (with `db` tags) are the main places that need updating when the table shape changes.
- Full checklist + drift-detection tests (including `TestNoteDBTagsPresent`) are documented in the root [AGENTS.md](../AGENTS.md) ("Evolving the Proto / Note Contract" section) and exercised by `pkg/notes` tests.

| File | Purpose |
|------|---------|
| `model.go` | Domain types (`Note`, `CreateNoteInput`, `UpdateNoteInput`, `SearchFilter`, `SearchResult`, `NoteStatus`) with `db` tags for named scanning |
| `repository.go` | `Repository` interface (all methods are owner-scoped) |
| `storage_postgres.go` | PostgreSQL implementation using `pgx/v5` + named struct scanning (`RowTo*ByNameLax` / `Collect*`) |
| `sql.go` | SQL query constants and column projections (noteColumns, searchNoteColumns, DML) |
| `service.go` | Business rules, input normalisation, tag handling, limits, validation |
| `connect_server.go` | Connect RPC adapter (scope checks, error mapping, mappers) |
| `mappers.go` | Converters between domain `Note` and generated `notesv1.Note` |
| `errors.go` | Sentinel errors (`ErrInvalidInput`, `ErrNoteNotFound`, `ErrUnauthenticated`) |
| `doc.go` | Package-level godoc (see root `AGENTS.md` for proto-evolution checklist) |

### `pkg/notes/module`

Importable domain module for bundle-ready architecture. Wraps `pkg/notes` with
lifecycle management, route registration, and embedded SQL migrations so another
repository can embed the notes domain in a shared binary.

| File | Purpose |
|------|---------|
| `module.go` | `Module` struct, `Config`, `Deps`, `New`, `Start`, `Stop` |
| `routes.go` | `ConnectHandlers`, `RegisterRoutes`, `RoutePatterns`, `ConnectPatterns` |
| `migrate.go` | `Migrate` (advisory lock), `ParseDBMateUp`, `Migrations` embed.FS |
| `module_test.go` | External tests; no real database required |
| `db/migrations/` | Embedded SQL migration files (DBMate format) |
| `AGENTS.md` | Detailed agent guidance for this package |

See the [Module & Bundle Strategy](../README.md#module--bundle-strategy) section
in the root README for operating modes and bundle integration instructions.

### `pkg/authadapter`

Token verification abstraction used by the Connect interceptor and the
standalone server.

| File | Purpose |
|------|---------|
| `interceptor.go` | Connect server interceptor: extracts and verifies bearer tokens |
| `context.go` | `ContextWithUser`, `UserFromContext`, `RequireUser`, `HasScope` helpers |
| `verifiers.go` | `JWTVerifier` (local JWT parsing) and `DevTokenVerifier` (dev mode only) |
| `pat_verifier.go` | `PatVerifier`: PAT introspection via auth service HTTP call, 60 s cache |
| `composite_verifier.go` | `CompositeVerifier`: routes `pat_…` tokens to PAT verifier, rest to JWT |
| `doc.go` | Package-level godoc |

`TokenVerifier` is the central interface. The `"notes:admin"` scope acts as a
wildcard and satisfies any `HasScope` check.

### `pkg/mcpnotes`

MCP stdio server that exposes note operations as tools for Claude Code and
Claude Desktop. Calls the notes server over Connect RPC using a bearer token.

| File | Purpose |
|------|---------|
| `server.go` | MCP server setup and tool registration |
| `client.go` | Authenticated Connect client for the notes service |
| `doc.go` | Package-level godoc |
| `mcp_auth_design.md` | Auth design notes (PAT flow, caching, token rotation) |

### `pkg/version`

Build-time version metadata (`AppName`, `Version`, `Revision`, `BuildStamp`,
`Repository`) injected via `-ldflags` at build time.
