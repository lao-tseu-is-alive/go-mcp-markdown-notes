#!make
SHELL := /bin/bash
DOCKER_BIN := nerdctl
VER_SOURCE_CODE := pkg/version/version.go
APP_NAME := $(shell grep -E 'AppName\s+=' $(VER_SOURCE_CODE)| awk '{ print $$3 }'  | tr -d '"')
APP_VERSION := $(shell grep -E 'Version\s+=' $(VER_SOURCE_CODE)| awk '{ print $$3 }'  | tr -d '"')
APP_REPOSITORY := $(shell grep -E 'Repository\s+=' $(VER_SOURCE_CODE)| awk '{ print $$3 }'  | tr -d '"')
$(info  Found APP_NAME:'$(APP_NAME)', APP_VERSION:'$(APP_VERSION)', APP_REPOSITORY:'$(APP_REPOSITORY)',  in file: $(VER_SOURCE_CODE) )
ifneq ("$(wildcard .env)","")
	ENV_EXISTS := "TRUE"
	include .env
	export $(shell sed 's/=.*//' .env)
else
    $(info env file was not found using default values for undefined variables )
    ENV_EXISTS := "FALSE"
	DB_DRIVER ?= postgres
	DB_HOST ?= 127.0.0.1
	DB_PORT ?= 5432
	DB_NAME ?= go_mcp_notes_db_schema
	DB_USER ?= go_mcp_notes_db_schema
	DB_SSL_MODE ?= disable
endif
APP_EXECUTABLE := notes-server
APP_REVISION := $(shell git describe --dirty --always)
BUILD := $(shell date -u '+%Y-%m-%d_%I:%M:%S%p')
PACKAGES := $(shell go list ./... | grep -v /vendor/)
LDFLAGS := -ldflags "-X ${APP_REPOSITORY}/pkg/version.Revision=${APP_REVISION} -X ${APP_REPOSITORY}/pkg/version.BuildStamp=${BUILD}"

MAKEFLAGS += --silent

.PHONY: run
## run:	will run the main server binary [DEFAULT RULE]
.PHONY: run-server
## run-server:	will run notes-server
run-server: run

.PHONY: build-frontend
build-frontend:
	@echo "  >  Building frontend assets using Bun..."
	cd cmd/notes-server/notesFront && bun run build

run: build-frontend mod-download
	go run $(LDFLAGS) ./cmd/$(APP_EXECUTABLE)

.PHONY: mod-download
mod-download:
	@echo "  >  Downloading go modules dependencies..."
	go mod download

.PHONY: build
## build:	will compile the main server binary and place it in the bin sub-folder
build: clean build-frontend mod-download test
	@echo "  >  Building your app binary inside bin directory..."
	CGO_ENABLED=0 go build ${LDFLAGS} -a -o bin/$(APP_EXECUTABLE) ./cmd/$(APP_EXECUTABLE)

.PHONY: generate
## generate:	will run buf generate to generate protobuf/connect code
generate:
	./scripts/buf_generate.sh

# Check if .env_testing exists and include it if it does
ifneq ("$(wildcard .env_testing)","")
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
	rm -rf bin/$(APP_EXECUTABLE) coverage.out coverage-all.out

.PHONY: release
## release:	will build & tag a clean repo with a version release
release: build
	@echo "  >  Preparing release $(APP_EXECUTABLE) v$(APP_VERSION) rev: $(APP_REVISION) ..."
ifeq ($(shell git status -s),)
	echo "OK : your repo is clean"
	@git fetch  ||  (echo "ERROR : git fetch failed" && exit 1)
	@git tag -l  "v${APP_VERSION}"  ||  (echo "ERROR : this git tag v${APP_VERSION} already exist" && exit 1)
	git tag "v${APP_VERSION}" -m "v${APP_VERSION} bump"
else
	(echo "ERROR : your local git repo is dirty" && ( git status -s) && exit 1)
endif

.PHONY: help
help: Makefile
	@echo
	@echo " Choose a make target from one of  :"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo

.PHONY: run-notes-client
## run-notes-client:	will run the notes-client binary
run-notes-client:
	go run cmd/notes-client/main.go $(ARGS)

.PHONY: run-notes-mcp
## run-notes-mcp:	will run the notes-mcp binary
run-notes-mcp:
	go run cmd/notes-mcp/main.go $(ARGS)

.PHONY: db-status
## db-status:	show dbmate migration status
db-status:
	cd cmd/notes-server && dbmate --env-file ../../.env status

.PHONY: db-up
## db-up:	apply pending dbmate migrations
db-up:
	cd cmd/notes-server && dbmate --env-file ../../.env up

.PHONY: db-down
## db-down:	roll back the latest dbmate migration
db-down:
	cd cmd/notes-server && dbmate --env-file ../../.env down
