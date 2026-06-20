#!/usr/bin/env bash
# scripts/check.sh — local CI-parity gate.
# Mirrors .github/workflows/ci.yml so failures surface BEFORE you push, not after.
# Run manually (`bash scripts/check.sh`) or automatically via the pre-push hook
# (.githooks/pre-push). A phase is not DONE until this exits 0.
#
# Exit 0 = all gates green. Exit 1 = at least one gate failed (see summary).
# ponytail: collects ALL failures before exiting (no set -e) so one push shows everything.

set -uo pipefail
cd "$(git rev-parse --show-toplevel)" || exit 1

FAILED=()
step() { # step "Label" cmd...
  local label="$1"; shift
  printf '\n\033[1m▶ %s\033[0m\n' "$label"
  if "$@"; then printf '\033[32m✓ %s\033[0m\n' "$label"
  else printf '\033[31m✗ %s\033[0m\n' "$label"; FAILED+=("$label"); fi
}

# ── Backend (Go) ────────────────────────────────────────────────────────────
# golangci-lint is THE gate that catches errcheck/unused/etc. Resolve it from
# PATH or GOPATH/bin; if missing, fail loudly with the CI-matching install line.
GOLANGCI="$(command -v golangci-lint || true)"
[ -z "$GOLANGCI" ] && [ -x "$(go env GOPATH)/bin/golangci-lint" ] && GOLANGCI="$(go env GOPATH)/bin/golangci-lint"
if [ -z "$GOLANGCI" ]; then
  printf '\033[31m✗ golangci-lint not installed.\033[0m Install the CI version:\n'
  printf '    go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8\n'
  FAILED+=("golangci-lint (not installed)")
else
  # --max-same-issues=0 / --max-issues-per-linter=0 → show EVERY issue locally.
  # CI's default caps display at 3 per message, which hides duplicates (the trap
  # that let 7 of 10 errcheck errors slip past the first review).
  step "backend: golangci-lint" bash -c "cd backend && '$GOLANGCI' run --timeout=5m --max-same-issues=0 --max-issues-per-linter=0"
fi
step "backend: go build"  bash -c "cd backend && go build ./..."
step "backend: go vet"    bash -c "cd backend && go vet ./..."

# go test needs Postgres + migrations applied (see backend/Makefile / ci.yml).
# Run it only when DATABASE_URL is set so the hook isn't blocked on Docker being up;
# CI always runs it. To include tests locally: start Docker Postgres, `make -C backend migrate`,
# then `export DATABASE_URL=postgres://myiu:myiu@localhost:5432/myiu_dev?sslmode=disable`.
if [ -n "${DATABASE_URL:-}" ]; then
  step "backend: go test" bash -c "cd backend && go test ./..."
else
  printf '\n\033[33m⚠ backend: go test SKIPPED — DATABASE_URL not set (CI will run it on Postgres).\033[0m\n'
fi

# ── Frontend (React/Vite) ─────────────────────────────────────────────────────
if [ ! -d frontend/node_modules ]; then
  step "frontend: npm ci" bash -c "cd frontend && npm ci"
fi
step "frontend: lint (eslint)"      bash -c "cd frontend && npm run lint"
step "frontend: build (tsc+vite)"   bash -c "cd frontend && npm run build"

# ── Summary ───────────────────────────────────────────────────────────────────
printf '\n────────────────────────────────────────\n'
if [ ${#FAILED[@]} -eq 0 ]; then
  printf '\033[32m✓ All checks passed.\033[0m\n'
  exit 0
fi
printf '\033[31m✗ %d gate(s) failed:\033[0m\n' "${#FAILED[@]}"
for f in "${FAILED[@]}"; do printf '   - %s\n' "$f"; done
printf 'Fix these before pushing. (Emergency bypass: git push --no-verify)\n'
exit 1
