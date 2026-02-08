#!/usr/bin/env bash
set -euo pipefail

ACTION="${1:-up}"
: "${POSTGRES_DSN:=postgres://app:app@localhost:5432/tgapp?sslmode=disable}"

if ! command -v goose >/dev/null 2>&1; then
  echo "goose not found, installing to GOPATH/bin..."
  go install github.com/pressly/goose/v3/cmd/goose@latest
fi

"$(go env GOPATH)/bin/goose" -dir migrations postgres "$POSTGRES_DSN" "$ACTION"
