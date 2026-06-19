# Stage 1 – Frontend assets
# Mirrors what `make build-frontend` does: runs bun run build.ts → notesFront/dist/.
FROM oven/bun:alpine AS frontend
WORKDIR /app
COPY cmd/notes-server/notesFront/package.json cmd/notes-server/notesFront/bun.lock ./
RUN bun install --frozen-lockfile
COPY cmd/notes-server/notesFront/ ./
RUN bun run build

# Stage 2 – Go binary
# Mirrors the go build step from `make build`, without the test step which requires a live DB.
FROM golang:1-alpine AS builder
LABEL maintainer="cgil"
RUN apk add --no-cache make git
WORKDIR /app
COPY go.mod go.sum ./
RUN make mod-download
COPY . .
# Inject the pre-built frontend assets so the Go embed directive can find them.
COPY --from=frontend /app/dist ./cmd/notes-server/notesFront/dist
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/notes-server ./cmd/notes-server

# Stage 3 – Minimal runtime image
FROM scratch
USER 1221:1221
WORKDIR /goapp
COPY --from=builder /app/bin/notes-server .

# --- Database ---------------------------------------------------------------
# Provide either DATABASE_URL (full DSN, takes precedence) or the individual
# DB_* variables. DB_PASSWORD is required when DATABASE_URL is not set.
ENV DATABASE_URL=""
ENV DB_HOST="127.0.0.1"
ENV DB_PORT="5432"
ENV DB_NAME="go_mcp_notes"
ENV DB_USER="go_mcp_notes"
ENV DB_PASSWORD=""
ENV DB_SSL_MODE="prefer"

# --- Authentication ----------------------------------------------------------
# NOTES_AUTH_MODE: "jwt" (default, production) or "dev" (local dev only).
ENV NOTES_AUTH_MODE="jwt"

# AUTH_SERVER_URL: base URL of the external auth service used for PAT
# introspection (/goapi/v1/auth/introspect) and the browser login flow.
ENV AUTH_SERVER_URL="http://localhost:9090"

# JWT settings — all three are required when NOTES_AUTH_MODE=jwt.
# JWT_SECRET: shared HMAC secret used to verify tokens locally.
ENV JWT_SECRET=""
# JWT_ISSUER_ID: expected "iss" claim in incoming JWTs.
ENV JWT_ISSUER_ID=""
# JWT_CONTEXT_KEY: key name used to store the parsed JWT in the request context.
ENV JWT_CONTEXT_KEY=""
# JWT_DURATION_MINUTES: token lifetime in minutes (1–1440).
# Leave empty to use the auth service default.
ENV JWT_DURATION_MINUTES=""

# --- Dev-mode auth (NOTES_AUTH_MODE=dev only) --------------------------------
# NOTES_DEV_TOKEN is required when auth mode is "dev". Never use in production.
ENV NOTES_DEV_TOKEN=""
ENV NOTES_DEV_USER_ID="1"
ENV NOTES_DEV_USER_EMAIL="dev@localhost"
ENV NOTES_DEV_USER_NAME="Local Notes User"

# --- Server tuning -----------------------------------------------------------
# NOTES_LISTEN_ADDRESS: host:port the HTTP server binds to.
# Use 0.0.0.0 in a container so the port is reachable from outside.
ENV NOTES_LISTEN_ADDRESS="0.0.0.0:8080"
# NOTES_DB_MAX_CONNECTIONS: pgxpool max connections (1–1000).
ENV NOTES_DB_MAX_CONNECTIONS="10"
# NOTES_SHUTDOWN_TIMEOUT_SECONDS: graceful shutdown window in seconds (1–300).
ENV NOTES_SHUTDOWN_TIMEOUT_SECONDS="10"
# NOTES_REQUEST_TIMEOUT_SECONDS: per-request deadline in seconds (1–300).
ENV NOTES_REQUEST_TIMEOUT_SECONDS="10"
# LOG_LEVEL: debug | info | warn | error.
ENV LOG_LEVEL="info"

EXPOSE 8080

# The /health endpoint returns {"status":"ok"} — use it for liveness probes at
# the orchestration layer (e.g. k8s or compose). No HEALTHCHECK here because
# the scratch image has no shell or curl.
CMD ["./notes-server"]
