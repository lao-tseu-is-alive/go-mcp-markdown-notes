# pkg/notes/module — Agent Instructions

## Purpose

This package exposes the notes domain as an importable Go module so that
another repository can embed it alongside other domains in a single binary —
sharing one HTTP server, one database pool, and one auth setup. The two
operating modes are:

- **Standalone**: `cmd/notes-server` calls `New` then `RegisterRoutes`.
- **Bundle**: a bundle repository calls `ConnectHandlers()` and mounts the
  returned handlers on its own shared `http.ServeMux`.

## Package Layout

```text
module.go          Module struct, Config, Deps, New, Start, Stop
routes.go          ConnectHandlers, RegisterRoutes, RoutePatterns, ConnectPatterns
migrate.go         Migrate (advisory lock), ParseDBMateUp, Migrations embed.FS
module_test.go     External tests (package module_test); no real DB required
db/migrations/     Embedded SQL files in DBMate format
```

## Design Invariants

- **`New` validates all required deps up front.** A nil `Pool` or nil `Verifier`
  is an immediate error. A nil `Logger` falls back to `slog.Default()`.
- **Interceptor chain is fixed and applied once.** `connectOption()` builds
  timeout → auth → proto validation. Do not split or reorder this chain.
- **`Migrate` acquires a PostgreSQL advisory lock** keyed to
  `'go-mcp-markdown-notes:migrations'`. In a bundle, every module must use a
  unique advisory lock key to avoid cross-module deadlocks.
- **`Migrations` is an exported `embed.FS`.** Bundle callers and tests may read
  it directly. Do not rename it without updating callers.
- **`ParseDBMateUp` is exported for testing only.** It is the public surface for
  verifying the parser against embedded SQL; do not use it as a general-purpose
  DBMate utility.

## Modifying Migrations

- Migration files live in `db/migrations/` and are embedded at build time.
  This is the single authoritative location; `make db-up/down` also reads here.
- Follow the existing zero-padded sequential naming convention
  (`000002_add_column.sql`, etc.).
- Never rewrite an already-applied migration. Add a new one instead.
- After adding a migration, verify that `TestModuleParseDBMateUp_ParsesMigration`
  counts the right number of statements (update the expected count in the test).

## Testing Without a Database

`module_test.go` uses `pgxpool.New` with a fake DSN. The pool is lazy — it
never dials during module construction, so tests pass without a running
PostgreSQL. Tests that need real DB interaction belong in integration test files
and must be gated behind a build tag or `testing.Short()` check.

## Adding a New RPC Method

1. Define the method in `proto/notes/v1/notes.proto`.
2. Run `make generate` from the repository root.
3. Implement the handler in `pkg/notes/connect_server.go`.
4. No changes to this package are needed unless the method requires a new route
   prefix (Connect handlers are discovered automatically through the generated
   service handler registration).

## Bundle Integration Checklist

When importing this module from another repository:

- Call `Migrate(ctx, pool)` once at startup before calling `New`.
- Pass the same `*pgxpool.Pool` you use for other modules; the notes module does
  not open its own connection.
- Use `ConnectHandlers()` to get `(pattern, http.Handler)` pairs and mount them
  on your shared mux. `RegisterRoutes` is a convenience wrapper for standalone
  use only.
- Check `RoutePatterns()` or `ConnectPatterns()` to confirm no prefix collisions
  with other modules before mounting.
- `Start` and `Stop` are currently no-ops but should still be called so future
  background workers are picked up automatically.

## What Not to Change Here

- Do not embed transport logic (TLS, port binding) in this package.
- Do not import `cmd/notes-server` packages. Dependencies flow one way: cmd →
  pkg.
- Do not add a `main` function or `init` side effects.
- Do not store module-level state outside the `Module` struct; `New` must be
  safe to call multiple times in the same process (e.g., in tests).
