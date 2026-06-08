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

### Database Setup

```bash
cp .env_sample .env
# Edit .env with your PostgreSQL credentials
```

### Available Make Targets

```bash
make help
```

## License

See [LICENSE](LICENSE) file.
