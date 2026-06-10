# MCP Server Authentication & Token Auto-Renewal Design

This document details the authentication design and programmatic token auto-renewal mechanism for the Model Context Protocol (MCP) server (`notes-mcp`).

---

## 1. Overview
The MCP server (`notes-mcp`) will be built to act as a client calling the `notes-server` Connect RPC service (using the Go Connect client package). Since the MCP server runs as a background subprocess managed by the LLM client (e.g. Claude Desktop), it cannot support interactive browser-based logins.

---

## 2. Authentication Strategies

### Strategy A: Local Dev Mode (Static Token)
- **Notes Server Setup**: Run notes-server with `NOTES_AUTH_MODE=dev NOTES_DEV_TOKEN=notes-dev-token`.
- **LLM Host Config** (e.g. `claude_desktop_config.json`):
  ```json
  "mcpServers": {
    "markdown-notes": {
      "command": "/path/to/go-mcp-markdown-notes/bin/notes-mcp",
      "env": {
        "NOTES_SERVER": "http://127.0.0.1:8080",
        "NOTES_TOKEN": "notes-dev-token"
      }
    }
  }
  ```
- **Behavior**: The static token is sent in the `Authorization: Bearer` header of all RPC calls and never expires.

---

### Strategy B: Personal Access Token (PAT) — IMPLEMENTED
This replaces the earlier "admin password auto-renewal" idea: instead of giving
the MCP process a password to mint short-lived JWTs, the user creates a
long-lived **Personal Access Token** once and configures it as an env var.
PATs are issued and revocable centrally by `go-cloud-k8s-auth` and verified by
`notes-server` through the public `POST /goapi/v1/auth/introspect` endpoint
(with a 60 s positive cache).

- **Creating a token**: sign in on the auth service and open
  `http://<AUTH_SERVER>/tokens.html` (also linked from the notes web app as
  "Manage MCP tokens"). Create a token (e.g. scopes `notes:read`,
  `notes:write`, `notes:mcp`); the full `pat_...` value is shown exactly once.
- **LLM Host Config**:
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
- **Behavior**:
  - The token is attached as `Authorization: Bearer <NOTES_TOKEN>` on every
    Connect RPC call (see `pkg/mcpnotes/client.go`).
  - `notes-server` routes `pat_`-prefixed tokens to the auth service
    introspection endpoint (composite verifier in `pkg/authadapter`); all data
    stays scoped to the token owner's `user_id`.
  - No renewal logic is needed: PATs live until they expire or are revoked in
    the tokens UI. Revocation takes effect within at most 60 seconds.
  - On `401 Unauthenticated`, the MCP tools return an explicit message asking
    the user to check `NOTES_TOKEN`.
