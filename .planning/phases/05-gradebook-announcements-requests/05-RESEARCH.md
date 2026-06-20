# Phase 5: Gradebook, Announcements & Requests - Research

**Researched:** 2026-06-20
**Domain:** Hierarchical weighted gradebook computation, fan-out notifications, directed requests — all on the committed Go/Gin/sqlc/pgx + Postgres stack
**Confidence:** HIGH (the entire substrate is confirmed against committed code; the only open items are deliberate design choices, not unknowns)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions (D-56 → D-67 — research THESE, do not re-litigate)

- **D-56 — Hierarchical weighted-component gradebook (not flat columns).** Overall = `Inclass + Midterm + Final = 100%`. Composite components contain sub-components whose weights sum to 100% of the parent. Each **leaf** has a source type: **AUTO** (Quiz Average, Assignment Average) or **MANUAL** (Project, Lab, Participation, Bonus, Midterm, Final). Principle: *the gradebook computes grades, it does not merely store them.* Must be a foundation for future best-N / drop-lowest without redesign.
- **D-57 — Single 0–100 scale; normalize first, aggregate second.** Quiz Average = `avg(student_score / quiz_max × 100)`; Assignment Average = `avg(student_score / assignment_max × 100)`. MANUAL entered directly on 0–100 (`0 ≤ score ≤ 100`). **Schema consequence: assignments SHALL define a max score** — Phase 4 `assignments` has none.
- **D-58 — AUTO includes all eligible items; missing = zero.** A student who did not attempt/submit an *eligible* item gets 0 for it and it stays in the aggregation set. Principle: *missing assessment is a grade of zero, not absence of data.*
- **D-64 — AUTO eligibility = when the score is FINALIZED (not window timing).** Quiz eligible at `close_at` (auto-graded). Assignment eligible **when the lecturer finalizes grading** — independent of deadline / accept-late / late-threshold. **Eligibility ≠ publication.**
- **D-59 — Component-level publication; grades hidden until published.** Publish each top-level component independently. After publish: students see that component's score + receive one grade notification each. **Overall visible only once all top-level components are published.** Editing a published score does NOT auto-notify unless the lecturer **republishes**.
- **D-65 — Grade scheme is immutable once created.** No add/remove components, no weight/structure changes after creation. Scores may be entered/published/updated-while-unpublished. Principle: *academic policy is immutable.*
- **D-66 — Published components are snapshots; computation stays live.** Student sees the value **frozen at publish**; lecturer always sees the live recomputed value. Republish replaces the snapshot + generates a new notification. **Persist the snapshot value — do not compute-on-read for the student view.**
- **D-67 — MANUAL grade CSV: one component per file, `student_id,score`.** Phase 3 CSV discipline: whole-file validation, all-or-nothing, row-level error report, HTTP 422 on failure, no partial imports.
- **D-60 — Announcements are first-class entities + fan-out delivery.** `announcements(id, course_id, author_id, title, body, audience_type, created_at)`; `audience_type ∈ {ALL_STUDENTS, SPECIFIC_STUDENTS}`; SPECIFIC targets in a join table. Creating one fans out notification rows to all targeted recipients. Two surfaces: a per-course Announcements page + the bell.
- **D-61 — Announcements are immutable after sending.** No edit/delete/recipient-change. **Drop `updated_at`.** Lifecycle = CREATED only.
- **D-62 — Requests are directed to one student-chosen lecturer.** Visible only to the requesting student and the targeted lecturer. No main/assistant distinction. Principle: *requests have a clear owner; directed, not broadcast.*
- **D-63 — Reply = required Decision + optional Note; one round-trip.** Type ∈ leave-early / absence / custom; title + plain-text body; starts PENDING; lecturer replies APPROVED or DENIED (required) + optional note; then closed permanently. Reply auto-generates a notification in the same transaction.

### Claude's Discretion (settled by substrate — confirm)
- Request creation notifies the targeted lecturer (symmetric to reply→student).
- Same-transaction notification writes (NOTIF-02) for grade-publish, republish, request-create, request-reply.
- Lecturer actions are NOT audit-logged (audit_log stays admin-only, ADMIN-08).
- Feature-folder layout (D-10) + migration `000008` (D-06).
- Recipient snapshot at send time (later enrollment changes don't retroactively alter delivered notifications).

### Deferred Ideas (OUT OF SCOPE — ignore)
- Per-institution configurable grading scale (0–10 / GPA). Fixed 0–100.
- Advanced policies: best-N quizzes, drop-lowest, weighted quiz groups.
- Grade export (CSV/PDF transcripts) — tracked as GRADE-V2-01.
- Request conversation threads / reopen / multi-reply.
- Announcement edit/delete / scheduled send / read receipts beyond the notification marker.
- Shared lecturer request inbox / request reassignment.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| GRADE-01 | Configure Inclass+Midterm+Final summing to 100% | Hierarchical schema (§Gradebook Schema); server-side sum-to-100 validation at each level |
| GRADE-02 | Configure Inclass sub-components summing to 100% of Inclass | Self-referencing `parent_id` 2-level tree; recursive sibling-sum validation |
| GRADE-03 | Enter/upload Midterm & Final manually | MANUAL leaf `source_type='MANUAL'`; direct entry + CSV import (D-67 pattern, §CSV) |
| GRADE-04 | Compute weighted overall grade | Normalize-then-aggregate algorithm (§Computation); live for lecturer, snapshot for student |
| GRADE-05 | Student views grades; availability auto-notified | Component publication + per-recipient grade notification in-transaction (§Publication) |
| ANNC-01 | Announce to all enrolled students | `audience_type=ALL_STUDENTS`; `ListCourseStudents` fan-out |
| ANNC-02 | Announce to specific students | `audience_type=SPECIFIC_STUDENTS` + `announcement_recipients` join table |
| ANNC-03 | Student receives announcements (persisted, no email) | Fan-out notification rows + per-course Announcements page (§Announcements) |
| REQ-01 | Student sends typed request to chosen course lecturer | `requests` table + `ListCourseLecturers` for selection (§Requests) |
| REQ-02 | Lecturer replies yes/no | `status` PENDING→APPROVED/DENIED + optional note, single round-trip |
| REQ-03 | Reply auto-delivered to student | In-transaction notification on reply (NOTIF-02 pattern) |
</phase_requirements>

## Summary

This phase is almost entirely an **application of confirmed existing patterns**, not new technology. Every dependency the CONTEXT.md flagged was verified against committed code: the same-transaction notification write (`assignments/service.go:202`), the notification table shape (`migration 000006`), the CSV import discipline (`enrollments/{csv,service,handler}.go`), the course-membership reads (`courses.sql` → `ListCourseStudents` / `ListCourseLecturers`), the quiz MAX-score read (`quiz_attempts.sql` → `GetMaxScore`, D-50), and the integration-test harness (`DATABASE_URL` → `pgxpool.New` against real Postgres). **No new external package is introduced** — the entire stack is already committed.

The genuinely-new design work is four-fold: (1) a **2-level self-referencing component tree** with server-side sum-to-100 validation at each level; (2) a **normalize-then-aggregate computation** that reads Phase-4 coursework, treats missing-but-eligible items as 0, and must distinguish *eligible* (counts in computation) from *published* (student-visible); (3) a **published-snapshot persistence model** where the student view is frozen and the lecturer view recomputes live; and (4) two **schema touches to the Phase 4 `assignments` table** — adding `max_score` (D-57) and an explicit grading-finalized marker (D-64).

**Primary recommendation:** Build three feature folders (`internal/grades/`, `internal/announcements/`, `internal/requests/`) following the handler/service/repository split, with one migration `000008` that (a) creates the new tables and (b) alters `assignments` to add `max_score NUMERIC` and `grading_finalized_at TIMESTAMPTZ NULL`. Use a single `grade_components` table with `parent_id` self-reference, a separate `grade_scores` table for leaf MANUAL/computed values, and a `published_at` + snapshot value carried on top-level components. Reuse the `pool.Begin → q.WithTx(tx) → InsertNotification` pattern verbatim for all four notify-on-mutation flows.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Grade-scheme definition + sum-to-100 validation | API / Backend (service) | — | Academic policy; ownership-from-JWT; never trust client weights |
| Weighted grade computation (normalize→aggregate) | API / Backend (service) | Database (reads) | Business rule; reads quiz_attempts/submissions; must be deterministic & testable |
| Published snapshot persistence | Database / Storage | API (service writes it) | Student-facing value is frozen; persisted, not computed-on-read |
| Live recompute for lecturer | API / Backend (service) | Database (reads) | Recomputed on every request from current underlying scores |
| MANUAL grade CSV import | API / Backend (service) | Frontend (upload form) | Whole-file validation + all-or-nothing commit is server-side; client is UX only |
| Announcement fan-out | API / Backend (service) | Database (notification rows) | Recipient snapshot resolved server-side at send time |
| Request routing + reply | API / Backend (service) | — | Directed-to-one-lecturer visibility enforced in service authz |
| Notification delivery (all flows) | Database (per-recipient rows) | API (in-tx insert) | Reuse Phase 4 primitive verbatim; no new UX |
| Grade-view / announcements-page / request-inbox UI | Frontend (SPA) | API (REST) | shadcn/ui under D-20 role route trees; data via TanStack Query |

## Standard Stack

No new libraries. This phase uses only the committed stack, all already present in `backend/go.mod` / `frontend/package.json`.

### Core (already committed — confirmed in use)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| sqlc | v1.31.1 | SQL → type-safe Go | `sqlc.yaml` confirmed: `engine: postgresql`, `sql_package: pgx/v5`, schema=`db/migrations`, queries=`db/queries` [VERIFIED: backend/sqlc.yaml] |
| pgx | v5.7.x | Postgres driver | `pgxpool.Pool` + `q.WithTx(tx)` used everywhere [VERIFIED: backend/internal/assignments/service.go] |
| golang-migrate | v4.18.x | Migrations | Sequential `00000N_*.up/.down.sql`; next is `000008` [VERIFIED: backend/db/migrations/] |
| Gin | v1.11.0 | HTTP | `r.Group("/api/lecturer")` + `middleware.RequireRole(...)` [VERIFIED: backend/internal/assignments/handler.go] |
| golang-jwt | v5 | Auth | ownership-from-JWT via `c.GetInt64("user_id")` [VERIFIED: backend/internal/enrollments/handler.go] |
| pgtype | (pgx/v5) | Nullable/Numeric scan | `pgtype.Numeric`, `pgtype.Int8`, `pgtype.Text`, `pgtype.Timestamptz` [VERIFIED: assignments/service.go] |

### Supporting (frontend — already committed)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| shadcn/ui | current | Components | Grade table, announcement form, request inbox — no hand-rolled components |
| TanStack Query | v5 | Server state | Fetch grades/announcements/requests; invalidate on mutation |
| React Hook Form + Zod | v7 / v4 | Forms | Scheme builder, announcement composer, request form, CSV upload |
| react-router | v7 | Role routes | Student grade-view, course Announcements page, lecturer request-inbox under D-20 trees |

### Testing (already committed — confirmed pattern)
| Tool | Purpose | Notes |
|------|---------|-------|
| Go stdlib `testing` + testify | Unit + integration | `assert`/`require` used in all `*_test.go` [VERIFIED: enrollments/import_test.go] |
| Real Postgres via `DATABASE_URL` | Integration | Tests `t.Skip` if `DATABASE_URL` unset; `pgxpool.New(ctx, dbURL)` then inline SQL setup/teardown. **No testcontainers in the Go layer** — CI provisions Postgres + runs migrations, then `go test` connects. [VERIFIED: backend/internal/enrollments/import_test.go] |

**Installation:** None required — all packages present. The only `go.mod` impact is zero (no new imports beyond stdlib + existing pgx/gin).

## Package Legitimacy Audit

> This phase introduces **no new external packages**. All libraries are already committed and in active use (verified in `backend/go.mod` usage across `internal/`). No registry verification needed.

| Package | Registry | Verdict | Disposition |
|---------|----------|---------|-------------|
| (none) | — | — | No new dependencies introduced |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
                    ┌─────────────────────────── FRONTEND (SPA) ───────────────────────────┐
  Lecturer ───────► │ Scheme Builder │ Grade Entry │ Announcement Composer │ Request Inbox  │
  Student  ───────► │ Grade View     │ Announcements Page │ Request Form │ Bell (Phase 4)   │
                    └──────────────────────────────┬───────────────────────────────────────┘
                                                    │ REST + JWT cookie (axios + TanStack Query)
                    ┌───────────────────────────────▼─────────────────────── API (Gin) ─────┐
                    │  /api/lecturer/...   /api/student/...   (RequireRole + AssertCourseMember)│
                    │  ┌─────────────┐   ┌──────────────────┐   ┌──────────────┐              │
                    │  │ grades/     │   │ announcements/   │   │ requests/    │              │
                    │  │ handler     │   │ handler          │   │ handler      │              │
                    │  │ service ◄───┼───┤ service          │   │ service      │              │
                    │  │ repository  │   │ repository       │   │ repository   │              │
                    │  └──────┬──────┘   └────────┬─────────┘   └──────┬───────┘              │
                    │         │ reads             │ ListCourseStudents │ ListCourseLecturers  │
                    │         ▼                   ▼                    ▼                      │
                    │   ┌──────────────────────────────────────────────────────────┐         │
                    │   │ SAME-TX:  pool.Begin → q.WithTx(tx) → mutation             │         │
                    │   │            → InsertNotification(per recipient) → Commit    │ (NOTIF-02)│
                    │   └──────────────────────────────┬───────────────────────────┘         │
                    └──────────────────────────────────┼──────────────────────────────────────┘
                                                        ▼
   ┌──────────────────────────────────── PostgreSQL (Docker) ───────────────────────────────┐
   │ grade_components (parent_id tree) │ grade_scores │ grade_publications(snapshot)          │
   │ announcements │ announcement_recipients │ requests                                        │
   │ ── reads ──►  quiz_attempts(score,status) │ submissions(score,graded_at) │ quizzes(...)  │
   │ ── ALTER ──►  assignments + max_score + grading_finalized_at                             │
   │ ── writes ─►  notifications (per-recipient rows, Phase 4)                                │
   └────────────────────────────────────────────────────────────────────────────────────────┘

Primary flow (grade publish): Lecturer POSTs publish(top-level component)
  → service recomputes live value for every enrolled student (normalize→aggregate, missing=0)
  → in ONE tx: write snapshot rows + InsertNotification per student → Commit
  → Student GET grades → reads frozen snapshot (not live recompute).
```

### Recommended Project Structure
```
backend/
  internal/
    grades/          # handler.go service.go repository.go model.go dto.go
    announcements/   # handler.go service.go repository.go model.go dto.go
    requests/        # handler.go service.go repository.go model.go dto.go
  db/
    migrations/000008_grades_announcements_requests.up.sql   (+ .down.sql)
    queries/grades.sql  announcements.sql  requests.sql
                        (+ ALTER touch reflected in assignments.sql new queries)
frontend/src/
  pages/  (Grades.tsx, CourseAnnouncements.tsx, Requests.tsx, RequestInbox.tsx)
  under the existing D-20 role route trees in routes/router.tsx
```

### Pattern 1: Hierarchical weighted-component schema (D-56/D-57/D-58) — RECOMMENDED

**What:** A single self-referencing `grade_components` table modelling the 2-level tree. Leanest shape that supports the tree without over-engineering deferred best-N/drop-lowest.

```sql
-- 000008 (recommended shape)
CREATE TABLE grade_schemes (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    course_id  BIGINT NOT NULL REFERENCES courses(id),
    created_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (course_id)                       -- one immutable scheme per course (D-65)
);

CREATE TABLE grade_components (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    scheme_id   BIGINT NOT NULL REFERENCES grade_schemes(id),
    parent_id   BIGINT NULL REFERENCES grade_components(id),   -- NULL = top-level
    name        TEXT NOT NULL,                                 -- 'Inclass','Midterm','Final','Quizzes',...
    weight      NUMERIC NOT NULL CHECK (weight > 0 AND weight <= 100),
    source_type TEXT NULL CHECK (source_type IN ('AUTO','MANUAL')),  -- NULL for composite (non-leaf)
    auto_kind   TEXT NULL CHECK (auto_kind IN ('QUIZ_AVERAGE','ASSIGNMENT_AVERAGE')), -- only when AUTO
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- Leaf = source_type IS NOT NULL. Composite = source_type IS NULL (its children are the leaves).
```

**Why this shape (Ponytail):** One table, one self-reference, depth enforced in the service (max 2 levels), not in SQL. A depth-typed two-table design (top_components + sub_components) duplicates columns and queries for no real gain at 2 fixed levels — and `parent_id` is the design that *generalizes* to future weighted groups (the D-56 requirement) without a redesign. **Sum-to-100 is validated in the service at scheme-creation, not via a DB trigger** — triggers fight the all-at-once immutable-create model and are harder to test.

**When to use:** This is the recommended default. Do not add a `depth` column or a closure table — YAGNI for 2 fixed levels.

### Pattern 2: Score storage + published snapshot (D-59/D-66) — RECOMMENDED

```sql
-- MANUAL leaf scores (and any persisted per-student leaf value); AUTO leaves are computed live, not stored here.
CREATE TABLE grade_scores (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    component_id BIGINT NOT NULL REFERENCES grade_components(id),
    student_id   BIGINT NOT NULL REFERENCES users(id),
    score        NUMERIC NOT NULL CHECK (score >= 0 AND score <= 100),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (component_id, student_id)         -- upsert per (component, student)
);

-- Published snapshot per TOP-LEVEL component per student (D-66: frozen student-facing value).
CREATE TABLE grade_publications (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    component_id BIGINT NOT NULL REFERENCES grade_components(id),  -- must be a top-level component
    student_id   BIGINT NOT NULL REFERENCES users(id),
    value        NUMERIC NOT NULL,            -- frozen normalized 0-100 value at publish time
    published_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (component_id, student_id)         -- republish = UPDATE value + published_at (one row per student)
);
```

**What:** Lecturer view = recompute live from `grade_scores` + Phase-4 coursework. Student view = read `grade_publications.value` (frozen). Republish = `UPDATE grade_publications SET value=$new, published_at=now()` + new notification, all in one tx. **Overall is visible to the student only when a `grade_publications` row exists for every top-level component** — compute the overall on the *snapshot* values, not live.

**Why:** D-66 explicitly says "persist the published snapshot value, don't compute-on-read for the student view." A snapshot table keyed `(component_id, student_id)` is the leanest persistence; the per-student grain is required because each student's value differs.

### Pattern 3: Same-transaction notification write (NOTIF-02) — REUSE VERBATIM

**Source:** `backend/internal/assignments/service.go:202-253` [VERIFIED].

```go
tx, err := s.pool.Begin(ctx)
if err != nil { return err }
defer func() { _ = tx.Rollback(ctx) }()
qtx := s.q.WithTx(tx)

// ...the mutation (upsert grade snapshot / insert announcement / insert request / update request)...

_, err = qtx.InsertNotification(ctx, db.InsertNotificationParams{
    RecipientID:  studentID,
    Type:         "GRADE_PUBLISHED",                         // planner picks the enum string
    Title:        "Grades available",
    Body:         fmt.Sprintf("Your %s grade is available.", componentName),
    ResourceType: pgtype.Text{String: "course", Valid: true},
    ResourceID:   pgtype.Int8{Int64: courseID, Valid: true},
    Link:         pgtype.Text{String: fmt.Sprintf("/courses/%d/grades", courseID), Valid: true},
})
if err != nil { return err }
return tx.Commit(ctx)
```

**Fan-out variant (publish / announcement):** loop the recipient list and call `qtx.InsertNotification` once per recipient *inside the same tx* (mirrors `enrollments/service.go` which loops parsed rows inside one tx). Either all snapshots + all notifications commit, or none do.

**`InsertNotificationParams` fields (confirmed):** `RecipientID, Type, Title, Body, ResourceType, ResourceID, Link` — `:one` returning the row [VERIFIED: db/queries/notifications.sql].

### Pattern 4: CSV import discipline (D-67) — REUSE the Phase 3 shape

**Source:** `enrollments/{csv.go, service.go, handler.go}` [VERIFIED]. The exact reusable shape:
1. `ParseCSV` reads header, finds the required column, then collects **every** malformed/empty/duplicate row into `[]RowError` (does not abort on first error).
2. Service validates existence (student in course) and appends any not-found rows to `RowError`.
3. **If `len(rowErrs) > 0` → return `ErrValidation` with the full list → handler emits `HTTP 422` with `gin.H{"errors": rowErrs}`.** No tx is opened.
4. Only when zero errors: `pool.Begin` → loop upserts → `Commit` (all-or-nothing).

**For grade CSV:** header `student_id,score`; per-row validate `student_id` exists-and-enrolled AND `0 ≤ score ≤ 100` AND numeric-parseable; collect all errors; target exactly one MANUAL component (component_id in the URL/body, not in the CSV — D-67 "structure must not leak into the CSV format"). Upsert into `grade_scores`. Reuse `http.MaxBytesReader` + extension/content-type check from `handleImport`.

### Anti-Patterns to Avoid
- **DB trigger for sum-to-100:** harder to test against real Postgres than a service check; the scheme is created all-at-once and immutable, so validate once in the service. Do it in Go.
- **Computing the student-facing value on read:** violates D-66 — student value must be the frozen snapshot. Compute live only for the lecturer.
- **Storing AUTO leaf values in `grade_scores`:** AUTO is derived from coursework; persisting it invites staleness. Compute AUTO live; only its *publication* is snapshotted.
- **Conflating "published coursework" (D-58 eligibility) with "published grade component" (D-59):** name the columns distinctly — `grading_finalized_at` (assignment eligibility) vs `grade_publications.published_at` (student release). Do not call either "published" in a way that overlaps.
- **`updated_at` on announcements / requests-as-threads:** D-61 drops `updated_at`; requests are single round-trip (D-63). No mutable-history columns.
- **Trusting client-supplied weights or student IDs:** validate sum-to-100 server-side; derive ownership from JWT via `AssertCourseMember` (never trust client course/student IDs).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Notification delivery | New notify table/service/UX | Phase 4 `notifications` + `InsertNotification` | Per-recipient rows + bell + list page already shipped (D-53/D-54) |
| Atomic mutation+notify | Manual rollback bookkeeping | `pool.Begin → q.WithTx(tx) → InsertNotification` | Canonical at assignments/service.go:202 |
| CSV validation/commit | New parser + partial-import logic | `enrollments` ParseCSV + all-or-nothing tx + 422 | Whole-file, row-error, no-partial discipline proven |
| Course rosters / lecturer list | New membership queries | `ListCourseStudents` / `ListCourseLecturers` | Already in courses.sql, filter deleted users |
| Quiz official score | Re-derive from attempts | `GetMaxScore` (MAX over SUBMITTED/AUTO_SUBMITTED) | D-50 already implemented |
| Ownership / role gating | Per-handler ad-hoc checks | `RequireRole` + `authz.AssertCourseMember` | Confirmed in middleware + authz pkg |
| Error envelope | New JSON shape | `errorEnvelope(code,message)` → `{error:{code,message}}` | Project-wide convention |
| Numeric handling | float math in SQL | `pgtype.Numeric` + `num.Scan(fmt.Sprintf("%f",x))` | Pattern at assignments/service.go:220 |

**Key insight:** Phase 5 is ~80% wiring of confirmed primitives. The only net-new *logic* is the grade computation (normalize→aggregate, missing=0, eligibility-on-finalize) and the tree/snapshot schema. Everything else is reuse.

## Runtime State Inventory

> This is a greenfield feature phase (three new folders + one additive migration). The only "existing runtime state" touched is the Phase 4 `assignments` table.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | `assignments` rows exist (Phase 4) but have **no `max_score`** | Migration `000008` adds `max_score NUMERIC`; **backfill** existing rows (set a sensible default e.g. 100, or NOT NULL DEFAULT 100) so Assignment Average can normalize. AUTO Assignment Average must skip assignments with NULL/0 max_score to avoid divide-by-zero. |
| Stored data | `assignments` rows have **no grading-finalized marker** | Migration `000008` adds `grading_finalized_at TIMESTAMPTZ NULL`; existing rows = NULL (not finalized → not eligible). Add a lecturer "finalize grading" action + query. |
| Live service config | None | None — no external service config embeds Phase-5 strings. |
| OS-registered state | None | None — no scheduled jobs or OS registrations added (publication is request-driven, not cron). |
| Secrets/env vars | None | None — reuses existing `DATABASE_URL`, JWT secret, Cloudinary creds. |
| Build artifacts | sqlc-generated `internal/shared/db/*` | Re-run `sqlc generate` after adding `db/queries/{grades,announcements,requests}.sql` + the assignment ALTER queries. CI/`check.sh` must regenerate or the build drifts. |

**Phase-4 table touch (the one real cross-phase edit):**
- `ALTER TABLE assignments ADD COLUMN max_score NUMERIC NOT NULL DEFAULT 100;` (D-57). Update `CreateAssignmentRequest` DTO + handler + `CreateAssignment` query + `AssignmentResponse` to carry `max_score`. **Run `impact({target:"CreateAssignment", direction:"upstream"})` before editing** (CLAUDE.md mandate) — the create flow has FE form + handler + service + query + tests.
- `ALTER TABLE assignments ADD COLUMN grading_finalized_at TIMESTAMPTZ NULL;` (D-64). Add a `FinalizeAssignmentGrading` lecturer action (sets the column) + query; eligibility = `grading_finalized_at IS NOT NULL`.

## Common Pitfalls

### Pitfall 1: Nullable quiz columns break AUTO normalization
**What goes wrong:** `quizzes.max_grade` and `quizzes.close_at` are **nullable** (declared `NUMERIC`/`TIMESTAMPTZ` with no NOT NULL) [VERIFIED: migration 000006]. Quiz Average divides by `max_grade` and gates eligibility on `close_at` — a NULL or 0 `max_grade` is a divide-by-zero / NaN; a NULL `close_at` means "never eligible".
**Why it happens:** Phase 4 didn't constrain these because quizzes weren't yet feeding a gradebook.
**How to avoid:** In the Quiz Average query, filter `WHERE q.close_at IS NOT NULL AND q.close_at <= now() AND q.max_grade IS NOT NULL AND q.max_grade > 0`. Document that a quiz with null/zero max_grade is simply not eligible (not an error). Same defensive guard for `assignments.max_score`.
**Warning signs:** NaN or Inf in computed averages; a student's overall jumping when a misconfigured quiz closes.

### Pitfall 2: Missing=0 only AFTER eligibility (D-58 vs D-64)
**What goes wrong:** Treating a not-yet-finalized assignment as 0 tanks every student's average before grading is done.
**Why it happens:** D-58 ("missing=0") and D-64 ("eligible only when finalized") interact — missing=0 applies *only to the eligible set*.
**How to avoid:** Two-step: (1) build the eligible-item set (quizzes past `close_at`; assignments with `grading_finalized_at IS NOT NULL`); (2) within that set, a student with no score = 0. A non-eligible item is excluded entirely (not 0). Test both branches explicitly.
**Warning signs:** Averages that drop the moment an assignment is *created* rather than *finalized*.

### Pitfall 3: Computing overall on live values for the student
**What goes wrong:** Student's overall changes after publish when an underlying AUTO score moves.
**Why it happens:** Forgetting D-66 — student overall must aggregate the **snapshots**, not live components.
**How to avoid:** Student overall = aggregate of `grade_publications.value` across top-level components; show it only when all top-level components have a publication row. Lecturer overall = live recompute. Two different code paths.
**Warning signs:** A student's grade silently changing without a republish notification.

### Pitfall 4: Numeric precision / pgtype.Numeric round-trips
**What goes wrong:** Direct `float64` math in SQL or sloppy Numeric scans produce drift (e.g. 89.9999999).
**Why it happens:** Postgres `NUMERIC` ↔ Go `float64` needs explicit handling.
**How to avoid:** Reuse the confirmed pattern: `var num pgtype.Numeric; num.Scan(fmt.Sprintf("%f", score))` for writes; for computed aggregates do the arithmetic in Go on `float64`, round to 2 decimals for display, and store the snapshot as NUMERIC. Clamp `0 ≤ score ≤ 100` (the assignments grade path already clamps).
**Warning signs:** Off-by-epsilon test failures; weights summing to 99.99999.

### Pitfall 5: Sum-to-100 must be validated at EACH level
**What goes wrong:** Top-level sums to 100 but Inclass sub-components sum to 95.
**Why it happens:** Validating only the root.
**How to avoid:** At scheme creation, group components by `parent_id` (including the NULL/root group) and assert each group's `weight` sums to exactly 100 (with a small epsilon tolerance). Reject otherwise with a 422-style field error before any insert.
**Warning signs:** Overall grades that don't reach 100 even with perfect scores.

### Pitfall 6: D-65 typo-recovery edge (flagged in CONTEXT specifics)
**What goes wrong:** A lecturer fat-fingers a weight and the scheme is now permanently wrong.
**Why it happens:** D-65 makes the scheme immutable once created.
**How to avoid (recommendation):** Allow **delete-and-recreate of the entire scheme only while no `grade_scores` and no `grade_publications` rows exist for it** — i.e. before any score is entered or published. This preserves "academic policy is immutable" (you can't edit a live scheme) while giving a clean typo escape hatch. Surface this as an explicit decision for the planner/user to confirm. [ASSUMED — needs user confirmation]

## Code Examples

### Quiz Average (AUTO) — eligible quizzes, missing=0, normalize first
```sql
-- db/queries/grades.sql  (recommended; one student)
-- name: ComputeQuizAverage :one
WITH eligible AS (
    SELECT q.id, q.max_grade
    FROM quizzes q
    JOIN courses c ON q.course_id = c.id
    WHERE q.course_id = $1 AND c.deleted_at IS NULL
      AND q.close_at IS NOT NULL AND q.close_at <= now()
      AND q.max_grade IS NOT NULL AND q.max_grade > 0
), per_quiz AS (
    SELECT e.id,
           COALESCE(
             (SELECT MAX(a.score) FROM quiz_attempts a
              WHERE a.quiz_id = e.id AND a.student_id = $2
                AND a.status IN ('SUBMITTED','AUTO_SUBMITTED')),
             0) / e.max_grade * 100 AS normalized   -- missing => MAX over empty = NULL => COALESCE 0
    FROM eligible e
)
SELECT COALESCE(AVG(normalized), 0)::numeric AS quiz_average,
       COUNT(*)::int AS eligible_count
FROM per_quiz;
```
*(Mirrors `GetMaxScore` D-50; eligible_count=0 means "no eligible quizzes yet" — component contributes 0 or is treated as not-yet-computable; planner decides display.)*

### Assignment Average (AUTO) — eligible = finalized, missing=0
```sql
-- name: ComputeAssignmentAverage :one
WITH eligible AS (
    SELECT a.id, a.max_score
    FROM assignments a
    JOIN courses c ON a.course_id = c.id
    WHERE a.course_id = $1 AND c.deleted_at IS NULL
      AND a.grading_finalized_at IS NOT NULL          -- D-64 eligibility
      AND a.max_score IS NOT NULL AND a.max_score > 0  -- D-57
), per_assignment AS (
    SELECT e.id,
           COALESCE(
             (SELECT s.score FROM submissions s
              WHERE s.assignment_id = e.id AND s.student_id = $2
              ORDER BY s.version DESC LIMIT 1),
             0) / e.max_score * 100 AS normalized
    FROM eligible e
)
SELECT COALESCE(AVG(normalized), 0)::numeric AS assignment_average,
       COUNT(*)::int AS eligible_count
FROM per_assignment;
```

### Sum-to-100 validation (service, Go)
```go
// group weights by parent (nil parent = root group); each group must sum to 100.
func validateWeights(comps []ComponentInput) error {
    sums := map[int64]float64{} // keyed by parent id, 0 for root
    for _, c := range comps {
        var k int64
        if c.ParentID != nil { k = *c.ParentID }
        sums[k] += c.Weight
    }
    for parent, total := range sums {
        if math.Abs(total-100) > 0.001 {
            return fmt.Errorf("weights under parent %d sum to %.2f, must be 100", parent, total)
        }
    }
    return nil
}
```

### Announcement fan-out (service)
```go
recipients := /* ALL_STUDENTS: ListCourseStudents | SPECIFIC: validated subset */
tx, _ := s.pool.Begin(ctx); defer tx.Rollback(ctx)
qtx := s.q.WithTx(tx)
ann, err := qtx.InsertAnnouncement(ctx, ...)           // immutable row, no updated_at
for _, r := range recipients {
    if audience == "SPECIFIC_STUDENTS" {
        _ = qtx.InsertAnnouncementRecipient(ctx, ann.ID, r.StudentID) // join table
    }
    _, _ = qtx.InsertNotification(ctx, db.InsertNotificationParams{
        RecipientID: r.StudentID, Type: "ANNOUNCEMENT",
        Title: ann.Title, Body: ann.Body,
        ResourceType: pgtype.Text{String:"announcement",Valid:true},
        ResourceID:   pgtype.Int8{Int64:ann.ID,Valid:true},
        Link:         pgtype.Text{String:fmt.Sprintf("/courses/%d/announcements/%d",courseID,ann.ID),Valid:true},
    })
}
return tx.Commit(ctx)
```

## State of the Art

| Old Approach | Current Approach | When | Impact |
|--------------|------------------|------|--------|
| Flat grade columns | Hierarchical `parent_id` component tree | D-56 | Supports future best-N/drop-lowest without redesign |
| Compute-on-read student grades | Persisted publish snapshot | D-66 | Student value frozen; lecturer live |
| Per-handler authz | `AssertCourseMember` + `RequireRole` | Phase 2/4 | Reuse, don't reinvent |

**Deprecated/outdated:** none introduced. Stack is current per CLAUDE.md (sqlc v1.31.1, pgx v5.7.x, Gin v1.11.0).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Delete-and-recreate whole scheme allowed only before any score/publication exists (D-65 typo escape) | Pitfall 6 | If user wants strict no-recovery, drop the escape hatch; if user wants edit-before-publish, widen it |
| A2 | `max_score` backfill default = 100 for existing Phase-4 assignments | Runtime State Inventory | Wrong default mis-normalizes historical Assignment Averages; confirm with user / check if any graded assignments exist |
| A3 | "Grading finalized" = explicit lecturer action setting `grading_finalized_at` (not derived "all active submissions graded") | Runtime Inventory / D-64 | Derived approach is alternative; explicit is leaner & matches D-64 "lecturer finalizes" wording — recommend explicit |
| A4 | A top-level AUTO/MANUAL leaf with no eligible items / no entered score contributes 0 to overall | Code Examples | Could instead block overall until populated; planner/user call |
| A5 | One grade scheme per course (`UNIQUE(course_id)`) | Pattern 1 | If multi-scheme/versioning ever needed (deferred), drop the constraint |
| A6 | Notification `type` enum strings (`GRADE_PUBLISHED`, `ANNOUNCEMENT`, `REQUEST_CREATED`, `REQUEST_REPLIED`) | Code Examples | Planner's discretion per D-53/D-54; names are illustrative |

## Open Questions (RESOLVED)

> All three resolved during planning — the resolutions are locked in 05-CONTEXT.md (prohibitions/decisions) and implemented by the Phase 5 plans (05-01/05-02). Marked RESOLVED here for record honesty.

1. **What exactly makes an assignment "finalized" (D-64)?**
   - What we know: Phase 4 has only per-submission `graded_at`/`graded_by`, no per-assignment marker.
   - What's unclear: explicit lecturer "finalize" button vs derived "all active submissions graded".
   - Recommendation: **explicit `grading_finalized_at` set by a lecturer action** — leaner, deterministic, matches D-64 wording ("when the lecturer finalizes grading"), and avoids a fragile "all submissions" definition (what about students who never submitted?). [A3]
   - **RESOLVED: explicit `grading_finalized_at` set by a lecturer finalize action** (05-01 Task 1/2; 05-02 compute reads it).

2. **Backfill value for `assignments.max_score` on existing rows.**
   - What we know: column is new; existing rows need a value.
   - Recommendation: `NOT NULL DEFAULT 100`; confirm no already-graded assignments would be mis-normalized. [A2]
   - **RESOLVED: `NOT NULL DEFAULT 100`** (05-01 Task 2 ALTER on `assignments`).

3. **D-65 typo recovery scope.** See Pitfall 6 / A1 — recommend allowing whole-scheme delete+recreate only before any score/publication exists. Surface for user confirmation.
   - **RESOLVED: whole-scheme delete+recreate allowed ONLY before any score/publication exists** (05-02 `DeleteSchemeIfEmpty`, counts==0 guard); scheme otherwise immutable per D-65.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| PostgreSQL (Docker) | All data | ✓ | 17 | — (Docker-only per D-08) |
| Go toolchain | Backend | ✓ | 1.24.x | — |
| sqlc | Codegen | ✓ (committed tool) | v1.31.1 | — |
| golang-migrate | Migrations | ✓ | v4.18.x | — |
| `DATABASE_URL` set | Integration tests | required for `go test` | — | tests `t.Skip` if unset (but check.sh now FAILs when unset) |

**No new external dependencies.** Cloudinary is NOT needed this phase (no file uploads — grade CSV is parsed in-memory, not stored to Cloudinary).

## Validation Architecture

> `config.json` has `nyquist_validation: false`, so the formal Nyquist section is optional — but the task explicitly requested validation coverage for the gradebook math and atomic notification writes, so this section maps each requirement to the **smallest real integration test against real Postgres** (the confirmed `DATABASE_URL` → `pgxpool.New` harness, inline SQL setup/teardown, `testify`).

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` + `stretchr/testify` |
| DB | Real Postgres via `DATABASE_URL` (CI runs migrations first; check.sh FAILs if unset) |
| Pattern | `pgxpool.New(ctx, os.Getenv("DATABASE_URL"))`, inline `INSERT ... RETURNING id`, `defer DELETE` cleanup [VERIFIED: enrollments/import_test.go] |
| Quick run | `go test ./internal/grades/... ./internal/announcements/... ./internal/requests/...` |
| Full suite | `bash scripts/check.sh` (golangci-lint + build + vet + `go test` on Postgres + FE) |

### Phase Requirements → Test Map
| Req | Behavior | Test Type | Smallest real test (against Postgres) |
|-----|----------|-----------|----------------------------------------|
| GRADE-01/02 | Sum-to-100 at each level | integration | Create scheme with Inclass(50)+Midterm(25)+Final(25) and Inclass subs summing 100 → success; mutate one sub to 95 → 422/ErrValidation. Assert reject is **per-level**. |
| GRADE-04 | normalize→aggregate, missing=0 | integration | Seed course+2 quizzes (max_grade 10 & 20), 1 finalized assignment (max_score 50); student attempts one quiz (score 8 → 80), skips the other (→0), submission 25/50 (→50). Assert Quiz Average=40, Assignment Average=50, overall = weighted. **Red-when-reverted:** if normalization removed, numbers diverge. |
| GRADE-04 (eligibility) | not-finalized assignment excluded | integration | Same seed but assignment `grading_finalized_at IS NULL` → assert it is **excluded** (not counted as 0). Then finalize → assert now counted as 0 for non-submitters. Proves D-58 vs D-64. |
| GRADE-05 | publish snapshot + per-student notify, atomic | integration | Publish a top-level component for N students in one tx → assert N `grade_publications` rows AND N `notifications` rows. Force a failure mid-loop (e.g. invalid recipient) → assert **zero** publications AND **zero** notifications (rollback leaves neither). |
| GRADE-05 (snapshot frozen) | student sees frozen value | integration | Publish (value=80); change underlying score so live=90; assert student read = 80, lecturer read = 90; republish → student read = 90 + a new notification row. |
| GRADE-03 (CSV) | all-or-nothing, 422 | integration | CSV with one bad row (score 150) → assert HTTP 422, full error list, **zero** `grade_scores` rows written. Clean CSV → all rows committed. |
| ANNC-01/02/03 | fan-out atomic | integration | ALL_STUDENTS → one notification per enrolled student + one `announcements` row, same tx. SPECIFIC → only targeted students get rows + join-table rows. |
| REQ-01/02/03 | directed + reply atomic | integration | Student creates request to lecturer L1 → L1 gets a notification (create), L2 does NOT see the request. L1 replies APPROVED+note → student gets a reply notification; both writes in one tx; reopen attempt rejected (closed permanently). |

### Sampling Rate
- **Per task commit:** `go test ./internal/<feature>/...`
- **Per wave merge:** `bash scripts/check.sh` (must be green; runs on real Postgres)
- **Phase gate:** full `scripts/check.sh` green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/grades/{compute_test.go, publish_test.go, csv_test.go}` — GRADE-01/04/05/03
- [ ] `internal/announcements/fanout_test.go` — ANNC-01/02/03
- [ ] `internal/requests/request_test.go` — REQ-01/02/03
- [ ] Migration `000008` must apply cleanly in CI before tests (golang-migrate up)
- [ ] **Anti-theater requirement (per project memory):** each test must be red-when-reverted, use real seeded fixtures (no empty bodies), and assert the atomic rollback property explicitly — not just the happy path.

## Security Domain

> `security_enforcement: true` — section required.

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V4 Access Control | yes | `RequireRole` + `AssertCourseMember`; ownership-from-JWT (`c.GetInt64("user_id")`); requests visible only to student+targeted lecturer (D-62) |
| V5 Input Validation | yes | Zod (FE, UX) + Go server-side: sum-to-100, `0≤score≤100`, CSV per-row validation, request type enum, `http.MaxBytesReader` on CSV upload |
| V6 Cryptography | no | No new crypto; JWT auth reused unchanged |
| V2 Authentication | no (reused) | Phase 2 JWT cookie spine unchanged |

### Known Threat Patterns for Go/Gin + Postgres
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| SQL injection | Tampering | sqlc parameterized queries only (no string-built SQL) |
| IDOR — student reads another student's grades | Info Disclosure | Student grade reads scoped to `student_id = JWT user_id`; never trust client-supplied student IDs |
| IDOR — lecturer publishes for a course they don't own | Elevation | `AssertCourseMember(...lecturer...)` before any scheme/publish/announce mutation |
| Request visible to wrong lecturer | Info Disclosure | Query requests filtered by `targeted_lecturer_id = JWT user_id` (D-62) |
| CSV bomb / oversized upload | DoS | `http.MaxBytesReader` (reuse 5<<20 cap from enrollments handler) + extension/content-type check |
| Tampered weights to inflate grades | Tampering | Server-side sum-to-100 + immutable scheme (D-65); reject client-recomputed overalls |
| Spreadsheet-formula injection in CSV | Tampering | Reuse `strings.TrimLeft(..., "=+-@\t\r ")` sanitization from enrollments ParseCSV |

## Sources

### Primary (HIGH confidence — verified against committed code)
- `backend/internal/assignments/service.go:202-253` — same-transaction grade+notify pattern, pgtype.Numeric, score clamp
- `backend/db/migrations/000006_*.up.sql` — notifications/assignments/submissions/quizzes/quiz_attempts schema; nullable quiz columns
- `backend/db/queries/{notifications,courses,submissions,quiz_attempts,assignments,quizzes}.sql` — InsertNotification, ListCourseStudents/Lecturers, GetMaxScore (D-50)
- `backend/internal/enrollments/{csv,service,handler}.go` — CSV all-or-nothing + 422 + formula sanitization
- `backend/internal/enrollments/import_test.go` — real-Postgres integration test harness
- `backend/sqlc.yaml` — engine/driver/paths
- `.planning/phases/05-gradebook-announcements-requests/05-CONTEXT.md` — D-56→D-67
- `.planning/REQUIREMENTS.md` — GRADE/ANNC/REQ/NOTIF acceptance wording
- `.claude/CLAUDE.md` / `.planning/PROJECT.md` — committed stack, D-04, D-10

### Secondary (MEDIUM)
- `.claude/projects/.../memory/MEMORY.md` — anti-theater tests; tests must run on real Postgres; check.sh FAILs when DATABASE_URL unset

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all packages committed and confirmed in code
- Architecture (schema/snapshot/computation): HIGH — derives directly from locked decisions + confirmed substrate; schema shapes are recommendations with rationale
- Pitfalls: HIGH — nullable quiz columns, eligibility-vs-publish, snapshot-vs-live all verified against actual schema/decisions
- Open design choices (A1/A2/A3): MEDIUM — flagged for user/planner confirmation

**Research date:** 2026-06-20
**Valid until:** ~2026-07-20 (stable; substrate is committed code, decisions are locked)

## Project Constraints (from CLAUDE.md)
- Committed stack only: sqlc v1.31.1 + pgx v5.7.x, golang-migrate v4.18, Gin v1.11.0, golang-jwt v5. No new deps.
- Feature-Oriented Monolith (D-10): handler=HTTP / service=business+authz / repository=SQL only. No interfaces/abstractions purely for purity (Ponytail).
- Incremental migration (D-06): append `000008`; CI runs migrations before tests.
- Postgres via Docker only (D-08).
- GitHub Flow: one `ft/<slug>` branch for the phase (backend+frontend together); squash-merge PR; never commit to `main`.
- Definition of Done: `bash scripts/check.sh` exits 0 (golangci-lint, build, vet, `go test` on real Postgres, FE eslint+tsc+vite build). Never claim complete without green check.sh.
- Tests must run against real Postgres (green check.sh ≠ tests ran — DATABASE_URL must be set).
- **MUST run `impact()` before editing any symbol** (esp. the `CreateAssignment` flow for the `max_score` touch) and `detect_changes()` before committing (GitNexus MCP).
- Mirror critical validation server-side in Go (client Zod is UX only).
- shadcn/ui only — no hand-rolled components.
