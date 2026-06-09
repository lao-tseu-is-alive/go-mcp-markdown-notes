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

### Strategy B: JWT Mode (Token Auto-Renewal)
Since JWT tokens expire after 15 minutes, the MCP server must dynamically renew its token programmatically in the background without user intervention.

- **LLM Host Config**:
  ```json
  "mcpServers": {
    "markdown-notes": {
      "command": "/path/to/go-mcp-markdown-notes/bin/notes-mcp",
      "env": {
        "NOTES_SERVER": "http://127.0.0.1:8080",
        "AUTH_SERVER": "http://127.0.0.1:9090",
        "ADMIN_USER": "<your_admin_user>",
        "ADMIN_PASSWORD": "<your_admin_password>"
      }
    }
  }
  ```
- **Login Request**:
  1. The MCP server reads `ADMIN_USER` and `ADMIN_PASSWORD` from its environment.
  2. It hashes the password using SHA-256: `hash = sha256(password)`.
  3. It sends a `POST` request to `http://<AUTH_SERVER>/login` with the payload `{"username": "$ADMIN_USER", "password_hash": "$hash"}` to retrieve a fresh JWT token.
- **In-Memory Caching & Verification**:
  - The fetched token is stored in-memory in the Go process.
  - The Connect client attaches the token in the `Authorization: Bearer <token>` header of every outgoing RPC request.
- **Auto-Healing Interceptor**:
  - The Go client wraps requests in an interceptor. If an RPC call returns a `401 Unauthenticated` error (indicating the cached token has expired), the interceptor catches it, automatically invokes the login routine to refresh the token, updates the cache, and retries the failed RPC call.
