# Phase 3 — CI Lint Fix Handoff for Antigravity

**CI gate failing:** `golangci-lint` (errcheck) on `backend/`. Branch `ft/phase-3-admin`.

**Nature:** Pure lint — unchecked error return values. No logic bugs. These are intentional ignores (rollback-after-commit is a safe no-op; test cleanup errors are intentionally discarded), so the fix is to make the "ignore" explicit so errcheck is satisfied.

## ⚠️ Important — CI only showed 3 of the 10 `tx.Rollback` errors
golangci-lint's default `max-same-issues: 3` caps how many instances of the *same* message are printed. The log listed only the 3 in the `enrollments` package, but there are **10 `defer tx.Rollback(ctx)` sites total**. **Fix ALL of them**, or the next CI run will fail again on the remaining 7. (Likewise `errcheck` will keep flagging until every site is clean.)

---

## Fix A — `defer tx.Rollback(ctx)` (10 sites)
Rollback after a successful Commit returns `pgx.ErrTxClosed`, which is expected and must be ignored. Make it explicit by wrapping in a func with `_ =`:

**Before:**
```go
defer tx.Rollback(ctx)
```
**After:**
```go
defer func() { _ = tx.Rollback(ctx) }()
```

Apply at ALL of these:
- `backend/internal/enrollments/service.go:74, 127, 154`
- `backend/internal/courses/service.go:63, 149, 184`
- `backend/internal/users/service.go:38, 140, 202`
- `backend/internal/lifecycle/sweep.go:50`

> Do NOT change the audit-write calls or the real `tx.Commit(ctx)` error handling — those must still propagate. Only the deferred `Rollback` becomes an explicit ignore.

## Fix B — `defer pool.Exec(...)` cleanup in tests (3 sites)
Test teardown deletes; the error is intentionally ignored.

**Before:**
```go
defer pool.Exec(ctx, `DELETE FROM courses WHERE id = $1`, courseID)
```
**After:**
```go
defer func() { _, _ = pool.Exec(ctx, `DELETE FROM courses WHERE id = $1`, courseID) }()
```
Apply at `backend/internal/enrollments/import_test.go:34, 45, 46` (each with its own SQL/args — keep them identical, just wrap).

## Fix C — unchecked `.Scan(...)` in a test (1 site)
`backend/internal/enrollments/import_test.go:61` — this is a test assertion, so check the error properly rather than ignoring it:

**Before:**
```go
pool.QueryRow(ctx, `SELECT COUNT(*) FROM student_enrollments WHERE course_id = $1`, courseID).Scan(&count)
```
**After:**
```go
require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM student_enrollments WHERE course_id = $1`, courseID).Scan(&count))
```
(`require` is already imported in this test file. If not, use `if err := ...Scan(&count); err != nil { t.Fatal(err) }`.)

---

## Verify before committing
```bash
cd backend
# Mirror the CI invocation:
golangci-lint run --path-prefix=backend --timeout=5m   # exit 0, no issues
go build ./... && go vet ./...                          # exit 0
```
If `golangci-lint` isn't installed locally: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8` (match the CI version) — and do NOT commit the binary (it's a CLI tool, like sqlc).

**Commit:** `fix(03): satisfy errcheck — explicitly ignore deferred Rollback/cleanup, check test Scan`

After push, CI should go green. Then ping Claude for a final confirm.

## Suggestion (optional, prevents recurrence)
Add a minimal `backend/.golangci.yml` so lint behavior is explicit and reproducible locally, e.g.:
```yaml
run:
  timeout: 5m
linters:
  enable: [errcheck, govet, ineffassign, staticcheck, unused]
issues:
  max-same-issues: 0   # show ALL instances, not just 3 — avoids the "hidden errors" trap above
```
Not required to pass CI, but it would have surfaced all 10 errors at once.
