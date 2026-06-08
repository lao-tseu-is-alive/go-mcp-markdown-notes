#!/bin/bash
echo "## $0 received NUM ARGS : " $#
APP_REPOSITORY=github.com/lao-tseu-is-alive/go-cloud-k8s-common-libs
NOW=$(date +%Y-%m-%dT%T)
REVISION="$(git describe --dirty --always)"
LDFLAGS="-X ${APP_REPOSITORY}/pkg/version.BuildStamp=${NOW} -X ${APP_REPOSITORY}/pkg/version.REVISION=${REVISION}"
ENV_FILENAME='.env'

if [[ $# -eq 1 ]]; then
  ENV_FILENAME=${1:-.env}    # ← fixed the default value syntax
else
  echo "## 💥💥 expecting first argument to be path to an .env file name (default is .env in current directory)"
  exit 1
fi

echo "## will try to run : go test -race -coverprofile coverage.out -v ./... with env variables in ${ENV_FILENAME} ..."

if [[ -r "$ENV_FILENAME" ]]; then
    echo "## will do : go test -race -coverprofile=coverage.txt -v ./... "
    set -a
    source <(sed -e '/^#/d;/^\s*$/d' -e "s/'/'\\\''/g" -e "s/=\(.*\)/='\1'/g" $ENV_FILENAME )
    go test -race -coverprofile=coverage.txt -v -coverpkg=./cmd/...,./pkg/... ./...
    set +a
    go tool cover -func=coverage.txt | tail -1
else
  echo "## 💥💥 env path argument : ${ENV_FILENAME} was not found"
  exit 1
fi