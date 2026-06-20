---
phase: 03-admin-provisioning-course-lifecycle
reviewed: 2026-06-20T00:00:00Z
depth: standard
round: 2
files_reviewed: 16
files_reviewed_list:
  - backend/internal/courses/service.go
  - backend/internal/courses/handler.go
  - backend/internal/courses/model.go
  - backend/internal/courses/repository.go
  - backend/internal/users/service.go
  - backend/internal/users/handler.go
  - backend/internal/users/csv.go
  - backend/internal/users/repository.go
  - backend/internal/users/reset_test.go
  - backend/internal/enrollments/service.go
  - backend/internal/enrollments/handler.go
  - backend/internal/enrollments/csv.go
  - backend/internal/enrollments/repository.go
  - backend/internal/auditlogs/handler.go
  - backend/internal/auditlogs/audit_helper.go
  - backend/internal/auth/service.go
  - backend/internal/lifecycle/sweep.go
  - backend/internal/lifecycle/sweep_test.go
  - backend/.gitignore
  - frontend/src/lib/admin-api.ts
findings:
  critical: 1
  warning: 0
  info: 0
  total: 1
status: issues_found
---

# Phase 3: Code Review Report — Round 2 (Re-review of fix commits)

**Reviewed:** 2026-06-20
**Depth:** standard
**Files Reviewed:** 16 (changed in fix commits + their callers)
**Status:** issues_found (1 regression)

## Summary

The fix commits resolve nearly every prior finding correctly. The high-risk transaction refactor was done well across all seven single-row admin mutations: each follows the canonical `tx := pool.Begin → defer tx.Rollback(ctx) → qtx := s.q.WithTx(tx) → mutation on qtx → WriteAudit on the SAME qtx → tx.Commit(ctx)` pattern. No audit error is discarded with `_ =` anymore, no audit write points at the pool, no tx leak on early return, and no double-commit. `UpdateCourse`/`SoftDeleteCourse` correctly surface `pgx.ErrNoRows` → `ErrCourseNotFound` (404).

**One genuine regression was introduced by the IN-04 fix**: the frontend was changed to read the roster as `{data:[...]}` (`admin-api.ts:76-81` → `res.data.data`), but the backend roster handlers (`courses/handler.go:210,239`) still return a **bare array**. The FE/BE contract now drifts in the opposite direction from round 1 — the roster lists will break at runtime. This is exactly the regression the round-2 brief flagged as a risk.

Everything else is RESOLVED.

## Critical Issues

### CR-R2-01: Roster FE/BE contract drift — frontend reads `{data}`, backend returns a bare array (IN-04 regression)

**File:** `backend/internal/courses/handler.go:210` and `:239`; `frontend/src/lib/admin-api.ts:75-82`
**Issue:** The IN-04 fix was applied only to the frontend. `admin-api.ts` now does:
```ts
listCourseStudents: async (id) => {
  const res = await api.get<{ data: RosterUser[] }>(`/admin/courses/${id}/students`);
  return res.data.data;   // expects { data: [...] }
},
listCourseLecturers: async (id) => {
  const res = await api.get<{ data: RosterUser[] }>(`/admin/courses/${id}/lecturers`);
  return res.data.data;   // expects { data: [...] }
},
```
But the backend handlers still emit a **bare array**:
```go
// ListStudents (handler.go:210)
c.JSON(http.StatusOK, res)   // res is []RosterUser, NOT { data: res }
// ListLecturers (handler.go:239)
c.JSON(http.StatusOK, res)
```
At runtime axios parses the body as a JS array, so `res.data` is the array and `res.data.data` is `undefined`. Both roster calls return `undefined`, breaking the CourseDetail roster display (likely an empty list or a `.map` of undefined crash downstream). This is a functional break of the admin course-roster view.
**Fix:** Make backend match the contract the FE now expects. In `courses/handler.go`:
```go
// ListStudents
c.JSON(http.StatusOK, gin.H{"data": res})
// ListLecturers
c.JSON(http.StatusOK, gin.H{"data": res})
```
(Alternative: revert the FE to `res.data`. Backend-side `{data: res}` is preferred — it matches every other list endpoint's `{data, total}` envelope and was the intent of IN-04.)

---

## Per-Finding Verdict Table

| ID | Finding | Verdict | Evidence |
|----|---------|---------|----------|
| **CR-00** | `sqlc` + `sqlc.zip` removed from tracking; `.gitignore` has `/sqlc` + `/sqlc.zip` | ✅ RESOLVED | `backend/.gitignore:1-2` = `/sqlc`, `/sqlc.zip`. `git ls-files` shows neither binary nor zip tracked (only `backend/sqlc.yaml` config remains, correct). |
| **CR-01 / CR-04 / IN-01** | All single-row mutations wrap mutation + WriteAudit in one tx; no `_ =`; audit on same qtx; no leak; pool present; `ErrNoRows`→404 | ✅ RESOLVED | `CreateCourse` courses/service.go:59-86; `UpdateCourse` :145-176 (ErrNoRows→ErrCourseNotFound :162-164); `SoftDeleteCourse` :179-203 (GetCourseByID→404 on qtx :188-191); `CreateAccount` users/service.go:34-67; `ResetPassword` :198-218; `RemoveStudent` enrollments/service.go:123-146; `UnassignLecturer` :149-173. Every path: `pool.Begin` → `defer tx.Rollback(ctx)` → `qtx := s.q.WithTx(tx)` → mutation on qtx → `WriteAudit(ctx, qtx, ...)` (SAME qtx) → `tx.Commit(ctx)`. No `_ =` on any WriteAudit (grep confirmed). Service structs hold `pool *pgxpool.Pool`. |
| **CR-02** | Duplicate-row message uses `fmt.Sprintf("%d", prev)` | ✅ RESOLVED | users/service.go:103 `fmt.Sprintf("duplicate ID in file (matches row %d)", prev)`. No `string(rune(...))` anywhere (grep clean). |
| **CR-03** | CreateAccount maps only `23505` via `errors.As(&pgconn.PgError)` to ErrDuplicateUser; others propagate | ✅ RESOLVED | users/service.go:50-56 `var pgErr *pgconn.PgError; if errors.As(err, &pgErr) && pgErr.Code == "23505" { return 0, ErrDuplicateUser }; return 0, err`. |
| **WR-01** | ResetPassword rejects SYSTEM (returns ErrUserNotFound) | ✅ RESOLVED | users/service.go:184-186 `if user.IsSystem { return ErrUserNotFound }`, before any mutation. Handler maps ErrUserNotFound→404 (users/handler.go:119-121). |
| **WR-02** | Login has explicit `if user.IsSystem` guard | ✅ RESOLVED | auth/service.go:29-31 `if user.IsSystem { return db.User{}, ErrInvalidCredentials }`, before bcrypt compare. |
| **WR-03 / WR-04** | courses/users/auditlogs list handlers clamp limit ≤200 | ✅ RESOLVED | courses/handler.go:78-80; users/handler.go:137-139; auditlogs/handler.go:40-42 — all `if limit > 200 { limit = 200 }`. |
| **WR-05** | CSV upload handlers validate Content-Type / `.csv` ext server-side | ✅ RESOLVED | users/handler.go:86-91 and enrollments/handler.go:61-66: reject unless ext==`.csv` OR content-type ∈ {`text/csv`, `application/vnd.ms-excel`}. Note: this is permissive (passes if EITHER ext or type matches), so it does NOT reject valid `text/csv`/`application/vnd.ms-excel`/`.csv` uploads — no false-positive regression. (See OBS-1 re: `application/octet-stream`.) |
| **WR-06** | Sweep ticker body wrapped in `recover()`; still exits on `ctx.Done()`; recover doesn't swallow loop | ✅ RESOLVED | lifecycle/sweep.go:25-41. `select` has `case <-ctx.Done(): return`; the `case <-ticker.C` body is an IIFE with `defer recover()` (:30-39). Recover is scoped to the IIFE, so a panic logs and the `for` loop continues; ctx cancellation still returns cleanly. |
| **WR-07** | Date parsing returns distinct format-error vs range-error | ✅ RESOLVED | courses/service.go:26-34 `parseDate` returns `ErrInvalidDateFormat`; range check at :55,141 returns `ErrInvalidDates`. Both defined in model.go:7,9 with distinct messages. Handler distinguishes them in the same INVALID_DATA bucket but with `err.Error()` so the message differs. |
| **IN-02** | CSV strips/neutralizes leading formula chars; doesn't corrupt mid-string names | ✅ RESOLVED | users/csv.go:81-83 and enrollments/csv.go:69 use `strings.TrimLeft(strings.TrimSpace(...), "=+-@\t\r ")`. TrimLeft strips a leading-char *set* only — "Anne-Marie" is untouched (hyphen is mid-string). Acceptable per finding intent. (Minor: a legitimate name like `-Anne` would lose its leading `-`; tolerated.) |
| **IN-03** | sqlc regenerated to v1.31.1 banner | ✅ RESOLVED | `backend/internal/shared/db/auditlogs.sql.go:3` → `// sqlc v1.31.1`. |
| **IN-04** | Roster endpoints wrapped in `{data:[...]}`; FE parses correctly | 🔁 REGRESSED | FE updated to expect `{data}` (admin-api.ts:76-81) but backend still returns bare array (courses/handler.go:210,239). Contract now broken at runtime. See **CR-R2-01**. |
| **IN-05** | reset_test.go uses unique username + cleanup | ✅ RESOLVED | reset_test.go:33 `username := "reset_test_" + strconv.FormatInt(time.Now().UnixNano(), 10)`; :37-39 deferred `DELETE FROM users WHERE id=$1`. Deterministic. |

## Regression-Hunt Results (round-2 focus)

- **Mutation succeeds on qtx but returns before Commit (silent data loss):** None. Every error path between mutation and Commit returns through the `defer tx.Rollback(ctx)`, so an un-committed mutation is always rolled back, never silently lost.
- **WriteAudit pointing at the pool instead of qtx (breaks atomicity):** None. All 7 mutations + the sweep + both CSV importers call `WriteAudit(ctx, qtx, ...)`. In enrollments `ImportMembers` the mutation runs on `repoTx := NewRepository(qtx)` and the audit on `qtx` directly — both wrap the same `tx`, so atomicity holds (verified enrollments/service.go:76-77,109 + repository.go:13-15).
- **Double-commit / missing `defer tx.Rollback`:** None. Each function has exactly one `defer tx.Rollback(ctx)` and one `tx.Commit(ctx)`. Rollback-after-commit is a documented pgx no-op.
- **Context misuse:** All Begin/mutation/audit/Commit use the same request `ctx`. Sweep goroutine uses the long-lived app `ctx` correctly.
- **CSV type validation rejecting valid uploads:** No false positives — the OR-logic accepts `.csv` extension, `text/csv`, and `application/vnd.ms-excel`. (`application/octet-stream` is NOT accepted unless the filename ends `.csv`; see OBS-1.)
- **IN-04 FE/BE contract:** DRIFTED — the one real regression (CR-R2-01).

## Observations (non-blocking, not counted as findings)

- **OBS-1 (info):** The CSV type check accepts a file when EITHER the extension is `.csv` OR content-type is `text/csv`/`application/vnd.ms-excel`. Browsers commonly send `application/octet-stream` (or empty) for CSV; those still pass *iff* the filename ends in `.csv`, which the FE enforces via `accept`. A curl/programmatic client sending octet-stream + a non-`.csv` filename would be rejected. This matches the round-2 guidance (don't reject valid `text/csv`) and is acceptable; just noting the extension dependency.
- **OBS-2 (info):** `lifecycle/sweep.go:45` `runSweep` still takes both `pool` and `q` and calls `q.WithTx(tx)` — the prior WR-06 note about `db.New(pool)` allocation was addressed (a single `q` is built once in `StartSweeper:14` and reused), so the per-run allocation is gone. The `pool` param is used for `pool.Begin`; fine.

---

## Carried-forward verified invariants (unchanged from R1, spot-checked)

- D-27 all-or-nothing CSV import; D-30/D-32 idempotent membership (`ON CONFLICT DO NOTHING`); D-37/D-39 sweep idempotency + startup catch-up + audit only when `n>0`; D-38 SYSTEM excluded from lists; SQL injection N/A (sqlc-parameterized). All still hold.

---

_Reviewed: 2026-06-20_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard — Round 2 re-review_
