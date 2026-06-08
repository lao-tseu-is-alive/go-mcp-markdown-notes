##  AI-Driven Domain Code Generation



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
- `cmd/notes-server/db/migrations/000001_create_notes.sql` — dbmate migration containing both `-- migrate:up` and `-- migrate:down` sections
