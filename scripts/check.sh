#!/bin/sh
set -eu
ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT"
gofmt -w $(find . -name '*.go' -not -name '*_test.go')
go vet ./...
go test ./...
find . -name '*.go' -not -name '*_test.go' -print0 | xargs -0 cat | wc -l

