# Repository Technical Review - 2026-06-17

## 1. Executive Summary

- This is a useful, coherent personal Markdown notes service: Go HTTP/ConnectRPC backend, PostgreSQL persistence, embedded Bun/TypeScript SPA, CLI client, and MCP stdio adapter.
- The core backend shape is strong: `cmd/notes-server` validates config, pings PostgreSQL, applies embedded migrations, wires a module boundary, exposes `/health` and `/readiness`, uses JSON `slog`, sets HTTP timeouts, and performs graceful shutdown.
- Package boundaries are mostly healthy: `pkg/notes` owns domain/service/storage/transport mapping, `pkg/authadapter` owns token verification and Connect auth middleware, `pkg/mcpnotes` owns MCP tooling, and `pkg/notes/module` supports bundle-style embedding.
- The current worktree is dirty: `Makefile`, `proto/notes/v1/notes.proto`, `gen/notes/v1/notes.pb.go`, and `scripts/buf_generate.sh` are modified, and `.codex-tasks/` is untracked. Findings that touch those files should be read as current-state observations, not necessarily committed design.
- The biggest correctness risk is schema drift: `proto/notes/v1/notes.proto` now declares `NoteStatus`, request status fields, and `deleted_note_id`, while the generated Go, domain model, mappers, storage, frontend, MCP, and CLI are only partly aligned.
- The biggest production-readiness risk is container/deployment quality: the Dockerfile does not build frontend assets for the embedded FS, uses `scratch` with a `curl` healthcheck, and deployment scripts reference missing or scaffold-like Kubernetes artifacts.
- The biggest security issue is frontend HTML injection: note metadata and debug responses are inserted through `innerHTML` without escaping in `cmd/notes-server/notesFront/src/main.ts`.
- Backend tests pass with `go test ./...`, but there is no observed CI workflow, no frontend typecheck/test/lint target, no integration test against real PostgreSQL migrations/storage, and no container build test.

## 2. Repository Purpose

The project appears to provide authenticated personal Markdown notes that can be accessed through three surfaces: an embedded browser UI, a Go CLI client, and an MCP stdio server for AI assistants. It solves the practical problem of storing and retrieving scoped notes over a ConnectRPC API, with PostgreSQL durability and token-based authentication compatible with an external `go-cloud-k8s-auth` service.

The intended users are likely the repository owner and related automation clients rather than a broad SaaS audience. The README describes local development, SSO-backed JWT mode, PAT-backed MCP access, and a bundle architecture where another Go service can import the notes module.

## 3. Main Technical Scope

- Go module: `github.com/lao-tseu-is-alive/go-mcp-markdown-notes`, Go `1.25.5` in `go.mod`.
- Backend API: ConnectRPC using generated code under `gen/notes/v1` from `proto/notes/v1/notes.proto`.
- Validation: `connectrpc.com/validate` plus `buf.validate` rules in protobuf.
- Persistence: PostgreSQL through `github.com/jackc/pgx/v5/pgxpool`.
- Migrations: DBMate-format SQL duplicated under `cmd/notes-server/db/migrations/` and `pkg/notes/module/db/migrations/`; module runtime applies embedded migrations through `notesmodule.Migrate`.
- Auth: JWT verifier adapter, explicit dev-token verifier, PAT introspection verifier with 60 second positive cache, and Connect interceptor.
- Frontend: Bun build, TypeScript entrypoint, static HTML/CSS, embedded by Go `embed` under `cmd/notes-server/notesFront/dist`.
- MCP: `github.com/modelcontextprotocol/go-sdk`, stdio transport, seven note tools.
- Build/dev commands: `make run`, `make build`, `make build-mcp`, `make test`, `make lint`, `make generate`, DBMate targets.

Verification run during this review:

- `go test ./...` with temp Go cache paths passed for all packages. The first sandboxed attempt failed due the execution sandbox; the rerun outside the sandbox was read-only and passed.

Not run:

- `make test`, because it writes `coverage.out` and can read/export `.env_testing`.
- `make generate`, because the review is read-only and generation mutates `buf.lock`, `gen/`, and `api/openapi/`.
- Frontend build/typecheck, because the repository has only a Bun build script and no test/typecheck/lint target, and the review did not install or mutate dependencies.
- Container build, because it is not read-only and would require build tooling/network/runtime assumptions.

## 4. Architecture and Design

The backend architecture is compact and mostly well separated. `cmd/notes-server/main.go` handles process setup, logging, signals, and listener creation. `cmd/notes-server/server.go` builds the PostgreSQL pool, pings the DB, runs `notesmodule.Migrate`, wires auth and module dependencies, mounts Connect routes, mounts `/health`, `/readiness`, `/goAppInfo`, `/config`, and serves the SPA. This is a good shape for a small cloud-native service.

The module boundary in `pkg/notes/module` is one of the better parts of the repository. `module.New` validates `Pool` and `Verifier`, falls back to `slog.Default()` for a nil logger, and returns route metadata and pre-wired Connect handlers. `routes.go` builds a single interceptor chain in one place: timeout, auth, proto validation. It also wraps Connect with `http.MaxBytesHandler` at 1 MiB, which is appropriate given the note body limit.

The domain package has a sensible split: `model.go`, `repository.go`, `service.go`, `storage_postgres.go`, `sql.go`, `connect_server.go`, `mappers.go`, and sentinel errors. Service methods validate owner IDs, normalize titles/tags/limits, and enforce owner isolation by requiring `ownerUserID` on every repository method. Connect handlers map domain errors to Connect status codes and avoid leaking unexpected internals.

Storage is straightforward and transaction-aware for mutating note/tag operations. The schema uses `(id, owner_user_id)` uniqueness, foreign keys on tags, length checks, normalized tag checks, owner/update indexes, and an `updated_at` trigger. Search is intentionally simple but currently uses `ILIKE '%query%'` over title/body, so it will not scale well without full-text search or trigram indexing.

Auth design is pragmatic. The interceptor requires a syntactically valid Bearer token, dev tokens use constant-time comparison, PATs are introspected and cached by SHA-256 token key, and `notes:admin` works as a wildcard. One risk is that ordinary JWT users are granted the default notes read/write/MCP scopes by the local verifier rather than preserving token-declared scopes, so authorization is coarse unless the external auth model intentionally delegates all notes access to authenticated users.

Operational design is partially cloud-native. The HTTP server has read header, read, write, and idle timeouts; shutdown uses `http.Server.Shutdown`; readiness pings the database with a 2 second context; logs are structured JSON. Missing pieces are metrics, tracing, request IDs/correlation IDs, response status logging, pprof/debug toggles, and CI/deployment manifests that prove these endpoints are wired into a runtime environment.

The current protobuf/API boundary is not clean. The source schema now includes `NoteStatus` and status fields, plus `DeleteNoteResponse.deleted_note_id`, but the rest of the repository is not consistently updated. `pkg/notes/mappers.go` maps only existing domain fields, `pkg/notes/model.go` has no status, migrations have no status column, frontend/MCP/CLI do not expose status, and `gen/notes/v1/notes.pb.go` only appears to include the `deleted_note_id` part of the dirty protobuf diff. Treat `proto/` as authoritative before merging or shipping this state.

## 5. Code Clarity and Maintainability

The Go code is generally readable and idiomatic. Constructors validate required dependencies, errors are wrapped with useful context, tests use small fakes, and the repository interface makes owner scoping hard to omit. Naming is clear in the backend packages.

Maintainability risks are concentrated outside the core backend:

- The Dockerfile is not aligned with the actual build path. The server embeds `notesFront/dist/*`, but the Dockerfile only copies source and runs `go build`; it does not run `bun run build` or copy prebuilt dist assets from a frontend stage. From a clean checkout this is likely to fail before runtime.
- The Dockerfile runtime is `scratch` but the healthcheck invokes `curl`, which will not exist in the image.
- `scripts/03_deploy_to_k8s.sh` still looks scaffolded: it references `scripts/k8s-deployment_template.yml`, but the inventory found no such file, and it contains hard-coded namespace/host assumptions.
- `scripts/create_k8s_configmap_from_env.sh` creates a ConfigMap directly from `.env`; that is unsafe for values like database passwords and JWT secrets, which belong in Kubernetes Secrets.
- `scripts/buf_generate.sh` runs `buf dep update` every generation, which can create avoidable lockfile churn and make generation less reproducible.
- `pkg/README.md` is slightly stale: it lists error names such as `ErrForbidden` and `ErrConflict`, but `pkg/notes/errors.go` currently defines `ErrInvalidInput`, `ErrNoteNotFound`, and `ErrUnauthenticated`.

The frontend is functional but not maintainable at the same level as the Go code. It is a single large TypeScript file with broad `any` usage, raw DOM manipulation, inline HTML strings, inline styles in rendered Markdown, and no test/typecheck/lint script. This is acceptable for a demo console but fragile as a real user-facing notes UI.

## 6. Documentation Quality

The root README is unusually thorough for a small service. It explains purpose, quick start, SSO flow, frontend operations, CLI usage, MCP registration, project structure, bundle mode, module API, schema isolation, routes, and Make targets. The MCP documentation is practical and calls out token rotation and stdio behavior. Package-level documentation exists for the module and shared packages.

Documentation gaps and inconsistencies:

- The README claims a smooth production-like flow, but the Dockerfile and deployment scripts do not yet support that confidently.
- The CLI README says it supports all RPC operations, but `cmd/notes-client/main.go` has no `delete` subcommand even though the service exposes `DeleteNote`.
- The root README warns not to quote `.env` values, but `.env_sample` has `JWT_SECRET="Use your nice and complicated token here"`, which directly contradicts that rule.
- The frontend README is still Bun boilerplate and says `bun run index.ts`, while the actual package scripts are `bun run build` and `bun run dev`.
- No CI status or workflow is documented or present in `.github/` from the repository inventory.

## 7. Strengths

- Clear service boundaries: process setup in `cmd/notes-server`, reusable domain in `pkg/notes`, auth in `pkg/authadapter`, MCP in `pkg/mcpnotes`, bundle wrapper in `pkg/notes/module`.
- Strong owner isolation pattern: every repository method requires `ownerUserID`, and misses are mapped to not-found rather than exposing cross-user existence.
- Good backend validation layering: proto validation, Connect auth interceptor, domain validation, and database constraints all exist.
- Practical operational basics: JSON logs, health/readiness endpoints, DB ping at startup, HTTP timeouts, graceful shutdown, panic recovery, and request logging.
- Embedded migration strategy is well thought through for standalone and bundle modes, including advisory locking and DBMate-format parser tests.
- Tests cover config parsing, health/readiness handlers, auth interceptors/verifiers, PAT caching behavior, service validation, Connect error mapping, module route registration, migration parsing, mappers, and MCP tool exposure.
- MCP implementation respects stdio constraints by writing operational logs to stderr, not stdout.
- README and AGENTS guidance are explicit about generated code, migrations, secret handling, and `.env` quoting.

## 8. Weaknesses and Risks

- Important: Current protobuf/generated/domain drift. `proto/notes/v1/notes.proto` declares `NoteStatus`, status fields, and `deleted_note_id`, but generated code and implementation are only partially aligned. This can mislead API consumers and makes `make generate` likely to produce a larger behavioral diff than the compiled code currently represents.
- Important: Container image is likely broken from a clean checkout. The Dockerfile does not build `cmd/notes-server/notesFront/dist` before compiling the Go server that embeds it, and the `scratch` runtime cannot execute the declared `curl` healthcheck.
- Important: Frontend stored/self-XSS exposure. `cmd/notes-server/notesFront/src/main.ts` inserts note title/category/tag metadata and debug response content through `innerHTML` without escaping.
- Important: Deployment scripts are not production-safe. Kubernetes deployment template is missing from the observed repository inventory, namespace/host values are hard-coded, and `.env` is turned into a ConfigMap rather than separating secrets.
- Moderate: Observability stops at logs and health/readiness. There are no metrics, traces, request IDs, response status/size logging, or structured per-RPC audit events.
- Moderate: Search implementation will degrade with data growth because it uses wildcard `ILIKE` over title/body without full-text or trigram indexing.
- Moderate: Test suite lacks real database integration coverage for the repository and migrations. Most tests use fakes or parser tests; storage SQL behavior is not validated against PostgreSQL.
- Moderate: Frontend has no automated verification. No typecheck, lint, unit, accessibility, or browser smoke target was observed.
- Moderate: No observed CI workflow. Local tests pass, but there is no repository-level enforcement of Go tests, `go vet`, `buf lint`, generation freshness, frontend build, or container build.
- Minor: `PatVerifier` positive-cache map has no cleanup or max size. For normal personal use the risk is low, but a high-cardinality token attack could grow memory until process restart.
- Minor: CLI is incomplete versus API and docs: no delete command, and update sends full replacement fields that can accidentally blank title/body/category/tags when flags are omitted.
- Minor: `.env_sample` includes quoted `JWT_SECRET`, contradicting project rules and README warning about Makefile `.env` inclusion.

## 9. Obvious Incomplete or Broken Areas

- Protobuf status feature is incomplete. The schema adds lifecycle status, but there is no domain field, migration, SQL, mapper, Connect handler usage, MCP input/output, CLI flag, frontend control, or tests for it.
- `DeleteNoteResponse.deleted_note_id` is in the dirty generated Go, but `ConnectServer.DeleteNote` returns an empty response and the MCP/CLI/frontend ignore the response body.
- Clean Docker build is likely non-functional because frontend assets are not produced inside the Dockerfile before Go embed compilation.
- Docker healthcheck is non-functional in the declared `scratch` image because `curl` is unavailable.
- Kubernetes deployment script references `scripts/k8s-deployment_template.yml`, which was not found in the inventory.
- Frontend debug and note rendering can execute injected HTML from note metadata or response strings.
- CLI docs and code do not match full CRUD because delete is missing.
- `cmd/notes-server/notesFront/README.md` is stale boilerplate.
- No `.github/` workflow was observed.

## 10. Quick Wins

1. Fix protobuf/generated consistency.
   Benefit: restores API contract integrity before more code builds on status. Effort: M. Risk: Medium.

2. Fix frontend escaping around note metadata and debug output.
   Benefit: removes the clearest user-facing security issue. Effort: S. Risk: Low.

3. Repair Dockerfile with a real frontend build stage and a valid healthcheck strategy.
   Benefit: makes container builds and runtime probes trustworthy. Effort: M. Risk: Medium.

4. Add CI for `go test ./...`, `go vet ./...`, `buf lint`, generation freshness, frontend build/typecheck, and Docker build.
   Benefit: prevents regressions that are currently easy to miss locally. Effort: M. Risk: Low.

5. Replace `.env` ConfigMap helper with Secret-aware deployment examples.
   Benefit: reduces credential leakage risk in Kubernetes usage. Effort: S. Risk: Low.

6. Add a CLI `delete` command and make update semantics explicit in help text.
   Benefit: aligns CLI with API and docs. Effort: S. Risk: Low.

7. Update stale docs: frontend README, `pkg/README.md` error list, `.env_sample` quoting.
   Benefit: reduces onboarding mistakes. Effort: XS. Risk: Low.

8. Add PostgreSQL integration tests behind an explicit env flag or testcontainer target.
   Benefit: validates migrations, constraints, SQL query behavior, and repository code. Effort: M. Risk: Medium.

9. Add basic request correlation and response status logging.
   Benefit: improves production debugging without a full observability stack. Effort: S. Risk: Low.

10. Replace wildcard `ILIKE` search with PostgreSQL full-text search or trigram indexes when note volume grows.
    Benefit: keeps search usable beyond small datasets. Effort: M. Risk: Medium.

## 11. Suggested Next Steps

Next 1 hour:

- Decide whether `NoteStatus` is in scope now. If not, revert or park the dirty proto status changes before merging. If yes, list every layer that must carry status.
- Fix `.env_sample` so `JWT_SECRET` is unquoted.
- Update frontend README command examples.
- Add a short note to the current branch/PR that the worktree is dirty and protobuf generation is not complete.

Next half-day:

- Complete or remove the status API change.
- Run `make generate` only after the schema decision, then update domain model, migrations in both migration directories, mappers, tests, frontend, MCP, CLI, and OpenAPI as needed.
- Fix `DeleteNoteResponse` so the server returns `deleted_note_id`, or remove that field from the schema.
- Patch frontend rendering to avoid `innerHTML` for untrusted values or use a sanitizer/DOM construction.

Next 1-2 days:

- Rework the Dockerfile into reproducible multi-stage frontend + Go build stages.
- Add a valid container healthcheck approach for the chosen runtime image, or remove Dockerfile healthcheck and define probes in Kubernetes.
- Add CI that runs Go tests, vet, buf lint, generation freshness, frontend build/typecheck, and container build.
- Add a minimal PostgreSQL-backed repository/migration integration test path.

Next 1-2 weeks:

- Replace scaffold deployment scripts with committed Kubernetes manifests, Helm/Kustomize, or documented examples that separate ConfigMaps from Secrets.
- Add metrics and tracing or at least Prometheus-compatible request metrics, request IDs, and structured status-code logs.
- Improve frontend structure: generated API types or typed DTOs, safer rendering helpers, smaller modules, build-time typecheck, and a smoke test.
- Revisit search design and indexing based on expected note volume.

## 12. Evaluation Scores

- Purpose clarity: 8/10. The README and package structure make the personal notes + MCP purpose clear.
- Architecture: 8/10. Backend/module boundaries are strong; frontend/container/deployment boundaries lag behind.
- Go idiomatic quality: 8/10. Constructors, interfaces, context use, error wrapping, and tests are generally idiomatic.
- Code maintainability: 7/10. Core Go is maintainable; generated/schema drift and single-file frontend reduce confidence.
- Error handling: 7/10. Domain-to-Connect mapping is good; CLI/frontend error behavior is thinner.
- Logging/observability: 5/10. Structured logs and request timing exist, but no metrics, tracing, request IDs, status logging, or audit detail.
- Testing: 6/10. Good unit coverage for core paths and passing `go test ./...`; missing DB integration, frontend verification, generation freshness, CI, and container tests.
- Documentation: 7/10. Root README is strong; several secondary docs/examples are stale or contradictory.
- Security posture: 6/10. Backend auth/owner scoping is solid, but frontend XSS, ConfigMap-from-env deployment helper, and broad JWT default scopes need attention.
- Production readiness: 5/10. Server runtime basics are present, but container/deployment/observability/CI gaps are material.
- Developer experience: 7/10. Make targets and docs help, but generation churn, missing CI, stale frontend README, and Docker issues create friction.

## 13. Final Verdict

The project is useful and technically promising. The backend has a clean enough shape to keep building on: small packages, explicit auth, owner-scoped storage, ConnectRPC validation, embedded migrations, and MCP integration are all worth preserving.

The most important thing to fix first is API contract consistency. Decide whether the dirty `NoteStatus`/`deleted_note_id` schema changes are real, then update generated code and every implementation layer together or remove the schema changes. After that, fix the frontend XSS surfaces and the Dockerfile, because those are the most direct security and deployment blockers.

Preserve the module boundary, the owner-scoped repository interface, the interceptor chain centralization, the migration discipline, and the MCP stdio separation. Those are the repository's strongest foundations.
