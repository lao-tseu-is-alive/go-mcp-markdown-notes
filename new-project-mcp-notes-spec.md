# Project Brief — go-mcp-markdown-notes

## 1. Goal

Create a small but complete Go proof of concept named:

```text
go-mcp-markdown-notes
````

The project is a personal Markdown notes application with a Connect RPC backend and an MCP server.

The purpose is to demonstrate the following architecture:

```text
Markdown notes app
+ Go backend
+ protobuf/Connect RPC contract
+ authentication integration
+ PostgreSQL storage
+ MCP tools for AI assistants
```

The application must remain much smaller and simpler than the existing `go-cloud-k8s-thing` project.

This is a proof of concept, not a production SaaS.

The project must be clear enough for a senior Go developer to understand in less than 30 minutes.

---

## 2. Main Architecture

The target architecture is:

```text
Browser / Web UI
      ↓
notes-server
      ↓
Connect RPC API
      ↓
PostgreSQL database

AI Assistant / Claude Code / Codex / Cursor
      ↓ MCP stdio
notes-mcp
      ↓ Connect client
notes-server
      ↓
PostgreSQL database
```

Authentication is provided by a separate reusable project:

```text
github.com/lao-tseu-is-alive/go-cloud-k8s-auth
```

The notes app must not implement its own OAuth provider, password handling, or identity system.

---

## 3. Repository Organization

Create a separate GitHub repository:

```text
github.com/lao-tseu-is-alive/go-mcp-markdown-notes
```

Do not put the notes application inside:

```text
github.com/lao-tseu-is-alive/go-cloud-k8s-auth
```

The auth project is a reusable authentication brick.
The notes project is the business application.

Expected structure:

```text
go-mcp-markdown-notes/
├── cmd/
│   ├── notes-server/
│   │   └── main.go
│   ├── notes-client/
│   │   └── main.go
│   └── notes-mcp/
│       └── main.go
├── proto/
│   └── notes/v1/
│       └── notes.proto
├── gen/
│   └── notes/v1/
│       └── ...
├── pkg/
│   ├── notes/
│   │   ├── model.go
│   │   ├── repository.go
│   │   ├── storage_postgres.go
│   │   ├── service.go
│   │   └── service_test.go
│   ├── authadapter/
│   │   ├── context.go
│   │   └── interceptor.go
│   └── mcpnotes/
│       ├── server.go
│       ├── tools.go
│       └── format.go
├── migrations/
│   └── 001_init.sql
├── web/
│   └── ...
├── buf.yaml
├── buf.gen.yaml
├── Makefile
├── go.mod
├── go.sum
├── README.md
└── CODEX.md
```

The frontend can be minimal in the first version.
The backend and MCP integration are more important than a polished UI.

---

## 4. Non-Goals for Version 1

Do not implement the following in v1:

```text
Kubernetes deployment
complex Docker setup
multi-tenant SaaS billing
OCR
image upload
web page summarization
LLM backend integration
advanced full-text search ranking
admin dashboard
delete-all operations
free SQL execution
filesystem access from MCP
shell access from MCP
```

These can be future extensions.

---

## 5. Version 1 Scope

Version 1 must implement:

```text
Markdown notes
PostgreSQL storage
tags
simple category
search
Connect RPC API
MCP stdio server
authentication hook/integration
personal access token support for MCP
basic tests
README
```

Minimal note fields:

```text
id
owner_user_id
title
body_markdown
category
tags
created_at
updated_at
```

Every note must belong to an authenticated internal user:

```text
owner_user_id
```

Never use the user email as the primary ownership key.

---

## 6. Authentication Design

The project must use the existing auth project as a reusable dependency:

```text
github.com/lao-tseu-is-alive/go-cloud-k8s-auth
```

The preferred integration model is:

```text
go-cloud-k8s-auth
    → provides reusable auth types, token verification, session/PAT helpers, or Connect interceptors

go-mcp-markdown-notes
    → imports the auth package
    → protects Connect endpoints
    → extracts AuthenticatedUser from context
    → filters data by owner_user_id
```

If needed during local development, use a `replace` directive:

```go
replace github.com/lao-tseu-is-alive/go-cloud-k8s-auth => ../go-cloud-k8s-auth
```

The notes app should depend on a small abstraction such as:

```go
type AuthenticatedUser struct {
    AppUserID    string
    Email        string
    DisplayName  string
    Scopes       []string
}

type TokenVerifier interface {
    VerifyBearerToken(ctx context.Context, token string) (*AuthenticatedUser, error)
}
```

The exact names can be adapted to the real API of `go-cloud-k8s-auth`.

The important rule is:

```text
The notes domain must not depend directly on Google, GitHub, Supabase, Keycloak, or Authentik concepts.
```

It must depend only on an internal authenticated user.

---

## 7. Authorization Rules

All note operations must be scoped by `owner_user_id`.

Examples:

```go
repo.CreateNote(ctx, user.AppUserID, note)
repo.GetNote(ctx, user.AppUserID, noteID)
repo.SearchNotes(ctx, user.AppUserID, query, limit)
repo.UpdateNote(ctx, user.AppUserID, noteID, patch)
repo.AddTags(ctx, user.AppUserID, noteID, tags)
```

Never write repository methods like:

```go
repo.GetNote(ctx, noteID)
```

unless the owner check is guaranteed elsewhere.

Preferred scopes:

```text
notes:read
notes:write
notes:mcp
notes:admin
```

MCP tokens should have:

```text
notes:read
notes:write
notes:mcp
```

No admin scope by default.

---

## 8. MCP Role

MCP is not the application backend.
MCP is not the Markdown editor.
MCP is not the database.

MCP is an agent-facing adapter that lets an AI assistant use the notes application through controlled tools.

The core idea:

```text
AI assistant
   ↓ MCP tools
notes-mcp
   ↓ Connect client
notes-server
   ↓ PostgreSQL
```

The MCP server must call the Connect API.
It must not access the repository directly.

This is important because the Connect API remains the main system contract.

---

## 9. MCP Use Cases

Target user scenarios:

### Create a note from a conversation

User asks an assistant:

```text
Summarize this discussion and save it as a note tagged go, mcp, connect.
```

The assistant calls:

```text
create_note
```

### Search existing notes

User asks:

```text
Find my notes about MCP and Connect.
```

The assistant calls:

```text
search_notes
get_note
```

### Create a synthesis note

User asks:

```text
Search my notes about MCP, Go and authentication, then create a new synthesis note.
```

The assistant calls:

```text
search_notes
get_note
create_note
```

### Add tags

User asks:

```text
Add the tags "architecture" and "poc" to this note.
```

The assistant calls:

```text
add_tags
```

---

## 10. MCP Tools for Version 1

Expose only safe and useful tools:

```text
create_note
get_note
search_notes
list_recent_notes
add_tags
```

Optional if simple:

```text
update_note
```

Do not expose in v1:

```text
delete_note
delete_all_notes
execute_sql
read_local_file
write_local_file
run_shell_command
fetch_any_url
```

Each MCP tool must:

```text
validate inputs
enforce sane limits
call Connect client
return human-readable text
return structured data when useful
use timeouts
respect authenticated user context through the API token
```

---

## 11. MCP Authentication

The MCP server should not implement Google/GitHub login.

Instead:

```text
1. User logs into the web application.
2. User creates a Personal Access Token for MCP.
3. notes-mcp runs locally with the token.
4. notes-mcp sends the token to notes-server.
5. notes-server verifies the token using go-cloud-k8s-auth.
```

Example environment variables:

```bash
NOTES_API_BASE_URL=http://127.0.0.1:8080
NOTES_API_TOKEN=pat_xxxxxxxxx
```

The MCP server sends:

```http
Authorization: Bearer pat_xxxxxxxxx
```

---

## 12. Connect / Protobuf Contract

Create a protobuf service:

```proto
syntax = "proto3";

package notes.v1;

option go_package = "github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1;notesv1";

service NotesService {
  rpc CreateNote(CreateNoteRequest) returns (CreateNoteResponse);
  rpc GetNote(GetNoteRequest) returns (GetNoteResponse);
  rpc ListRecentNotes(ListRecentNotesRequest) returns (ListRecentNotesResponse);
  rpc SearchNotes(SearchNotesRequest) returns (SearchNotesResponse);
  rpc AddTags(AddTagsRequest) returns (AddTagsResponse);
  rpc UpdateNote(UpdateNoteRequest) returns (UpdateNoteResponse);
}

message Note {
  string id = 1;
  string title = 2;
  string body_markdown = 3;
  string category = 4;
  repeated string tags = 5;
  string created_at = 6;
  string updated_at = 7;
}

message CreateNoteRequest {
  string title = 1;
  string body_markdown = 2;
  string category = 3;
  repeated string tags = 4;
}

message CreateNoteResponse {
  Note note = 1;
}

message GetNoteRequest {
  string id = 1;
}

message GetNoteResponse {
  Note note = 1;
}

message ListRecentNotesRequest {
  int32 limit = 1;
}

message ListRecentNotesResponse {
  repeated Note notes = 1;
}

message SearchNotesRequest {
  string query = 1;
  repeated string tags = 2;
  string category = 3;
  int32 limit = 4;
}

message SearchNotesResponse {
  repeated Note notes = 1;
}

message AddTagsRequest {
  string note_id = 1;
  repeated string tags = 2;
}

message AddTagsResponse {
  Note note = 1;
}

message UpdateNoteRequest {
  string note_id = 1;
  string title = 2;
  string body_markdown = 3;
  string category = 4;
  repeated string tags = 5;
}

message UpdateNoteResponse {
  Note note = 1;
}
```

The proto may be adjusted, but keep it simple.

---

## 13. PostgreSQL Schema

Use PostgreSQL for v1 (consistent with go-cloud-k8s-auth patterns).

Initial migration (`db/migrations/000001_create_notes.up.sql`):

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS notes (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_user_id INTEGER NOT NULL,
    title       TEXT NOT NULL,
    body_markdown TEXT NOT NULL,
    category    TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS note_tags (
    note_id       UUID NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    owner_user_id INTEGER NOT NULL,
    tag           TEXT NOT NULL,
    PRIMARY KEY (note_id, tag)
);

CREATE INDEX IF NOT EXISTS idx_notes_owner_updated
ON notes(owner_user_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_note_tags_owner_tag
ON note_tags(owner_user_id, tag);
```

Optional PostgreSQL full-text search for later:

```sql
ALTER TABLE notes ADD COLUMN IF NOT EXISTS search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(body_markdown, '')), 'B')
    ) STORED;

CREATE INDEX IF NOT EXISTS idx_notes_fts ON notes USING GIN(search_vector);
```

For v1, simple `ILIKE` search is acceptable.

---

## 14. Search Behavior

For v1:

```text
Search title and body_markdown using simple LIKE
Filter by owner_user_id
Filter by tags if provided
Filter by category if provided
Limit results
Default limit: 10
Maximum limit: 50
```

Never allow unlimited search result output.

---

## 15. Server Binary

Create:

```text
cmd/notes-server
```

Responsibilities:

```text
load configuration
open PostgreSQL database connection
run migrations
initialize auth verifier
initialize notes service
register Connect handler
listen on 127.0.0.1:8080 by default
```

Default listen address should be local only:

```text
127.0.0.1:8080
```

Do not listen on `0.0.0.0` by default.

---

## 16. Client Binary

Create:

```text
cmd/notes-client
```

It should demonstrate basic API usage:

```text
create note
list recent notes
search notes
get note
```

It can use a dev token from environment:

```bash
NOTES_API_TOKEN=...
```

This binary is primarily for testing the Connect API without MCP.

---

## 17. MCP Binary

Create:

```text
cmd/notes-mcp
```

Responsibilities:

```text
read NOTES_API_BASE_URL
read NOTES_API_TOKEN
create Connect client
register MCP tools
run MCP stdio transport
```

Use the official Go MCP SDK if possible:

```text
github.com/modelcontextprotocol/go-sdk/mcp
```

MCP tools should be implemented in:

```text
pkg/mcpnotes
```

Example input struct:

```go
type CreateNoteInput struct {
    Title        string   `json:"title" jsonschema:"note title"`
    BodyMarkdown string   `json:"body_markdown" jsonschema:"note body in Markdown format"`
    Category     string   `json:"category,omitempty" jsonschema:"optional note category"`
    Tags         []string `json:"tags,omitempty" jsonschema:"optional tags"`
}
```

Tool behavior:

```text
validate title is not empty
validate body is not too large
normalize tags
call NotesService.CreateNote through Connect client
return a readable summary with the note ID
```

---

## 18. Formatting MCP Results

MCP tools should return concise but useful text.

Example:

```text
Created note:
ID: note_123
Title: Architecture MCP over Connect
Category: architecture
Tags: go, mcp, connect
```

Search result example:

```text
Found 3 notes:

1. Architecture MCP over Connect
   ID: note_123
   Tags: go, mcp, connect
   Updated: 2026-06-08T10:30:00Z

2. Authentication strategy
   ID: note_456
   Tags: auth, oauth, pat
   Updated: 2026-06-08T11:00:00Z
```

Avoid dumping huge Markdown bodies in search results.
Use `get_note` to retrieve full note content.

---

## 19. Security Rules

Even for a POC:

```text
listen on localhost by default
require authentication on Connect API
require authentication for MCP calls through API token
do not expose shell commands
do not expose arbitrary filesystem access
do not expose SQL execution
do not expose delete-all operations
limit search results
limit body size
limit tag count
validate all inputs
log MCP write actions
```

Recommended limits:

```text
max note title length: 200 characters
max note body length: 200_000 characters
max tags per note: 20
max tag length: 50 characters
default search limit: 10
max search limit: 50
```

---

## 20. Future Features

Future extensions, not v1:

```text
Markdown web editor
image upload
OCR on uploaded images
web page to note
automatic tag suggestions
note linking
full-text search with PostgreSQL tsvector
LLM-based summarization
import/export Markdown files
sync
sharing
collaboration
mobile-friendly UI
```

Recommended order:

```text
v1: notes + tags + search + Connect + MCP + auth
v2: simple web Markdown editor
v3: CreateNoteFromURL
v4: image upload
v5: OCR
v6: PostgreSQL full-text search (tsvector)
v7: automatic classification/tags
```

---

## 21. Important Design Principle

The application must keep these layers separate:

```text
Auth layer
    identifies the user

Notes domain
    manages notes owned by a user

Connect API
    exposes the application contract

MCP server
    exposes controlled tools to AI assistants

Frontend
    provides direct human UI
```

Avoid this:

```text
MCP tool → repository directly
frontend → database directly
notes domain → Google/GitHub directly
repository → global user state
```

Prefer this:

```text
MCP tool → Connect client → Connect API → notes service → repository
frontend → Connect API → notes service → repository
auth → context user → service methods
```

---

## 22. Development Commands

Provide a Makefile with commands similar to:

```makefile
.PHONY: generate test run-server run-client run-mcp fmt

generate:
	buf generate

fmt:
	gofmt -w .

test:
	go test ./...

run-server:
	go run ./cmd/notes-server

run-client:
	go run ./cmd/notes-client

run-mcp:
	go run ./cmd/notes-mcp
```

---

## 23. README Requirements

The README must explain:

```text
what the project is
why it exists
architecture diagram
how to generate proto code
how to run tests
how to run notes-server
how to run notes-client
how to run notes-mcp
how auth is expected to work
how to create/use a MCP token
which features are intentionally not implemented
```

Include example commands:

```bash
make generate
make test
make run-server
make run-client
NOTES_API_TOKEN=pat_xxx make run-mcp
```

---

## 24. Coding Style

Use idiomatic Go.

Priorities:

```text
clarity
small functions
explicit errors
context.Context
timeouts
simple interfaces
testability
no premature abstraction
```

Avoid:

```text
large generic frameworks
global mutable state
hidden dependencies
magic configuration
over-engineering
too many packages
copy-pasted auth logic
```

---

## 25. Testing Requirements

Implement at least:

```text
repository tests
service tests
auth scoping tests
MCP tool tests if simple enough
```

Important test cases:

```text
user A cannot read user B note
search returns only current user notes
limit is enforced
empty title is rejected
too many tags are rejected
MCP create_note validates input
MCP search_notes limits results
```

---

## 26. Suggested First Implementation Plan

1. Create repository skeleton.
2. Add `go.mod`.
3. Add `notes.proto`.
4. Add `buf.yaml` and `buf.gen.yaml`.
5. Generate Connect/protobuf code.
6. Implement PostgreSQL repository.
7. Implement notes service.
8. Implement Connect server.
9. Add simple token-auth interceptor.
10. Integrate `go-cloud-k8s-auth` abstraction if available.
11. Implement `notes-client`.
12. Implement `notes-mcp`.
13. Add tests.
14. Add README.
15. Run:

```bash
gofmt -w .
go test ./...
```

---

## 27. Local Development Mode

If the auth project is not ready as a stable library, implement a temporary dev verifier behind the same interface:

```go
type DevTokenVerifier struct {
    Token string
    User  AuthenticatedUser
}
```

It should only be enabled explicitly with a dev flag or environment variable:

```bash
NOTES_AUTH_MODE=dev
NOTES_DEV_TOKEN=dev-local-token
```

Do not hardcode tokens.

This allows the notes project to progress while the auth package is stabilized.

---

## 28. Final Expected Result

A successful v1 allows this workflow:

Terminal 1:

```bash
make run-server
```

Terminal 2:

```bash
NOTES_API_TOKEN=dev-local-token make run-client
```

Terminal 3:

```bash
NOTES_API_TOKEN=dev-local-token make run-mcp
```

Then from an MCP client such as MCP Inspector, Claude Code, Cursor, or another MCP-capable assistant:

```text
Create a note titled "MCP over Connect" with a summary of this architecture and tags go, mcp, connect.
```

Then:

```text
Search my notes about MCP and Connect.
```

The assistant should call:

```text
create_note
search_notes
get_note
```

through MCP.

The MCP server must call the Connect API.

The Connect API must enforce authentication.

The repository must enforce ownership by `owner_user_id`.

---

## 29. Core Principle to Preserve

This POC is not only a notes app.

It is a small demonstration of a future architecture pattern:

```text
Human UI + Connect API + Auth + MCP adapter
```

The same pattern can later be reused for larger systems such as:

```text
gothing
goaffaire
godocument
goacteurs
gouser
```

The notes domain is intentionally simple so that the architecture can be validated without the complexity of geospatial data, municipal workflows, or document management.

````
----
