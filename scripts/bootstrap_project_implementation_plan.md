# Implementation Plan: Bootstrap `go-mcp-markdown-notes`

## Context

Create a reusable project bootstrapping workflow, then use it to scaffold `go-mcp-markdown-notes` — a Markdown notes app with Connect RPC + MCP + PostgreSQL, using `go-cloud-k8s-auth` as both the reference architecture and the auth dependency.

### Key Decisions (confirmed)
- **Database:** PostgreSQL (same as `go-cloud-k8s-auth`)
- **Package layout:** `pkg/` (exportable, consistent with auth project)
- **Tooling:** `gh` CLI v2.93.0 available at `/usr/bin/gh`
- **Reference codebase:** the living `go-cloud-k8s-auth` repo (not the stale template)

---

## Phase 1: Create `scripts/bootstrap_project.sh`

> [!NOTE]
> This is a one-time investment. The script lives in `go-cloud-k8s-auth/scripts/` and can bootstrap any future project (`gothing`, `goaffaire`, `godocument`, etc.)

### [NEW] [bootstrap_project.sh](file:///home/cgil/cgdev/golang/go-cloud-k8s-auth/scripts/bootstrap_project.sh)

A bash script that takes parameters and generates a complete project skeleton:

```bash
./scripts/bootstrap_project.sh \
  --name "go-mcp-markdown-notes" \
  --module "github.com/lao-tseu-is-alive/go-mcp-markdown-notes" \
  --target-dir "../go-mcp-markdown-notes" \
  --binaries "notes-server,notes-client,notes-mcp" \
  --packages "notes,authadapter,mcpnotes" \
  --proto-package "notes/v1" \
  --db-schema "go_mcp_notes_db_schema" \
  --go-version "1.25.5"
```

**What it generates:**

| File/Directory | Generated From |
|---------------|---------------|
| `go.mod` | Fresh, correct module path + Go version |
| `buf.yaml` | Adapted from [buf.yaml](file:///home/cgil/cgdev/golang/go-cloud-k8s-auth/buf.yaml) |
| `buf.gen.yaml` | Adapted from [buf.gen.yaml](file:///home/cgil/cgdev/golang/go-cloud-k8s-auth/buf.gen.yaml) |
| `Makefile` | Simplified from [Makefile](file:///home/cgil/cgdev/golang/go-cloud-k8s-auth/Makefile) |
| `.github/workflows/test.yml` | Based on [test.yml](file:///home/cgil/cgdev/golang/go-cloud-k8s-auth/.github/workflows/test.yml) |
| `Dockerfile` | Based on [Dockerfile](file:///home/cgil/cgdev/golang/go-cloud-k8s-auth/Dockerfile) |
| `cmd/<binary>/main.go` | Placeholder `package main` for each binary |
| `pkg/<package>/` | Empty directories with placeholder `.go` files |
| `pkg/version/version.go` | Same pattern as [version.go](file:///home/cgil/cgdev/golang/go-cloud-k8s-auth/pkg/version/version.go) |
| `proto/<proto-pkg>/v1/` | Empty dir ready for `.proto` files |
| `db/migrations/` | Empty dir for golang-migrate SQL files |
| `.gitignore` | Based on auth's `.gitignore` |
| `.dockerignore` | Based on auth's `.dockerignore` |
| `.env_sample` | Skeleton with PostgreSQL vars |
| `README.md` | Basic skeleton |
| `scripts/buf_generate.sh` | Copy from auth |

**What it does NOT generate** (AI fills these in Phase 3):
- Proto service definitions
- Domain logic (model, repository, service)
- SQL migration content
- Auth adapter integration
- MCP server implementation
- Tests

---

## Phase 2: Create GitHub Repo + Run Bootstrap

Using `gh` CLI:

```bash
# 1. Create the GitHub repo
gh repo create lao-tseu-is-alive/go-mcp-markdown-notes \
  --public \
  --description "Markdown notes app with Connect RPC, MCP, and PostgreSQL" \
  --clone

# 2. Run the bootstrap script into the cloned repo
cd /home/cgil/cgdev/golang
./go-cloud-k8s-auth/scripts/bootstrap_project.sh \
  --name "go-mcp-markdown-notes" \
  --module "github.com/lao-tseu-is-alive/go-mcp-markdown-notes" \
  --target-dir "./go-mcp-markdown-notes" \
  --binaries "notes-server,notes-client,notes-mcp" \
  --packages "notes,authadapter,mcpnotes" \
  --proto-package "notes/v1" \
  --db-schema "go_mcp_notes_db_schema" \
  --go-version "1.25.5"

# 3. Initial commit + push
cd go-mcp-markdown-notes
git add .
git commit -m "chore: initial project scaffold from go-cloud-k8s-auth patterns"
git push -u origin main
```

---

## Phase 3: AI-Driven Domain Code Generation

Once the skeleton exists, open the new project with me and I will:

### 3.1 Proto + Generated Code
- Write `proto/notes/v1/notes.proto` from the [spec section 12](file:///home/cgil/cgdev/golang/go-cloud-k8s-auth/new-project-mcp-notes-spec.md#L456)
- Configure `buf.yaml` + `buf.gen.yaml` properly
- Run `buf generate`

### 3.2 Domain Logic in `pkg/`
Following the patterns established in [pkg/auth/](file:///home/cgil/cgdev/golang/go-cloud-k8s-auth/pkg/auth):

| File | Pattern Source | Purpose |
|------|---------------|---------|
| `pkg/notes/model.go` | [domain.go](file:///home/cgil/cgdev/golang/go-cloud-k8s-auth/pkg/auth/domain.go) | Domain types |
| `pkg/notes/repository.go` | [storage.go](file:///home/cgil/cgdev/golang/go-cloud-k8s-auth/pkg/auth/storage.go) | Repository interface |
| `pkg/notes/storage_postgres.go` | [storage_postgres.go](file:///home/cgil/cgdev/golang/go-cloud-k8s-auth/pkg/auth/storage_postgres.go) | PostgreSQL implementation (pgx) |
| `pkg/notes/sql.go` | [sql.go](file:///home/cgil/cgdev/golang/go-cloud-k8s-auth/pkg/auth/sql.go) | SQL query constants |
| `pkg/notes/service.go` | [auth_service.go](file:///home/cgil/cgdev/golang/go-cloud-k8s-auth/pkg/auth/auth_service.go) | Business service |
| `pkg/notes/connect_server.go` | [auth_connect_server.go](file:///home/cgil/cgdev/golang/go-cloud-k8s-auth/pkg/auth/auth_connect_server.go) | Connect handler |
| `pkg/notes/errors.go` | [errors.go](file:///home/cgil/cgdev/golang/go-cloud-k8s-auth/pkg/auth/errors.go) | Domain errors |
| `pkg/notes/mappers.go` | [mappers.go](file:///home/cgil/cgdev/golang/go-cloud-k8s-auth/pkg/auth/mappers.go) | Proto ↔ domain mappers |

### 3.3 Auth Adapter
| File | Purpose |
|------|---------|
| `pkg/authadapter/interceptor.go` | Connect interceptor that imports from `go-cloud-k8s-auth` |
| `pkg/authadapter/context.go` | Context helpers for authenticated user |

### 3.4 MCP Server
| File | Purpose |
|------|---------|
| `pkg/mcpnotes/server.go` | MCP server using `github.com/modelcontextprotocol/go-sdk/mcp` |
| `pkg/mcpnotes/tools.go` | Tool definitions (create_note, search_notes, etc.) |
| `pkg/mcpnotes/format.go` | Human-readable output formatting |

### 3.5 Binaries
Following the pattern from [goCloudAuthServer.go](file:///home/cgil/cgdev/golang/go-cloud-k8s-auth/cmd/goCloudAuthServer/goCloudAuthServer.go):

| Binary | Purpose |
|--------|---------|
| `cmd/notes-server/main.go` | PostgreSQL + migrations + Connect + Vanguard + Echo |
| `cmd/notes-client/main.go` | CLI client for testing the API |
| `cmd/notes-mcp/main.go` | MCP stdio server |

### 3.6 Database Migration
- `cmd/notes-server/db/migrations/000001_create_notes.up.sql` — from spec section 13
- `cmd/notes-server/db/migrations/000001_create_notes.down.sql` — DROP statements

### 3.7 Tests
Following [goCloudAuthServer_test.go](file:///home/cgil/cgdev/golang/go-cloud-k8s-auth/cmd/goCloudAuthServer/goCloudAuthServer_test.go) patterns.

---

## Verification Plan

### Automated Tests
```bash
cd go-mcp-markdown-notes
go vet ./...
go test -race ./...
buf lint
buf build
```

### Manual Verification
1. `make run-server` — server starts, connects to PostgreSQL, runs migrations
2. `make run-client` — creates a note, searches, retrieves
3. `make run-mcp` — MCP server responds to tool calls
4. `gh workflow view test` — CI pipeline passes

---

## Execution Order

| Step | Description | Effort |
|------|-------------|--------|
| **1** | Create `scripts/bootstrap_project.sh` in go-cloud-k8s-auth | ~30 min |
| **2** | Create GitHub repo with `gh` + run bootstrap | ~5 min |
| **3** | Open new project, AI generates domain code from spec | ~2-3 hours |
| **4** | Run tests, verify, push | ~30 min |

---

## Open Questions

> [!IMPORTANT]
> **Q1:** Should I create the GitHub repo as **public** or **private**? The auth project is public.

> [!IMPORTANT]
> **Q2:** For the `go.mod`, should I start with a `replace` directive pointing to the local `go-cloud-k8s-auth`?
> ```go
> replace github.com/lao-tseu-is-alive/go-cloud-k8s-auth => ../go-cloud-k8s-auth
> ```
> This is useful for local dev but should be removed before pushing. Should the bootstrap script include it commented out?

> [!IMPORTANT]
> **Q3:** Do you want me to start with **Phase 1** (create `bootstrap_project.sh`)? Once you approve it, I'll run it to scaffold the project.
