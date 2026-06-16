# pkg — Shared Packages

Importable packages that are independent of any specific binary. Binaries under
`cmd/` depend on these; packages here must not import from `cmd/`.

## Package Overview

### `pkg/notes`

Core notes domain: types, repository interface, PostgreSQL implementation,
service layer, Connect RPC handler, proto mappers, and domain errors.

| File | Purpose |
|------|---------|
| `model.go` | Domain types (`Note`, `NoteFilter`) |
| `repository.go` | `Repository` interface |
| `storage_postgres.go` | PostgreSQL implementation using `pgx/v5` |
| `sql.go` | SQL query constants |
| `service.go` | Business rules, validation, tag normalisation |
| `connect_server.go` | Connect RPC handler (`NotesServiceHandler`) |
| `mappers.go` | Proto ↔ domain type converters |
| `errors.go` | Sentinel errors (`ErrNotFound`, `ErrForbidden`, `ErrConflict`) |
| `doc.go` | Package-level godoc |

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
