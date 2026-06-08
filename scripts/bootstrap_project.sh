#!/bin/bash
# scripts/bootstrap_project.sh
#
# Reusable project scaffolder for Go microservices following
# the patterns established in go-cloud-k8s-auth.
#
# Usage:
#   ./scripts/bootstrap_project.sh \
#     --name "go-mcp-markdown-notes" \
#     --module "github.com/lao-tseu-is-alive/go-mcp-markdown-notes" \
#     --target-dir "../go-mcp-markdown-notes" \
#     --binaries "notes-server,notes-client,notes-mcp" \
#     --packages "notes,authadapter,mcpnotes" \
#     --proto-package "notes/v1" \
#     --db-schema "go_mcp_notes_db_schema" \
#     --go-version "1.25.5"

set -euo pipefail

# ─── Color helpers ──────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

info()  { echo -e "${GREEN}✅ $*${NC}"; }
warn()  { echo -e "${YELLOW}⚠️  $*${NC}"; }
err()   { echo -e "${RED}💥 $*${NC}" >&2; }
step()  { echo -e "${CYAN}▶  $*${NC}"; }

# ─── Default values ────────────────────────────────────────────────────
NAME=""
MODULE=""
TARGET_DIR=""
BINARIES=""
PACKAGES=""
PROTO_PACKAGE=""
DB_SCHEMA=""
GO_VERSION="1.25.5"
DB_PORT="5432"
SERVER_PORT="8080"

# ─── Parse arguments ───────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case "$1" in
    --name)        NAME="$2";          shift 2 ;;
    --module)      MODULE="$2";        shift 2 ;;
    --target-dir)  TARGET_DIR="$2";    shift 2 ;;
    --binaries)    BINARIES="$2";      shift 2 ;;
    --packages)    PACKAGES="$2";      shift 2 ;;
    --proto-package) PROTO_PACKAGE="$2"; shift 2 ;;
    --db-schema)   DB_SCHEMA="$2";     shift 2 ;;
    --go-version)  GO_VERSION="$2";    shift 2 ;;
    --db-port)     DB_PORT="$2";       shift 2 ;;
    --server-port) SERVER_PORT="$2";   shift 2 ;;
    -h|--help)
      echo "Usage: $0 --name NAME --module MODULE --target-dir DIR --binaries BIN1,BIN2 --packages PKG1,PKG2 --proto-package PKG/V --db-schema SCHEMA [--go-version VER] [--db-port PORT] [--server-port PORT]"
      exit 0
      ;;
    *) err "Unknown option: $1"; exit 1 ;;
  esac
done

# ─── Validate required args ────────────────────────────────────────────
missing=()
[[ -z "$NAME" ]]          && missing+=("--name")
[[ -z "$MODULE" ]]        && missing+=("--module")
[[ -z "$TARGET_DIR" ]]    && missing+=("--target-dir")
[[ -z "$BINARIES" ]]      && missing+=("--binaries")
[[ -z "$PACKAGES" ]]      && missing+=("--packages")
[[ -z "$PROTO_PACKAGE" ]] && missing+=("--proto-package")
[[ -z "$DB_SCHEMA" ]]     && missing+=("--db-schema")

if [[ ${#missing[@]} -gt 0 ]]; then
  err "Missing required arguments: ${missing[*]}"
  echo "Run $0 --help for usage."
  exit 1
fi

# ─── Derived values ────────────────────────────────────────────────────
IFS=',' read -ra BIN_ARRAY <<< "$BINARIES"
IFS=',' read -ra PKG_ARRAY <<< "$PACKAGES"

# First binary is assumed to be the main server
MAIN_SERVER="${BIN_ARRAY[0]}"

# Extract repo short name (last component of module path)
REPO_SHORT="${MODULE##*/}"

# Derive naming variants
APP_NAME_KEBAB="$NAME"
APP_NAME_SNAKE="${NAME//-/_}"
# CamelCase from kebab: go-mcp-markdown-notes -> goMcpMarkdownNotes
APP_NAME_CAMEL=$(echo "$NAME" | sed -E 's/(^|-)([a-z])/\U\2/g; s/^(.)/\l\1/')

# Proto package base name (e.g., "notes" from "notes/v1")
PROTO_BASE="${PROTO_PACKAGE%%/*}"
PROTO_VERSION="${PROTO_PACKAGE##*/}"

echo ""
echo "╔══════════════════════════════════════════════════════════╗"
echo "║          Go Project Bootstrap (go-cloud-k8s style)      ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""
echo "  Name:           $NAME"
echo "  Module:         $MODULE"
echo "  Target:         $TARGET_DIR"
echo "  Binaries:       ${BIN_ARRAY[*]}"
echo "  Packages:       ${PKG_ARRAY[*]}"
echo "  Proto:          $PROTO_PACKAGE"
echo "  DB Schema:      $DB_SCHEMA"
echo "  Go Version:     $GO_VERSION"
echo "  Server Port:    $SERVER_PORT"
echo "  DB Port:        $DB_PORT"
echo "  CamelCase:      $APP_NAME_CAMEL"
echo "  snake_case:     $APP_NAME_SNAKE"
echo ""

read -p "Continue with these values? (y/n): " confirm
[[ "$confirm" =~ ^[yY]$ ]] || { echo "Aborted."; exit 0; }

echo "creating the github repository : ${MODULE}"
gh repo create "${MODULE}" --public --clone

# ─── Create target directory ───────────────────────────────────────────
if [[ -d "$TARGET_DIR" ]]; then
  # Check if non-empty (excluding .git)
  NON_GIT_FILES=$(find "$TARGET_DIR" -maxdepth 1 -not -name '.git' -not -name '.' | head -5)
  if [[ -n "$NON_GIT_FILES" ]]; then
    warn "Target directory '$TARGET_DIR' already contains files."
    read -p "Overwrite existing files? (y/n): " overwrite
    [[ "$overwrite" =~ ^[yY]$ ]] || { echo "Aborted."; exit 0; }
  fi
else
  mkdir -p "$TARGET_DIR"
fi

step "Creating directory structure..."

# ─── Create directories ───────────────────────────────────────────────
for bin in "${BIN_ARRAY[@]}"; do
  mkdir -p "$TARGET_DIR/cmd/$bin"
done

for pkg in "${PKG_ARRAY[@]}"; do
  mkdir -p "$TARGET_DIR/pkg/$pkg"
done

mkdir -p "$TARGET_DIR/pkg/version"
mkdir -p "$TARGET_DIR/proto/$PROTO_BASE/$PROTO_VERSION"
mkdir -p "$TARGET_DIR/gen"
mkdir -p "$TARGET_DIR/cmd/$MAIN_SERVER/db/migrations"
mkdir -p "$TARGET_DIR/scripts"
mkdir -p "$TARGET_DIR/.github/workflows"
mkdir -p "$TARGET_DIR/bin"

info "Directory structure created"

# ─── go.mod ────────────────────────────────────────────────────────────
step "Generating go.mod..."
cat > "$TARGET_DIR/go.mod" << GOMOD
module ${MODULE}

go ${GO_VERSION}
GOMOD
info "go.mod created"

# ─── pkg/version/version.go ───────────────────────────────────────────
step "Generating pkg/version/version.go..."
cat > "$TARGET_DIR/pkg/version/version.go" << 'VERSIONGO'
package version

var (
	// EDIT THESE VALUES AFTER CREATING THE REPO FROM TEMPLATE
	// -----------------------------------------------------

	// AppName  is the CamelCase name of your app (e.g., "User", "Product")
	AppName = "APP_NAME_CAMEL_PLACEHOLDER"

	// GoPackage  is the name of your main service go package (e.g., "user", "product")
	// should be: all lowercase, short no hyphens, no underscores, no camelCase, usually one word
	GoPackage = "PROTO_BASE_PLACEHOLDER"

	// ServiceName is the name of your main entity/service first letter Capital (e.g., "User", "Product")
	ServiceName = "SERVICE_NAME_PLACEHOLDER"

	// DbSchemaName is the name of your main entity/service database schema
	DbSchemaName = "DB_SCHEMA_PLACEHOLDER"

	// AppNameKebab is the kebab-case version for your github repository
	AppNameKebab = "APP_NAME_KEBAB_PLACEHOLDER"

	// AppNameSnake is the snake-case version for database or directory
	AppNameSnake = "APP_NAME_SNAKE_PLACEHOLDER"

	// Repository is the full GitHub module path
	Repository = "MODULE_PLACEHOLDER"

	// Version starting point
	Version = "0.0.1"

	// Revision is auto-filled by build (do not edit manually)
	Revision = "unknown"
	// BuildStamp is auto-filled by build (do not edit manually)
	BuildStamp = "unknown"
)
VERSIONGO

# Replace placeholders in version.go
# Derive ServiceName: capitalize first letter of PROTO_BASE
SERVICE_NAME="$(echo "${PROTO_BASE:0:1}" | tr '[:lower:]' '[:upper:]')${PROTO_BASE:1}"

sed -i \
  -e "s|APP_NAME_CAMEL_PLACEHOLDER|$APP_NAME_CAMEL|g" \
  -e "s|PROTO_BASE_PLACEHOLDER|$PROTO_BASE|g" \
  -e "s|SERVICE_NAME_PLACEHOLDER|$SERVICE_NAME|g" \
  -e "s|DB_SCHEMA_PLACEHOLDER|$DB_SCHEMA|g" \
  -e "s|APP_NAME_KEBAB_PLACEHOLDER|$APP_NAME_KEBAB|g" \
  -e "s|APP_NAME_SNAKE_PLACEHOLDER|$APP_NAME_SNAKE|g" \
  -e "s|MODULE_PLACEHOLDER|$MODULE|g" \
  "$TARGET_DIR/pkg/version/version.go"

info "pkg/version/version.go created"

# ─── buf.yaml ──────────────────────────────────────────────────────────
step "Generating buf.yaml..."
cat > "$TARGET_DIR/buf.yaml" << 'BUFYAML'
# For details on buf.yaml configuration, visit https://buf.build/docs/configuration/v2/buf-yaml
version: v2
modules:
  - path: proto

deps:
  - buf.build/bufbuild/protovalidate    #https://github.com/bufbuild/protovalidate
  - buf.build/googleapis/googleapis
lint:
  use:
    - STANDARD
breaking:
  use:
    - FILE
BUFYAML
info "buf.yaml created"

# ─── buf.gen.yaml ──────────────────────────────────────────────────────
step "Generating buf.gen.yaml..."
cat > "$TARGET_DIR/buf.gen.yaml" << BUFGENYAML
version: v2
plugins:
  # 1. Generate Go Code (gRPC + Models)
  - local: protoc-gen-go
    out: gen
    opt: paths=source_relative

  # 2. Generate ConnectRPC Code (Interfaces for HTTP)
  - local: protoc-gen-connect-go
    out: gen
    opt:
      - paths=source_relative

  # 3. Generate OpenAPI (The Verification Step)
  - remote: buf.build/grpc-ecosystem/openapiv2
    out: api/openapi
    opt:
      - json_names_for_fields=true
      - output_format=yaml
      - allow_merge=true
      - merge_file_name=${PROTO_BASE}
      - simple_operation_ids=true
managed:
  enabled: true
  disable:
    - file_option: go_package_prefix
      module: buf.build/googleapis/googleapis
    - file_option: go_package_prefix
      module: buf.build/bufbuild/protovalidate
  override:
    - file_option: go_package_prefix
      value: ${MODULE}/gen
BUFGENYAML
info "buf.gen.yaml created"

# ─── scripts/buf_generate.sh ──────────────────────────────────────────
step "Generating scripts/buf_generate.sh..."
cat > "$TARGET_DIR/scripts/buf_generate.sh" << 'BUFGEN'
#!/bin/bash

if [ ! -d "./gen" ]; then
  mkdir gen
fi
# see https://buf.build/docs/lint/
buf lint
buf dep update
buf generate
BUFGEN
chmod +x "$TARGET_DIR/scripts/buf_generate.sh"
info "scripts/buf_generate.sh created"

# ─── Makefile ──────────────────────────────────────────────────────────
step "Generating Makefile..."
cat > "$TARGET_DIR/Makefile" << MAKEFILE
#!make
SHELL := /bin/bash
DOCKER_BIN := nerdctl
VER_SOURCE_CODE := pkg/version/version.go
APP_NAME := \$(shell grep -E 'AppName\\s+=' \$(VER_SOURCE_CODE)| awk '{ print \$\$3 }'  | tr -d '"')
APP_VERSION := \$(shell grep -E 'Version\\s+=' \$(VER_SOURCE_CODE)| awk '{ print \$\$3 }'  | tr -d '"')
APP_REPOSITORY := \$(shell grep -E 'Repository\\s+=' \$(VER_SOURCE_CODE)| awk '{ print \$\$3 }'  | tr -d '"')
\$(info  Found APP_NAME:'\$(APP_NAME)', APP_VERSION:'\$(APP_VERSION)', APP_REPOSITORY:'\$(APP_REPOSITORY)',  in file: \$(VER_SOURCE_CODE) )
ifneq ("\$(wildcard .env)","")
	ENV_EXISTS := "TRUE"
	include .env
	export \$(shell sed 's/=.*//' .env)
else
    \$(info env file was not found using default values for undefined variables )
    ENV_EXISTS := "FALSE"
	DB_DRIVER ?= postgres
	DB_HOST ?= 127.0.0.1
	DB_PORT ?= ${DB_PORT}
	DB_NAME ?= ${DB_SCHEMA}
	DB_USER ?= ${DB_SCHEMA}
	DB_SSL_MODE ?= disable
endif
APP_EXECUTABLE := ${MAIN_SERVER}
APP_REVISION := \$(shell git describe --dirty --always)
BUILD := \$(shell date -u '+%Y-%m-%d_%I:%M:%S%p')
PACKAGES := \$(shell go list ./... | grep -v /vendor/)
LDFLAGS := -ldflags "-X \${APP_REPOSITORY}/pkg/version.Revision=\${APP_REVISION} -X \${APP_REPOSITORY}/pkg/version.BuildStamp=\${BUILD}"

MAKEFLAGS += --silent

.PHONY: run
## run:	will run the main server binary [DEFAULT RULE]
run: mod-download
	go run \$(LDFLAGS) cmd/\$(APP_EXECUTABLE)/main.go

.PHONY: mod-download
mod-download:
	@echo "  >  Downloading go modules dependencies..."
	go mod download

.PHONY: build
## build:	will compile the main server binary and place it in the bin sub-folder
build: clean mod-download test
	@echo "  >  Building your app binary inside bin directory..."
	CGO_ENABLED=0 go build \${LDFLAGS} -a -o bin/\$(APP_EXECUTABLE) cmd/\$(APP_EXECUTABLE)/main.go

.PHONY: generate
## generate:	will run buf generate to generate protobuf/connect code
generate:
	./scripts/buf_generate.sh

# Check if .env_testing exists and include it if it does
ifneq ("\$(wildcard .env_testing)","")
include .env_testing
env-test-export:
	@echo "Exporting environment variables from .env_testing..."
	sed -ne '/^export / {p;d}; /.*=/ s/^/export / p' .env_testing > .env-testing-export

.PHONY: test
## test:	will run all tests
test: clean mod-download env-test-export
	@echo "  >  Running all tests..."
	. .env-testing-export && go test -race -coverprofile coverage.out -coverpkg=./... ./...

else
env-test-export:
	@echo ".env_testing file does not exist, skipping export..."

.PHONY: test
test: clean mod-download env-test-export
	@echo "  >  Running all tests..."
	go test -race -coverprofile coverage.out -coverpkg=./... ./...

endif

.PHONY: lint
## lint:	will run go vet and buf lint
lint:
	go vet ./...
	buf lint

.PHONY: fmt
## fmt:	will format all Go source files
fmt:
	gofmt -w .

.PHONY: clean
## clean:	will remove binaries and coverage files
clean:
	@echo "  >  Removing binaries and coverage..."
	rm -rf bin/\$(APP_EXECUTABLE) coverage.out coverage-all.out

.PHONY: release
## release:	will build & tag a clean repo with a version release
release: build
	@echo "  >  Preparing release \$(APP_EXECUTABLE) v\$(APP_VERSION) rev: \$(APP_REVISION) ..."
ifeq (\$(shell git status -s),)
	echo "OK : your repo is clean"
	@git fetch  ||  (echo "ERROR : git fetch failed" && exit 1)
	@git tag -l  "v\${APP_VERSION}"  ||  (echo "ERROR : this git tag v\${APP_VERSION} already exist" && exit 1)
	git tag "v\${APP_VERSION}" -m "v\${APP_VERSION} bump"
else
	(echo "ERROR : your local git repo is dirty" && ( git status -s) && exit 1)
endif

.PHONY: help
help: Makefile
	@echo
	@echo " Choose a make target from one of  :"
	@echo
	@sed -n 's/^##//p' \$< | column -t -s ':' |  sed -e 's/^/ /'
	@echo
MAKEFILE

# Add run targets for each binary
for bin in "${BIN_ARRAY[@]}"; do
  if [[ "$bin" != "$MAIN_SERVER" ]]; then
    cat >> "$TARGET_DIR/Makefile" << MAKETARGET

.PHONY: run-${bin}
## run-${bin}:	will run the ${bin} binary
run-${bin}:
	go run cmd/${bin}/main.go \$(ARGS)
MAKETARGET
  fi
done

info "Makefile created"

# ─── Dockerfile ────────────────────────────────────────────────────────
step "Generating Dockerfile..."
cat > "$TARGET_DIR/Dockerfile" << DOCKERFILE
# Start from the latest golang base image
FROM golang:${GO_VERSION%%.*}-alpine AS builder

LABEL maintainer="cgil"

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/${MAIN_SERVER} ./${MAIN_SERVER}
COPY pkg ./pkg
COPY gen ./gen

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ${MAIN_SERVER} ./${MAIN_SERVER}

######## Start a new stage  #######
FROM scratch
USER 1221:1221
WORKDIR /goapp

COPY --from=builder /app/${MAIN_SERVER} .

ENV PORT="\${PORT}"
ENV DB_DRIVER="\${DB_DRIVER}"
ENV DB_HOST="\${DB_HOST}"
ENV DB_PORT="\${DB_PORT}"
ENV DB_NAME="\${DB_NAME}"
ENV DB_USER="\${DB_USER}"
ENV DB_PASSWORD="\${DB_PASSWORD}"
ENV DB_SSL_MODE="\${DB_SSL_MODE}"

EXPOSE ${SERVER_PORT}

HEALTHCHECK --start-period=5s --interval=30s --timeout=3s \\
    CMD curl --fail http://localhost:${SERVER_PORT}/health || exit 1

CMD ["./${MAIN_SERVER}"]
DOCKERFILE
info "Dockerfile created"

# ─── .github/workflows/test.yml ───────────────────────────────────────
step "Generating .github/workflows/test.yml..."
cat > "$TARGET_DIR/.github/workflows/test.yml" << TESTYML
name: test

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:17
        env:
          POSTGRES_PASSWORD: postgres
        ports:
          - 5432:5432
        options: --health-cmd pg_isready --health-interval 10s --health-timeout 5s --health-retries 5

    steps:
    - uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version-file: 'go.mod'
    - run: go version

    - name: Test
      env:
        PORT: ${SERVER_PORT}
        DB_DRIVER: postgres
        DB_HOST: 127.0.0.1
        DB_PORT: 5432
        DB_SSL_MODE: prefer
        DB_NAME: postgres
        DB_USER: postgres
        DB_PASSWORD: postgres
        PGPASSWORD: postgres
      run: go test -race -coverprofile coverage.out -coverpkg=./cmd/...,./pkg/... ./...

    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v4
      with:
        token: \${{ secrets.CODECOV_TOKEN }}
        file: ./coverage.out
TESTYML
info ".github/workflows/test.yml created"

# ─── .gitignore ────────────────────────────────────────────────────────
step "Generating .gitignore..."
cat > "$TARGET_DIR/.gitignore" << 'GITIGNORE'
# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, built with `go test -c`
*.test

# Coverage output
.coverage.out
/coverage.out
/coverage-all.out
/coverage.txt

# Environment files (secrets!)
/.env
/.env_testing
/.env-testing-export
/.env_for_github.txt

# IDE
.idea
.vscode

# Dependencies
# vendor/

# Generated / build
/bin/
proto_third_party/*
**/node_modules
/openapitools.json
GITIGNORE
info ".gitignore created"

# ─── .dockerignore ─────────────────────────────────────────────────────
step "Generating .dockerignore..."
cat > "$TARGET_DIR/.dockerignore" << 'DOCKERIGNORE'
.idea
*.md
.env
.env_sample
.gitignore
Dockerfile
Makefile
scripts/*
api/*
deployments/*
.github/*
DOCKERIGNORE
info ".dockerignore created"

# ─── .env_sample ──────────────────────────────────────────────────────
step "Generating .env_sample..."
cat > "$TARGET_DIR/.env_sample" << ENVSAMPLE
# rename this file to .env and adapt it to your needs
# do not put this file in your git, it will expose your passwords!
# in github use github secrets: https://docs.github.com/en/actions/security-guides/encrypted-secrets

######### SERVER CONFIGURATION #########
PORT=${SERVER_PORT}

######### DATABASE CONFIGURATION #########
DB_DRIVER=postgres
DB_HOST=127.0.0.1
DB_PORT=${DB_PORT}
DB_NAME=${DB_SCHEMA}
DB_USER=${DB_SCHEMA}
DB_PASSWORD=Choose_your_own_password_here
DB_SSL_MODE=prefer

######### JSON WEB TOKEN CONFIGURATION #########
JWT_SECRET="Use your nice and complicated token here"
JWT_ISSUER_ID=2490AD68-3AA9-4C17-BB49-33C2F202B754
JWT_CONTEXT_KEY=yourContextKey
JWT_DURATION_MINUTES=15
JWT_AUTH_URL=/login
JWT_STATUS_URL=/status

######### ADMIN USER #########
ADMIN_USER=goadmin
ADMIN_EMAIL=goadmin@yourdomain.org
ADMIN_ID=960901
ADMIN_PASSWORD=Choose_your_own_admin_password_here

######### LOG CONFIGURATION #########
# LOG_FILE: string containing the filename, OR stdout, stderr, DISCARD. default: stderr
LOG_FILE=stderr
# LOG_LEVEL: debug, info, warn, error, fatal (or 0-4)
LOG_LEVEL=info
ENVSAMPLE
info ".env_sample created"

# ─── cmd/*/main.go placeholder files ──────────────────────────────────
step "Generating cmd/ placeholder binaries..."
for bin in "${BIN_ARRAY[@]}"; do
  cat > "$TARGET_DIR/cmd/$bin/main.go" << MAINGO
package main

import (
	"fmt"
	"${MODULE}/pkg/version"
)

func main() {
	fmt.Printf("🚀 Starting %s v%s (rev: %s, build: %s)\\n",
		version.AppName, version.Version, version.Revision, version.BuildStamp)
	// TODO: implement ${bin}
	fmt.Println("⚠️  ${bin} is not yet implemented")
}
MAINGO
  info "cmd/$bin/main.go created"
done

# ─── Migration directory for main server ──────────────────────────────
step "Generating db/migrations placeholder..."
cat > "$TARGET_DIR/cmd/$MAIN_SERVER/db/migrations/.gitkeep" << 'GITKEEP'
GITKEEP
info "db/migrations/.gitkeep created"

# ─── pkg/*/placeholder files ──────────────────────────────────────────
step "Generating pkg/ placeholder packages..."
for pkg in "${PKG_ARRAY[@]}"; do
  cat > "$TARGET_DIR/pkg/$pkg/doc.go" << DOCGO
// Package ${pkg} provides the ${pkg} functionality for ${NAME}.
package ${pkg}
DOCGO
  info "pkg/$pkg/doc.go created"
done

# ─── proto placeholder ─────────────────────────────────────────────────
step "Generating proto/ placeholder..."
cat > "$TARGET_DIR/proto/$PROTO_BASE/$PROTO_VERSION/.gitkeep" << 'GITKEEP'
GITKEEP
# Also add a .gitignore for the gen directory
cat > "$TARGET_DIR/proto/.gitignore" << 'PROTOGITIGNORE'
third_party/
PROTOGITIGNORE
info "proto/ placeholder created"

# ─── README.md ─────────────────────────────────────────────────────────
step "Generating README.md..."
cat > "$TARGET_DIR/README.md" << README
# ${NAME}

> ${NAME} — a Go microservice built with ConnectRPC, PostgreSQL, and the go-cloud-k8s patterns.

## Quick Start

\`\`\`bash
# Generate protobuf/Connect code
make generate

# Run tests
make test

# Run the main server
make run
\`\`\`

## Project Structure

\`\`\`text
${NAME}/
├── cmd/                    # Application binaries
$(for bin in "${BIN_ARRAY[@]}"; do echo "│   ├── ${bin}/"; done)
├── pkg/                    # Shared packages
$(for pkg in "${PKG_ARRAY[@]}"; do echo "│   ├── ${pkg}/"; done)
│   └── version/            # Build version info
├── proto/                  # Protobuf definitions
│   └── ${PROTO_PACKAGE}/
├── gen/                    # Generated protobuf/Connect code
├── .github/workflows/      # CI/CD pipelines
├── Makefile
├── Dockerfile
└── README.md
\`\`\`

## Development

### Prerequisites

- Go ${GO_VERSION}+
- PostgreSQL
- [buf](https://buf.build/docs/installation) for protobuf generation
- protoc-gen-go, protoc-gen-connect-go

### Database Setup

\`\`\`bash
cp .env_sample .env
# Edit .env with your PostgreSQL credentials
\`\`\`

### Available Make Targets

\`\`\`bash
make help
\`\`\`

## License

See [LICENSE](LICENSE) file.
README
info "README.md created"

# ─── LICENSE (AGPL-3.0 placeholder) ───────────────────────────────────
step "Generating LICENSE placeholder..."
cat > "$TARGET_DIR/LICENSE" << 'LICENSE'
GNU AFFERO GENERAL PUBLIC LICENSE
Version 3, 19 November 2007

See https://www.gnu.org/licenses/agpl-3.0.html for the full license text.
LICENSE
info "LICENSE placeholder created"

# ─── Summary ───────────────────────────────────────────────────────────
# Nettoie un éventuel slash final avant le basename
MODULE_CLEAN="${MODULE%/}"
rsync -av "${MODULE_CLEAN##*/}/" "$TARGET_DIR/"

echo ""
echo "╔══════════════════════════════════════════════════════════╗"
echo "║          ✅ Project scaffold complete!                   ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""
echo "  Target:  $TARGET_DIR"
echo ""
echo "  Generated files:"
find "$TARGET_DIR" -type f -not -path '*/.git/*' | sort | while read -r f; do
  echo "    ${f#$TARGET_DIR/}"
done
echo ""
echo "  Next steps:"
echo "    1. cd $TARGET_DIR"
echo "    2. Write your .proto files in proto/$PROTO_PACKAGE/"
echo "    3. Run: make generate"
echo "    4. Implement domain logic in pkg/"
echo "    5. Implement server in cmd/$MAIN_SERVER/"
echo "    6. Run: go mod tidy && make test"
echo ""
