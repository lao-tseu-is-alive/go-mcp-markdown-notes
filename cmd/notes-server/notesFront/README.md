# notesFront — Markdown Notes SPA

Browser UI for the notes service. It is a **Connect RPC test console** and day-to-day notes client, bundled with Bun and embedded into the `notes-server` binary.

The Go server serves static assets from `notesFront/dist/` (see `//go:embed` in `cmd/notes-server/server.go`). Run `make build-frontend` (or `make run` / `make build`) before building the server from a clean checkout.

## Prerequisites

- [Bun](https://bun.sh) (runtime + bundler)
- TypeScript (peer dependency; Bun can use its built-in tooling)

```bash
cd cmd/notes-server/notesFront
bun install
```

## Commands

| Command | Purpose |
|---------|---------|
| `bun run typecheck` | `tsc --noEmit` — strict TypeScript check |
| `bun run test` | Unit tests for pure `src/lib/` modules |
| `bun run build` | Production bundle → `dist/` |
| `bun run dev` | Watch build of `src/main.ts` → `dist/` |
| `bun run clean` | Remove `dist/` |

From the repo root:

| Makefile target | What it runs |
|-----------------|--------------|
| `make build-frontend` | typecheck + test + build |
| `make lint-frontend` | typecheck + test |
| `make test-frontend` | `bun test` only |

`dist/` is generated and git-ignored. Do not edit it by hand.

## Source layout

```text
src/
  main.ts           Entry point (DOMContentLoaded → bootstrap)
  app.ts            Form handlers, /config load, startup sequence
  types.ts          Wire-format and UI types
  dom.ts            Typed references to index.html elements
  auth.ts           JWT / dev auth state and flows
  connect.ts        Connect JSON-over-HTTP RPC client
  notes-ui.ts       Note cards, list/search fetchers, pagination bar
  lib/              Pure, testable helpers (no DOM)
    escape.ts       HTML escaping (XSS mitigation)
    markdown.ts     Lightweight note body preview renderer
    note-status.ts  Proto NoteStatus → badge metadata
    tags.ts         Comma-separated tag parsing
    notes-pagination.ts  page_token cursor state
  ui/
    console-log.ts  Connect protocol debugger panel
    toast.ts        Toast notifications
    tabs.ts         Workspace tab switching
  index.html        Page shell, styles, markup
```

**Dependency direction:** `main` → `app` → feature modules → `lib` / `ui` / `dom`. Pure logic lives in `lib/` so it can be covered by `bun test` without a browser.

## Authentication

Auth mode comes from `GET /config` on the notes-server:

| Mode | Behavior |
|------|----------|
| `jwt` (default) | SSO via external auth service. JWT minted silently from session cookie (`/auth/token`). Token kept **in memory only** — never `localStorage`. Auto re-mint at ~80% of TTL; one retry on RPC `401`. |
| `dev` | Static token entered manually (`NOTES_DEV_TOKEN` on the server). |

In jwt mode, “Manage MCP tokens” links to `{authBaseUrl}/tokens.html`.

## Connect RPC

All note operations go through `callConnectRPC()` in `connect.ts`:

- Protocol: Connect JSON POST to `/notes.v1.NotesService/{Method}`
- Headers: `Content-Type: application/json`, `Connect-Protocol-Version: 1`, optional `Authorization: Bearer …`
- Request/response bodies are logged in the Debug Console tab (auth header masked)

Supported procedures match the web forms: `CreateNote`, `ListRecentNotes`, `SearchNotes`, `AddTags`, `UpdateNote`, `DeleteNote`.

## Pagination

List and search share one pagination bar below the note cards.

- **List recent:** `ListRecentNotes` with `limit` + optional `page_token`
- **Search:** `SearchNotes` with filters + `limit` + optional `page_token`
- Server returns `pageResponse.nextPageToken` and `pageResponse.totalSize`
- Client state in `lib/notes-pagination.ts` — token stack enables Previous without re-querying metadata

Pass `reset=true` on a fresh list/search to clear the page stack (form submit and “Fetch Recent Notes”).

## Testing

Tests live beside the code they cover: `src/lib/*.test.ts`. They run with Bun’s built-in test runner:

```bash
bun run test
```

Coverage today is intentionally limited to **pure functions** (escape, markdown, status mapping, tag parsing, pagination math). DOM wiring and RPC calls are exercised manually through the UI or the Connect debugger.

## Local development loop

**Option A — full stack (recommended):**

```bash
# from repo root
make run
```

This rebuilds the frontend, then starts `notes-server` with embedded assets.

**Option B — frontend watch only:**

```bash
cd cmd/notes-server/notesFront
bun run dev
```

Then run/restart `notes-server` separately so it picks up `dist/` changes. You still need a running Postgres instance and valid auth config (see root `README.md`).

## Security notes

- User-supplied note fields are escaped before insertion into card HTML (`escapeHtml`).
- Markdown preview escapes raw HTML in source before applying regex transforms.
- Debug console masks bearer tokens in displayed headers.
- Do not commit real tokens or `.env` values.

## Related docs

- Root [`README.md`](../../../README.md) — server setup, auth, MCP
- [`CLAUDE.md`](../../../CLAUDE.md) — agent/build rules for the whole repo
- [`proto/notes/v1/notes.proto`](../../../proto/notes/v1/notes.proto) — RPC contract