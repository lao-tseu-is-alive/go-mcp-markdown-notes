# Connect RPC Notes CLI Client (`notes-client`)

`notes-client` is a command-line interface utility to interact with the Connect RPC `NotesService` server. It supports all RPC operations (create, retrieve, list, search, tag, update) and offers both pretty-printed terminal layouts and structured JSON outputs.

---

## Installation / Building

To build the client binary, compile it from the root of the repository:
```bash
go build -o bin/notes-client ./cmd/notes-client
```

---

## Authentication Configuration

All RPC operations require authentication. You must configure the client with a bearer token by either setting the `NOTES_TOKEN` environment variable or passing the `-token` (or `-t`) flag.

The token to use depends on the **Authentication Mode** of the notes-server.

### Mode A: Local Dev Mode (Easiest for testing)
In development mode, the notes-server accepts a static predefined token.

1. **Start the notes-server** in dev mode:
   ```bash
   NOTES_AUTH_MODE=dev NOTES_DEV_TOKEN=notes-dev-token make run
   ```
2. **Configure the client** in your shell using the dev token:
   ```bash
   export NOTES_TOKEN=notes-dev-token
   ```

---

### Mode B: JWT Token Mode (Production/Integration)
In JWT mode, the notes-server verifies tokens issued by the `go-cloud-k8s-auth` server.

1. **Start the notes-server** in JWT mode:
   ```bash
   make run
   ```
2. **Obtain the JWT token** from the auth server:
   - Navigate to the auth portal at **`http://localhost:9090/`**.
   - Authenticate (e.g., click GitHub login).
   - Once redirected back, scroll down to the **CONSOLE DE DEBUG & RÉPONSES API** and copy the long `"jwt"` string value from the JSON response.
   - Alternatively, open your browser's Developer Tools (F12) -> Console, and type `localStorage.getItem("goJWT_token")` to print it.
3. **Configure the client** in your shell using the copied token:
   ```bash
   export NOTES_TOKEN="eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ..."
   ```

---

## Command Reference

### Global Flags
Global flags must be specified **before** the subcommand:
- `-server` / `-s`: Target server base URL (default: `http://127.0.0.1:8080`).
- `-token` / `-t`: Override bearer token.
- `-format` / `-f`: Output formatting mode: `text` (default, colored layout) or `json` (raw JSON).

```bash
# Example using global flags
bin/notes-client -s http://127.0.0.1:8080 -f json list
```

---

### Subcommands

#### 1. `create`
Create a new note with a title, markdown body, optional category, and optional tags.
```bash
bin/notes-client create \
  -title "Docker Tutorial" \
  -body "# Getting Started with Docker\nRun 'docker run hello-world'." \
  -category "devops" \
  -tags "docker,tutorial,containers"
```

#### 2. `list`
List recent notes (returns note metadata in a tabular format).
```bash
# Retrieve the default 10 recent notes
bin/notes-client list

# Retrieve up to 25 notes
bin/notes-client list -limit 25
```

#### 3. `get`
Retrieve details and body of a specific note using its UUID (supports positional ID or `-id` flag).
```bash
# Positional ID (recommended)
bin/notes-client get "e4db7c12-32b4-4b55-a4b7-34cb052db8a8"

# Using the -id flag
bin/notes-client get -id "e4db7c12-32b4-4b55-a4b7-34cb052db8a8"
```

#### 4. `search`
Search note content (title/body text), filter by category, or filter by tags.
```bash
# Search for the term "docker"
bin/notes-client search -query "docker"

# Filter search by category
bin/notes-client search -query "setup" -category "devops"

# Filter search by tags
bin/notes-client search -tags "docker,tutorial"
```

#### 5. `tag`
Append new tags to an existing note.
```bash
bin/notes-client tag -id "e4db7c12-32b4-4b55-a4b7-34cb052db8a8" -tags "extra-tag,kubernetes"
```

#### 6. `update`
Modify the title, body, category, or tags of an existing note.
```bash
bin/notes-client update \
  -id "e4db7c12-32b4-4b55-a4b7-34cb052db8a8" \
  -title "Updated Docker Tutorial" \
  -body "# Getting Started with Docker\nUpdated text..."
```

---

## Scripting Integration (JSON Piping)

If you use the `-format json` flag, output is emitted as clean JSON. This makes it easy to integrate with other command-line tools like `jq`.

```bash
# Extract the ID of the last created note
NOTE_ID=$(bin/notes-client -f json list -limit 1 | jq -r '.[0].id')
echo "Last Note ID is: $NOTE_ID"

# Get details and extract just the note category
bin/notes-client -f json get "$NOTE_ID" | jq -r '.note.category'
```
