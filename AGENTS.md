# go-mcp-markdown-notes - Agent Instructions

## Security

- Never copy, print, log, or commit real credentials, passwords, tokens, keys,
  cookies, or connection strings from `.env`, `.env_testing`, local config, or
  command output.
- Use obvious placeholders in examples, such as `<db-password>`,
  `<jwt-secret>`, `<dev-token>`, and `pat_<redacted>`.
- Do not read or display secret-bearing environment files unless the task
  explicitly requires inspecting their structure. Redact all values.
- Never print or copy environment values from `.env`, `.env_testing`,
  `.env-testing-export`, or the shell environment while diagnosing tests or
  database configuration.
- Local environment files and `.env-testing-export` are ignored by Git, but
  that does not make their contents safe to expose.
- Never put a real MCP token in committed configuration such as `.mcp.json`.
- Values in `.env` must not be surrounded by quotes. The Makefile includes and
  exports the file, so quotes become part of the value.

## Repository Overview

This repository contains:

- A Go notes service using ConnectRPC and PostgreSQL.
- A Bun/TypeScript single-page frontend embedded in the server binary.
- A generic MCP stdio server that calls the notes service over ConnectRPC.
- JWT, personal access token (PAT), and explicit local-development
  authentication support.

Key paths:

```text
cmd/notes-server/                 HTTP/Connect server
cmd/notes-server/notesFront/src/  Frontend source
cmd/notes-mcp/                    MCP stdio entry point
cmd/notes-client/                 Example CLI client
pkg/notes/                        Notes service and PostgreSQL repository
pkg/notes/module/                 Importable domain module for bundle-ready architecture
pkg/authadapter/                  JWT, PAT, and development token verification
pkg/mcpnotes/                     MCP tools and authenticated Connect client
proto/                            Protobuf source
gen/                              Generated Go and Connect code
api/openapi/                      Generated OpenAPI output
```

## Commands

- `make run`: build frontend assets, download Go modules, and run the server
  from source. Default listen address: `127.0.0.1:8080`.
- `make build`: run tests and build `bin/notes-server`.
- `make build-mcp`: build `bin/notes-mcp`.
- `make run-mcp`: run the MCP server from source; requires `NOTES_TOKEN`.
- `make test`: run all Go package tests with the race detector and write
  `coverage.out`. If `.env_testing` exists, it is exported through the ignored
  `.env-testing-export` file.
- `make lint`: run `go vet ./...` and `buf lint`.
- `make fmt`: run `gofmt -w .`.
- `make generate`: lint protobuf files, update Buf dependencies, and regenerate
  Go protobuf, Connect, and OpenAPI outputs.
- `make db-status`, `make db-up`, `make db-down`: inspect, apply, or roll back
  dbmate migrations using `.env`.

Do not claim that `make test` runs frontend tests or that `make lint` checks
frontend code; neither target currently does so.

## Authentication

`NOTES_AUTH_MODE` supports:

- `jwt` (default): non-`pat_` bearer tokens are verified locally. The verifier
  requires `JWT_SECRET`, `JWT_ISSUER_ID`, and `JWT_CONTEXT_KEY` compatible with
  the token issuer. JWT duration is configurable with
  `JWT_DURATION_MINUTES`.
- `jwt` with PATs: `pat_` bearer tokens are verified through
  `<AUTH_SERVER_URL>/goapi/v1/auth/introspect`. Positive results are cached for
  60 seconds.
- `dev`: accepts the configured `NOTES_DEV_TOKEN`. There is no built-in token
  default; the server rejects dev mode when it is unset. Development user
  fields come from `NOTES_DEV_USER_ID`, `NOTES_DEV_USER_EMAIL`, and
  `NOTES_DEV_USER_NAME`.

`AUTH_SERVER_URL` defaults to `http://localhost:9090`. It is used by the
frontend login/token flow and PAT introspection; ordinary JWT verification is
local. Browser login providers, including Google or GitHub, depend on the
external auth service configuration and are not implemented in this
repository.

## MCP

- `notes-mcp` communicates with its MCP client over stdio. Do not write protocol
  logs to stdout; application logging belongs on stderr.
- It calls the notes server over HTTP/ConnectRPC.
- `NOTES_TOKEN` is required. It may contain a PAT or the configured dev token.
- `NOTES_SERVER` defaults to `http://127.0.0.1:8080`.
- Keep client registration examples generic and use redacted token
  placeholders.

## Generated Code and Protobuf

- Treat `proto/` as the source of truth for RPC schemas.
- Never hand-edit files under `gen/` or generated files under `api/openapi/`.
- After changing protobuf definitions, run `make generate`.
- `make generate` may change `buf.lock`, `gen/`, and `api/openapi/`; review and
  commit all intended generated changes together.
- Generation requires Buf, local `protoc-gen-go` and
  `protoc-gen-connect-go` executables, and access to the configured remote
  OpenAPI plugin.
- Run `make lint` and relevant Go tests after regeneration.

## Evolving the Proto / Note Contract (keeping layers in sync)

`proto/notes/v1/notes.proto` is the contract for the public RPC API. Changing it
automatically updates the generated client/server types via `make generate`.

However, the following layers are **not** generated and must be updated manually:

- Database schema: `pkg/notes/module/db/migrations/*.sql` (always add a new migration; never rewrite an applied one).
- Raw SQL and column projections: `pkg/notes/sql.go` (noteColumns, searchNoteColumns, DML).
- Internal domain model: `pkg/notes/model.go` (Note struct + *Input types + NoteStatus). The `db:"..."` tags power named scanning.
- Mappers: `pkg/notes/mappers.go`.
- Business rules + normalization: `pkg/notes/service.go` (when adding visible fields).
- Repository scanning: `pkg/notes/storage_postgres.go` (now uses pgx named scanning via `RowTo*ByNameLax`, which relies on db tags).
- Wire adapters: `pkg/notes/connect_server.go`.
- MCP tools: `pkg/mcpnotes/server.go` (new Input structs + conversion).
- Example client: `cmd/notes-client/`.
- Frontend: `cmd/notes-server/notesFront/src/`.

### Recommended checklist when adding a field to Note (or similar change)

1. Edit `proto/notes/v1/notes.proto` (add to message + validation if appropriate).
2. Run `make generate` + `make lint`.
3. If the field is stored: add a new migration under `pkg/notes/module/db/migrations/`.
4. Update `noteColumns` / `searchNoteColumns` and any affected SQL in `pkg/notes/sql.go`.
5. Add the field + `json` + `db` tag to `Note` (and to Create/UpdateInput if user-supplied).
6. Update `DomainNoteToProto` / `ProtoNoteToDomain` in `mappers.go`.
7. Update normalization / validation / defaults in `service.go` if needed.
8. Wire the field in `connect_server.go` (Create/Update paths).
9. Update MCP tool inputs/outputs + conversion in `pkg/mcpnotes/`.
10. Update `cmd/notes-client` flag handling + request building if the field is useful from CLI.
11. Update tests (especially roundtrips in `pkg/notes/mappers_test.go` and any storage behavior tests).
12. Run focused tests: `go test ./pkg/notes/... -run 'Note|Mapper|Proto|Round' -count=1`
13. Run `make lint`.
14. If you changed migrations, also update the expected statement count in `TestModuleParseDBMateUp_ParsesMigration` (in `pkg/notes/module/module_test.go`).

Named scanning (introduced 2026) means you usually do **not** need to touch every `Scan(...)` call when adding columns — only the struct + SQL projection.

See `pkg/notes/model.go` and `pkg/notes/sql.go` for more detailed comments.

## Database Migrations

Migration SQL files live in exactly one place:

- `pkg/notes/module/db/migrations/` — embedded in the importable module; applied
  at startup via `notesmodule.Migrate` under a PostgreSQL advisory lock keyed to
  `'go-mcp-markdown-notes:migrations'`. `make db-up/down` also reads this
  directory for manual inspection and rollbacks.

Additional rules:

- Follow the existing zero-padded sequential naming convention (`000002_…sql`),
  with dbmate `-- migrate:up` and `-- migrate:down` sections.
- Never rewrite a migration that may already have been applied. Add a new one
  instead.
- After adding a migration, update the expected statement count in
  `TestModuleParseDBMateUp_ParsesMigration` in `pkg/notes/module/module_test.go`.
- `make db-up` is available for explicit pre-application; it is not required
  before every server start.
- Use `make db-down` only deliberately: it rolls back the latest dbmate
  migration. There is no module-level rollback; the advisory lock only covers
  forward migrations.
- Never run `make db-up`, `make db-down`, or migration-enabled server startup
  against an unknown, shared, staging, or production database without explicit
  approval and verified configuration.
- Schema changes require focused migration/parser tests and repository tests
  where behavior changes.

## Module & Bundle Architecture

`pkg/notes/module` exposes the notes domain as an importable Go package. The
two operating modes are:

- **Standalone** (`cmd/notes-server`): calls `notesmodule.Migrate`, then
  `notesmodule.New`, then `mod.RegisterRoutes(mux)`. This is the only mode
  currently in production.
- **Bundle**: a foreign repository imports the module, passes its own shared
  `*pgxpool.Pool`, `TokenVerifier`, and `*slog.Logger`, and mounts the returned
  `ConnectHandlers()` on its shared `http.ServeMux`.

Key rules:

- `New` validates deps at construction time: `Pool` and `Verifier` are required;
  a nil `Logger` silently falls back to `slog.Default()`.
- The interceptor chain (timeout → auth → proto validation) is assembled once
  inside `connectOption()`. Do not duplicate or reorder it.
- `Start` and `Stop` are currently no-ops but must be called by bundle callers
  so future background workers are picked up automatically.
- Do not import `cmd/notes-server` packages from this module. Dependency flow is
  one-way: `cmd` → `pkg`.
- `pkg/notes/module/AGENTS.md` contains detailed guidance for changes scoped to
  that package.

## Frontend

- Edit frontend source under `cmd/notes-server/notesFront/src/`.
- Use Bun for frontend dependency and build commands.
- `make build-frontend` writes `notesFront/dist/`; that directory is generated
  and ignored by Git.
- Never hand-edit or commit `dist/`.
- The Go server embeds `notesFront/dist/*`, so build frontend assets before a
  direct Go build from a clean checkout. `make run` and `make build` do this
  automatically.
- There is currently no frontend test or lint target in the root Makefile.

## Testing and Change Discipline

- Keep changes scoped and follow existing package boundaries and patterns.
- Add or update tests for behavioral changes. Prefer focused
  `go test ./path/...` for local changes; reserve `make test` for broader
  verification.
- Run `make lint` for Go or protobuf changes. Run `gofmt` only on touched Go
  files; use `make fmt` only when repository-wide formatting is intended.
- Do not manually edit generated files to make tests pass.
- Do not revert unrelated work in a dirty worktree.

## API Routing

Connect procedures use:

```text
/notes.v1.NotesService/<Method>
```

Additional server endpoints include `/config`, `/health`, `/readiness`, and
`/goAppInfo`.
