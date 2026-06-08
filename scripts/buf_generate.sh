#!/bin/bash

if [ ! -d "./gen" ]; then
  mkdir gen
fi
# see https://buf.build/docs/lint/
buf lint
buf dep update
buf generate
