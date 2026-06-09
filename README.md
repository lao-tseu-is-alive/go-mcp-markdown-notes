# go-mcp-markdown-notes

> go-mcp-markdown-notes — a Go microservice built with ConnectRPC, PostgreSQL, and the go-cloud-k8s patterns.

## Quick Start

```bash
# Generate protobuf/Connect code
make generate

# Run tests
make test

# Run the main server
make run
```

## Project Structure

```text
go-mcp-markdown-notes/
├── cmd/                    # Application binaries
│   ├── notes-server/
│   ├── notes-client/
│   ├── notes-mcp/
├── pkg/                    # Shared packages
│   ├── notes/
│   ├── authadapter/
│   ├── mcpnotes/
│   ├── version/
│   └── version/            # Build version info
├── proto/                  # Protobuf definitions
│   └── notes/v1/
├── gen/                    # Generated protobuf/Connect code
├── .github/workflows/      # CI/CD pipelines
├── Makefile
├── Dockerfile
└── README.md
```

## Development

### Prerequisites

- Go 1.25.5+
- PostgreSQL
- [buf](https://buf.build/docs/installation) for protobuf generation
- protoc-gen-go, protoc-gen-connect-go
- dbmate

### Database Setup

```bash
cp .env_sample .env
# Edit .env with your PostgreSQL credentials
# Inspect and apply the dbmate migration
make db-status
make db-up
```

### Run The Server

The default authentication mode verifies JWTs issued by `go-cloud-k8s-auth`
using the shared `JWT_SECRET` and `JWT_ISSUER_ID` configuration.

For explicit local development:

```bash
make run NOTES_AUTH_MODE=dev NOTES_DEV_TOKEN=dev-local-token
```

The server listens on `127.0.0.1:8080` by default and exposes:

```text
/notes.v1.NotesService/*  Connect RPC
/health                   liveness
/readiness                PostgreSQL readiness
/goAppInfo                build metadata
```

All Notes RPCs require `Authorization: Bearer <token>`.

### Available Make Targets

```bash
make help
```

## License

See [LICENSE](LICENSE) file.
