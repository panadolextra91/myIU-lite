---
phase: 03-admin-provisioning-course-lifecycle
reviewed: 2026-06-20T00:00:00Z
depth: standard
files_reviewed: 47
files_reviewed_list:
  - backend/db/migrations/000004_admin_schema.up.sql
  - backend/db/migrations/000004_admin_schema.down.sql
  - backend/db/migrations/000005_audit_append_only_and_system.up.sql
  - backend/db/migrations/000005_audit_append_only_and_system.down.sql
  - backend/db/queries/auditlogs.sql
  - backend/db/queries/courses.sql
  - backend/db/queries/enrollments.sql
  - backend/db/queries/users.sql
  - backend/internal/auditlogs/audit_helper.go
  - backend/internal/auditlogs/dto.go
  - backend/internal/auditlogs/handler.go
  - backend/internal/auditlogs/repository.go
  - backend/internal/auditlogs/service.go
  - backend/internal/auditlogs/append_only_test.go
  - backend/internal/auth/system_test.go
  - backend/internal/courses/dto.go
  - backend/internal/courses/handler.go
  - backend/internal/courses/model.go
  - backend/internal/courses/repository.go
  - backend/internal/courses/service.go
  - backend/internal/enrollments/csv.go
  - backend/internal/enrollments/dto.go
  - backend/internal/enrollments/handler.go
  - backend/internal/enrollments/model.go
  - backend/internal/enrollments/repository.go
  - backend/internal/enrollments/service.go
  - backend/internal/enrollments/import_test.go
  - backend/internal/lifecycle/sweep.go
  - backend/internal/lifecycle/sweep_test.go
  - backend/internal/users/csv.go
  - backend/internal/users/dto.go
  - backend/internal/users/handler.go
  - backend/internal/users/model.go
  - backend/internal/users/repository.go
  - backend/internal/users/service.go
  - backend/internal/users/derive_test.go
  - backend/internal/users/import_test.go
  - backend/internal/users/reset_test.go
  - backend/cmd/api/main.go
  - frontend/src/components/AppLayout.tsx
  - frontend/src/components/admin/AdminSidebar.tsx
  - frontend/src/lib/admin-api.ts
  - frontend/src/pages/admin/Accounts.tsx
  - frontend/src/pages/admin/AuditLogs.tsx
  - frontend/src/pages/admin/CourseDetail.tsx
  - frontend/src/pages/admin/Courses.tsx
  - frontend/src/pages/admin/Enrollment.tsx
  - frontend/src/pages/admin/LecturerAssignment.tsx
  - frontend/src/routes/router.tsx
findings:
  critical: 4
  warning: 7
  info: 5
  total: 16
status: issues_found
---

# Phase 3: Code Review Report

**Reviewed:** 2026-06-20
**Depth:** standard
**Files Reviewed:** 47
**Status:** issues_found

## Summary

Phase 3 implements admin provisioning (account/CSV import, course CRUD, enrollment/lecturer assignment, password reset), the append-only audit log, the SYSTEM account, and the soft-delete sweep. The all-or-nothing CSV import transaction pattern (D-27) is implemented correctly in both `users.ImportAccounts` and `enrollments.ImportMembers` (validate-all-then-tx, deferred rollback, commit at end). Idempotent membership (D-30/D-32) via `ON CONFLICT DO NOTHING` is correct. The sweep query, idempotency, startup catch-up, and 0-affected audit suppression (D-37/D-39) are correct. Append-only triggers (D-35) are present and tested.

However, the phase **systematically violates the "audit row in the same transaction as the mutation" invariant** for every non-CSV admin mutation: `CreateAccount`, `CreateCourse`, `UpdateCourse`, `SoftDeleteCourse`, `RemoveStudent`, `UnassignLecturer`, and `ResetPassword` all perform the mutation on the pool, then write the audit row in a *separate* call (several with the error discarded via `_ =`). A failed or partially-failed audit write leaves a committed mutation with no audit trail — a compliance/data-integrity defect given the "audit log for all admin actions" constraint. Plus a string-formatting bug that corrupts duplicate-row error messages, and an error-masking bug that reports all `CreateUser` DB failures as "user already exists".

## Critical Issues

### CR-01: Admin mutations are not atomic with their audit write (D-35 violation)

**File:** `backend/internal/courses/service.go:68,140,155`; `backend/internal/users/service.go:46,183`; `backend/internal/enrollments/service.go:134,149`
**Issue:** The single-row admin mutations execute the data change and the audit write as two independent operations against the connection pool (`s.q`), not inside one transaction. In courses (`CreateCourse`, `UpdateCourse`, `SoftDeleteCourse`) and users (`CreateAccount`, `ResetPassword`) the audit error is additionally discarded with `_ =`. If the audit `INSERT` fails (DB hiccup, constraint, append-only trigger edge case) the mutation is already committed and there is **no audit row** — directly breaking the "exactly one audit row per admin mutation, written in the same transaction" requirement and the project's "audit log for all admin actions" compliance constraint. In `RemoveStudent`/`UnassignLecturer` the error is returned, but the row was already deleted on the pool, so the caller sees a 500 while the deletion silently succeeded (mutation/response inconsistency).
**Fix:** Wrap each mutation + its audit write in one `pgx.Tx`, mirroring the CSV importers. Example for `SoftDeleteCourse`:
```go
tx, err := s.pool.Begin(ctx)
if err != nil { return err }
defer tx.Rollback(ctx)
qtx := s.q.WithTx(tx)
if err := qtx.SoftDeleteCourse(ctx, id); err != nil { return err }
if err := auditlogs.WriteAudit(ctx, qtx, actorID, auditlogs.COURSE_DELETE, auditlogs.TargetTypeCourse, &id, nil, nil); err != nil {
    return err
}
return tx.Commit(ctx)
```
Never use `_ =` on an audit write — propagate the error so the transaction rolls back.

### CR-02: Duplicate-row error message corrupts row numbers >= 10

**File:** `backend/internal/users/service.go:84`
**Issue:** `string(rune(prev+'0'))` is used to render the previously-seen row number into the duplicate-ID error message. `rune(prev+'0')` only maps 0–9 to '0'–'9'; for `prev == 10` it yields `':'`, for `prev == 13` `'='`, etc., and for `prev >= 75` it produces non-ASCII garbage. Any CSV with a duplicate ID first seen at row >= 10 returns a misleading/garbled error to the admin. (The enrollments importer does this correctly with `fmt.Sprintf`.)
**Fix:**
```go
rowErrs = append(rowErrs, RowError{
    Row: p.RowIndex, Field: "id",
    Message: fmt.Sprintf("duplicate ID in file (matches row %d)", prev),
})
```

### CR-03: All CreateUser DB errors are masked as "user already exists"

**File:** `backend/internal/users/service.go:40-43`
**Issue:** `CreateAccount` maps *any* error from `repo.CreateUser` to `ErrDuplicateUser`, which the handler turns into HTTP 409 "User already exists". A transient DB error, a constraint violation unrelated to uniqueness, a context cancellation, or a connection failure will all be reported to the admin as a duplicate-username conflict, hiding real failures and producing incorrect client behavior (the admin will assume the username is taken when it is not).
**Fix:** Distinguish the unique-violation case from other errors using the pgx error code, e.g.:
```go
var pgErr *pgconn.PgError
if errors.As(err, &pgErr) && pgErr.Code == "23505" {
    return 0, ErrDuplicateUser
}
return 0, err
```

### CR-04: Course update/create/delete audited even when audit may silently no-op, and UpdateCourse audits a possibly-missing row

**File:** `backend/internal/courses/service.go:128-141`
**Issue:** `UpdateCourse` calls `repo.UpdateCourse` whose SQL has `WHERE id=$1 AND deleted_at IS NULL ... RETURNING *`. If the course was concurrently soft-deleted, the `:one` query returns `pgx.ErrNoRows` and the function returns an error before auditing — that path is fine. But combined with CR-01, the larger problem is that the audit write happens unconditionally after the mutation with its error dropped. Because the audit is outside any transaction, a successful course update can be committed with the COURSE_UPDATE audit row missing, leaving the audit trail inconsistent with course state. This is the same root cause as CR-01 but is called out separately because course mutations are the highest-frequency admin action and the most visible audit gap.
**Fix:** Resolve via CR-01 (single transaction). Additionally confirm `UpdateCourse`/`SoftDeleteCourse` surface `pgx.ErrNoRows` as `ErrCourseNotFound` so a vanished course returns 404 rather than 500.

## Warnings

### WR-01: ResetPassword silently succeeds against the SYSTEM account

**File:** `backend/internal/users/service.go:159-184`; `backend/db/queries/users.sql:18-19`
**Issue:** `GetUserByID` does not exclude `is_system`, so an admin can target the `__system__` user's id. The `ResetUserPassword` SQL guards with `AND is_system = FALSE`, so the update affects 0 rows — but because the query is `:exec` it returns no error. Result: the handler returns `200 {"status":"success"}`, an audit row is written, yet nothing was reset. The admin is misled and a spurious PASSWORD_RESET audit entry is created for the system account.
**Fix:** Reject system/non-existent targets explicitly. Either add `is_system = FALSE` to `GetUserByID` for this path, or check `user.IsSystem` after fetch and return `ErrUserNotFound`. Optionally change the reset query to `:execrows` and return `ErrUserNotFound` when 0 rows affected.

### WR-02: SYSTEM no-login depends on an invalid bcrypt hash, not an explicit guard

**File:** `backend/internal/auth/service.go:23-34`; `backend/db/migrations/000005...up.sql:13-14`
**Issue:** The SYSTEM account is prevented from logging in only because its `password_hash` is the sentinel `'!'`, which makes `bcrypt.CompareHashAndPassword` fail. There is no explicit `is_system = FALSE` check in `Login`. If anyone ever writes a real hash to that row (e.g. a future "reset all passwords" job that doesn't exclude system, or a manual fix), the account becomes loginable as an admin. The invariant is currently upheld only by an implementation accident.
**Fix:** Add a defense-in-depth check in `Login`: after fetching the user, `if user.IsSystem { return db.User{}, ErrInvalidCredentials }`. Cheap and removes the reliance on the sentinel.

### WR-03: Audit log list endpoint leaks raw metadata and is unbounded by max page size

**File:** `backend/internal/auditlogs/handler.go:33-46`; `backend/internal/auditlogs/dto.go:20`
**Issue:** (a) `limit` accepts any positive int with no upper bound, so a client can request `limit=100000000` and force a very large response/scan. (b) `Metadata []byte` is serialized directly into JSON; since it stores arbitrary admin-supplied content (e.g. usernames from CSV), it is echoed back verbatim — combined with CSV-injection (see IN-02) this can round-trip attacker-controlled strings to the admin UI.
**Fix:** Cap `limit` (e.g. `if limit > 200 { limit = 200 }`), matching the other list endpoints which share the same unbounded-limit issue (WR-04).

### WR-04: All list endpoints accept an unbounded limit

**File:** `backend/internal/courses/handler.go:75-79`; `backend/internal/users/handler.go:125-129`; `backend/internal/auditlogs/handler.go:37-41`
**Issue:** `limit` is parsed as "any positive int" with no ceiling across courses, users, and audit logs. A caller can pass an arbitrarily large limit. The frontend's Enrollment/LecturerAssignment pages already request `limit: 1000`. Without a server cap this is a denial-of-resource vector and produces oversized payloads.
**Fix:** Clamp to a max (e.g. 200) in each handler, ideally via a shared helper.

### WR-05: CSV import MIME/type not validated; only client-side `accept=".csv"` enforced

**File:** `backend/internal/users/handler.go:75-82`; `backend/internal/enrollments/handler.go:50-57`
**Issue:** The size cap (`http.MaxBytesReader`, 5MB) is correctly applied, but the uploaded file's content-type/extension is never checked server-side. `accept=".csv"` on the `<input>` is UX-only and trivially bypassed. The parser will attempt to read any bytes as CSV. While not a code-execution risk (encoding/csv is safe), it allows uploading binary blobs that consume the full 5MB and CPU during parsing. The project conventions explicitly require validating file type server-side, not trusting the client.
**Fix:** Read the multipart file header's `Content-Type` and/or the filename extension and reject non-CSV uploads before parsing, returning `INVALID_FILE`.

### WR-06: Sweeper goroutine builds a fresh `db.New(pool)` per run and a panic in the audit write would crash the process

**File:** `backend/internal/lifecycle/sweep.go:19-33,54`
**Issue:** (a) `runSweep` is invoked from a bare goroutine with no `recover()`; an unexpected panic (e.g. nil pointer in `WriteAudit`/serialization) in the daily tick would crash the whole API server rather than logging and continuing. (b) `db.New(pool)` is allocated inside `runSweep` on every run; minor, but `.WithTx(tx)` is what binds the queries to the tx so the `pool` argument to `db.New` is effectively discarded — slightly confusing and wasteful.
**Fix:** Wrap the ticker body in a `func(){ defer func(){ if r:=recover(); r!=nil { log.Printf("sweep panic: %v", r) } }(); ... }()`. Construct the queries from the tx directly (e.g. a package-level `q := db.New(pool)` reused, then `q.WithTx(tx)`).

### WR-07: Course date validation rejects equal-day single-day courses inconsistently and `parseDate` swallows real errors

**File:** `backend/internal/courses/service.go:24-32,43-55`
**Issue:** `parseDate` returns the generic `ErrInvalidDates` for *both* a bad format and a logically-invalid range, so the handler cannot distinguish "your date string is malformed" from "end before start" — both surface as the same message. Additionally, a course `end_date` equal to `start_date` is allowed (`Before` check), which is fine, but `time.Parse` of a date like `2026-02-30` will fail format parsing and be reported as a range error, confusing the admin.
**Fix:** Return distinct errors: a dedicated `ErrInvalidDateFormat` from `parseDate`, and keep `ErrInvalidDates` for the `endDate.Before(startDate)` case, mapping each to a specific message.

## Info

### IN-01: `ResetUserPassword` and reset audit are not transactional (consistency)

**File:** `backend/internal/users/service.go:175-184`
**Issue:** Same root cause as CR-01 but lower frequency: the password update and PASSWORD_RESET audit are separate pool calls with the audit error dropped. Folded into CR-01's fix.

### IN-02: CSV formula/injection not neutralized on import or display

**File:** `backend/internal/users/csv.go:81-89`; `backend/internal/enrollments/csv.go:69`
**Issue:** `full_name`/IDs are stored verbatim. A value like `=cmd|...` or `@SUM(...)` is preserved and later rendered in the admin UI and echoed in audit metadata. React escapes HTML so XSS is not the risk, but if any admin ever exports these back to CSV/Excel, formula injection is possible. Low severity for this app but worth a note.
**Fix:** Optionally strip/prefix leading `= + - @ \t \r` on import, or sanitize on any future CSV export.

### IN-03: Unused `RowIndex` / dead-code comments and stale sqlc version banner

**File:** `backend/internal/enrollments/csv.go:10-13`; `backend/internal/users/service.go:42,84`; `backend/internal/shared/db/auditlogs.sql.go:3`
**Issue:** Comments like `// mapping simplified for now`, `// simple int to string for MVP`, and `// handle existing or just use something random` (reset_test.go:34) flag known shortcuts. The generated banner says sqlc `v1.27.0` while CLAUDE.md pins `v1.31.1` — generated code may be stale relative to the pinned toolchain.
**Fix:** Remove resolved TODO comments; regenerate with the pinned sqlc version to keep the banner accurate.

### IN-04: Audit list and roster endpoints return bare arrays / inconsistent envelopes

**File:** `backend/internal/courses/handler.go:207,236`
**Issue:** `ListStudents`/`ListLecturers` return a bare JSON array, while every other list endpoint returns `{data, total}`. Error responses use the `{error:{code,message}}` envelope consistently, but success shapes diverge. Minor API-consistency nit.
**Fix:** Consider wrapping rosters in a `{data: [...]}` object for consistency, or document the intentional difference.

### IN-05: `reset_test.go` swallows the duplicate-user branch, making the test non-deterministic

**File:** `backend/internal/users/reset_test.go:32-36`
**Issue:** If `reset_test_user` already exists from a prior run, the `if err ... {}` branch is empty and the subsequent `require.NoError(t, err)` fails. The test is order/state dependent.
**Fix:** Use a unique username per run (timestamp/uuid suffix) and clean up with a deferred delete.

---

## Verified-correct invariants (no action needed)

- **D-27 all-or-nothing CSV import:** `users.ImportAccounts` and `enrollments.ImportMembers` validate the entire file first, then insert inside one `pgx.Tx` with `defer tx.Rollback` and a final `Commit`. No early return leaks a started tx. Confirmed by `import_test.go`.
- **D-30/D-32 idempotent membership:** `EnrollStudent`/`AssignLecturer` use `ON CONFLICT DO NOTHING`; re-import returns the newly-added count without error. Confirmed by `import_test.go`.
- **D-29/D-40 soft-delete only:** `SoftDeleteCourse` sets `deleted_at` only; no cascade. All course reads filter `deleted_at IS NULL`.
- **D-37/D-39 sweep:** query is `deleted_at IS NULL AND end_date < now() - interval '1 month'`, idempotent, startup catch-up + 24h ticker, SYSTEM actor, audit row only when `n > 0`. Confirmed by `sweep_test.go`.
- **D-38 SYSTEM excluded from lists:** `ListUsers`/`CountUsers`/`GetUserIDsByRole`/`GetActiveUsernames` all filter `is_system = FALSE`.
- **D-26 default password:** never stored/returned in plaintext; bcrypt cost=12; DOB parsed DD/MM/YYYY, password formatted DDMMYYYY. Confirmed by `derive_test.go`.
- **SQL injection:** all queries are sqlc-parameterized; no string-built SQL. ILIKE search uses bound params.
- **Frontend audit viewer:** strictly read-only (no edit/delete controls); TanStack Query invalidation present after mutations; no passwords/secrets displayed.
- **Authorization:** all `/admin/*` routes sit behind `AuthMiddleware + RequireRole(admin)`; actor id is taken from the verified JWT (`c.GetInt64("user_id")`), not the request body.

---

_Reviewed: 2026-06-20_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
