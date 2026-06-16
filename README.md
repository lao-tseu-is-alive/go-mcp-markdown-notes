# go-mcp-markdown-notes

> go-mcp-markdown-notes — a Go microservice built with ConnectRPC, PostgreSQL, and go-cloud-k8s patterns, featuring an embedded responsive web frontend.

---

## Features

- **Connect RPC Notes Service**: Exposes robust RPC methods for creating, listing, searching, tagging, updating, and deleting notes.
- **Embedded Web Client**: A responsive single-page application built with Bun, TypeScript, and a modern glassmorphic UI.
- **Seamless SSO Sign-in**: One-click "Sign in with Google / GitHub" via [go-cloud-k8s-auth](https://github.com/lao-tseu-is-alive/go-cloud-k8s-auth) — no token copy-paste. The frontend silently mints short-lived JWTs from the SSO session cookie and keeps them in memory only.
- **MCP Server (`notes-mcp`)**: A stdio Model Context Protocol server exposing all note operations as tools for Claude Code / Claude Desktop, authenticated with a Personal Access Token.
- **Zero-Dependency Serving**: Frontend assets are compiled and embedded directly inside the compiled Go binary using Go `embed` and served at `/`.
- **Connect Debugger & Logger**: Interactive UI to inspect exact Connect RPC request and response payloads.
- **Auth Modes** (`NOTES_AUTH_MODE`):
  - **jwt** (default): accepts JWT Bearer tokens (shared `JWT_SECRET` with go-cloud-k8s-auth) **and** `pat_...` Personal Access Tokens, verified via the auth service introspection endpoint with a 60 s cache.
  - **dev**: simple predefined token for local development.

---

## Quick Start

### 1. Configure the Environment
Copy the environment template and ensure surrounding double quotes are removed from configurations:
```bash
cp .env_sample .env
# Edit .env and customize database / auth variables
```
> [!IMPORTANT]
> **Environment Quotation Rule**: Surrounding double quotes `""` on values must be avoided in the `.env` file. Because the `Makefile` exports variables to child processes via `include .env`, surrounding quotes are preserved literally, leading to signature verification and OAuth redirect mismatches.

### 2. Setup the Database
Use `dbmate` to run migrations:
```bash
# Check migration status
make db-status

# Apply database migrations
make db-up
```

### 3. Build and Run
```bash
# Compile frontend, download go modules, and run the server
make run
```
The server will start on `127.0.0.1:8080` by default. Open your browser and navigate to **`http://localhost:8080/`** to interact with the embedded client.

### 4. Sign in (jwt mode)

With `NOTES_AUTH_MODE=jwt`, the auth service must also be running (default
`AUTH_SERVER_URL=http://localhost:9090`, see
[go-cloud-k8s-auth](https://github.com/lao-tseu-is-alive/go-cloud-k8s-auth)).
Click **Sign in with Google / GitHub**: you are redirected to the auth
service's hosted login page and straight back, already authenticated. The SSO
session cookie lives on the auth service; this app silently calls
`GET <AUTH_SERVER_URL>/auth/token` (with `credentials: 'include'`) to obtain
short-lived JWTs and refreshes them automatically before they expire. Nothing
is persisted in browser storage.

> One-time setup steps (OAuth console callbacks, allowlists, shared JWT
> secret) are listed in the auth repo's
> [sso_setup_checklist.md](https://github.com/lao-tseu-is-alive/go-cloud-k8s-auth/blob/main/documentation/sso_setup_checklist.md).

---

## Embedded Web Client UI

The built-in single-page client is located under `cmd/notes-server/notesFront`.
It is built using **Bun** and **TypeScript** and compiles into the `notesFront/dist/` directory.

### Key Capabilities:
- **SSO Sign-in (jwt mode)**: one click on "Sign in with Google / GitHub", powered by the auth service's hosted login page and silent `/auth/token` mint. A "Manage MCP tokens" link opens the auth service's PAT management UI.
- **Local Dev Login (dev mode)**: when the server runs with `NOTES_AUTH_MODE=dev`, the UI shows a dev-token form instead (default token: `notes-dev-token`). The UI discovers the active mode from `GET /config`.
- **Connect Debugger**: Prints outgoing headers, JSON body payloads, response status codes, and returned responses in real-time.
- **Full CRUD**: create, search, edit, tag, and delete notes (with confirmation) interactively, with a custom markdown previewer.

### Manual Frontend Operations:
If you want to manage the frontend build separately:
```bash
cd cmd/notes-server/notesFront

# Install dev dependencies
bun install

# Build the assets once
bun run build

# Run in watch mode for development
bun run dev
```

---

## Connect CLI Client

The repository includes a Go command-line tool `notes-client` under `cmd/notes-client/` to execute all RPC operations directly from your terminal.

### Quick Start:
1. Compile the CLI tool:
   ```bash
   go build -o bin/notes-client ./cmd/notes-client
   ```
2. Configure authentication (either dev token, or programmatically fetching a JWT token):
   ```bash
   # For Local Dev Mode
   export NOTES_TOKEN=notes-dev-token

   # For JWT Mode (fetches from auth server running on port 9090)
   export NOTES_TOKEN=$(./scripts/get_jwt_token.sh /home/cgil/cgdev/golang/go-cloud-k8s-auth/.env)
   ```
3. Run CLI commands:
   ```bash
   # List notes
   bin/notes-client list

   # Create a note
   bin/notes-client create -title "Hello" -body "Content" -category "general" -tags "cli,demo"
   ```

For advanced configuration, server authentication modes (dev vs jwt), and automation scripting guide, see the dedicated [notes-client guide](file:///home/cgil/cgdev/golang/go-mcp-markdown-notes/cmd/notes-client/README.md).

---

## MCP Server (`notes-mcp`)

`notes-mcp` lets AI assistants (Claude Code, Claude Desktop, any MCP client)
manage your notes. It runs locally over **stdio** and calls the notes-server
Connect API with a bearer token. Implementation: `pkg/mcpnotes/` (built on
[modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk)).

### Exposed tools

`create_note`, `get_note`, `list_recent_notes`, `search_notes`, `add_tags`,
`update_note`, `delete_note` — all scoped to the owner of the configured token.

### Configuration (environment variables)

| Variable | Default | Purpose |
|----------|---------|---------|
| `NOTES_SERVER` | `http://127.0.0.1:8080` | Base URL of the notes-server |
| `NOTES_TOKEN` | *(required)* | A `pat_...` Personal Access Token, or the dev token in dev mode |

### Getting a token

- **jwt mode (recommended)**: sign in on the notes web app, click **Manage MCP
  tokens** (or open `<AUTH_SERVER_URL>/tokens.html`), create a token — the
  `pat_...` value is shown exactly once. Revoking it there cuts MCP access
  within ≤ 60 seconds.
- **dev mode**: use the value of `NOTES_DEV_TOKEN`.

### Build & register in Claude Code

```bash
make build-mcp          # produces bin/notes-mcp

claude mcp add --scope user notes \
  -e NOTES_SERVER=http://127.0.0.1:8080 \
  -e NOTES_TOKEN=pat_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx \
  -- /path/to/go-mcp-markdown-notes/bin/notes-mcp
```

Good to know:

- **Scope matters.** Without `--scope`, `claude mcp add` registers the server
  with `local` scope: it is only visible to Claude Code sessions started from
  the current directory. `--scope user` makes it available in every session
  (recommended for personal notes); `--scope project` writes a shareable
  `.mcp.json` into the repo instead.
- **MCP servers are loaded at session startup.** After adding, removing or
  rotating a token, restart Claude Code (or open a new session) for the change
  to take effect. Check with the `/mcp` command inside a session.
- **Runtime prerequisites.** The `notes` tools need `notes-server` running on
  `NOTES_SERVER` (and, in jwt mode, the go-cloud-k8s-auth server reachable to
  introspect the PAT). A connection error from the tools usually just means
  those servers are not started.

Claude Desktop (`claude_desktop_config.json`):
```json
"mcpServers": {
  "markdown-notes": {
    "command": "/path/to/go-mcp-markdown-notes/bin/notes-mcp",
    "env": {
      "NOTES_SERVER": "http://127.0.0.1:8080",
      "NOTES_TOKEN": "pat_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
    }
  }
}
```

To rotate the token (compromised or expired PAT): revoke it in `tokens.html`,
create a new one, then re-register:

```bash
claude mcp remove notes
claude mcp add notes -e NOTES_SERVER=http://127.0.0.1:8080 -e NOTES_TOKEN=pat_...new... -- /path/to/go-mcp-markdown-notes/bin/notes-mcp
```

Then ask things like *“create a note titled X”*, *“search my notes about
docker”*, *“delete that note”*. Authentication design details:
[pkg/mcpnotes/mcp_auth_design.md](./pkg/mcpnotes/mcp_auth_design.md).

---

## Project Structure

```text
go-mcp-markdown-notes/
├── cmd/                    # Application entry points
│   ├── notes-server/       # Go HTTP / Connect Server
│   │   └── notesFront/     # Bun & TS Frontend Client source & compiler
│   │       ├── src/        # HTML and TypeScript code
│   │       ├── dist/       # Built static assets (HTML/JS)
│   │       └── build.ts    # Bun build/bundler script
│   ├── notes-client/       # CLI client example
│   └── notes-mcp/          # MCP stdio server entry point (env: NOTES_SERVER, NOTES_TOKEN)
├── pkg/                    # Shared packages
│   ├── notes/              # Notes business & repository layers
│   │   └── module/         # Importable domain module (bundle-ready); see Module & Bundle Strategy
│   ├── authadapter/        # TokenVerifier abstraction: JWT, dev token, PAT introspection, composite
│   ├── mcpnotes/           # MCP server (7 tools) + authenticated Connect client
│   └── version/            # Build version meta information
├── proto/                  # Protobuf definitions
│   └── notes/v1/           # notes.proto defining RPC schemas
├── gen/                    # Generated Go / Connect code
├── Makefile                # Automated building and dev orchestration
├── Dockerfile              # Multi-stage production container definition
└── README.md
```

---

## Module & Bundle Strategy

`pkg/notes/module` exposes the notes domain as an importable Go package so that
another repository can embed it alongside other domains in a single binary —
sharing one HTTP server, one database pool, and one auth setup — without
forking or duplicating code.

### Standalone mode (default)

`cmd/notes-server` is the standard standalone binary. It calls
`notesmodule.Migrate` and `notesmodule.New`, then mounts all routes via
`mod.RegisterRoutes(mux)`:

```go
import notesmodule "github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/notes/module"

mod, _ := notesmodule.New(ctx, notesmodule.Config{}, notesmodule.Deps{
    Pool:     pool,
    Verifier: verifier,
    Logger:   log,
})
mod.RegisterRoutes(mux) // mounts /notes.v1.NotesService/ on the shared mux
```

Nothing changes for existing deployments. The standalone binary continues to
work exactly as before.

### Bundle mode

A bundle repository imports this module alongside others and composes them at
compile time — no plugins, no dynamic loading. Each module gets its own
config, schema namespace, and handler prefix; they share the process, pool,
and mux.

```go
// In the bundle repo's application setup:
notesMod, _ := notesmodule.New(ctx, notesmodule.Config{
    RequestTimeout: 10 * time.Second,
}, notesmodule.Deps{
    Pool:     sharedPool,   // one pool for all modules
    Verifier: sharedVerifier,
    Logger:   logger,
})

// Option A – mount with RegisterRoutes (simple)
notesMod.RegisterRoutes(sharedMux)

// Option B – iterate ConnectHandlers for fine-grained control
for _, ch := range notesMod.ConnectHandlers() {
    sharedMux.Handle(ch.Pattern, ch.Handler)
}
```

`ConnectHandlers()` returns `[]ConnectHandler{Pattern, Handler}` pairs with the
full interceptor chain (timeout → auth → proto validation) already applied.
`RoutePatterns()` and `ConnectPatterns()` list the owned URL prefixes so the
bundle can detect conflicts before mounting.

### Module API

| Method | Returns | Purpose |
|--------|---------|---------|
| `New(ctx, Config, Deps)` | `*Module, error` | Validate deps, build all service layers |
| `RegisterRoutes(mux)` | `error` | Mount Connect handlers on a shared mux (standalone or bundle) |
| `ConnectHandlers()` | `[]ConnectHandler` | Return `(pattern, handler)` pairs for bundle callers |
| `RoutePatterns()` | `[]RoutePattern` | URL patterns owned by this module |
| `ConnectPatterns()` | `[]string` | Connect/gRPC path prefixes |
| `Start(ctx)` | `error` | No-op (hook for future background workers) |
| `Stop(ctx)` | `error` | No-op (hook for future graceful shutdown) |
| `Migrate(ctx, pool)` | `error` | Apply pending SQL migrations (package-level function) |
| `Migrations` | `embed.FS` | Embedded SQL files; bundle callers can inspect directly |
| `ParseDBMateUp(content)` | `[]string, error` | Parse DBMate migration format (exported for testing) |

### Database / schema isolation

The module owns the `notes` and `note_tags` tables under the `public` schema.
`Migrate` creates a `schema_migrations` table (if absent) and uses a
PostgreSQL advisory lock keyed to `'go-mcp-markdown-notes:migrations'` to
prevent concurrent migration runs across instances.

In bundle mode, each domain module runs `Migrate` independently against the
shared pool; advisory lock keys must be unique per module to avoid deadlocks.
If the bundle needs full schema isolation, point each module at a different
PostgreSQL `search_path` schema by setting that on the pool before passing it in.

### Local multi-repository development with `go.work`

When developing the bundle repo and this repo side by side, use a Go workspace
so that the bundle picks up local (uncommitted) changes to this module without
publishing a new version:

```bash
# From the directory that contains both repos
go work init
go work use ./go-mcp-markdown-notes
go work use ./my-bundle-repo
```

The `go.work` file is conventionally git-ignored. Drop it to switch back to the
released module version.

### Importing this module from another repository

```go
// In go.mod of the bundle repo:
require github.com/lao-tseu-is-alive/go-mcp-markdown-notes v0.x.y

// In code:
import (
    notesmodule "github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/notes/module"
)
```

The SQL migration files are embedded in the module binary via `go:embed`, so no
external SQL files need to be copied into the bundle repo. Only the Go source is
needed at build time.

---

## API & Protocol Schema

All RPC actions are routed via POST to `/{package}.{service}/{method}` (e.g., `/notes.v1.NotesService/ListRecentNotes`) with the following headers:
- `Content-Type: application/json`
- `Connect-Protocol-Version: 1`
- `Authorization: Bearer <token>`

### Service RPC Definitions:
1. `CreateNote` (`CreateNoteRequest` -> `CreateNoteResponse`)
2. `GetNote` (`GetNoteRequest` -> `GetNoteResponse`)
3. `ListRecentNotes` (`ListRecentNotesRequest` -> `ListRecentNotesResponse`)
4. `SearchNotes` (`SearchNotesRequest` -> `SearchNotesResponse`)
5. `AddTags` (`AddTagsRequest` -> `AddTagsResponse`)
6. `UpdateNote` (`UpdateNoteRequest` -> `UpdateNoteResponse`)
7. `DeleteNote` (`DeleteNoteRequest` -> `DeleteNoteResponse`) — requires `notes:write`; deleting another user's note returns `not_found`

### Plain HTTP endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/config` | Frontend bootstrap: `{"authMode": "jwt"\|"dev", "authBaseUrl": "..."}` |
| `GET` | `/health`, `/readiness`, `/goAppInfo` | Health checks & build info |

---

## Available Make Targets

Run `make help` to inspect available Makefile actions:
```text
  run                  will run the main server binary
  build-frontend       will compile frontend assets using Bun
  build                will compile the main server binary and place it in bin/
  build-mcp            will compile the notes-mcp stdio binary into bin/notes-mcp
  run-mcp              will run notes-mcp over stdio (requires NOTES_TOKEN)
  generate             will run buf generate to generate protobuf/connect code
  test                 will run all tests
  lint                 will run go vet and buf lint
  fmt                  will format all Go source files
  clean                will remove binaries and coverage files
  db-status            show dbmate migration status
  db-up                apply pending dbmate migrations
  db-down              roll back the latest dbmate migration
```

---

## License

See [LICENSE](LICENSE) file.
