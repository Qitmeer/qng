#!/usr/bin/env bash
set -ex

export GO111MODULE=on
#go mod tidy

if [ ! -x "$(type -p golangci-lint)" ]; then
  exit 1
fi

golangci-lint --version
golangci-lint run -v --timeout=2m --disable-all --enable=govet --tests=false ./...
