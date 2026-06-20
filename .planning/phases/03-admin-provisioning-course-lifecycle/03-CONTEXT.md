# Phase 3: Admin Provisioning & Course Lifecycle - Context

**Gathered:** 2026-06-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Admin can provision an entire term — accounts, enrollment, courses — from CSV or UI, every admin mutation is append-only audit-logged, and stale courses auto-soft-delete. Delivers **ADMIN-01 → ADMIN-08**:

- Create student/lecturer accounts manually or by **whole-file, all-or-nothing CSV** with a per-row error report (no partial inserts).
- New accounts default to `username = ID`, `password = birthday DDMMYYYY`, `must_change_password = TRUE`; admin can reset any user back to that default.
- CRUD courses (with start/end dates) and assign students + lecturers to a course from CSV (idempotent).
- Every admin mutation writes an **append-only** audit row (actor, action, target, timestamp) that cannot be edited/deleted.
- Courses **auto soft-delete one month after their end date** with no manual action; each sweep is itself audit-logged.

This phase introduces the first real admin feature pages + the admin sidebar nav (deferred from Phase 2 by D-21). Builds directly on the Phase 2 auth/RBAC spine: `RequireRole(admin)`, the `{error:{code,message}}` envelope, cookie JWT, and the `password_changed_at` session-kill primitive (reused by ADMIN-04 reset). It does NOT build any course-scoped coursework features (assignments/quizzes/grades/announcements/requests — Phases 4–5); those tables are referenced only as "do not cascade" targets in D-40.

</domain>

<decisions>
## Implementation Decisions

Decision IDs continue the project sequence. Phase 2 CONTEXT ended at D-23; **D-24 → D-43 are new and owned by this phase.**

### Account provisioning & CSV (ADMIN-01/02/03/04)
- **D-24 — Separate CSV files/actions for students vs lecturers.** Two distinct imports (Import Students / Import Lecturers), each validated against its own schema. **Student CSV** required columns: `student_id, full_name, dob` (`DD/MM/YYYY`). **Lecturer CSV** required columns: `lecturer_id, full_name, dob` (`DD/MM/YYYY`). Additional columns are allowed and ignored. Account generation: `username = student_id|lecturer_id`; `password = dob` formatted as `DDMMYYYY`; `must_change_password = TRUE` (per D-07 onboarding pattern). Rationale: student & lecturer data come from different administrative sources with different schemas; separate flows simplify validation, give clearer errors, avoid mixed-role files. (Manual single-account create — ADMIN-01 — uses the same per-role fields.)
- **D-25 — Store `full_name`; no email for MVP.** `full_name` is needed across student lists, lecturer views, announcements, requests, grades, assignment management — IDs alone hurt usability. Email omitted: MVP sends no email, auth is username-based, reset is admin-driven, no workflow depends on email. `users` gains `full_name` (and `date_of_birth` per D-26).
- **D-26 — Store `date_of_birth (DATE)` for every account.** ADMIN-04 reset uses the stored DOB to auto-regenerate the default `DDMMYYYY` with no admin input (one-click, deterministic). Ties to D-01 (reset returns to default) and reuses Phase 2's `password_changed_at` kill-switch (D-13/D-16) to invalidate old sessions on reset. DOB is used **only** for provisioning + ADMIN-04 reset; the system never stores plaintext passwords and never re-displays generated passwords.
- **D-27 — All-or-nothing CSV validation.** Validate the entire file before creating anything. If ≥1 error: no inserts/updates, return **HTTP 422** with a structured `{ "errors": [ { "row", "field", "message" } ] }` list; FE renders the errors as a table. Rationale: partial imports create inconsistent states and hard recovery; all-or-nothing is deterministic, prevents duplicate-import scenarios, simplifies retry.

### Courses & enrollment (ADMIN-05/06)
- **D-28 — Course identity fields:** `code, name, term, start_date, end_date` (no `description` in MVP), plus `deleted_at`. `term` (e.g. `2026.1`) is a first-class academic concept, **not** derived from dates — multiple offerings across terms share the same `name`.
- **D-29 — Course deletion is soft-delete only.** Admin "delete" sets `deleted_at` and excludes the course from normal queries; no physical delete. Preserves referential integrity + academic history and aligns with the ADMIN-07 sweep. Accepted trade-off: soft-deleted rows stay in the DB indefinitely.
- **D-30 — Student enrollment import is per-course, additive, idempotent.** Admin selects a course, then uploads a CSV of `student_id` (one column). Already-enrolled rows are ignored (no duplicate, no error); imports **never remove** existing enrollments. Follows D-27 all-or-nothing: any nonexistent/invalid student ID rejects the entire file. Rationale: enrollment lists arrive after registration and may be re-imported (late registrations).
- **D-31 — Course membership model = two separate tables.** A course may have **one or more lecturers** and **zero or more students**. `student_enrollments(course_id, student_id)` and `course_lecturers(course_id, lecturer_id)` are modeled separately because student-enroll and lecturer-assign are different business processes. myIU is **not** the course-registration system — it manages membership after registration is determined elsewhere.
- **D-32 — Lecturer assignment mirrors D-30.** Admin selects a course, uploads a CSV of `lecturer_id` (one column). Additive + idempotent; never removes; all-or-nothing (D-27). Equivalent business operations behave consistently.

### Audit log (ADMIN-08)
- **D-33 — Bulk op = one audit row + `operation_id` + `affected_count`.** Each bulk admin action (IMPORT_STUDENTS, IMPORT_LECTURERS, ENROLL_IMPORT, LECTURER_IMPORT) writes a single audit row (e.g. `affected_count=253`, `operation_id=…`), **not** one row per entity. Per-entity detail, if finer traceability is ever needed, belongs in dedicated operation-detail tables linked by `operation_id`. Principle: audit records business actions; operational detail is stored separately.
- **D-34 — Audit payload: `actor, action, target_type, target_id, timestamp, metadata`; NO before/after diffs.** Action taxonomy includes `ACCOUNT_CREATE, PASSWORD_RESET, COURSE_CREATE, COURSE_UPDATE, COURSE_DELETE, ENROLL_IMPORT, LECTURER_IMPORT` (plus removal actions in D-43 and `COURSE_SWEEP` in D-39). The system can detect an update happened but cannot reconstruct previous field values from audit alone — accepted for MVP.
- **D-35 — Append-only enforced by DB triggers.** `BEFORE UPDATE` and `BEFORE DELETE` triggers on `audit_log` raise an exception; only INSERT is permitted. Trade-off accepted for MVP: no dedicated DB roles / privilege separation (the app DB user may still hold UPDATE/DELETE at the role level, but triggers block mutation). Future hardening (deferred): dedicated app DB roles + `REVOKE UPDATE/DELETE` + migration/runtime role separation as defense-in-depth.
- **D-36 — Admin audit-log viewer ships this phase.** A dedicated **read-only** admin page: pagination, filter by actor, filter by action, filter by date range, view metadata. Admin-only; no edit/delete (the viewer is presentation only — `audit_log` stays append-only per D-35). Rationale: audit data is low-value if admins can't inspect it; shipping it now also validates audit generation and supports troubleshooting.

### Course lifecycle sweep (ADMIN-07)
- **D-37 — In-process daily sweep + startup catch-up.** A Go-native scheduled job (e.g. `time.Ticker` or equivalent) inside the backend runs **once per day**, plus **one catch-up sweep at application startup** to process windows missed while offline. Logic: find courses whose `end_date` passed by ≥ 1 month, set `deleted_at`, ignore already-soft-deleted; idempotent. No Redis/queues/workers. Trade-off: if the app is offline, sweeps are delayed to the next startup/run (not time-critical). Future (multi-instance): dedicated scheduler / external cron / distributed locking.
- **D-38 — Dedicated SYSTEM account is the actor for automated actions.** Chosen over a NULL actor for explicit attribution + referential integrity. Seeded at app initialization; **cannot log in, cannot authenticate, cannot own a session** — exists solely for audit attribution. Used for `COURSE_SWEEP` and future automated jobs. System-generated audit rows are append-only like human rows (D-35). Trade-off: the `users` table holds one special-purpose non-human record. *(Implementation wrinkle flagged below: the SYSTEM account does not match the `user_role` enum — see Research items.)*
- **D-39 — Sweep audit granularity follows D-33.** A single audit row (`action=COURSE_SWEEP, actor=SYSTEM, affected_count=N`) **only when ≥ 1 course is affected**. No audit row when `affected_count = 0` (no business change). Trade-off: audit alone cannot prove the scheduler ran on a no-op day — process-execution visibility belongs to app logs/metrics/monitoring, not the audit log.
- **D-40 — Soft-delete does NOT cascade.** Only `courses.deleted_at` is set; related records stay unchanged (`student_enrollments`, `course_lecturers`, and future `assignments`, `submissions`, `quizzes`, `quiz_attempts`, `grades`, `announcements`, `requests`). Normal queries exclude soft-deleted courses, so related records become naturally hidden (they are accessed only through active courses). Trade-off: orphaned historical relationships remain. Future: an archival mechanism moves inactive courses + related records to long-term storage without cascading.

### Admin UI surface
- **D-41 — Phase 3 introduces the first full admin sidebar** (fulfils D-21, follows D-05: shadcn/ui, collapsible/hamburger, light+dark). Sections: **Dashboard**; **User Management** → Accounts; **Academic Management** → Courses, Student Enrollment, Lecturer Assignment; **System** → Audit Logs.
- **D-42 — Read-only course detail / roster page.** Each course has a detail page with tabs: **Overview**, **Students** (`student_id, full_name`), **Lecturers** (`lecturer_id, full_name`). Admin views all rosters. Rationale: bulk CSV imports need a verification surface (confirm imports succeeded, counts correct). Lecturer/student-facing roster visibility may come in later phases.
- **D-43 — Manual membership removal ships this phase, from the roster page (UI, not CSV).** Admin can remove an individual student from a course and unassign an individual lecturer. CSV imports stay additive/idempotent (D-30/D-32); removal is an explicit UI action. Each removal writes an audit row (`STUDENT_REMOVED_FROM_COURSE`, `LECTURER_UNASSIGNED_FROM_COURSE`). Accepted trade-off: slight scope increase — justified because enrollment management is incomplete without a correction path (fix import mistakes / registration corrections) and avoids requiring DB access or a future phase.

### Claude's Discretion (settled without a user question)
- **Backend feature-folder layout (per D-10).** New code organizes by business feature with the `handler.go / service.go / repository.go / model.go / dto.go` split and sqlc queries under `backend/db/queries/`. The exact folder grouping for this phase (e.g. `internal/admin/` vs `internal/users/` + `internal/courses/` + `internal/enrollments/` + `internal/auditlogs/`) is the planner's call; CLAUDE.md's architecture lists `courses/` and `auditlogs/` as feature folders. Audit writes are explicit `INSERT` SQL (and may use a Gin middleware on admin routes per CLAUDE.md's pattern).
- **Error envelope + role gate reuse.** All admin endpoints sit behind `middleware.RequireRole(db.UserRoleAdmin)` and return the established `{ "error": { "code", "message" } }` envelope; `422` for CSV validation failures (D-27).
- **Admin Dashboard content (D-41).** Exact widgets are discretionary — sensible default: simple counts (total students / lecturers / courses / recent audit events). Keep it lean (Ponytail).
- **Concrete action-code strings & target_type values, accounts-list search/pagination, and manual-create form layout** — planner picks specifics consistent with D-34's taxonomy and D-05.
- **Reset is single-user (ADMIN-04).** A per-user reset action from the Accounts list (no bulk reset); reuses D-26 stored DOB.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project-level (locked stack, constraints & architecture)
- `.claude/CLAUDE.md` — committed stack + versions and the admin-relevant **Stack Patterns**: `encoding/csv` stdlib for CSV; bcrypt **cost=12**; explicit `INSERT INTO audit_log (...)` in admin handlers or a Gin middleware on admin routes; one daily `gocron`/`time.Ticker` sweep `UPDATE courses SET deleted_at = now() WHERE deleted_at IS NULL AND end_date < now() - interval '1 month'`; sqlc v1.31.1 + pgx v5.7.x; golang-migrate v4.18 owns schema. Authoritative for all library choices.
- `.planning/PROJECT.md` — vision, constraints, Key Decisions table (incl. **D-01** admin-only reset to `DDMMYYYY`; **D-10** Feature-Oriented Monolith + handler/service/repository split).
- `.planning/REQUIREMENTS.md` §"Admin — Provisioning & Courses (ADMIN)" — ADMIN-01 → ADMIN-08 acceptance wording.
- `.planning/ROADMAP.md` §"Phase 3: Admin Provisioning & Course Lifecycle" — goal + 5 success criteria.

### Design (frontend)
- `.planning/DESIGN-SYSTEM.md` (D-05) — global UI ruleset: shadcn/ui only, light+dark themes, 6px radius, Lucide icons, Skeleton loaders, WCAG AA, **expandable sidebar** (now realized for admin per D-41).

### Prior phase context (patterns this phase reuses)
- `.planning/phases/02-auth-rbac-forced-first-login/02-CONTEXT.md` — D-13/D-16 `password_changed_at` session-kill (reused by ADMIN-04 reset), the `{error:{code,message}}` envelope, `RequireRole`, cookie JWT, the FE 401/403 interceptor + `ProtectedRoute`/`RoleGuard`/`AppLayout` shells, and D-21 (sidebar deferred → now built here).
- `.planning/phases/01-foundation-data-core/01-CONTEXT.md` — D-06 incremental per-phase migrations (Phase 3 appends `000004+`), D-07 bootstrap admin + bcrypt cost=12 + `must_change_password` onboarding pattern, D-08 Docker = Postgres-only.

### Existing code to read (Phase 1–2 output)
- `backend/db/migrations/000001_init_foundation.up.sql` — current `users` (`id, username, password_hash, role user_role, must_change_password, created_at, updated_at, deleted_at`) + `audit_log` (`id, actor_id→users, action, target, metadata JSONB, created_at`) + `users_username_active_uq` partial unique index. **Phase 3 adds migration(s) `000004+`** for: `users.full_name`, `users.date_of_birth`; `courses`; `student_enrollments`; `course_lecturers`; `audit_log` extensions (D-33/D-34); the append-only triggers (D-35); and the SYSTEM seed (D-38).
- `backend/db/migrations/000003_add_password_changed_at.*.sql` — the `password_changed_at` column reused by ADMIN-04.
- `backend/db/queries/users.sql` — existing sqlc queries (`GetUserByUsername`, `GetUserByID`, `UpdatePasswordAndStamp`) to extend.
- `backend/internal/shared/middleware/role.go` — `RequireRole(db.UserRoleAdmin)` gate for all admin routes.
- `backend/internal/shared/middleware/auth.go` — middleware that loads the user, enforces `password_changed_at` + `must_change_password`; note the SYSTEM account (D-38) must never pass this.
- `backend/internal/auth/handler.go` / `service.go` / `repository.go` — the `errorEnvelope(...)` helper + handler/service/repository feature pattern to mirror; `RegisterRoutes(r, pool, cfg)` wiring style.
- `backend/cmd/api/main.go` — entrypoint where admin routes register and where the in-process sweep scheduler (D-37) + SYSTEM seed (D-38) are wired.
- `frontend/src/routes/router.tsx`, `frontend/src/components/AppLayout.tsx`, `frontend/src/stores/auth.ts`, `frontend/src/lib/api.ts` — the `/admin/*` route tree, header/logout shell to extend with a sidebar (D-41), the axios client (cookies + 401/403 interceptor), and the auth store.

No external ADRs beyond the above — requirements are fully captured in the decisions here + the locked stack in CLAUDE.md.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`users` + `audit_log` tables already exist** (Phase 1). Phase 3 extends `users` (`full_name`, `date_of_birth`) and `audit_log` (operation_id / target_type / target_id / affected_count per D-33/D-34, or via existing `metadata` JSONB — planner's call), and adds the append-only triggers (D-35).
- **Phase 2 auth spine is fully reusable:** `RequireRole(admin)`, cookie JWT, `errorEnvelope`, the `password_changed_at` kill-switch (load-bearing for ADMIN-04), and the FE `/admin/*` route tree + interceptor.
- **Frontend shell exists** (`AppLayout` header + logout, `ProtectedRoute`, `RoleGuard`, shadcn `card/button/form/input/label/sonner`, TanStack Query client, axios `withCredentials`). Phase 3 adds the admin sidebar + feature pages inside this shell.

### Established Patterns
- **Feature-Oriented Monolith (D-10):** handler = HTTP only, service = business + authz, repository = SQL only; sqlc queries in `backend/db/queries/`.
- **Incremental migrations (D-06):** append `000004+`; CI runs migrations before tests.
- **CSV via stdlib `encoding/csv`; bcrypt cost=12; explicit audit INSERTs** — all pre-locked in CLAUDE.md.

### Integration Points
- The append-only `audit_log` + action taxonomy established here is the audit substrate Phases 4–5 also write to (NOTIF/grade/announcement mutations).
- `courses` / `student_enrollments` / `course_lecturers` are the tables every Phase 4–5 feature reads (assignments, quizzes, grades, announcements, requests derive scope/recipients from enrollments).
- The soft-delete-by-`deleted_at` + "reads filter active courses" discipline (D-29/D-40) is the pattern all later course-scoped reads must follow.
- The SYSTEM actor (D-38) is the attribution mechanism for every future automated job.

</code_context>

<specifics>
## Specific Ideas

- Each decision D-24→D-43 was authored by the user as a full decision record (title + decision + rationale + relationships + accepted trade-off + design principle) — preserve that intent verbatim; the summaries above are faithful condensations.
- **CSV `dob` is `DD/MM/YYYY` (slashes) but the derived password is `DDMMYYYY` (no slashes)** — a deliberate distinction (D-24).
- **SYSTEM account over NULL actor** (D-38) was an explicit "every action has a traceable actor" call by the user — keep it; it generalizes to all future automated jobs.
- **Removal is UI-only, never via CSV** (D-43): CSV stays a pure additive/idempotent channel; destructive corrections require explicit per-row UI intent.
- The recurring user design principle across this phase: **explicit business workflows over generic mechanisms; preserve history over destructive cleanup; enforce invariants close to the data.**

</specifics>

<deferred>
## Deferred Ideas

- **Course retention / archival policy** — archive courses whose `end_date` exceeds a retention threshold (~5 years) and move inactive courses + related records to long-term storage **without** cascading soft-deletes. Explicitly not MVP (D-29, D-40).
- **Audit-history mechanism** — before/after snapshots / field-level diffs for full state reconstruction. Not MVP (D-34).
- **Audit-log security hardening (defense-in-depth)** — dedicated app DB roles + `REVOKE UPDATE/DELETE` on `audit_log` + migration/runtime role separation, layered on top of the D-35 triggers. Future hardening phase.
- **Multi-instance sweep hardening** — dedicated scheduler process / external cron / distributed locking to prevent duplicate execution across instances. Deferred until a multi-instance deployment (D-37).
- **Lecturer/student-facing roster visibility** — D-42's roster page is admin-only this phase; role-scoped roster views may come in later phases.

None of these are scope creep into Phase 3 — discussion stayed within the admin-provisioning / course-lifecycle boundary.

## Research items for gsd-phase-researcher
- **`audit_log` schema shape for D-33/D-34.** The table currently has only `target TEXT`. Decide whether to add columns `operation_id`, `target_type`, `target_id`, `affected_count` (migration `000004`) or pack them into the existing `metadata` JSONB. Also decide whether dedicated operation-detail tables (linked by `operation_id`, D-33) are needed for MVP or whether `metadata` suffices.
- **SYSTEM account vs `user_role` enum (D-38).** The `user_role` enum is `('student','lecturer','admin')` — the SYSTEM account fits none. Decide: extend the enum with a `'system'` value (migration), OR add an `is_system` boolean flag. Either way ensure the SYSTEM account is non-loginable (invalid/empty `password_hash`, never passes `auth.go`), excluded from every user listing, and handled w.r.t. the `users_username_active_uq` index.
- **CSV streaming/limits.** Confirm reasonable upload-size handling for admin CSVs (stdlib `encoding/csv`; reject oversized files) — no Cloudinary involved this phase.
- **Sweep scheduler choice.** `time.Ticker` vs `gocron/v2` for the single daily job + startup catch-up (D-37); CLAUDE.md leans either — pick the leaner fit and ensure idempotency.

</deferred>

---

*Phase: 3-Admin Provisioning & Course Lifecycle*
*Context gathered: 2026-06-20*
