# Phase 5: Gradebook, Announcements & Requests - Context

**Gathered:** 2026-06-20
**Status:** Ready for planning

<domain>
## Phase Boundary

A course can be **run to completion without email** — lecturers compute **weighted final grades**, broadcast or target **announcements**, and answer student **requests**, all auto-delivered to the right students through the **Phase 4 notification primitive**. Delivers **GRADE-01 → GRADE-05**, **ANNC-01 → ANNC-03**, **REQ-01 → REQ-03**:

- A lecturer configures a course grade scheme as **Inclass + Midterm + Final = 100%**, with **Inclass sub-components summing to 100% of Inclass**, enters Midterm/Final manually, and the system computes each student's **weighted overall grade**; a student views their grades with **availability auto-notified**.
- A lecturer sends an announcement to **all enrolled students or specific ones**; targeted students see it **persisted on next login (no email)**.
- A student sends a **leave-early / absence / custom** request to a chosen course lecturer; the lecturer's **yes/no reply is auto-delivered** back to that student.

This phase reuses, not rebuilds: the **notification primitive** (D-53/D-54 — per-recipient rows, rendered title+body, bell + list page, mark-read-on-click), the **same-transaction notification write** (NOTIF-02, `pool.Begin → q.WithTx(tx) → InsertNotification`, see `assignments/service.go:202`), the **course/enrollment substrate** (`courses` / `student_enrollments` / `course_lecturers`), the **auth/ownership spine** (`RequireRole`, ownership-from-JWT, `{error:{code,message}}` envelope), and the **Phase 4 coursework scores** (quiz official score = MAX per D-50, assignment grades) as the gradebook's raw AUTO inputs. It adds three feature folders (`internal/grades/`, `internal/announcements/`, `internal/requests/`) and migration **`000008`**. It does **not** add new notification UX, new auth, or new course-admin capability.

</domain>

<decisions>
## Implementation Decisions

Decision IDs continue the project sequence. Phase 4 CONTEXT ended at D-55; **D-56 → D-67 are new and owned by this phase.** Each was authored by the user as a full decision record (decision + rationale + design principle); the summaries below are faithful condensations — the full records live in `05-DISCUSSION-LOG.md`.

### Gradebook — model & computation (GRADE-01/02/03/04)
- **D-56 — Hierarchical weighted-component gradebook (not flat columns).** Overall = `Inclass + Midterm + Final = 100%`. Composite components contain sub-components whose weights **sum to 100% of the parent**. Each **leaf** component has a source type: **AUTO** (computed from coursework — e.g. Quiz Average, Assignment Average) or **MANUAL** (entered by lecturer directly or via CSV — Project, Laboratory, Participation, Bonus, Midterm, Final). Design principle: **the gradebook computes grades, it does not merely store them.** Foundation for future policies (best-N, drop-lowest, weighted groups) without a redesign.
- **D-57 — Single 0–100 scale; normalize first, aggregate second.** All computations on a **0–100 scale** (HCMIU convention; per-institution configurable scale is **out of MVP**). AUTO normalizes before aggregation: **Quiz Average** = `avg(student_score / quiz_max × 100)` across eligible quizzes; **Assignment Average** = `avg(student_score / assignment_max × 100)`. MANUAL is entered directly on 0–100 (**validate `0 ≤ score ≤ 100`**). **⚠ Schema consequence:** assignments **SHALL define a max score** — the Phase 4 `assignments` table has no such column (see Planner Notes).
- **D-58 — AUTO includes all eligible items; missing = zero.** Quiz/Assignment Average include **all eligible items** in the course; a student who did not attempt/submit gets **0** for that item and it **stays in the aggregation set**. Only eligible (per D-64) items participate. Design principle: **missing assessment is a grade of zero, not absence of data.** Deterministic, no per-student lecturer intervention.
- **D-64 — AUTO eligibility = when the score is FINALIZED (not window timing).** A **quiz** becomes eligible when `close_at` has passed and auto-grading completed (effectively at `close_at`). An **assignment** becomes eligible **when the lecturer finalizes grading** — **independent of deadline, accept-late, or late threshold** (D-02). Refines D-58: an assessment is **not "missing=0" until finalized & eligible**, so ungraded/not-yet-due coursework never tanks an average. **Eligibility ≠ publication** — an item is eligible (counts in computation, lecturer sees it) potentially long before it is published (student sees it).

### Gradebook — visibility, publication & integrity (GRADE-05)
- **D-59 — Component-level publication; grades hidden until published.** Visibility is controlled by **publishing each top-level component independently** (Midterm first, Inclass later, Final last). Before publish: only the lecturer sees scores. After publish: students see **that component's** score **and receive one grade notification per affected student**. **Overall grade becomes visible only once all top-level components required for it are published.** Editing an already-published score does **not** auto-notify unless the lecturer **republishes**. Design principle: **grades exist before students can see them; publication is a separate academic action.**
- **D-65 — Grade scheme is immutable once created.** After creation, components, hierarchy, weights, and aggregation structure **cannot change**. Lecturers may enter scores, publish, and update **unpublished** scores; they may **not** add/remove components or change weights/structure. Rationale: grade weights are approved academic policy, not operational data. Design principle: **academic policy is immutable.**
- **D-66 — Published components are snapshots; computation stays live.** The value **students see is frozen at publish** and does not auto-change when underlying scores change. **Lecturers always see the live recomputed value** (AUTO recomputes on any eligible underlying change). To push a new value to students the lecturer **republishes** — which **replaces the snapshot and generates a new notification**. Aligns with D-59/D-65. Design principle: **publication creates a snapshot; computation remains live.** (Planner: persist the published snapshot value, don't compute-on-read for the student view.)
- **D-67 — MANUAL grade CSV: one component per file, `student_id,score`.** Each upload targets exactly one MANUAL component (Midterm, Final, Project, …) with rows `student_id,score`. Follows the **Phase 3 CSV discipline**: whole-file validation, all-or-nothing commit, row-level error report, **HTTP 422** on failure, no partial imports. Design principle: **grades are imported by component; gradebook structure must not leak into the CSV format.**

### Announcements (ANNC-01/02/03)
- **D-60 — Announcements are first-class entities + fan-out delivery.** An `announcements` row (`id, course_id, author_id, title, body, audience_type, created_at`) with `audience_type ∈ {ALL_STUDENTS, SPECIFIC_STUDENTS}`; `SPECIFIC_STUDENTS` stores targeted IDs in a **join table**. Creating an announcement **generates notification rows for all targeted recipients** (linking back to the announcement). Two surfaces: a per-course **Announcements page** (browse history, persists after notifications are read) **and** the **bell** (click → navigate to the announcement). Design principle: **announcements are content; notifications are delivery** — the notification system is reused without becoming the source of truth.
- **D-61 — Announcements are immutable after sending.** No edit, no delete, no recipient change. Corrections = **create a new announcement**. **⚠ Schema:** **drop `updated_at`** from the D-60 sketch — lifecycle is **CREATED only** (no EDITED/DELETED state). No re-notification or notification updates exist; history stays consistent. Design principle: **an announcement is a notice, not a document.**

### Requests (REQ-01/02/03)
- **D-62 — Requests are directed to one student-chosen lecturer.** The student **chooses a specific lecturer** assigned to the course; the request is visible **only to the requesting student and the targeted lecturer**. Other course lecturers do **not** see it. No main/assistant distinction; academic responsibility is handled outside the system. Eliminates shared-inbox races / first-reply-wins. Design principle: **requests have a clear owner; communication is directed, not broadcast.**
- **D-63 — Reply = required Decision + optional Note; one round-trip.** A request (type **leave-early / absence / custom**, title + plain-text body per REQ-01) starts **PENDING**; the lecturer replies **APPROVED or DENIED (required)** with an **optional note**, then the request is **closed permanently** — no reopen, no further replies, no conversation thread. The reply **auto-generates a notification** for the student (decision + optional note), written in the same transaction (NOTIF-02). Consistent with D-46. Design principle: **a decision is required; an explanation is encouraged, not required.**

### Claude's Discretion (settled by the established substrate — listed for researcher/planner to confirm)
- **Request creation notifies the targeted lecturer.** When a student submits a request, the targeted lecturer (D-62) gets a bell notification (the whole app is "no email"). Symmetric to the reply→student notification (D-63).
- **Same-transaction notification writes (NOTIF-02).** Grade-publish (D-59), republish (D-66), and request reply (D-63) each insert their notification(s) in the **same transaction** as the mutation, via the existing `pool.Begin → q.WithTx(tx) → InsertNotification` pattern (`assignments/service.go:202`).
- **Lecturer actions are NOT audit-logged.** The append-only `audit_log` stays **admin-only** (ADMIN-08) — grade entry/publish, announcements, and request replies are not audit-logged unless planning surfaces a reason (same default as Phase 4 discretion).
- **Feature-folder layout (D-10) + migration (D-06).** New code in `internal/grades/`, `internal/announcements/`, `internal/requests/` with the `handler/service/repository/model/dto` split; sqlc queries under `backend/db/queries/`; migration **`000008`** adds the new tables. Notification `type` enum strings, `link` URL shapes, and student grade-view / lecturer request-inbox UI layouts are the planner's call (consistent with D-53/D-54 and the DESIGN-SYSTEM).
- **Recipient snapshot at send time.** ALL_STUDENTS / SPECIFIC_STUDENTS fan-out (D-60) resolves recipients at creation (per D-53 persist-at-creation); later enrollment changes do not retroactively add/remove delivered notifications.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project-level (locked stack, constraints & architecture)
- `.claude/CLAUDE.md` — committed stack (sqlc v1.31.1 + pgx v5.7.x, golang-migrate v4.18, Gin v1.11.0, golang-jwt v5) and the Feature-Oriented Monolith handler/service/repository split. Authoritative for all library choices.
- `.planning/PROJECT.md` — vision, constraints, Key Decisions table — esp. **D-04** (weighted gradebook: Inclass with sub-weights + Midterm + Final, each a % of overall, Midterm/Final entered manually) and **D-10** (feature-folder architecture).
- `.planning/REQUIREMENTS.md` §"Gradebook (GRADE)", §"Announcements (ANNC)", §"Requests (REQ)", §"Notifications (NOTIF)" — GRADE-01→05, ANNC-01→03, REQ-01→03, NOTIF-01→02 acceptance wording; plus §"Out of Scope" (no email; grade export is GRADE-V2-01).
- `.planning/ROADMAP.md` §"Phase 5" — goal + 4 success criteria; overview note that gradebook/announcements/requests all share **one persisted-fan-out notification primitive** and honor append-only-audit + soft-delete discipline.

### Design (frontend)
- `.planning/DESIGN-SYSTEM.md` (D-05) — global UI ruleset: shadcn/ui only (no hand-rolled components), light+dark, 6px radius, Lucide icons, Skeleton loaders, WCAG AA, expandable sidebar. The student grade-view, course Announcements page (D-60), and lecturer request-inbox (D-62) live under the D-20 role route trees; the bell (D-54) is the shared notification surface.

### Prior phase context (patterns this phase reuses)
- `.planning/phases/04-assignments-quizzes/04-CONTEXT.md` — the **notification primitive** (D-53/D-54), the **same-transaction write** (NOTIF-02, D-55), and the coursework scores that feed the gradebook: assignment grades (D-46) and quiz **official score = MAX across attempts (D-50)**. Note D-55: quiz grading did NOT notify (synchronous); Phase 5 grade-publish DOES notify (D-59).
- `.planning/phases/03-admin-provisioning-course-lifecycle/03-CONTEXT.md` — `courses` / `student_enrollments` / `course_lecturers` substrate; **CSV import discipline** (whole-file validation, all-or-nothing, per-row 422 errors) that D-67 reuses; append-only `audit_log` + SYSTEM actor; soft-delete-by-`deleted_at` with reads filtering active courses (D-40 no-cascade).
- `.planning/phases/02-auth-rbac-forced-first-login/02-CONTEXT.md` — `RequireRole`, ownership-from-JWT (never trust client IDs), `{error:{code,message}}` envelope, cookie JWT, FE `ProtectedRoute`/`RoleGuard`/`AppLayout` + axios 401/403 interceptor.
- `.planning/phases/01-foundation-data-core/01-CONTEXT.md` — **D-06** incremental per-phase migrations (Phase 5 appends `000008`), **D-08** Docker = Postgres-only.

### Existing code to read (Phase 1–4 output)
- `backend/db/migrations/000006_assignments_quizzes_notifications.up.sql` — the `notifications` table (recipient_id, type, title, body, resource_type, resource_id, link, created_at, read_at + `notifications_recipient_read_idx`); `assignments` (**no `max_score` — D-57 needs one**); `submissions` (`score`, `graded_at`, `graded_by`); `quizzes` (`max_grade`, `close_at`); `quiz_attempts` (`score`, `status`).
- `backend/internal/notifications/{service,repository,model,dto,handler}.go` — `InsertNotification`, `ListNotifications`, `CountUnread`, `MarkRead`; the bell + list page wiring to reuse.
- `backend/internal/assignments/service.go:202` — the canonical **same-transaction grade+notify** pattern (`pool.Begin → q.WithTx(tx) → InsertNotification`) that grade-publish and request-reply follow.
- `backend/internal/{courses,enrollments}/` + `backend/db/queries/courses.sql` (`ListCourseStudents`, `ListCourseLecturers`) — course/membership reads for gradebook rosters, announcement audiences, and request lecturer-selection.
- `backend/internal/auth/` — `errorEnvelope(...)`, `RegisterRoutes(r, pool, cfg)` wiring style to mirror; `backend/internal/shared/middleware/{auth,role}.go` — `RequireRole`.
- `frontend/src/routes/router.tsx`, `frontend/src/components/AppLayout.tsx`, `frontend/src/stores/auth.ts`, `frontend/src/lib/api.ts`, `frontend/src/pages/Notifications.tsx` — role route trees, header/bell mount, auth store, axios client, notification list page.

No external ADRs beyond the above — requirements are fully captured in the decisions here + the locked stack in CLAUDE.md.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **Notification primitive (Phase 4):** `notifications` table + `internal/notifications/` service (insert/list/count-unread/mark-read) + FE bell & list page — reused verbatim for grade-publish (D-59/D-66), announcement fan-out (D-60), and request replies (D-63). No new notification UX.
- **Same-transaction write (`assignments/service.go:202`):** `pool.Begin → q.WithTx(tx) → InsertNotification` is the exact pattern grade-publish, republish, and request-reply follow (NOTIF-02).
- **Coursework scores (Phase 4):** quiz official score = MAX across attempts (D-50) and assignment grades (D-46) are the **AUTO inputs** for Quiz Average / Assignment Average (D-56/D-57/D-58).
- **CSV import discipline (Phase 3):** whole-file validation, all-or-nothing, per-row 422 errors — reused by MANUAL grade CSV (D-67).
- **Course/membership substrate + auth/ownership spine** — gradebook rosters, announcement audiences, request routing, and all authz derive from `student_enrollments` / `course_lecturers` + `RequireRole` + ownership-from-JWT.

### Established Patterns
- **Feature-Oriented Monolith (D-10):** `internal/grades/`, `internal/announcements/`, `internal/requests/`, each handler=HTTP / service=business+authz / repository=SQL only.
- **Incremental migration (D-06):** append `000008`; CI runs migrations before tests.
- **Immutable historical records:** notifications (D-53), announcements (D-61), published grade snapshots (D-66), and the grade scheme (D-65) all follow the same "freeze on commit, correct by re-issuing" stance.

### Integration Points
- **Gradebook ⇄ Phase 4 coursework:** AUTO components read `quiz_attempts`/`submissions`; **assignment needs a `max_score` (D-57)** and a **"grading finalized" marker (D-64)** that Phase 4 does not yet provide → migration `000008` + a small Phase-4-table touch.
- **All three features ⇄ notification primitive:** every student-facing delivery is a notification row (NOTIF-01) written in-transaction (NOTIF-02).
- **Requests ⇄ course_lecturers:** the student picks from the course's assigned lecturers (D-62); reply notifies the student, request-creation notifies the lecturer (discretion).

</code_context>

<specifics>
## Specific Ideas

- Each decision **D-56 → D-67 was authored by the user as a complete decision record** (decision + rationale + relationships + design principle) — preserve that intent; the full text is in `05-DISCUSSION-LOG.md`, the condensations above are faithful.
- Recurring user design principles this phase: **the gradebook computes grades, it does not merely store them** (D-56); **normalize first, aggregate second** (D-57); **missing assessment is a grade of zero, not absence of data** (D-58); **grades exist before students can see them — publication is a separate academic action** (D-59); **an assessment contributes when its result is finalized; eligibility ≠ publication** (D-64); **academic policy is immutable** (D-65); **publication creates a snapshot; computation remains live** (D-66); **an announcement is a notice, not a document** (D-61); **requests have a clear owner — directed, not broadcast** (D-62); **a decision is required, an explanation is encouraged** (D-63).
- The user consistently chose the **richer, real-university-aligned** option over the leaner default (hierarchical components over flat columns, 0–100 + normalization over assume-same-scale, component-level publication over single release, directed requests over shared inbox) and reinforced an **immutability theme** across grades / announcements / scheme / snapshots.
- The user flagged the **assignment late-threshold** interaction unprompted — eligibility is keyed to **finalized grading**, not the deadline, precisely because accept-late assignments stay open past the deadline (D-02/D-44).

## Planner / Researcher Notes (technical consequences to resolve)
- **D-57:** add **`max_score`** to `assignments` in migration `000008`; update the Phase 4 assignment-create form/handler/DTO + any existing-data backfill so AUTO Assignment Average can normalize.
- **D-64:** add an explicit **"grading finalized" marker for an assignment** (Phase 4 has only per-submission `graded_at`/`graded_by`, no per-assignment finalize). Define precisely what "finalized" means (lecturer action vs. derived "all active submissions graded").
- **D-66:** persist the **published snapshot value** per published component (and `published_at`) so the student-facing value is stable; lecturer view recomputes live. Define republish semantics (replace snapshot + new notification).
- **D-58 vs D-59 terminology:** "published coursework" (D-58 eligibility) ≠ "published grade component" (D-59 student release) — do not conflate in schema/code naming.
- **D-65:** decide the typo-recovery edge — is delete-and-recreate of the whole scheme allowed **before any score is entered**, given the scheme is otherwise immutable.
- **Overall computation requires all top-level weights to be defined** (D-56 sum-to-100 validation server-side); reject scheme creation that doesn't sum correctly at each level.

</specifics>

<deferred>
## Deferred Ideas

- **Per-institution configurable grading scale** (0–10 / GPA-based) — out of MVP; 0–100 fixed (D-57 future evolution).
- **Advanced grading policies** — best-N quizzes, drop-lowest, weighted quiz groups, alternative aggregation rules — the D-56 hierarchical model is designed to support these later without redesign.
- **Grade export (CSV/PDF transcripts)** — already tracked as **GRADE-V2-01** (v2).
- **Request conversation threads / reopen / multi-reply** — explicitly excluded; requests are one-request-one-decision (D-63).
- **Announcement edit/delete / scheduled send / read receipts beyond the notification marker** — excluded; announcements are immutable notices (D-61).
- **Shared lecturer request inbox / request reassignment between co-lecturers** — excluded; requests are directed to one chosen lecturer (D-62).

None of these are scope creep into Phase 5 — discussion stayed within the gradebook / announcements / requests boundary.

</deferred>

---

*Phase: 5-Gradebook, Announcements & Requests*
*Context gathered: 2026-06-20*
