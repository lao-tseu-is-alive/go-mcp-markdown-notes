#!/bin/bash

if [ ! -d "./gen" ]; then
  mkdir gen
fi
# see https://buf.build/docs/lint/
echo "## about to buf lint"
buf lint
echo "## about to buf dep update"
buf dep update
echo "## about to buf generate"
buf generate
