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

# Local dev convenience (gitignored, absent in CI): load scripts/check.env so the
# pre-push gate picks up DATABASE_URL without re-exporting in every shell. The file
# only sets DATABASE_URL if you haven't already set one, so an explicit shell export
# still wins. See scripts/check.env.example.
[ -f scripts/check.env ] && . scripts/check.env

# CI env parity — keep env-dependent tests behaving identically local vs CI so the
# pre-push gate predicts CI. These are the SAME fixed test placeholders committed in
# .github/workflows/ci.yml `env:` (public test values, not secrets). `:-` so a real
# value already in your shell / check.env still wins. DATABASE_URL stays per-dev in
# check.env. KEEP IN SYNC with ci.yml's env block.
export JWT_SECRET="${JWT_SECRET:-test-secret}"
export CLOUDINARY_URL="${CLOUDINARY_URL:-cloudinary://test:test@test}"

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
# HONEST GATE: tests are never SILENTLY skipped. Either they run (DATABASE_URL set),
# or you consciously opt out (SKIP_DB_TESTS=1). A bare run with neither FAILS, so the
# summary can never print a green "all passed" while the test gate didn't run — that
# false-green is what let untested code feel "done". CI always sets DATABASE_URL.
# To run locally: start Docker Postgres, `make -C backend migrate`, then
#   export DATABASE_URL=postgres://myiu:myiu@localhost:5432/myiu_dev?sslmode=disable
DB_TESTS_SKIPPED=""
if [ -n "${DATABASE_URL:-}" ]; then
  # DB schema sync — CI applies migrations before testing; do the same so a new migration
  # that isn't applied to your local DB can't let the gate pass while CI (current schema)
  # fails. `migrate up` is idempotent (exit 0 + "no change" when already current) and
  # NON-destructive — it never drops data, so it's safe on any DATABASE_URL. (A full
  # empty-DB reset isn't needed: the suite is residue-tolerant — verified green on both a
  # fresh and a populated DB — and `migrate drop` can't remove ENUM types anyway.)
  MIGRATE="$(command -v migrate || true)"
  [ -z "$MIGRATE" ] && [ -x "$(go env GOPATH)/bin/migrate" ] && MIGRATE="$(go env GOPATH)/bin/migrate"
  if [ -z "$MIGRATE" ]; then
    printf '\n\033[33m⚠ migrate CLI not found — skipping schema sync; local schema may drift from CI.\033[0m\n'
    printf '   Install: go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@latest\n'
  else
    step "backend: migrate up" bash -c "'$MIGRATE' -path backend/db/migrations -database '$DATABASE_URL' up"
  fi
  # -count=1 disables Go's test cache so the gate ALWAYS re-runs against the live DB; a
  # cached 'ok' can mask a real DB-state failure, and CI never caches.
  # -p 1 serializes package binaries: the suite shares ONE Postgres and the assignments
  # same-tx rollback test briefly RENAMEs the notifications table globally — parallel
  # packages race that window. KEEP IN SYNC with ci.yml.
  step "backend: go test" bash -c "cd backend && go test -count=1 -p 1 ./..."
elif [ "${SKIP_DB_TESTS:-}" = "1" ]; then
  DB_TESTS_SKIPPED=1
  printf '\n\033[33m⚠ backend: go test SKIPPED via SKIP_DB_TESTS=1 — CI will run it on Postgres.\033[0m\n'
else
  printf '\n\033[31m✗ backend: go test NOT RUN — DATABASE_URL not set.\033[0m\n'
  printf '   Run tests: start Docker Postgres + `make -C backend migrate`, then\n'
  printf '              export DATABASE_URL=postgres://myiu:myiu@localhost:5432/myiu_dev?sslmode=disable\n'
  printf '   Or bypass: SKIP_DB_TESTS=1 (intentional skip; CI still runs them)\n'
  FAILED+=("backend: go test (set DATABASE_URL, or SKIP_DB_TESTS=1 to opt out)")
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
  if [ -n "$DB_TESTS_SKIPPED" ]; then
    printf '\033[32m✓ All checks passed\033[0m \033[33m(DB tests intentionally skipped — not a full gate; CI will run them).\033[0m\n'
  else
    printf '\033[32m✓ All checks passed.\033[0m\n'
  fi
  exit 0
fi
printf '\033[31m✗ %d gate(s) failed:\033[0m\n' "${#FAILED[@]}"
for f in "${FAILED[@]}"; do printf '   - %s\n' "$f"; done
printf 'Fix these before pushing. (Emergency bypass: git push --no-verify)\n'
exit 1
