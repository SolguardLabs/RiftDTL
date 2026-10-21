#!/usr/bin/env bash
set -euo pipefail
test -z "$(gofmt -l src)"
go test ./...
go vet ./...
npm test
npm run loc
