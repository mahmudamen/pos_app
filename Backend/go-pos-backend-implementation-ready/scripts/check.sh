#!/usr/bin/env bash
set -euo pipefail

gofmt -l . | tee /tmp/pos-gofmt.txt
if [ -s /tmp/pos-gofmt.txt ]; then
  echo "ERROR: files need gofmt" >&2
  exit 1
fi

go vet ./...
go test ./...
go test -race ./...

if command -v golangci-lint >/dev/null 2>&1; then
  golangci-lint run ./...
else
  echo "WARNING: golangci-lint not installed; skipped."
fi

if command -v gosec >/dev/null 2>&1; then
  gosec ./...
else
  echo "WARNING: gosec not installed; skipped."
fi
