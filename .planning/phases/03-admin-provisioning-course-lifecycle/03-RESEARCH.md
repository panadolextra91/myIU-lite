# Phase 3: Admin Provisioning & Course Lifecycle - Research

**Researched:** 2026-06-20
**Domain:** Go/Gin admin CRUD + CSV bulk import, Postgres audit triggers + soft-delete sweep, React/shadcn admin console
**Confidence:** HIGH (stack and patterns are locked in CLAUDE.md and grounded in existing Phase 1-2 code; only two facts needed external verification)

## Summary

Phase 3 is almost entirely an *application* of patterns the codebase already established in Phases 1-2 plus a small set of new Postgres-side primitives. The auth spine (`RequireRole(admin)`, `errorEnvelope`, cookie JWT, `password_changed_at` kill-switch, sqlc+pgx, golang-migrate, feature-folder split) is fully reusable and must be mirrored, not re-invented. The genuinely new technical work is: (1) two new migrations adding `users.full_name`/`date_of_birth`, the `courses`/`student_enrollments`/`course_lecturers` tables, audit_log columns, append-only triggers, and the SYSTEM seed; (2) a transactional all-or-nothing CSV importer over stdlib `encoding/csv`; (3) idempotent `ON CONFLICT DO NOTHING` membership imports; (4) an in-process daily sweep goroutine with a startup catch-up; (5) the first real admin UI (sidebar + data tables + CSV upload + course forms + roster tabs) using shadcn components not yet vendored.

Two decisions the CONTEXT flagged resolve cleanly toward the *leaner* option (Ponytail): the SYSTEM account should be an `is_system BOOLEAN` flag (NOT a new enum value — that avoids the `ALTER TYPE ... ADD VALUE` transaction caveat under golang-migrate entirely), and the sweep should be a plain `time.Ticker` goroutine (NOT gocron — a single daily job in a single-instance app does not justify a dependency). The audit_log shape should be extended with real columns (`operation_id`, `target_type`, `target_id`, `affected_count`) rather than packed into `metadata` JSONB, because D-36's viewer filters by actor/action/date and groups by operation — typed columns + indexes serve that far better than JSONB extraction.

**Primary recommendation:** Add migrations `000004` (schema: users columns, courses, enrollments, lecturers, audit_log columns) and `000005` (append-only triggers + SYSTEM seed), keep CSV import in one `pgx.Tx` with validate-all-then-insert, use `is_system` + `time.Ticker`, and build the UI from newly-added shadcn `sidebar`/`table`/`tabs`/`dialog`/`select`/`badge`/`skeleton` components. Organize as five vertical slices: Accounts, Courses, Enrollment/Lecturer, Audit-viewer, Sweep.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| ADMIN-01 | Create student/lecturer account manually | Reuse bcrypt cost=12 + `must_change_password` pattern from `000002` seed and `auth.Service`; new `INSERT` query per role; audit `ACCOUNT_CREATE`. See Accounts slice. |
| ADMIN-02 | Bulk-create from CSV, whole-file validation, dup/invalid reported | All-or-nothing `encoding/csv` + single `pgx.Tx` pattern (Pitfall 1, Code Examples). 422 + `{errors:[{row,field,message}]}` (D-27). |
| ADMIN-03 | Defaults: username=ID, password=DDMMYYYY, must_change_password | `dob DD/MM/YYYY` parse → `DDMMYYYY` derive → bcrypt (Code Examples §CSV row mapping). `date_of_birth DATE` column (D-26). |
| ADMIN-04 | Reset any user back to default DDMMYYYY, re-set flag | Reuse stored `date_of_birth` → regenerate hash, set `must_change_password=true`, bump `password_changed_at` (kill-switch, D-13/16). New `ResetPassword` query. |
| ADMIN-05 | CRUD courses with start/end dates | `courses` table (D-28), soft-delete on "delete" (D-29). Standard handler/service/repo + RHF/Zod forms. |
| ADMIN-06 | Assign students+lecturers from CSV | Idempotent `ON CONFLICT DO NOTHING` per-course import (D-30/D-32), all-or-nothing validation. Two tables (D-31). |
| ADMIN-07 | Auto soft-delete 1 month after end_date, no manual action | `time.Ticker` daily + startup catch-up (D-37), idempotent sweep SQL. SYSTEM actor (D-38). |
| ADMIN-08 | Every admin mutation → append-only audit row | DB triggers block UPDATE/DELETE (D-35); explicit `INSERT INTO audit_log` in services; bulk = one row + operation_id + affected_count (D-33). |
</phase_requirements>

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions (D-24 → D-43)

- **D-24** — Separate CSV files/actions for students vs lecturers. Student CSV cols: `student_id, full_name, dob` (`DD/MM/YYYY`). Lecturer CSV cols: `lecturer_id, full_name, dob`. Extra columns allowed and ignored. `username=ID`, `password=DDMMYYYY`, `must_change_password=TRUE`. Manual single-create uses same per-role fields.
- **D-25** — Store `full_name`; no email for MVP. `users` gains `full_name`.
- **D-26** — Store `date_of_birth (DATE)` for every account. ADMIN-04 reset regenerates `DDMMYYYY` from stored DOB; reuses `password_changed_at` kill-switch. Never store/redisplay plaintext passwords.
- **D-27** — All-or-nothing CSV validation. ≥1 error ⇒ no inserts, HTTP **422** with `{ "errors": [ { "row","field","message" } ] }`; FE renders error table.
- **D-28** — Course fields: `code, name, term, start_date, end_date` (no description), plus `deleted_at`. `term` (e.g. `2026.1`) is first-class, not derived from dates.
- **D-29** — Course deletion is soft-delete only (`deleted_at`); excluded from normal queries; no physical delete.
- **D-30** — Student enrollment import: per-course, additive, idempotent. CSV of `student_id` (one column). Already-enrolled rows ignored. Never removes. All-or-nothing: any invalid ID rejects whole file.
- **D-31** — Membership = two separate tables: `student_enrollments(course_id, student_id)`, `course_lecturers(course_id, lecturer_id)`. ≥1 lecturer, ≥0 students per course. Membership-after-registration only.
- **D-32** — Lecturer assignment mirrors D-30 (CSV of `lecturer_id`, additive/idempotent/all-or-nothing).
- **D-33** — Bulk op = ONE audit row + `operation_id` + `affected_count` (not per-entity). Per-entity detail (if ever needed) goes in dedicated operation-detail tables linked by `operation_id`.
- **D-34** — Audit payload: `actor, action, target_type, target_id, timestamp, metadata`; NO before/after diffs. Actions: `ACCOUNT_CREATE, PASSWORD_RESET, COURSE_CREATE, COURSE_UPDATE, COURSE_DELETE, ENROLL_IMPORT, LECTURER_IMPORT` (+ D-43 removals, + `COURSE_SWEEP`).
- **D-35** — Append-only enforced by DB triggers. `BEFORE UPDATE`/`BEFORE DELETE` on `audit_log` raise exception; only INSERT allowed. No DB-role separation for MVP (deferred).
- **D-36** — Admin audit-log viewer ships this phase: read-only page, pagination, filter by actor/action/date range, view metadata. Admin-only, no edit/delete.
- **D-37** — In-process daily sweep + startup catch-up. Go-native (`time.Ticker` or equivalent), once/day + one catch-up at startup. Set `deleted_at` for courses ≥1 month past `end_date`, ignore already-deleted, idempotent. No Redis/queues/workers.
- **D-38** — Dedicated SYSTEM account is actor for automated actions. Seeded at init; cannot log in / authenticate / own a session. Used for `COURSE_SWEEP` + future jobs. System rows append-only like human rows.
- **D-39** — Sweep audit granularity follows D-33: one row (`action=COURSE_SWEEP, actor=SYSTEM, affected_count=N`) ONLY when ≥1 course affected. No row when affected_count=0.
- **D-40** — Soft-delete does NOT cascade. Only `courses.deleted_at` set; related records unchanged. Normal queries exclude soft-deleted courses (related records become naturally hidden).
- **D-41** — First full admin sidebar (fulfils D-21, follows D-05: shadcn/ui, collapsible/hamburger, light+dark). Sections: Dashboard; User Management → Accounts; Academic Management → Courses, Student Enrollment, Lecturer Assignment; System → Audit Logs.
- **D-42** — Read-only course detail/roster page with tabs: Overview, Students (`student_id, full_name`), Lecturers (`lecturer_id, full_name`). Admin views all rosters.
- **D-43** — Manual membership removal ships this phase, from roster page (UI, not CSV). Remove individual student / unassign individual lecturer. Each writes audit row (`STUDENT_REMOVED_FROM_COURSE`, `LECTURER_UNASSIGNED_FROM_COURSE`). CSV stays additive/idempotent.

### Claude's Discretion
- Backend feature-folder grouping (planner picks `internal/courses/`, `internal/auditlogs/`, etc.). Audit writes are explicit `INSERT` SQL (optionally a Gin middleware on admin routes).
- All admin endpoints behind `middleware.RequireRole(db.UserRoleAdmin)`, established `{error:{code,message}}` envelope, 422 for CSV failures.
- Admin Dashboard content discretionary — sensible default: counts (students/lecturers/courses/recent audit events). Keep lean.
- Concrete action-code strings, target_type values, accounts-list search/pagination, manual-create form layout — planner picks, consistent with D-34 taxonomy + D-05.
- Reset is single-user (per-user action from Accounts list, no bulk), reuses D-26 stored DOB.

### Deferred Ideas (OUT OF SCOPE)
- Course retention/archival (~5yr threshold, move to long-term storage, no cascade).
- Audit before/after snapshots / field-level diffs.
- Audit-log DB-role hardening (`REVOKE UPDATE/DELETE`, dedicated app roles).
- Multi-instance sweep hardening (external cron / distributed locking).
- Lecturer/student-facing roster visibility (D-42 is admin-only this phase).
</user_constraints>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| CSV parse + all-or-nothing validation | API (service) | — | Security + business rule; never trust client. `encoding/csv` is server-side. |
| Password derivation (DDMMYYYY → bcrypt) | API (service) | — | Secret material; bcrypt cost=12 server-only. |
| Audit append-only enforcement | Database (triggers) | API (explicit INSERT) | Invariant enforced closest to data (D-35); app issues the INSERTs. |
| Course soft-delete sweep | API (in-process goroutine) | Database (UPDATE) | Single-instance scheduler in the binary (D-37); SQL does the set. |
| SYSTEM actor non-loginability | Database (`is_system` flag) | API (auth middleware never loads it) | Flag in `users`; auth.go's existing username lookup never returns it (no valid password / excluded). |
| Idempotent membership | Database (`ON CONFLICT DO NOTHING` + unique index) | API (tx orchestration) | Idempotency is a constraint property; service wraps in tx. |
| Admin console (sidebar, tables, forms, roster) | Frontend (React/shadcn) | API (REST JSON) | shadcn components; TanStack Query owns server state. |
| Pagination/filtering of accounts + audit | API (SQL LIMIT/OFFSET/WHERE) | Frontend (query params) | Server paginates; FE sends params, renders. |

## Standard Stack

All libraries are LOCKED in CLAUDE.md — this phase adds **zero new backend dependencies** and only **new shadcn components** (copied source, not deps) on the frontend.

### Core (already present, reuse)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| sqlc | v1.31.1 | SQL→Go codegen | Locked; regenerate after adding queries. `sql_package: pgx/v5`, `package: db`, out `internal/shared/db`. [VERIFIED: backend/sqlc.yaml] |
| pgx/v5 | v5.7.x | Postgres driver + `Tx`/`CopyFrom`/`Batch` | Locked; tx is the all-or-nothing mechanism. [VERIFIED: codebase imports] |
| golang-migrate | v4.18.x | Migrations | Locked; append `000004`,`000005`. [CITED: CLAUDE.md] |
| golang-jwt/jwt v5 + bcrypt | v5.3.1 / x/crypto | Auth/hash (reuse) | bcrypt cost=12 already used in `auth.Service` and `000002` seed. [VERIFIED: backend/internal/auth/service.go] |
| Gin | v1.11.0 | HTTP | Locked; `RegisterRoutes` pattern. [VERIFIED: codebase] |
| `encoding/csv` | stdlib | CSV parse | Locked; no third-party CSV dep. [CITED: CLAUDE.md] |
| `time` (Ticker) | stdlib | Sweep scheduler | RECOMMENDED over gocron (see Don't Hand-Roll / Pitfalls). |

### Supporting (frontend — already in package.json)
| Library | Version | Purpose |
|---------|---------|---------|
| @tanstack/react-query | ^5.101.0 | Server state for all admin lists/mutations. [VERIFIED: package.json] |
| react-hook-form | ^7.79.0 | Course CRUD + manual-create forms. [VERIFIED: package.json] |
| zod | ^4.4.3 + @hookform/resolvers ^5.4.0 | Client form validation. [VERIFIED: package.json] |
| zustand | ^5.0.14 | Auth/UI state only (sidebar collapse state). [VERIFIED: package.json] |
| react-router | ^8.0.1 | `/admin/*` nested routes (note: actual installed is v8, not v7). [VERIFIED: package.json] |
| axios | ^1.18.0 | `api` client w/ cookie + 401/403 interceptor (reuse `src/lib/api.ts`). [VERIFIED: codebase] |
| lucide-react | ^1.21.0 | Sidebar/section icons. [VERIFIED: package.json] |

### New shadcn components to add (CLI `npx shadcn@latest add ...`)
Currently vendored: `button, card, form, input, label, sonner` [VERIFIED: ls components/ui].
Add: **`sidebar`** (D-41 collapsible nav), **`table`** (accounts list, audit viewer, rosters), **`tabs`** (D-42 roster Overview/Students/Lecturers), **`dialog`** (course create/edit, manual account create, confirm removal), **`select`** (course picker for enrollment import, audit action filter), **`badge`** (role/status chips), **`skeleton`** (D-05 loaders), **`alert-dialog`** (destructive confirm for soft-delete + removal), **`popover`+`calendar`** OR a plain date `input type=date` (course start/end + audit date-range — see note), **`pagination`** (or hand-driven page buttons via `button`). Sidebar pulls in `sheet`, `separator`, `tooltip`, `skeleton` transitively via the CLI.

> **Ponytail note on dates:** shadcn `calendar` depends on `react-day-picker` (a new dep). For course start/end dates and the audit date-range filter, a native `<input type="date">` styled with the existing `input` component is sufficient and avoids the dependency. Recommend native date inputs unless the planner wants the calendar UX.

**Installation:**
```bash
# Frontend (run from frontend/)
npx shadcn@latest add sidebar table tabs dialog alert-dialog select badge skeleton

# Backend: NO new go deps. After adding queries/migrations:
cd backend && sqlc generate
```

**Version verification:** No new packages introduced — backend deps already pinned and building (Phase 2 merged). Frontend deps already in `package.json` and matching CLAUDE.md pins (TanStack v5.101, RHF v7.79, Zod v4, shadcn CLI present). shadcn `add` copies source into `src/components/ui/` (no version drift risk).

## Package Legitimacy Audit

> No new packages are installed this phase. Backend adds zero deps; frontend adds only shadcn-vendored component source (Radix primitives already transitively present). Audit not applicable.

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
                         ┌─────────────────────── React Admin SPA ───────────────────────┐
                         │  Sidebar (D-41) → routes: /admin/accounts /admin/courses        │
                         │   /admin/enrollment /admin/lecturers /admin/audit /admin/courses/:id│
                         │  TanStack Query (server state)  RHF+Zod (forms)  axios(cookie)  │
                         └───────────────────────────────┬─────────────────────────────────┘
                                                          │ JSON over HTTPS (JWT cookie)
                                                          ▼
            ┌──────────────────────── Gin router ────────────────────────────────┐
            │  /admin/* group → AuthMiddleware → RequireRole(admin)               │
            │                                                                     │
            │  handler.go (parse multipart CSV / JSON, shape response, 422 errs)  │
            │        │                                                            │
            │        ▼                                                            │
            │  service.go (validate WHOLE file → derive password → bcrypt;        │
            │              orchestrate ONE pgx.Tx; write explicit audit INSERT)   │
            │        │                                                            │
            │        ▼                                                            │
            │  repository.go (sqlc queries; WithTx(tx) for atomic bulk)           │
            └────────┬──────────────────────────────────────────────┬────────────┘
                     │                                               │
        ┌────────────▼───────────────┐                ┌──────────────▼─────────────────┐
        │  in-process sweep goroutine │                │          PostgreSQL              │
        │  (time.Ticker, started in   │ writes via     │  users(+full_name,+dob,+is_system│
        │   main.go): startup catch-up│ SYSTEM actor   │  courses / student_enrollments / │
        │   then daily UPDATE courses │───────────────▶│  course_lecturers / audit_log    │
        │   SET deleted_at; audit if  │                │  TRIGGERS: BEFORE UPD/DEL on     │
        │   ≥1 affected (D-39)        │                │  audit_log RAISE EXCEPTION (D-35)│
        └─────────────────────────────┘                └──────────────────────────────────┘
```

Trace the CSV-import use case: multipart upload → handler reads file under `MaxBytesReader` → service parses all rows with `encoding/csv`, validates each (collecting `{row,field,message}`), checks dup IDs in-file and against DB → if any error, return 422 (no writes) → else open `pgx.Tx`, insert all users with derived bcrypt hashes via `repo.WithTx(tx)`, INSERT one audit row (`IMPORT_STUDENTS`, `affected_count=N`, `operation_id`), commit.

### Recommended Project Structure
Planner's discretion (D-10), but CLAUDE.md names `courses/` and `auditlogs/` as feature folders. Suggested:
```
backend/internal/
├── users/          # ADMIN-01..04: handler/service/repository/model/dto.go (account CRUD, CSV import, reset)
├── courses/        # ADMIN-05: course CRUD + soft-delete
├── enrollments/    # ADMIN-06/D-43: student & lecturer membership import + removal
├── auditlogs/      # ADMIN-08/D-36: read-only viewer queries + the shared writeAudit helper
├── lifecycle/      # ADMIN-07: sweep scheduler (time.Ticker) wired from main.go
└── shared/db/      # sqlc-generated (regenerate after new queries)
backend/db/migrations/000004_*.{up,down}.sql   # schema
backend/db/migrations/000005_*.{up,down}.sql   # triggers + SYSTEM seed
backend/db/queries/{users,courses,enrollments,auditlogs}.sql
frontend/src/pages/admin/{Accounts,Courses,CourseDetail,Enrollment,LecturerAssignment,AuditLogs,Index}.tsx
frontend/src/components/admin/AdminSidebar.tsx
```

> A shared `writeAudit(ctx, tx, actorID, action, targetType, targetID, affectedCount, metadata)` helper (in `auditlogs/`) keeps audit INSERTs DRY across services. The CONTEXT permits a Gin middleware alternative, but explicit in-service INSERTs inside the same tx as the mutation are clearer and let bulk ops emit ONE row (D-33) — recommend explicit helper over middleware.

### Pattern 1: Resolved — audit_log schema shape (D-33/D-34)
**Decision: add typed columns via migration `000004`; do NOT pack into `metadata` JSONB.**
The D-36 viewer filters by actor (`actor_id` exists), action (`action` exists), and date (`created_at` exists) — those already work. The new needs are `target_type`, `target_id` (D-34 taxonomy), `operation_id` (group a bulk op, D-33), and `affected_count` (D-33/D-39). Typed nullable columns are filterable/indexable and let the viewer show counts directly; JSONB extraction (`metadata->>'affected_count'`) is harder to query and not type-safe in sqlc. Keep `metadata JSONB` for incidental extras (e.g. uploaded filename). **No dedicated operation-detail tables for MVP** — D-33 explicitly says those come only "if finer traceability is ever needed"; `affected_count` on one row suffices.

```sql
-- 000004 (excerpt)
ALTER TABLE audit_log
    ADD COLUMN target_type    TEXT,
    ADD COLUMN target_id      BIGINT,
    ADD COLUMN operation_id   UUID,
    ADD COLUMN affected_count INTEGER;
CREATE INDEX audit_log_action_idx ON audit_log (action);
-- existing actor_idx + created_idx already cover the other two filters
```
sqlc implication: a `WriteAuditLog :exec` with all columns (nullable ones via `pgtype.Text`/`pgtype.Int8`/`pgtype.UUID`/`pgtype.Int4`), and a `ListAuditLogs :many` taking optional `actor_id`/`action`/`from`/`to` + `LIMIT`/`OFFSET`. Use `sqlc.narg()` for optional filters so one query handles all filter combinations.

### Pattern 2: Resolved — SYSTEM account (D-38)
**Decision: add `is_system BOOLEAN NOT NULL DEFAULT FALSE`; do NOT extend the `user_role` enum.**
Rationale (and the decisive caveat): `ALTER TYPE user_role ADD VALUE 'system'` **cannot run inside a transaction block** in Postgres, and golang-migrate runs each migration in a transaction by default — you'd hit `ALTER TYPE ... ADD cannot run inside a transaction block` [VERIFIED: web search — github.com/flyway/flyway#350, golang-migrate TUTORIAL]. Working around it (split files, `COMMIT;`/`BEGIN;` hacks, or `x-no-tx`) is fragile. The `is_system` flag is the Ponytail-lean choice: no enum change, the SYSTEM user keeps a real `role` (use `'admin'` purely as a placeholder value — it is never used because the account can't authenticate).

Non-loginability is achieved for free by the existing auth path:
- `auth.Service.Login` calls `GetUserByUsername` then `bcrypt.CompareHashAndPassword`. Seed the SYSTEM row with a **non-bcrypt sentinel** `password_hash` (e.g. `'!'` or `'x'`); `bcrypt.CompareHashAndPassword` returns an error for any non-bcrypt hash, so login always fails. [VERIFIED: backend/internal/auth/service.go lines 29-31]
- Exclude from listings: every account-list query adds `AND is_system = FALSE` (alongside `deleted_at IS NULL`).
- Partial unique index coexistence: `users_username_active_uq` is on `(username) WHERE deleted_at IS NULL`. Seed SYSTEM with a username that never collides (e.g. `__system__`) and `deleted_at IS NULL`; it occupies one slot in the index, which is fine.
- `auth.go` middleware loads users by ID from a valid JWT only; since SYSTEM can never log in, no JWT ever carries its ID, so it never reaches the middleware. (Defense-in-depth: account-list and any future "manage user" endpoints filter `is_system = FALSE`.)

```sql
-- 000004: ALTER TABLE users ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT FALSE;
-- 000005 seed (after triggers):
INSERT INTO users (username, password_hash, role, must_change_password, is_system)
VALUES ('__system__', '!', 'admin', FALSE, TRUE);
```
> Capture the SYSTEM user's `id` at startup (a `GetSystemUserID :one` query: `SELECT id FROM users WHERE is_system = TRUE LIMIT 1`) and pass it to the sweep as the audit actor.

### Pattern 3: Resolved — append-only triggers (D-35)
A `BEFORE UPDATE` and `BEFORE DELETE` trigger calling one trigger function that `RAISE EXCEPTION`. INSERT is unaffected. This is fully compatible with golang-migrate transactional migrations (creating functions/triggers is transaction-safe). The down migration drops both triggers then the function.

```sql
-- 000005 up
CREATE OR REPLACE FUNCTION audit_log_append_only() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'audit_log is append-only: % not permitted', TG_OP;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_log_no_update BEFORE UPDATE ON audit_log
    FOR EACH ROW EXECUTE FUNCTION audit_log_append_only();
CREATE TRIGGER audit_log_no_delete BEFORE DELETE ON audit_log
    FOR EACH ROW EXECUTE FUNCTION audit_log_append_only();

-- 000005 down
DROP TRIGGER IF EXISTS audit_log_no_delete ON audit_log;
DROP TRIGGER IF EXISTS audit_log_no_update ON audit_log;
DROP FUNCTION IF EXISTS audit_log_append_only();
```
Verification step for the plan: an integration test that INSERTs an audit row (succeeds), then attempts UPDATE and DELETE (both must error with the raised message). [CONFIRMED: standard plpgsql trigger semantics; INSERT not intercepted because no BEFORE INSERT trigger exists.]

### Pattern 4: Resolved — sweep scheduler (D-37)
**Decision: `time.Ticker` goroutine launched in `main.go`; do NOT add gocron.** Single daily job + startup catch-up in a single-instance binary is the textbook case where gocron's builder/cron-parser/locker buys nothing (Ponytail; CLAUDE.md "What NOT to over-build" explicitly says a plain ticker is enough for one daily sweep). The sweep SQL is idempotent so the startup catch-up and the daily tick are the same operation.

```go
// lifecycle/sweep.go
func StartSweeper(ctx context.Context, pool *pgxpool.Pool, systemID int64) {
    run := func() {
        n, err := runSweep(ctx, pool, systemID) // see runSweep below
        if err != nil { log.Printf("sweep error: %v", err); return }
        if n > 0 { log.Printf("sweep soft-deleted %d courses", n) }
    }
    run() // startup catch-up immediately
    go func() {
        t := time.NewTicker(24 * time.Hour)
        defer t.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-t.C:
                run()
            }
        }
    }()
}

// runSweep: idempotent UPDATE + conditional audit (D-39)
func runSweep(ctx context.Context, pool *pgxpool.Pool, systemID int64) (int64, error) {
    tx, err := pool.Begin(ctx)
    if err != nil { return 0, err }
    defer tx.Rollback(ctx)
    tag, err := tx.Exec(ctx,
        `UPDATE courses SET deleted_at = now()
         WHERE deleted_at IS NULL AND end_date < now() - interval '1 month'`)
    if err != nil { return 0, err }
    n := tag.RowsAffected()
    if n > 0 { // D-39: audit row ONLY when ≥1 affected
        _, err = tx.Exec(ctx,
            `INSERT INTO audit_log (actor_id, action, target_type, affected_count)
             VALUES ($1, 'COURSE_SWEEP', 'course', $2)`, systemID, n)
        if err != nil { return 0, err }
    }
    return n, tx.Commit(ctx)
}
```
Wire in `main.go` after the pool and config load and route registration: resolve `systemID` via `GetSystemUserID`, then `lifecycle.StartSweeper(ctx, pool, systemID)`. Use a cancelable context tied to shutdown. [VERIFIED: matches CLAUDE.md sweep SQL verbatim.]

### Pattern 5: Resolved — all-or-nothing CSV import (D-24/D-27)
Phases: **(1) parse** all rows with `encoding/csv`; **(2) validate** every row, collecting `[]RowError{Row, Field, Message}` (header presence, required fields non-empty, `dob` parses as `DD/MM/YYYY`, ID well-formed, no duplicate ID *within the file*, no existing active username *in DB*); **(3) if len(errors)>0 → 422, write nothing**; **(4) else** open ONE `pgx.Tx`, insert every user (derive `DDMMYYYY` → bcrypt cost=12), INSERT one audit row, commit. A late insert failure rolls the whole tx back (true all-or-nothing). See Code Examples. Duplicate-against-DB check: `SELECT username FROM users WHERE username = ANY($1) AND deleted_at IS NULL AND is_system = FALSE` over all candidate IDs in one round-trip, then diff. Within-file dups: a `map[string]int` of first-seen row.

### Pattern 6: Resolved — idempotent membership import (D-30/D-32)
Unique constraint per table + `ON CONFLICT DO NOTHING`. Validation (all-or-nothing) rejects the whole file if any `student_id`/`lecturer_id` is unknown or wrong role; but rows for *already-enrolled* members insert as no-ops (no error, no duplicate).

```sql
-- migration: UNIQUE (course_id, student_id) / UNIQUE (course_id, lecturer_id)
-- query (per row, inside the tx):
INSERT INTO student_enrollments (course_id, student_id)
VALUES ($1, $2) ON CONFLICT (course_id, student_id) DO NOTHING;
```
`affected_count` for the audit row = sum of `RowsAffected()` (counts only newly-inserted rows), so the audit reflects real additions, not skipped duplicates. Validate that each ID exists AND has the right role (`role='student'` / `role='lecturer'`, `is_system=FALSE`, `deleted_at IS NULL`) and that the course exists and is not soft-deleted.

### Anti-Patterns to Avoid
- **Per-entity audit rows for bulk ops** — violates D-33. One row + `affected_count`.
- **Packing audit fields in `metadata` JSONB** — defeats D-36 filtering; use typed columns.
- **`ALTER TYPE ADD VALUE 'system'`** — transaction-block failure under golang-migrate; use `is_system`.
- **gocron for one daily job** — needless dependency; `time.Ticker`.
- **Partial CSV inserts / per-row commit** — violates D-27; one tx, all-or-nothing.
- **Trusting client Zod for CSV file type/size** — validate server-side (`MaxBytesReader`, content checks).
- **Hand-rolled UI components / hand-rolled date picker beyond native input** — D-05 shadcn-only.
- **Physical course delete** — D-29 soft-delete only; reads filter `deleted_at IS NULL`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Append-only audit enforcement | App-layer "please don't update" convention | Postgres `BEFORE UPDATE/DELETE` trigger (D-35) | Invariant must hold even for buggy/future code paths; DB is the only reliable gate. |
| Daily scheduler | gocron / cron lib | stdlib `time.Ticker` + startup run | One job, one instance — ticker is dependency-free and idempotent. |
| Bulk-insert atomicity | Manual rollback bookkeeping | `pgx.Tx` (Begin/Commit/Rollback) + `repo.WithTx(tx)` | Tx gives all-or-nothing for free; sqlc `WithTx` is built-in. |
| Idempotent membership | "SELECT then INSERT if absent" | `INSERT ... ON CONFLICT DO NOTHING` + unique index | Race-free, single statement, exactly D-30/D-32 semantics. |
| CSV parsing | Custom split-on-comma | `encoding/csv` (stdlib) | Handles quoting/escaping/CRLF; locked in CLAUDE.md. |
| Admin tables/sidebar/forms | Custom components | shadcn `table`/`sidebar`/`form`/`dialog` | D-05 "no hand-rolled components". |
| Server-state caching | `useEffect`+`fetch` | TanStack Query | Cache/invalidation/loading states; pairs with mutations (re-fetch after import). |
| SYSTEM account modeling | New enum value | `is_system BOOLEAN` | Avoids enum-in-transaction caveat; keeps migrations clean. |

**Key insight:** every "deceptively complex" piece of this phase already has a one-liner answer in the locked stack or in Postgres. The phase's risk is *consistency* (every list filters `deleted_at IS NULL AND is_system=FALSE`; every mutation writes exactly one audit row), not novel engineering.

## Runtime State Inventory

> This phase is additive (new tables/columns/seed), not a rename/refactor. Inventory included because it seeds a SYSTEM account and adds a scheduler — both runtime state.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | New tables `courses`, `student_enrollments`, `course_lecturers`; new `users` columns `full_name`, `date_of_birth`, `is_system`; new `audit_log` columns. One seeded SYSTEM row. | Migrations `000004`/`000005` + `sqlc generate`. |
| Live service config | None — no external services configured this phase (no Cloudinary). | None. |
| OS-registered state | None — sweep is in-process (`time.Ticker`), not OS cron (D-37). | None. |
| Secrets/env vars | None new — reuses existing `JWTSecret`, `DatabaseURL`, `CookieSecure`, `FrontendOrigin` from `config.Config`. | None — verified `config.Load()` covers current needs. |
| Build artifacts | sqlc-generated `internal/shared/db/*.sql.go` + `models.go` regenerate after new queries; `User` struct gains `FullName`/`DateOfBirth`/`IsSystem` fields, new `Course`/`StudentEnrollment`/`CourseLecturer` models. | Run `sqlc generate`; commit generated files (repo commits them — verified `models.go` is tracked). |

## Common Pitfalls

### Pitfall 1: Validating row-by-row and inserting as you go
**What goes wrong:** A failure on row 200 leaves rows 1-199 created — violates D-27, produces inconsistent state, makes retry create duplicates.
**Why it happens:** Naive loop that inserts inside the validation loop.
**How to avoid:** Strict two-phase: validate ALL rows first (collect every error), only then open the tx and insert all. Any insert error rolls the whole tx back.
**Warning signs:** Inserts inside the same loop as `csv.Read()`; no `tx.Rollback` defer.

### Pitfall 2: SYSTEM account leaking into account lists or becoming loginable
**What goes wrong:** Admin sees a `__system__` row; or someone sets a real password and logs in as SYSTEM.
**Why it happens:** Forgetting `AND is_system = FALSE` in list queries; seeding a real bcrypt hash.
**How to avoid:** Sentinel non-bcrypt `password_hash` (`'!'`), `is_system=FALSE` filter in every list/lookup that isn't the explicit `GetSystemUserID`. Integration test: login as `__system__` must 401.
**Warning signs:** Account count is off by one; `__system__` appears in the accounts table.

### Pitfall 3: `dob` format confusion (DD/MM/YYYY input vs DDMMYYYY password)
**What goes wrong:** Password derived as `2006-01-02` or with slashes; users can't log in with their birthday.
**Why it happens:** Reusing the parse layout as the password format.
**How to avoid:** Parse with `time.Parse("02/01/2006", raw)`, derive password with `t.Format("02012006")` (D-24). Distinct layouts. Cover with a unit test (e.g. `09/03/2001` → `09032001`).
**Warning signs:** Generated password contains `/` or `-`.

### Pitfall 4: Sweep audit row written on no-op days
**What goes wrong:** Audit log fills with `COURSE_SWEEP affected_count=0` noise — violates D-39.
**Why it happens:** Unconditional audit INSERT after the UPDATE.
**How to avoid:** Guard `if n > 0` before the audit INSERT (see Pattern 4).
**Warning signs:** Daily `COURSE_SWEEP` rows with `affected_count=0`.

### Pitfall 5: Forgetting `deleted_at IS NULL` on course-scoped reads
**What goes wrong:** Soft-deleted courses reappear in lists / enrollment targets (D-29/D-40 broken).
**Why it happens:** Copy-pasting a query without the filter.
**How to avoid:** Every `courses` SELECT and every enrollment-target lookup filters `deleted_at IS NULL`. Roster reads go through an active course.
**Warning signs:** A swept course still selectable in the enrollment course-picker.

### Pitfall 6: CSV upload with no size guard
**What goes wrong:** A huge or malicious upload exhausts memory (no Cloudinary streaming this phase — file is read server-side).
**Why it happens:** Reading `c.Request.Body` / multipart without a cap.
**How to avoid:** Set `router.MaxMultipartMemory` and wrap with `http.MaxBytesReader` (CLAUDE.md pattern). Admin CSVs are small (hundreds–thousands of rows); a few MB cap is generous. Reject oversized with 413.
**Warning signs:** Importer accepts arbitrarily large files.

## Code Examples

### CSV row → user mapping (DD/MM/YYYY → DDMMYYYY → bcrypt)
```go
// Source: stdlib time/encoding/csv + golang.org/x/crypto/bcrypt; pattern per D-24
const dobInputLayout = "02/01/2006"     // DD/MM/YYYY (input)
const dobPasswordLayout = "02012006"    // DDMMYYYY (derived password)

type RowError struct {
    Row     int    `json:"row"`
    Field   string `json:"field"`
    Message string `json:"message"`
}

func deriveDefaults(rawDOB string) (dob time.Time, passwordHash string, err error) {
    dob, err = time.Parse(dobInputLayout, rawDOB)
    if err != nil {
        return time.Time{}, "", fmt.Errorf("dob must be DD/MM/YYYY")
    }
    pw := dob.Format(dobPasswordLayout)            // e.g. "09032001"
    h, err := bcrypt.GenerateFromPassword([]byte(pw), 12) // cost=12 (CLAUDE.md)
    if err != nil { return time.Time{}, "", err }
    return dob, string(h), nil
}
```

### All-or-nothing import inside one transaction (sqlc + pgx.Tx)
```go
// Source: sqlc docs (WithTx) + pgx/v5 Tx; D-27 all-or-nothing
// Validate first (omitted): build []RowError; if len>0 -> handler returns 422.
func (s *Service) ImportStudents(ctx context.Context, rows []studentRow, actorID int64) (int, error) {
    tx, err := s.pool.Begin(ctx)
    if err != nil { return 0, err }
    defer tx.Rollback(ctx) // no-op after Commit
    qtx := s.q.WithTx(tx)  // sqlc-generated: scopes queries to the tx

    opID := uuid.New()
    for _, r := range rows {
        if err := qtx.CreateUser(ctx, db.CreateUserParams{
            Username: r.StudentID, PasswordHash: r.Hash, Role: db.UserRoleStudent,
            FullName: r.FullName, DateOfBirth: pgtype.Date{Time: r.DOB, Valid: true},
            MustChangePassword: true,
        }); err != nil {
            return 0, err // rolls back EVERYTHING
        }
    }
    if err := qtx.WriteAuditLog(ctx, db.WriteAuditLogParams{
        ActorID: pgtype.Int8{Int64: actorID, Valid: true},
        Action:  "IMPORT_STUDENTS", TargetType: pgtype.Text{String: "user", Valid: true},
        OperationID: pgtype.UUID{Bytes: opID, Valid: true},
        AffectedCount: pgtype.Int4{Int32: int32(len(rows)), Valid: true},
    }); err != nil {
        return 0, err
    }
    return len(rows), tx.Commit(ctx)
}
```
> sqlc generates `func (q *Queries) WithTx(tx pgx.Tx) *Queries` automatically for the pgx/v5 driver — no manual plumbing. [VERIFIED: web search — docs.sqlc.dev insert howto + sqlc 1.31.1]. `uuid` comes from `github.com/google/uuid` — **check whether it's already an indirect dep** (`go list -m all | grep google/uuid`); if not present, planner adds it OR uses `pgtype.UUID` populated from `gen_random_uuid()` in SQL (DEFAULT on the column) to avoid a new dep. Recommend the SQL `DEFAULT gen_random_uuid()` route (Ponytail — no Go dep). [ASSUMED: google/uuid not yet a direct dep — planner verifies.]

### Handler 422 shape (D-27)
```go
if len(rowErrors) > 0 {
    c.JSON(http.StatusUnprocessableEntity, gin.H{"errors": rowErrors})
    return
}
```

### Optional-filter list query (sqlc narg) for the audit viewer (D-36)
```sql
-- name: ListAuditLogs :many
SELECT * FROM audit_log
WHERE (sqlc.narg('actor_id')::bigint IS NULL OR actor_id = sqlc.narg('actor_id'))
  AND (sqlc.narg('action')::text   IS NULL OR action = sqlc.narg('action'))
  AND (sqlc.narg('from')::timestamptz IS NULL OR created_at >= sqlc.narg('from'))
  AND (sqlc.narg('to')::timestamptz   IS NULL OR created_at <= sqlc.narg('to'))
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;
```

### Frontend: TanStack mutation + invalidate after import
```tsx
// Source: TanStack Query v5 pattern
const qc = useQueryClient();
const importStudents = useMutation({
  mutationFn: (file: File) => {
    const fd = new FormData(); fd.append('file', file);
    return api.post('/admin/students/import', fd);
  },
  onSuccess: () => { qc.invalidateQueries({ queryKey: ['accounts'] }); toast.success('Imported'); },
  onError: (e) => {/* e.response.status===422 -> render e.response.data.errors as <Table> */},
});
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `lib/pq` driver | `pgx/v5` + sqlc | 2023+ | Already adopted; use `WithTx`, `pgtype.*`. |
| ORM bulk insert | `pgx.Tx` + sqlc loop / `:copyfrom` | ongoing | For ≤few-thousand rows a tx loop is fine and clearer than CopyFrom; CopyFrom only if a single import is very large. |
| `react-router` v6 data APIs | v8 (installed) | — | Nested route objects (already used in `router.tsx`); `createBrowserRouter`. |
| Hand-rolled scheduler libs | stdlib `time.Ticker` for single-instance | — | Lean default for one daily job. |

**Deprecated/outdated:**
- Adding enum values in migrations — avoid via boolean flag (transaction caveat).
- `useEffect`+`fetch` for admin lists — use TanStack Query.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `github.com/google/uuid` is not yet a direct dependency; recommend SQL `DEFAULT gen_random_uuid()` for `operation_id` to avoid adding it. | Code Examples | Low — if it's already present, planner can use it directly; if not and SQL default is used, zero new deps. Planner verifies with `go list -m all`. |
| A2 | shadcn `calendar` would add `react-day-picker`; native `<input type="date">` is preferred to avoid the dep. | Standard Stack | Low — purely UX; both satisfy "shadcn styling" since the input wrapper is shadcn. User may prefer the calendar. |
| A3 | A few-MB upload cap is "generous" for admin CSVs. | Pitfall 6 | Low — class sizes are small; planner sets a concrete number (e.g. 5MB). |
| A4 | The bootstrap `admin` (000002) is the actor for manual/CSV admin actions via its JWT `user_id`; SYSTEM actor is only for automated sweeps. | Patterns | Low — matches D-38 ("SYSTEM for automated actions") and existing auth (`c.Get("user_id")`). |

**If this table is empty:** Not empty — four low-risk assumptions, all verifiable by the planner in one command each.

## Open Questions

1. **`operation_id` generation site (Go vs SQL DEFAULT).**
   - What we know: bulk ops need a UUID grouping key (D-33). Postgres has `gen_random_uuid()` built in (pgcrypto/`pg_catalog` in PG13+).
   - What's unclear: whether the team wants the ID echoed back to the client in the import response.
   - Recommendation: column `operation_id UUID NOT NULL DEFAULT gen_random_uuid()`; if the response must carry it, use `INSERT ... RETURNING operation_id`. Avoids a Go uuid dep.

2. **Accounts-list scope: separate student/lecturer lists vs one filtered list.**
   - What we know: D-24 separates *imports* by role; D-41 has a single "Accounts" nav item.
   - What's unclear: whether the Accounts page filters by role or shows all.
   - Recommendation: one Accounts table with a role filter (shadcn `select`) + search by username/full_name; server paginates. Discretionary per CONTEXT.

## Environment Availability

> External dependencies are the same as Phase 2 (Postgres via Docker, Go toolchain, Node for frontend). No new runtime deps (no Cloudinary this phase).

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| PostgreSQL (Docker) | All DB work, triggers, `gen_random_uuid()` | ✓ (Phase 1) | postgres:17-alpine | — |
| Go toolchain | Backend build + `sqlc generate` | ✓ | 1.24.x | — |
| sqlc CLI | Regenerate after new queries | ✓ (used Phases 1-2) | v1.31.1 | — |
| golang-migrate | Apply `000004`/`000005` | ✓ | v4.18.x | — |
| Node + shadcn CLI | Add new UI components | ✓ | shadcn ^4.11 | components can be hand-copied from registry if CLI fails |
| Docker | CI integration tests (testcontainers) | ✓ | — | — |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** shadcn CLI (registry copy fallback).

## Validation Architecture

> nyquist_validation assumed enabled (no `.planning/config.json` key found stating otherwise during research — planner confirms).

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` + testify; testcontainers-go postgres for integration (per CLAUDE.md / Phase 1-2). Frontend: Vitest + RTL (per CLAUDE.md). |
| Config file | `backend/sqlc.yaml` (codegen); CI workflow runs migrate + `go test`. Existing test: `internal/shared/db/seed_test.go`. |
| Quick run command | `cd backend && go test ./internal/... -run <Name> -count=1` |
| Full suite command | `cd backend && go test ./... ` (+ frontend `npm test` if Vitest configured) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| ADMIN-02/27 | All-or-nothing: invalid row ⇒ zero inserts + 422 | integration | `go test ./internal/users -run TestImportAllOrNothing` | ❌ Wave 0 |
| ADMIN-03 | DD/MM/YYYY → DDMMYYYY derivation | unit | `go test ./internal/users -run TestDeriveDefaults` | ❌ Wave 0 |
| ADMIN-04 | Reset regenerates DDMMYYYY + sets must_change + bumps password_changed_at | integration | `go test ./internal/users -run TestResetPassword` | ❌ Wave 0 |
| ADMIN-06 | Idempotent enrollment: re-import skips dups, no error | integration | `go test ./internal/enrollments -run TestEnrollIdempotent` | ❌ Wave 0 |
| ADMIN-07 | Sweep soft-deletes ≥1mo-past courses, idempotent | integration | `go test ./internal/lifecycle -run TestSweep` | ❌ Wave 0 |
| ADMIN-08 | audit_log append-only: UPDATE/DELETE raise, INSERT ok | integration | `go test ./internal/auditlogs -run TestAppendOnly` | ❌ Wave 0 |
| D-38 | SYSTEM account cannot log in | integration | `go test ./internal/auth -run TestSystemNoLogin` | ❌ Wave 0 |
| D-39 | No audit row on zero-affected sweep | integration | `go test ./internal/lifecycle -run TestSweepNoOp` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** the slice's quick `go test ./internal/<slice> -run ... -count=1`.
- **Per wave merge:** full `go test ./...` with migrations applied (testcontainers).
- **Phase gate:** full suite green + CI `ci` job green before verify-work.

### Wave 0 Gaps
- [ ] `internal/users/import_test.go` — ADMIN-02/03 (validation + derivation + all-or-nothing).
- [ ] `internal/users/reset_test.go` — ADMIN-04.
- [ ] `internal/enrollments/import_test.go` — ADMIN-06 idempotency.
- [ ] `internal/lifecycle/sweep_test.go` — ADMIN-07/D-39.
- [ ] `internal/auditlogs/append_only_test.go` — D-35 (trigger).
- [ ] `internal/auth/system_test.go` — D-38 non-loginability.
- [ ] Shared testcontainers fixture that applies migrations `000001..000005` (extend Phase 1-2 harness; `seed_test.go` shows the existing pattern).

## Security Domain

> `security_enforcement` assumed enabled.

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | bcrypt cost=12 for derived passwords; SYSTEM non-loginable (sentinel hash); reset re-arms `must_change_password` + `password_changed_at` kill-switch. |
| V3 Session Management | yes | Reuse cookie JWT + `password_changed_at` invalidation (Phase 2) — reset must bump it. |
| V4 Access Control | yes | All `/admin/*` behind `RequireRole(admin)`; SYSTEM + soft-deleted excluded from listings. |
| V5 Input Validation | yes | Server-side CSV validation (Zod is UX only); `MaxBytesReader` size cap; validate IDs/roles/course existence before insert. |
| V6 Cryptography | yes | bcrypt only; never store/echo plaintext or generated passwords (D-26). UUIDs via `gen_random_uuid()`. |
| V7 Logging | yes | Append-only audit_log via DB triggers (D-35); every mutation audited (D-34 taxonomy). |

### Known Threat Patterns for Go/Gin + Postgres
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| SQL injection via CSV field / filter params | Tampering | sqlc parameterized queries only (no string-built SQL). |
| Audit tampering (cover tracks) | Repudiation | `BEFORE UPDATE/DELETE` triggers RAISE EXCEPTION (D-35). |
| Privilege escalation via SYSTEM login | Elevation of Privilege | Sentinel non-bcrypt hash + `is_system` excluded from auth/listings. |
| Memory exhaustion via huge CSV | Denial of Service | `http.MaxBytesReader` + `MaxMultipartMemory` (Pitfall 6). |
| Resurrecting soft-deleted course | Tampering | Reads filter `deleted_at IS NULL`; no un-delete endpoint this phase. |
| Plaintext password leakage | Information Disclosure | Never store/return generated passwords; only the bcrypt hash persists (D-26). |

## Sources

### Primary (HIGH confidence)
- Codebase (read this session): `backend/sqlc.yaml`, `db/migrations/000001-000003`, `db/queries/users.sql`, `internal/shared/db/models.go`, `internal/auth/{handler,service,repository,dto}.go`, `internal/shared/middleware/{auth,role}.go`, `cmd/api/main.go`, `frontend/src/{routes/router.tsx,components/AppLayout.tsx,lib/api.ts}`, `frontend/package.json`, `frontend/components.json` — grounds every reuse pattern.
- `.claude/CLAUDE.md` — locked stack, versions, Stack Patterns (CSV stdlib, bcrypt cost=12, sweep SQL, explicit audit INSERT). HIGH.
- `.planning/phases/03-.../03-CONTEXT.md` — D-24→D-43 locked decisions. HIGH (authoritative).

### Secondary (MEDIUM confidence)
- docs.sqlc.dev (insert howto, 1.31.1) — `WithTx`, `:copyfrom`/`:batch` for pgx/v5. [CITED]
- github.com/flyway/flyway#350, golang-migrate postgres TUTORIAL — `ALTER TYPE ADD VALUE cannot run inside a transaction block`. [VERIFIED: web search]

### Tertiary (LOW confidence)
- None load-bearing; all key facts cross-checked against code or official docs.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — fully locked in CLAUDE.md and present/building in repo.
- Architecture: HIGH — mirrors existing Phase 1-2 handler/service/repo + grounded SQL.
- Pitfalls: HIGH — derived from the locked decisions + verified Postgres/migrate behavior.
- Two resolved research items (enum caveat, scheduler): HIGH — verified via web search + Ponytail principle in CLAUDE.md.

**Research date:** 2026-06-20
**Valid until:** 2026-07-20 (stable stack; no fast-moving deps introduced)

Sources:
- [Flyway issue #350 — ALTER TYPE ADD inside transaction](https://github.com/flyway/flyway/issues/350)
- [golang-migrate postgres TUTORIAL](https://github.com/golang-migrate/migrate/blob/master/database/postgres/TUTORIAL.md)
- [sqlc Inserting rows (1.31.1)](https://docs.sqlc.dev/en/latest/howto/insert.html)
- [pgx/v5 package docs](https://pkg.go.dev/github.com/jackc/pgx/v5)
</content>
</invoke>
