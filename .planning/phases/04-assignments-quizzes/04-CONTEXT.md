# Phase 4: Assignments & Quizzes - Context

**Gathered:** 2026-06-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Lecturers can run graded coursework — **file-upload assignments** (with deadline + late policy) and **auto-graded MCQ quizzes** — and students are auto-notified of new assignment grades through a **shared persisted notification primitive**. Delivers **ASMT-01 → ASMT-06**, **QUIZ-01 → QUIZ-06**, **NOTIF-01 → NOTIF-02**:

- A lecturer creates an assignment with a deadline + accept-late/threshold policy (D-02); a student submits a single **PDF or ZIP (≤10MB, magic-byte validated server-side)**; the server enforces the late policy by its own timestamp.
- Submitted files are stored on **Cloudinary as authenticated (non-public) assets**, downloadable only via **backend-generated short-lived signed URLs gated by role/ownership** (ROADMAP existential pitfall #2/#3).
- A lecturer grades a submission; saving the grade **auto-notifies the student in the same transaction**.
- A lecturer creates an **MCQ quiz** (CSV or UI; configurable max questions, max grade, shuffle, retake count per D-03); the take-quiz API **never exposes which option is correct**; shuffle preserves the correct-answer mapping by **stable option ID** (ROADMAP existential pitfall #4).
- The system **auto-grades a quiz on submit (idempotent per attempt)**, records the score, and enforces the retake limit with attempts tracked distinctly.

This phase introduces the **first Cloudinary integration** (config has `CLOUDINARY_URL` but no client yet) and the first course-scoped coursework features. It builds on the Phase 2 auth/RBAC spine (`RequireRole`, ownership-from-JWT, `{error:{code,message}}` envelope, cookie JWT) and the Phase 3 course substrate (`courses` / `student_enrollments` / `course_lecturers`, append-only `audit_log`, soft-delete-by-`deleted_at` with reads filtering active courses, D-40 no-cascade). It does **not** build the weighted gradebook, announcements, or requests (Phase 5) — but the notification primitive built here is the shared substrate Phase 5 reuses.

</domain>

<decisions>
## Implementation Decisions

Decision IDs continue the project sequence. Phase 3 CONTEXT ended at D-43; **D-44 → D-55 are new and owned by this phase.** Each was authored by the user as a full decision record (decision + rationale + relationships + accepted trade-off + design principle); the summaries below are faithful condensations.

### Assignment submission & grading (ASMT-01/03/04/06)
- **D-44 — Versioned submissions; never overwrite.** Students may submit multiple times while the submission window is open; **each submission creates a new version record** and prior versions are preserved (viewable). The submission window is defined by the deadline + late config (D-02) + late threshold; while open, first-submit / resubmit / unlimited versions are allowed; once closed, all new/resubmissions are rejected. The **most recent version is the active version** and is what lecturers grade by default. Rationale: academic submissions are historical records and should be versioned rather than overwritten (traceability, dispute resolution, no accidental data loss). Trade-off: extra submission metadata stored — accepted because files live in Cloudinary and DB growth stays minimal.
- **D-45 — Late submissions are flagged only; no automatic penalty.** When a submission lands after the deadline but within the allowed late window (D-02), the system records `is_late`, `submitted_at`, and a human-readable `late_duration` (e.g. "5 minutes", "2 days", "6 days 12 hours") and surfaces a "late by X" indicator to the lecturer. The system **records facts; the lecturer makes the academic judgment** on any penalty. Late resubmissions still create new versions while the window is open (D-44). Trade-off: late policy is not standardized/automated across lecturers — intentionally accepted (grading is an academic responsibility, not a system responsibility).
- **D-46 — Grading inputs: score required, feedback optional.** The grading form requires a **score** and allows **optional feedback text**; students view both. Grade applies to the active submission version (D-44); feedback may explain late-related adjustments (D-45) but is never required. Rationale: in large classes, mandatory feedback adds workload with limited value; optional feedback still serves exceptional cases. Design principle: required data supports the primary workflow; optional data supports exceptional cases without taxing routine ones.

### Quiz authoring & question model (QUIZ-01/02/04)
- **D-47 — Question-bank model with random per-attempt selection.** Lecturers maintain a **pool of N questions**; each attempt **randomly selects M questions (M ≤ N)** from the pool. Per-quiz config: Pool Size N, Max Questions M, Max Grade, Shuffle (yes/no), Retake Count. **Each retake attempt gets a newly generated question set** from the same pool. Shuffle = Yes → randomize question selection + question order + answer order; Shuffle = No → fixed configured order. Rationale: reduces cheating, increases variety, gives retakes meaning, lets lecturers maintain one master pool. Trade-off: different students may get different combinations — accepted because all questions come from the same lecturer-approved bank. Design principle: question pools **generate** assessments rather than store fixed ones.
- **D-48 — Two authoring modes; exact-match auto-grading.** (1) **CSV import** uses a fixed format — 4 answer choices A–D, exactly 1 correct: `question,A,B,C,D,correct` (e.g. `What is 2 + 2?,1,2,3,4,D`). (2) **Manual UI** allows **single-choice** (1 correct, radio) and **multi-choice** (multiple correct, checkbox). Auto-grading: single-choice correct when `selected == correct`; multi-choice correct **only** on exact set match `selected_set == correct_set` (**all-or-nothing**, no partial credit). Questions from either mode feed quiz pools (D-47). Trade-off: multi-choice uses all-or-nothing — partial credit / weighted / negative marking intentionally excluded from MVP (extensible later without changing question data). Design principle: prefer simple, deterministic grading for MVP while preserving future extensibility.
- **D-49 — Quizzes use an availability window (Open At / Close At); no late, no timer.** Lecturers configure **Open At** and **Close At**; students may start and submit attempts only while the window is open; retakes are only permitted while open (subject to retake count). **After Close At: no new attempts, retakes, or submissions.** Unlike assignments, **quizzes do not support late submission** — they are time-bounded assessment events. Students may review completed attempts after submission (subject to the reveal policy in D-51). Trade-off: per-attempt countdown timers excluded from MVP (availability enforced by open/close timestamps only); future may add timers / duration limits / lockdown browser without changing the attempt model. Design principle: quiz access is governed by availability windows, not late-submission policies.

### Quiz attempts, scoring & answer reveal (QUIZ-03/05/06)
- **D-50 — Official quiz score = MAX across completed attempts.** `official_score = MAX(attempt_scores)`. The gradebook (Phase 5) stores the **official score only**; full attempt history stays available for review. All attempts must fall within the availability window (D-49) and within the retake count (D-03). Rationale: retakes support learning — highest-score rewards mastery, encourages practice, doesn't penalize experimentation, matches common LMS expectations. Trade-off: students may use early attempts as practice — accepted (retakes are a learning mechanism, not a penalty).
- **D-51 — Answer-reveal policy is window-bound, not attempt-bound.** **While the window is open**, a student reviewing a completed attempt sees: final score, their submitted answers, per-question correct/incorrect status — but **NOT** the correct answers or explanations. **After the window closes**, correct answers + per-question results become visible. This rule applies **regardless of remaining/completed retakes or retake config**. Rationale: revealing correct answers before close would let early finishers share answers with students still taking the quiz or holding remaining attempts; integrity is tied to the **window**, not individual attempt completion. This is the concrete enforcement of the QUIZ-03 answer-non-leakage pitfall. Trade-off: delayed correct-answer feedback — accepted to preserve fairness within a shared window.
- **D-52 — Attempt consumed on START; resumable; auto-submit on window close.** A new attempt is created and **consumes one available attempt when the student starts** (not on submit). States: `IN_PROGRESS`, `SUBMITTED`, `AUTO_SUBMITTED`. While `IN_PROGRESS` the student may leave and return (resume the same attempt — no new attempt, no extra retake consumed) but may not start another. On submit → `SUBMITTED` + `submitted_at`. If the window closes while an attempt is still `IN_PROGRESS` → it is **automatically submitted** (`AUTO_SUBMITTED`). A new attempt may start only when the previous is `SUBMITTED`/`AUTO_SUBMITTED`, the window is still open, and attempts remain (D-03). Rationale: consuming on start prevents opening quizzes just to inspect the pool; resume protects against crashes/disconnects. Trade-off: starting consumes a retake even if few questions are answered — accepted because the attempt already exposed quiz content. Design principle: opening an assessment is participation; participation creates an attempt.

### Notifications (NOTIF-01/02)
- **D-53 — Notifications persist fully-rendered content at creation.** Each notification row: `recipient_id, type, title, body, resource_type (opt), resource_id (opt), link (opt), created_at, read_at`. **Title + body are rendered when the notification is created and stored directly** (e.g. title "Assignment Graded", body `Your assignment "SE Lab 03" has been graded. Score: 8.5/10.`). One row per recipient with `read_at` as the read marker (NOTIF-01). Notifications stay readable even if the referenced resource is later archived / soft-deleted / modified (ties to D-29/D-40). Rationale: notifications are **historical events**, not live projections — persisting rendered content simplifies reads, avoids runtime rendering, and prevents broken notifications when resources change. Trade-off: text is duplicated in the DB — accepted (small content, modest volume). Future: templates / localization while still persisting rendered content.
- **D-54 — Centralized notification center (bell in header).** A **bell icon in the app header** with an **unread badge count** (e.g. 🔔 5), a **notification list page**, **mark-as-read on click**, and **deep-link navigation** to the related resource (e.g. "Quiz Result Available" → `/courses/se101/quizzes/quiz1/result`; "Assignment Graded" → `/courses/se101/assignments/lab03`). One center aggregates all notification sources (assignments, quizzes, grades, announcements, courses, future types). **MVP scope:** bell + unread badge + list page + mark-read-on-click + deep-link. **Out of scope:** real-time push, dropdown previews, notification categories, notification preferences. Design principle: notifications should bring users to information, not require them to search for it.
- **D-55 — Phase 4 notifications fire only on assignment grading.** When a lecturer saves an assignment grade: the grade is persisted **and** a notification is created for the student in the **same transaction** (NOTIF-02), stored per D-53, delivered via D-54. **Quiz grading does NOT generate notifications** — quiz scores are shown immediately after submission (D-49), so a separate notification would be redundant. Rationale: assignment grading is asynchronous (students don't know when it's done → notify); quiz grading is synchronous (score delivered inline). Future phases add announcement / assignment-creation / enrollment / gradebook notifications without changing the architecture. Design principle: notify about *new* information; don't duplicate information already presented directly.

### Claude's Discretion (settled without a user question — locked by D-10 + the auth/course substrate)
- **Feature-folder layout (D-10).** New code organizes by business feature: `internal/assignments/`, `internal/quizzes/`, and a notification feature (e.g. `internal/notifications/`), each with the `handler.go / service.go / repository.go / model.go / dto.go` split; sqlc queries under `backend/db/queries/`. A new `internal/shared/cloudinary/` client is added (config `CloudinaryURL` already loaded). Migration(s) **`000006+`** (D-06) add: `assignments`, `submissions` (versioned per D-44), `quizzes`, `quiz_questions` (pool), `quiz_question_options` (stable IDs for shuffle), `quiz_attempts`, `quiz_attempt_answers`, `notifications`.
- **Ownership/authorization (AUTH-05 pattern).** All coursework endpoints sit behind `RequireRole(...)` and derive scope from membership: only a **lecturer assigned to the course** (`course_lecturers`) may author/grade its assignments & quizzes; a **student enrolled in the course** (`student_enrollments`) may submit/take. `user_id` is always from the JWT, never client-supplied. Reads filter soft-deleted courses (D-29/D-40). Error responses use the established `{error:{code,message}}` envelope.
- **Audit-logging of lecturer actions — default OFF.** The append-only `audit_log` is **admin-only** (ADMIN-08); Phase 4 lecturer actions (create assignment/quiz, grade) are **not** audit-logged unless planning surfaces a reason. (Listed for researcher/planner to confirm.)
- **Notification read-marker UX.** Mark-as-read on click (D-54); a "mark all read" affordance is discretionary. Exact `type` enum strings, `link` URL shapes, and badge-count query are the planner's call, consistent with D-53/D-54.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project-level (locked stack, constraints & architecture)
- `.claude/CLAUDE.md` — committed stack + the coursework-relevant **Stack Patterns**: Gin `MaxMultipartMemory` + reject early; validate real MIME via `http.DetectContentType` on the first 512 bytes (allow `application/pdf`, `application/zip`/`application/x-zip-compressed`), reject by extension **AND** sniffed type; enforce 10MB via `c.Request.ContentLength` + `http.MaxBytesReader`; Cloudinary `uploader.Upload(ctx, file, uploader.UploadParams{ResourceType: "raw"})` for PDF/ZIP (NOT "image") as **authenticated** assets; sqlc v1.31.1 + pgx v5.7.x; golang-migrate v4.18 owns schema; `gocron/v2` or `time.Ticker` for scheduled work. Authoritative for all library choices.
- `.planning/PROJECT.md` — vision, constraints, Key Decisions table, esp. **D-02** (per-assignment late policy: deadline + accept-late + optional threshold days) and **D-03** (per-quiz config: max questions, max grade, shuffle, CSV-or-UI source, retake 0..N), and **D-10** (Feature-Oriented Monolith + handler/service/repository split).
- `.planning/REQUIREMENTS.md` §"Assignments & Submissions (ASMT)", §"Quizzes (QUIZ)", §"Notifications (NOTIF)" — ASMT-01→06, QUIZ-01→06, NOTIF-01→02 acceptance wording; plus §"Out of Scope" (no server-side ZIP extraction; PDF/ZIP ≤10MB only; MCQ-only auto-grading).
- `.planning/ROADMAP.md` §"Phase 4: Assignments & Quizzes" — goal + 5 success criteria + the existential pitfalls baked in here: **authenticated Cloudinary delivery**, **magic-byte upload validation**, **quiz answer non-leakage**.

### Design (frontend)
- `.planning/DESIGN-SYSTEM.md` (D-05) — global UI ruleset: shadcn/ui only (no hand-rolled components), light+dark, 6px radius, Lucide icons, Skeleton loaders, WCAG AA, expandable sidebar. The **notification bell** (D-54) mounts in the header established here; student/lecturer feature pages live under the D-20 role route trees.

### Prior phase context (patterns this phase reuses)
- `.planning/phases/03-admin-provisioning-course-lifecycle/03-CONTEXT.md` — the `courses` / `student_enrollments` / `course_lecturers` substrate every coursework feature reads; the append-only `audit_log` + SYSTEM actor (D-38); **soft-delete-by-`deleted_at` with reads filtering active courses, D-40 no-cascade** (assignments/submissions/quizzes/attempts/notifications are non-cascading dependents).
- `.planning/phases/02-auth-rbac-forced-first-login/02-CONTEXT.md` — the auth spine: `RequireRole`, ownership-from-JWT (never trust client IDs), the `{error:{code,message}}` envelope, cookie JWT + `withCredentials`, the FE `ProtectedRoute`/`RoleGuard`/`AppLayout` shells and axios 401/403 interceptor.
- `.planning/phases/01-foundation-data-core/01-CONTEXT.md` — **D-06** incremental per-phase migrations (Phase 4 appends `000006+`), **D-08** Docker = Postgres-only.

### Existing code to read (Phase 1–3 output)
- `backend/db/migrations/000004_admin_schema.up.sql` — `courses` (`id BIGINT`, `code, name, term, start_date, end_date, deleted_at`), `student_enrollments(course_id, student_id)`, `course_lecturers(course_id, lecturer_id)` — the FK targets for assignments/quizzes (course_id) and submissions/attempts (student_id → `users.id`).
- `backend/db/migrations/000005_audit_append_only_and_system.up.sql` — append-only triggers + SYSTEM seed (context for the audit-OFF discretion + any reuse).
- `backend/internal/shared/config/config.go` — `CloudinaryURL` env field already loaded (no client yet — Phase 4 adds `internal/shared/cloudinary/`).
- `backend/internal/shared/middleware/auth.go` / `role.go` — `RequireRole` + the auth middleware all Phase 4 routes register behind.
- `backend/internal/auth/handler.go` / `service.go` / `repository.go` — the `errorEnvelope(...)` helper + feature pattern + `RegisterRoutes(r, pool, cfg)` wiring style to mirror.
- `backend/internal/courses/` + `backend/internal/enrollments/` + `backend/db/queries/courses.sql` (`ListCourseStudents`, `ListCourseLecturers`) — course/membership read patterns to derive coursework scope/recipients.
- `backend/cmd/api/main.go` — entrypoint where coursework routes register and where the in-process daily/startup scheduler (D-37) lives — candidate to reuse for the D-52 `AUTO_SUBMITTED`-on-window-close mechanism.
- `frontend/src/routes/router.tsx`, `frontend/src/components/AppLayout.tsx`, `frontend/src/stores/auth.ts`, `frontend/src/lib/api.ts` — role route trees, the header where the notification bell (D-54) mounts, the auth store, and the axios client (cookies + 401/403 interceptor).

No external ADRs beyond the above — requirements are fully captured in the decisions here + the locked stack in CLAUDE.md.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **Course + membership substrate exists** (Phase 3): `courses`, `student_enrollments`, `course_lecturers` with `ListCourseStudents` / `ListCourseLecturers` queries — coursework scope and notification recipients derive directly from these.
- **Auth spine fully reusable** (Phase 2): `RequireRole`, ownership-from-JWT, `errorEnvelope`, cookie JWT, the FE `AppLayout` header (notification bell mounts here), `ProtectedRoute`/`RoleGuard`, axios interceptor.
- **Config carries `CloudinaryURL`** already (Phase 1) — Phase 4 wires the first actual Cloudinary client under `internal/shared/cloudinary/`.
- **Append-only `audit_log` + SYSTEM actor** present — available if any Phase 4 action is later deemed audit-worthy (default OFF per discretion).

### Established Patterns
- **Feature-Oriented Monolith (D-10):** handler = HTTP only, service = business + authz/ownership, repository = SQL only; sqlc queries in `backend/db/queries/`.
- **Incremental migrations (D-06):** append `000006+`; CI runs migrations before tests.
- **Soft-delete discipline (D-29/D-40):** reads filter `deleted_at IS NULL` courses; coursework dependents don't cascade and are naturally hidden via their active-course gate.
- **Stack patterns pre-locked in CLAUDE.md:** Cloudinary `ResourceType:"raw"` authenticated + signed URLs, magic-byte validation + `MaxBytesReader` 10MB, sqlc + pgx, golang-migrate.

### Integration Points
- The **notification primitive** (D-53/D-54) built here is the shared substrate **Phase 5** reuses for grade-availability, announcement fan-out, and request replies (NOTIF-01).
- **Quiz scores + assignment grades** produced here become the inputs the **Phase 5 weighted gradebook** consumes (D-50 official score feeds the gradebook).
- Coursework reads hang off the **course/enrollment scope** + the **auth ownership** pattern; the **same-transaction notification write** (NOTIF-02) sets the pattern Phase 5 mutations follow.

</code_context>

<specifics>
## Specific Ideas

- Each decision **D-44 → D-55 was authored by the user as a complete decision record** (decision + rationale + relationships + accepted trade-off + design principle) — preserve that intent verbatim; the condensations above are faithful.
- Recurring user design principles across this phase: **the system records facts, lecturers make academic judgments** (D-45); **question pools generate assessments rather than store fixed ones** (D-47); **assessment integrity is governed by the quiz window, not individual attempt status** (D-51); **opening an assessment is participation** (D-52); **notifications are historical records that stay stable when resources change** (D-53); **notify only about genuinely new information** (D-55).
- **Versioning over overwriting** (D-44) and **flag-don't-penalize** (D-45) are deliberate "preserve history / keep policy human" calls, mirroring Phase 3's "preserve history over destructive cleanup" stance.
- The **CSV quiz format is fixed at 4 choices / 1 correct** (`question,A,B,C,D,correct`), while the **UI additionally allows multi-choice** (D-48) — a deliberate asymmetry (bulk import stays simple; UI handles specialized questions).
- **Quiz open/close window (D-49) was explicitly chosen in scope** by the user over the lean "always available while course active" default — it is not scope creep.

</specifics>

<deferred>
## Deferred Ideas

- **Per-attempt quiz timers / duration limits / lockdown-browser integration** — explicitly out of MVP; availability enforced by open/close timestamps only (D-49 future evolution).
- **Partial-credit / weighted-selection / negative-marking quiz grading** — MVP uses exact-match all-or-nothing; future may add configurable grading strategies without changing question data (D-48 future evolution).
- **Notification real-time push / dropdown previews / categories / preferences** — out of MVP (D-54); pull-based persisted center only.
- **Notification templates / localization** — future, while still persisting rendered content (D-53 future evolution).
- **Announcement / assignment-creation / enrollment / gradebook-publication notifications** — additional notification events arrive in **Phase 5** without changing the notification architecture (D-55 future evolution).
- **Late submission for quizzes** — explicitly excluded; quizzes are time-bounded events, not late-policy-governed (D-49).

None of these are scope creep into Phase 4 — discussion stayed within the assignments / quizzes / notification boundary.

## Research items for gsd-phase-researcher
- **Cloudinary authenticated storage + signed delivery (ASMT-05).** `uploader.Upload(..., ResourceType:"raw")` as a non-public/authenticated asset; backend generation of **short-lived signed download URLs** (TTL choice) gated by role/ownership (or a backend download-proxy endpoint). Confirm import path `github.com/cloudinary/cloudinary-go/v2` ≥ Jun-2025 security release. Wire `internal/shared/cloudinary/` from `CloudinaryURL`.
- **Magic-byte upload validation (ASMT-03).** `http.DetectContentType` on first 512 bytes for PDF + ZIP (incl. `application/x-zip-compressed`), reject by **extension AND sniffed type**; enforce 10MB via `ContentLength` check + `http.MaxBytesReader`; never extract ZIPs server-side (zip-bomb out-of-scope).
- **Submission versioning schema (D-44/D-46).** Model `submissions` with a version dimension keyed by (assignment, student); how a grade (D-46) attaches to a version and the **edge case** where a student submits a newer version after grading while the window is still open.
- **Quiz data model (D-47/D-48/QUIZ-04).** Pool questions + options with **stable option IDs** so shuffle preserves correct-answer mapping; `quiz_attempts` (states `IN_PROGRESS`/`SUBMITTED`/`AUTO_SUBMITTED`, D-52) + `quiz_attempt_answers`; **random draw of M from N** per attempt (D-47); **idempotent auto-grade per attempt** (QUIZ-05); retake counting **consumed on start** (D-52); the take-quiz API must **never return correct flags** while the window is open (D-51/QUIZ-03).
- **Quiz `AUTO_SUBMITTED` on window close (D-52).** Lazy evaluation (on next access after `close_at`) vs a scheduled sweep — candidate to reuse the Phase 3 in-process scheduler (D-37). Decide idempotency + how official-score = MAX (D-50) is computed/stored.
- **Notification schema + same-transaction write (NOTIF-01/02, D-53/D-55).** `notifications` table per D-53; assignment-grade write + notification insert in **one transaction**; unread-badge-count query + mark-read-on-click (D-54); answer-reveal gating (D-51) must be enforced server-side off `close_at`.

</deferred>

---

*Phase: 4-Assignments & Quizzes*
*Context gathered: 2026-06-20*
