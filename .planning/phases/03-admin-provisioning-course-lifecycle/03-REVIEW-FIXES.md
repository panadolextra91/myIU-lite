# Phase 3 — Fix Handoff for Antigravity

**Source:** Code review of Phase 3 (Admin Provisioning & Course Lifecycle) — see `03-REVIEW.md` for full findings.
**Branch:** `ft/phase-3-admin` (do NOT branch off; continue here).
**Context to read first:** `.claude/CLAUDE.md` (locked stack + Feature-Oriented Monolith rules), `.planning/phases/03-admin-provisioning-course-lifecycle/03-CONTEXT.md` (locked decisions D-24→D-43). Honor the GSD workflow: atomic commits per fix, run `go build ./...` + `go test ./...` (and `cd frontend && npm run lint && npx tsc --noEmit && npm run build`) before committing.

**Ground rule — do NOT regress these verified-correct invariants** while fixing: all-or-nothing CSV import (D-27, single tx in `users.ImportAccounts` + `enrollments.ImportMembers`), idempotent membership via `ON CONFLICT DO NOTHING` (D-30/32), soft-delete-only/no-cascade (D-29/40), sweep query + 0-affected audit suppression (D-37/39), SYSTEM exclusion from list queries (D-38), bcrypt cost=12 + never store/return plaintext passwords (D-26), sqlc-parameterized SQL (no string-built SQL), read-only audit viewer.

---

## MUST FIX — Critical

### CR-00 — Remove the committed `sqlc` binary + zip from git (repo hygiene)
**What's wrong:** `backend/sqlc` (49 MB Mach-O arm64 executable) and `backend/sqlc.zip` (13 MB) are committed and tracked in git. There is no `.gitignore` rule. This bloats the repo permanently, is platform-specific (useless on CI/Linux/other devs), and ships an opaque 49 MB executable in version control. sqlc is a build-time CLI — it must be installed, never vendored as a binary.
**Fix:**
1. `git rm --cached backend/sqlc backend/sqlc.zip` (remove from tracking; keep your local copy if you want).
2. Add to `backend/.gitignore` (create if absent):
   ```
   # sqlc is a build-time CLI tool, never commit the binary or archive
   /sqlc
   /sqlc.zip
   ```
3. sqlc should be invoked via an installed CLI (`go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1` or `brew install sqlc`), per CLAUDE.md. Confirm `sqlc.yaml` still drives codegen.
4. Commit: `chore(03): remove vendored sqlc binary from version control`.
*(Note: this only stops tracking it going forward — the 62 MB still lives in history. Acceptable for now; a history rewrite is out of scope unless the team wants it.)*

### CR-01 / CR-04 / IN-01 — Make every admin mutation atomic with its audit write (D-35 / ADMIN-08)
**Files:** `backend/internal/courses/service.go:68,128-141,155`; `backend/internal/users/service.go:46,175-184`; `backend/internal/enrollments/service.go:134,149`
**What's wrong:** The single-row mutations (`CreateCourse`, `UpdateCourse`, `SoftDeleteCourse`, `CreateAccount`, `ResetPassword`, `RemoveStudent`, `UnassignLecturer`) run the data change on the connection pool (`s.q`), then write the audit row in a *separate* call — several discard the audit error with `_ =`. If the audit INSERT fails, the mutation is already committed with **no audit row**, violating "exactly one audit row per admin mutation, in the same transaction" (D-35) and the "audit log for all admin actions" compliance constraint. In `RemoveStudent`/`UnassignLecturer` the delete happens on the pool then the audit error is returned → caller sees 500 but the row is already gone (mutation/response mismatch).
**Fix:** Wrap each mutation + its audit write in ONE `pgx.Tx`, exactly like the CSV importers already do. Each affected service struct must hold the `*pgxpool.Pool` (the importers already do — add the field where missing). Pattern:
```go
tx, err := s.pool.Begin(ctx)
if err != nil { return err }
defer tx.Rollback(ctx)
qtx := s.q.WithTx(tx)
if err := qtx.SoftDeleteCourse(ctx, id); err != nil { return err }      // mutation
if err := auditlogs.WriteAudit(ctx, qtx, actorID, auditlogs.COURSE_DELETE,
        auditlogs.TargetTypeCourse, &id, nil, nil); err != nil {        // audit, SAME tx
    return err
}
return tx.Commit(ctx)
```
Apply to all 7 mutations. **Never `_ =` an audit write** — propagate the error so the tx rolls back. Match the real `WriteAudit` signature + action/target constants in `auditlogs/audit_helper.go`.
Additionally (CR-04): make sure `UpdateCourse`/`SoftDeleteCourse` map `pgx.ErrNoRows` → `ErrCourseNotFound` (404) rather than a 500 when the course vanished.

### CR-02 — Duplicate-row error message corrupts row numbers ≥ 10
**File:** `backend/internal/users/service.go:84`
**What's wrong:** `string(rune(prev+'0'))` only maps 0–9 correctly; `prev==10` renders `':'`, `prev==13` `'='`, etc. Any CSV whose duplicate ID was first seen at row ≥ 10 produces a garbled error.
**Fix:**
```go
rowErrs = append(rowErrs, RowError{
    Row: p.RowIndex, Field: "id",
    Message: fmt.Sprintf("duplicate ID in file (matches row %d)", prev),
})
```
(The enrollments importer already uses `fmt.Sprintf` correctly — mirror it.)

### CR-03 — All `CreateUser` DB errors masked as "user already exists"
**File:** `backend/internal/users/service.go:40-43`
**What's wrong:** `CreateAccount` maps *any* `repo.CreateUser` error to `ErrDuplicateUser` → HTTP 409. Transient DB errors, unrelated constraint violations, context cancellation, connection failures all get reported as a duplicate-username conflict, hiding real failures.
**Fix:** Only treat the unique-violation pgx code as duplicate; propagate everything else:
```go
var pgErr *pgconn.PgError
if errors.As(err, &pgErr) && pgErr.Code == "23505" {
    return 0, ErrDuplicateUser
}
return 0, err
```
(import `github.com/jackc/pgx/v5/pgconn`)

---

## SHOULD FIX — Warnings

### WR-01 — `ResetPassword` silently "succeeds" against the SYSTEM account
**Files:** `backend/internal/users/service.go:159-184`; `backend/db/queries/users.sql:18-19`
**What's wrong:** `GetUserByID` doesn't exclude `is_system`, so admin can target `__system__`. The reset SQL guards `AND is_system = FALSE` so it updates 0 rows, but the `:exec` query returns no error → handler returns `200 success` + writes a spurious PASSWORD_RESET audit row, while nothing changed.
**Fix:** After fetching the target, `if user.IsSystem { return ErrUserNotFound }` (or add `is_system = FALSE` to the fetch). Optionally change `ResetUserPassword` to `:execrows` and return `ErrUserNotFound` on 0 rows.

### WR-02 — SYSTEM no-login relies on the sentinel hash, not an explicit guard
**File:** `backend/internal/auth/service.go:23-34`
**What's wrong:** `__system__` can't log in only because `password_hash='!'` is an invalid bcrypt hash. No explicit `is_system` check in `Login`. If a real hash ever lands on that row, it becomes a loginable admin.
**Fix (defense-in-depth):** after fetching the user in `Login`, `if user.IsSystem { return db.User{}, ErrInvalidCredentials }`.

### WR-03 / WR-04 — List endpoints accept an unbounded `limit`
**Files:** `backend/internal/courses/handler.go:75-79`; `backend/internal/users/handler.go:125-129`; `backend/internal/auditlogs/handler.go:37-41`
**What's wrong:** `limit` is "any positive int" with no ceiling (the FE already requests `limit:1000`). A caller can pass a huge limit → oversized payload / DoS.
**Fix:** Clamp to a max (e.g. `if limit > 200 { limit = 200 }`) in each handler — ideally a shared helper. Also for audit (WR-03): the `limit` cap covers the same endpoint.

### WR-05 — CSV upload type not validated server-side
**Files:** `backend/internal/users/handler.go:75-82`; `backend/internal/enrollments/handler.go:50-57`
**What's wrong:** The 5 MB `http.MaxBytesReader` cap is correct, but the file's content-type/extension is never checked server-side (`accept=".csv"` on the input is UX-only). CLAUDE.md requires validating file type server-side, not trusting the client.
**Fix:** Check the multipart header `Content-Type` (`text/csv`, `application/vnd.ms-excel`, `application/octet-stream`) and/or the `.csv` filename extension before parsing; reject with an `INVALID_FILE` code otherwise.

### WR-06 — Sweeper goroutine has no panic recovery
**File:** `backend/internal/lifecycle/sweep.go:19-33,54`
**What's wrong:** `runSweep` runs from a bare goroutine with no `recover()`; a panic in the daily tick (e.g. nil in `WriteAudit`/serialization) crashes the whole API server. Minor: `db.New(pool)` is rebuilt every run and its `pool` arg is effectively discarded once `.WithTx(tx)` is used.
**Fix:** Wrap the ticker body:
```go
func() {
    defer func() { if r := recover(); r != nil { log.Printf("sweep panic: %v", r) } }()
    runSweep(ctx, ...)
}()
```
Also: stop cleanly on `ctx.Done()` (confirm the ticker loop selects on the context). Reuse one `q := db.New(pool)` and bind per-tx with `q.WithTx(tx)`.

### WR-07 — Course date errors are indistinguishable
**File:** `backend/internal/courses/service.go:24-32,43-55`
**What's wrong:** `parseDate` returns the same `ErrInvalidDates` for a malformed date string AND for `end < start`, so the admin can't tell "bad format" from "end before start". A value like `2026-02-30` fails parsing but is reported as a range error.
**Fix:** Return a distinct `ErrInvalidDateFormat` from `parseDate`; keep `ErrInvalidDates` for `endDate.Before(startDate)`; map each to its own message in the handler.

---

## OPTIONAL — Info (fix if cheap; not blocking)

- **IN-02 — CSV formula injection:** `full_name`/IDs stored verbatim; values like `=cmd|…` / `@SUM(…)` survive and round-trip to the UI + audit metadata. React escapes HTML (no XSS), but a future CSV/Excel export would be vulnerable. Optionally strip/prefix leading `= + - @ \t \r` on import.
- **IN-03 — Cleanup:** remove resolved TODO comments (`// simple int to string for MVP` etc. in `users/service.go`, `enrollments/csv.go`); regenerate sqlc with the pinned **v1.31.1** so the generated banner stops saying `v1.27.0` (it currently mismatches CLAUDE.md).
- **IN-04 — Envelope consistency:** `courses/handler.go:207,236` `ListStudents`/`ListLecturers` return a bare JSON array while every other list returns `{data, total}`. Wrap rosters in `{data:[...]}` for consistency.
- **IN-05 — Flaky test:** `users/reset_test.go:32-36` is state-dependent (empty `if err {}` branch then `require.NoError`). Use a unique username per run + deferred cleanup.

---

## Suggested commit sequence (atomic)
1. `chore(03): remove vendored sqlc binary from version control` (CR-00)
2. `fix(03): make admin mutations atomic with audit writes` (CR-01/04/IN-01)
3. `fix(03): correct duplicate-row error formatting and CreateUser error mapping` (CR-02, CR-03)
4. `fix(03): harden SYSTEM account (reset no-op + explicit login guard)` (WR-01, WR-02)
5. `fix(03): cap list limits and validate CSV upload type` (WR-03/04, WR-05)
6. `fix(03): recover sweeper panics and distinguish course date errors` (WR-06, WR-07)
7. (optional) `chore(03): info-level cleanups` (IN-02..05)

After fixes: run full `go test ./...` + frontend build/lint, push, and ping Claude to re-review (`/gsd-code-review 3`).
