#!/usr/bin/env bash
set -euo pipefail

# ==========================================================================
# Full verification pipeline: typecheck → unit tests → Go tests
# Usage: bash scripts/check.sh
# ==========================================================================

ENV_FILE="${ENV_FILE:-.env}"
if [ ! -f "$ENV_FILE" ]; then
  echo "Missing env file: $ENV_FILE"
  echo "Create .env from .env.example, or run 'make worktree-env' and use .env.worktree."
  exit 1
fi

set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
set +a

# shellcheck disable=SC1091
. scripts/local-env.sh

EXIT_CODE=0

cleanup() {
  echo ""
  if [ "$EXIT_CODE" -eq 0 ]; then
    echo "✓ All checks passed."
  else
    echo "✗ Checks FAILED."
  fi
  exit "$EXIT_CODE"
}
trap cleanup EXIT

echo "==> Using env file: $ENV_FILE"
echo "==> Checking PostgreSQL..."
bash scripts/ensure-postgres.sh "$ENV_FILE"

echo ""
echo "==> [1/3] TypeScript typecheck..."
pnpm typecheck || { EXIT_CODE=1; exit 1; }

echo ""
echo "==> [2/3] TypeScript unit tests..."
pnpm test || { EXIT_CODE=1; exit 1; }

echo ""
echo "==> [3/3] Go tests..."
echo "==> Running database migrations..."
(cd server && go run ./cmd/migrate up) || { EXIT_CODE=1; exit 1; }
(cd server && go test ./...) || { EXIT_CODE=1; exit 1; }
