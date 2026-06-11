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
cmd/notes-server/                 HTTP/Connect server and embedded migrations
cmd/notes-server/notesFront/src/  Frontend source
cmd/notes-mcp/                    MCP stdio entry point
cmd/notes-client/                 Example CLI client
pkg/notes/                        Notes service and PostgreSQL repository
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

## Database Migrations

- Migration source lives in `cmd/notes-server/db/migrations/`.
- Follow the existing zero-padded sequential naming convention for new `.sql`
  migration files, with dbmate
  `-- migrate:up` and `-- migrate:down` sections.
- Never rewrite a migration that may already have been applied. Add a new
  migration instead.
- The server embeds migrations and automatically applies pending up migrations
  during startup under a PostgreSQL advisory lock.
- `make db-up` is available for explicit pre-application; it is not required
  before every server start.
- Use `make db-down` only deliberately: it rolls back the latest migration.
- Never run `make db-up`, `make db-down`, or migration-enabled server startup
  against an unknown, shared, staging, or production database without explicit
  approval and verified configuration.
- Schema changes require focused migration/parser tests and repository tests
  where behavior changes.

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
