# myIU-lite — Decision Logs

The record of **how this product evolved**: every design and engineering decision agreed during planning, with the reasoning, the design principle behind it, and the trade-off / technical debt accepted. This is the source of record — code comments and commits show *what*; this shows *why*.

**76 decisions total.** Each is `Locked` unless noted. The numbered sequence `D-01 → D-67` is contiguous with no gaps; nine unnumbered project-level stack/process decisions are catalogued here as `PD-A … PD-I`.

| Group | IDs | Theme |
|-------|-----|-------|
| Project-level | `PD-A…PD-I`, `D-01…D-05`, `D-10` | Stack, process, architecture, cross-cutting product rules |
| Phase 1 — Foundation & Data Core | `D-06…D-09` | Migrations, bootstrap admin, Docker scope, CI proof |
| Phase 2 — Auth, RBAC & Forced First-Login | `D-11…D-23` | Cookie JWT, stateless invalidation, forced reset, RBAC routing |
| Phase 3 — Admin Provisioning & Course Lifecycle | `D-24…D-43` | CSV provisioning, audit log, soft-delete + sweep, membership |
| Phase 4 — Assignments & Quizzes | `D-44…D-55` | Versioned submissions, question banks, attempts, notifications |
| Phase 5 — Gradebook, Announcements & Requests | `D-56…D-67` | Weighted gradebook, publication snapshots, fan-out, directed requests |

> Format per entry: **Decision** (what) · **Why** (rationale) · **Principle** (the one-line design principle, where the author stated one) · **Trade-off** (what was given up / debt) · **Status**.

---

## Project-level decisions

### PD-A — Go + Gin + PostgreSQL backend
- **Decision:** Backend is Go + Gin on PostgreSQL.
- **Why:** Chosen stack; Gin is a lightweight, common Go web framework.
- **Trade-off:** Commits to Go's relatively thin web/ORM ecosystem.
- **Status:** Locked.

### PD-B — ORM optional, raw SQL allowed
- **Decision:** No mandated ORM; raw SQL acceptable. Realized as **sqlc + pgx** — you write SQL, type-safe Go is generated.
- **Why:** Go ORM ecosystem is thin; avoid forcing a poor fit.
- **Principle:** "raw SQL but type-safe."
- **Trade-off:** Hand-written SQL + a codegen step instead of active-record ergonomics.
- **Status:** Locked.

### PD-C — React + Zustand + shadcn/ui frontend
- **Decision:** React + Zustand (UI/auth state) + shadcn/ui; no hand-rolled components.
- **Why:** Wants ready-made, accessible, good-looking components.
- **Principle:** "no hand-rolled components."
- **Trade-off:** Server state must NOT live in Zustand (TanStack Query owns it) to avoid stale-cache bugs.
- **Status:** Locked.

### PD-D — Cloudinary for file storage
- **Decision:** Uploaded files on Cloudinary via `.env`.
- **Why:** Offload upload/storage rather than self-hosting.
- **Principle:** PDF/ZIP are "raw" assets — `ResourceType:"raw"`, never "image".
- **Trade-off:** External dependency + credentials; realized as authenticated assets + signed URLs in Phase 4.
- **Status:** Locked.

### PD-E — Default password = birthday `DDMMYYYY`, forced change
- **Decision:** New accounts default password = birthday `DDMMYYYY`, forced change on first login.
- **Why:** Simple CSV provisioning, secured by a first-login reset.
- **Trade-off:** Default passwords are guessable until first change; mitigated by the forced-change flag.
- **Status:** Locked (elaborated by `D-01`, `D-26`, `D-07`).

### PD-F — Auto soft-delete courses 1 month after end date
- **Decision:** Courses auto soft-deleted one month after end date.
- **Why:** Removes manual cleanup; soft delete keeps history.
- **Principle:** Soft delete keeps history.
- **Trade-off:** Soft-deleted rows stay in the DB indefinitely.
- **Status:** Locked (implemented by `D-37`/`D-39`/`D-40`).

### PD-G — Audit log for admin actions
- **Decision:** All admin actions recorded in an audit log.
- **Why:** Admin can change others' passwords — needs accountability.
- **Status:** Locked (implemented by `D-33…D-36`).

### PD-H — PostgreSQL via Docker only
- **Decision:** Postgres runs via Docker only, never natively.
- **Why:** Reproducible DB env, no native install drift.
- **Trade-off:** Requires Docker for any local dev/test.
- **Status:** Locked (scoped to Postgres-only compose by `D-08`).

### PD-I — Monorepo + GitHub Flow + CI gate
- **Decision:** Monorepo (`backend/` + `frontend/`). GitHub Flow: `main` is the only long-lived, protected branch; short-lived `ft/`/`fix/`/`chore/`/`docs/` branches → PR → squash-merge → delete. GitHub Actions `ci` gates merge on tests + migrations + lint/build.
- **Why:** Vertical phases span both stacks; per-stack branches forced an awkward sync. No broken code reaches protected branches.
- **Principle:** Definition of Done = green `scripts/check.sh` before push (mirrors CI locally).
- **Trade-off:** Supersedes the former 3-branch (`main`/`backend`/`frontend`) model. Admin may bypass the `ci` gate on `main` (accepted).
- **Status:** Locked.

### D-01 — No self-service forgot-password; admin resets to default
- **Decision:** No self-service recovery. Admin resets a password to the default `DDMMYYYY` with forced change next login.
- **Why:** No email/SMS channel exists, so self-service recovery is impossible.
- **Trade-off:** All recovery is admin-mediated; self-service deferred (`AUTH-V2-01`).
- **Status:** Locked.

### D-02 — Per-assignment late policy
- **Decision:** Each assignment has its own deadline + accept-late flag + optional threshold days. No global lateness rule.
- **Why:** Lecturer controls lateness per assignment.
- **Trade-off:** Late policy is not standardized across lecturers (intentional, see `D-45`).
- **Status:** Locked (consumed by `D-44`, `D-45`, `D-64`).

### D-03 — Per-quiz config
- **Decision:** Each quiz independently configured: max questions, max grade, shuffle, CSV-or-UI question source, retake count 0..N.
- **Why:** Lecturer-tunable quizzes per assessment.
- **Status:** Locked (elaborated by `D-47`, `D-48`, `D-49`, `D-52`).

### D-04 — Weighted gradebook (Inclass + Midterm + Final)
- **Decision:** Weighted gradebook: Inclass (with sub-weights) + Midterm + Final, each a % of overall; midterm/final entered manually.
- **Why:** Reflects real course grading; midterm/final are offline exams entered by hand.
- **Status:** Locked (implemented by `D-56…D-67`).

### D-05 — One project-wide DESIGN-SYSTEM.md
- **Decision:** One global `.planning/DESIGN-SYSTEM.md` instead of per-phase UI-SPECs: shadcn/ui-only, light + dark, 6px radius, Lucide, WCAG AA.
- **Why:** A single global UI ruleset, not per-phase specs.
- **Status:** Locked.

### D-10 — Feature-Oriented Monolith architecture
- **Decision:** Backend organized by business feature, each with `handler/service/repository/model/dto`; cross-cutting infra under `internal/shared/`. Inside a feature: handler = HTTP only, service = business + authz, repository = SQL only. Cross-feature deps allowed when business rules require; no hard bounded-context enforcement, no abstractions for purity.
- **Why:** Optimizes feature discoverability, AI/graph navigation, and small-team maintainability over module-isolation purity.
- **Principle:** Navigate by "which feature?" before "which layer?"; add an interface only where a real second implementation or test seam needs it (Ponytail).
- **Trade-off:** Supersedes the layered layout in `research/ARCHITECTURE.md`. Numbered `D-10` deliberately, leaving `D-06…D-09` to Phase 1.
- **Status:** Locked.

---

## Phase 1 — Foundation & Data Core (D-06…D-09)

### D-06 — Incremental per-phase migrations
- **Decision:** Migrations are incremental per-phase, not big-bang. Phase 1 creates only foundational tables (`users` + `audit_log`); each later phase adds its own migration.
- **Why:** Lean, less rework than creating all tables up front.
- **Principle:** "the DB will grow alongside the codebase."
- **Trade-off:** Schema is fragmented across many migrations.
- **Status:** Locked.

### D-07 — Seed one bootstrap admin
- **Decision:** Seed one bootstrap admin (username `admin`, password `123456`, bcrypt cost 12, `must_change_password=TRUE`) via migration.
- **Why:** Avoids the chicken-and-egg problem (no account exists to create accounts).
- **Principle:** Never seed plaintext; force reset on first login.
- **Trade-off:** A bootstrap credential lives in a migration; rotated on first login once Phase 2 ships.
- **Status:** Locked.

### D-08 — Docker runs Postgres only
- **Decision:** `docker compose` runs Postgres only (`postgres:17-alpine`); Go/Vite run natively in dev; migrations apply via a separate command.
- **Why:** Keeps hot-reload fast; avoids extra Dockerfiles.
- **Trade-off:** No full-stack one-command bring-up (deferred to a deploy story).
- **Status:** Locked.

### D-09 — Prove the CI merge-block with a throwaway failing PR
- **Decision:** Prove the merge-block by opening a throwaway PR that deliberately fails, capturing the blocked merge as evidence, then closing it. Plan includes a branch-protection setup guide.
- **Why:** Satisfies "verified, not just configured."
- **Trade-off:** Some setup is manual GitHub admin action.
- **Status:** Locked.

---

## Phase 2 — Auth, RBAC & Forced First-Login (D-11…D-23)

### D-11 — Cookie storage, not localStorage
- **Decision:** Tokens in HttpOnly + Secure + SameSite=Lax cookies; axios `withCredentials`; credential-aware CORS (exact origin).
- **Why:** Holds grades/submissions/admin actions; HttpOnly blocks JS from reading the token (shrinks XSS impact).
- **Principle:** Security-over-simplicity for a business app with sensitive data.
- **Trade-off:** Accepts CORS-credentials + CSRF complexity (overrode the localStorage+Bearer "recommended" option).
- **Status:** Locked.

### D-12 — Access + refresh tokens
- **Decision:** Access = 15 min, refresh = 7 days, both HttpOnly cookies; `POST /auth/refresh` mints a new access token.
- **Why:** Smoother UX than one long-lived token with re-login on expiry.
- **Trade-off:** More moving parts than a single stateless token.
- **Status:** Locked.

### D-13 — Stateless logout via `password_changed_at`, no blacklist
- **Decision:** Logout clears cookies; no JWT blacklist/Redis/session table. Any JWT whose `iat` predates `users.password_changed_at` is rejected — so password change / admin reset / forced reset kill prior sessions with no server-side store.
- **Why:** Stateless invalidation without Redis/blacklist.
- **Principle:** "Stateless" = no session store, not no DB read.
- **Trade-off:** Auth middleware loads the user row per request (accepted). Non-goals: blacklist, device mgmt, global logout.
- **Status:** Locked.

### D-14 — Forced-reset enforced via live DB flag, not a JWT claim
- **Decision:** Middleware reads `must_change_password` live from the user row it already loads. JWT carries identity; mutable state read from DB. Order: verify JWT → load user → account status → `password_changed_at` → `must_change_password`.
- **Why:** No extra query, always live, no stale-claim/reissue problems.
- **Principle:** JWT = identity; mutable state = DB.
- **Status:** Locked.

### D-15 — Locked-state allow-list = exactly three endpoints
- **Decision:** While `must_change_password=true`, only `POST /auth/change-password`, `POST /auth/logout`, `GET /auth/me` are reachable; everything else returns 403 with a machine-readable code.
- **Why:** Strict — bypassing the SPA must not unlock other endpoints.
- **Principle:** Server-side enforcement, not client-trust.
- **Status:** Locked.

### D-16 — Password change ends the session; force re-login
- **Decision:** On change: update hash → `password_changed_at` → clear `must_change_password` → clear cookies → 200 "log in again". No auto-reissue.
- **Why:** A new credential should establish a new session; the one-time onboarding login is acceptable.
- **Principle:** Credential change establishes a new session.
- **Trade-off:** Slightly less seamless than landing on the dashboard.
- **Status:** Locked.

### D-17 — Require current password on change
- **Decision:** Body = `current_password`, `new_password`, `confirm_password`; verify current before accepting.
- **Why:** Possessing a session isn't enough — prove knowledge of the current credential (blocks stolen-session change).
- **Trade-off:** Slightly more friction than new-password-only.
- **Status:** Locked.

### D-18 — Minimum length 6, no complexity rules
- **Decision:** Min length 6; no composition rules.
- **Why:** Friction-free for an internal app with admin-assisted recovery; matches the bootstrap `123456`; length over composition.
- **Principle:** Length over composition.
- **Trade-off:** Weaker on paper than ≥8 + complexity (intentional).
- **Status:** Locked.

### D-19 — New password must differ from current; no history
- **Decision:** New must differ from current (`bcrypt.CompareHashAndPassword`); no "last N" history.
- **Why:** Prevents re-setting the default birthday password at forced change.
- **Trade-off:** No password-history protection (out of MVP scope).
- **Status:** Locked.

### D-20 — Separate role route trees on the frontend
- **Decision:** `/student/*`, `/lecturer/*`, `/admin/*` trees; redirect to the role root after login. Each tree owns its nav/pages; shared shadcn primitives stay reusable.
- **Why:** Clear boundaries; roles separated at the routing boundary, not via in-page conditionals.
- **Principle:** Separate roles at the routing boundary.
- **Trade-off:** Some shell duplication across trees.
- **Status:** Locked.

### D-21 — Minimal Phase-2 app shell only
- **Decision:** Phase 2 ships Login, Change-Password, role landing pages, `AppLayout`, `ProtectedRoute`, `RoleGuard`. No sidebar/feature menus/placeholder pages.
- **Why:** Navigation arrives when real feature pages exist (Phase 3+); no links to nonexistent pages.
- **Trade-off:** Sidebar deferred (→ realized in `D-41`).
- **Status:** Locked.

### D-22 — Transparent 401 refresh on the frontend
- **Decision:** axios interceptor catches 401 → calls `/auth/refresh` once → retries once; on failure clears state + redirects to `/login`. One refresh per request, concurrent 401s share one in-flight refresh.
- **Why:** Smooth UX that actually uses the refresh token vs logging out on any 401.
- **Trade-off:** Interceptor complexity (loop/concurrency guards).
- **Status:** Locked.

### D-23 — No dedicated CSRF token for the MVP
- **Decision:** No CSRF token. Rely on HttpOnly + SameSite=Lax cookies, strict CORS (exact origin, credentials, no wildcard), JSON-only state-changing endpoints. Same for `/auth/refresh`.
- **Why:** Single origin + cookie auth where SameSite=Lax blocks cross-site POST cookies — a token would add real plumbing for limited benefit at this threat model.
- **Principle:** "browser-native protections first; add a dedicated CSRF mechanism only when the threat model or deployment requires it."
- **Trade-off:** No defense if the deployment later goes multi-origin.
- **Status:** Locked.

---

## Phase 3 — Admin Provisioning & Course Lifecycle (D-24…D-43)

### D-24 — Separate CSV files for students vs lecturers
- **Decision:** Two imports, each validated against its own schema. Student CSV: `student_id, full_name, dob (DD/MM/YYYY)`; Lecturer CSV: `lecturer_id, full_name, dob`. `username = ID`; `password = dob` as `DDMMYYYY`; `must_change_password=TRUE`.
- **Why:** Student & lecturer data come from different sources/schemas; separate flows simplify validation and errors.
- **Principle:** Explicit business workflows over a generic import mechanism.
- **Trade-off:** Note the deliberate `DD/MM/YYYY` (dob) vs `DDMMYYYY` (derived password) distinction.
- **Status:** Locked.

### D-25 — Store `full_name`; no email for MVP
- **Decision:** Store `full_name`; omit email.
- **Why:** Names are needed across lists/announcements/requests/grades; email omitted because the system sends none and auth is username-based.
- **Trade-off:** No email channel anywhere (consistent with `D-01`).
- **Status:** Locked.

### D-26 — Store `date_of_birth (DATE)` for every account
- **Decision:** Store DOB so admin reset auto-regenerates the default `DDMMYYYY` with no input (one-click, deterministic).
- **Why:** Enables deterministic one-click reset; reuses the `password_changed_at` kill-switch.
- **Principle:** Never store plaintext passwords, never re-display generated ones.
- **Trade-off:** Stores PII (DOB) solely for password derivation.
- **Status:** Locked.

### D-27 — All-or-nothing CSV validation
- **Decision:** Validate the whole file before creating anything. On ≥1 error: no writes, HTTP 422 with `{errors:[{row,field,message}]}`; FE renders a table.
- **Why:** Partial imports create inconsistent states; all-or-nothing is deterministic and retry-safe.
- **Trade-off:** Whole file rejected on a single bad row (intended). This is the reusable CSV discipline (reused by `D-30`/`D-32`/`D-67`).
- **Status:** Locked.

### D-28 — Course identity fields
- **Decision:** `code, name, term, start_date, end_date` (+ `deleted_at`); no `description` in MVP. `term` (e.g. `2026.1`) is first-class.
- **Why:** `term` is a real academic concept distinct from dates.
- **Trade-off:** No course `description` in MVP.
- **Status:** Locked.

### D-29 — Course deletion is soft-delete only
- **Decision:** Admin "delete" sets `deleted_at` and excludes the course from normal queries; no physical delete.
- **Why:** Preserves referential integrity + academic history; aligns with the sweep.
- **Principle:** Preserve history over destructive cleanup.
- **Trade-off:** Soft-deleted rows stay indefinitely (archival policy deferred).
- **Status:** Locked.

### D-30 — Student enrollment import: per-course, additive, idempotent
- **Decision:** Per-course one-column `student_id` CSV; already-enrolled rows ignored; imports never remove. All-or-nothing (`D-27`).
- **Why:** Enrollment lists arrive after registration and may be re-imported for late registrations.
- **Trade-off:** CSV cannot remove enrollments — removal is a separate UI action (`D-43`).
- **Status:** Locked.

### D-31 — Membership model = two separate tables
- **Decision:** `student_enrollments(course_id, student_id)` and `course_lecturers(course_id, lecturer_id)`. A course has ≥1 lecturers and ≥0 students.
- **Why:** Enroll vs assign are different processes; myIU manages membership after registration is decided elsewhere.
- **Trade-off:** Two tables instead of one polymorphic membership table.
- **Status:** Locked.

### D-32 — Lecturer assignment mirrors D-30
- **Decision:** Per-course one-column `lecturer_id` CSV; additive + idempotent; all-or-nothing.
- **Why:** Equivalent operations behave consistently.
- **Principle:** Equivalent operations behave consistently.
- **Trade-off:** Removal is UI-only (`D-43`).
- **Status:** Locked.

### D-33 — Bulk op = one audit row + `operation_id` + `affected_count`
- **Decision:** Each bulk action writes a single audit row (e.g. `affected_count=253`), not one per entity. Per-entity detail (if needed) goes in dedicated tables linked by `operation_id`.
- **Why:** Audit records business actions; operational detail stored separately.
- **Principle:** Audit records business actions; operational detail is stored separately.
- **Trade-off:** No per-entity traceability in the audit log itself.
- **Status:** Locked.

### D-34 — Audit payload; NO before/after diffs
- **Decision:** `actor, action, target_type, target_id, timestamp, metadata`; no diffs. Actions: `ACCOUNT_CREATE, PASSWORD_RESET, COURSE_CREATE/UPDATE/DELETE, ENROLL_IMPORT, LECTURER_IMPORT` (+ removals + `COURSE_SWEEP`).
- **Why:** Records that an event happened with the new state; field-level history is overkill for MVP.
- **Trade-off:** Can detect an update but not reconstruct previous field values (deferred).
- **Status:** Locked.

### D-35 — Append-only enforced by DB triggers
- **Decision:** `BEFORE UPDATE`/`BEFORE DELETE` triggers on `audit_log` raise an exception; only INSERT permitted.
- **Why:** Enforce the invariant close to the data, not just in app code.
- **Principle:** Enforce invariants close to the data.
- **Trade-off:** No dedicated DB roles / privilege separation in MVP (defense-in-depth deferred).
- **Status:** Locked.

### D-36 — Admin audit-log viewer ships this phase
- **Decision:** Read-only admin page: pagination, filter by actor/action/date, view metadata. Admin-only, presentation only.
- **Why:** Audit data is low-value if admins can't inspect it; also validates audit generation.
- **Status:** Locked.

### D-37 — In-process daily sweep + startup catch-up
- **Decision:** A Go-native daily job + a catch-up sweep at startup. Find courses past `end_date` by ≥1 month, set `deleted_at`, ignore already-deleted; idempotent. No Redis/queues.
- **Why:** Simplest sufficient mechanism for a single-instance app.
- **Trade-off:** Offline → sweeps delayed to next startup/run. Multi-instance hardening deferred.
- **Status:** Locked.

### D-38 — Dedicated SYSTEM account is the actor for automated actions
- **Decision:** A seeded SYSTEM account is the actor for automated actions (e.g. `COURSE_SWEEP`), not a NULL actor. Cannot log in; exists for audit attribution; rows append-only.
- **Why:** Explicit attribution + referential integrity; generalizes to all future jobs.
- **Principle:** Every action has a traceable actor.
- **Trade-off:** One special non-human `users` row; realized as an `is_system` flag (non-loginable, excluded from listings + the active-username index).
- **Status:** Locked.

### D-39 — Sweep audit granularity follows D-33
- **Decision:** One audit row (`COURSE_SWEEP, actor=SYSTEM, affected_count=N`) only when ≥1 course is affected; none on a no-op day.
- **Why:** No business change → no audit row.
- **Principle:** Audit records business change, not process execution.
- **Trade-off:** Audit alone can't prove the scheduler ran on a no-op day (use logs/metrics).
- **Status:** Locked.

### D-40 — Soft-delete does NOT cascade
- **Decision:** Only `courses.deleted_at` is set; dependents stay unchanged. Normal queries exclude soft-deleted courses, so dependents are naturally hidden.
- **Why:** Avoid cascading deletes; rely on the active-course gate.
- **Principle:** Reads filter active courses; dependents hidden via the gate.
- **Trade-off:** Orphaned historical relationships remain (future archival, no cascade).
- **Status:** Locked.

### D-41 — Phase 3 introduces the first full admin sidebar
- **Decision:** First full admin sidebar (shadcn, collapsible, light+dark): Dashboard; Accounts; Courses / Student Enrollment / Lecturer Assignment; Audit Logs.
- **Why:** Real feature pages now exist, so navigation arrives. Fulfills `D-21`.
- **Status:** Locked.

### D-42 — Read-only course detail / roster page
- **Decision:** Course detail with tabs: Overview, Students, Lecturers. Admin views all rosters.
- **Why:** Bulk CSV imports need a verification surface.
- **Trade-off:** Roster is admin-only this phase (lecturer/student visibility later).
- **Status:** Locked.

### D-43 — Manual membership removal ships this phase (UI, not CSV)
- **Decision:** Admin can remove a student / unassign a lecturer from the roster page. CSV stays additive/idempotent. Each removal writes an audit row.
- **Why:** Enrollment management is incomplete without a correction path.
- **Principle:** CSV is a pure additive/idempotent channel; destructive corrections require explicit per-row UI intent.
- **Trade-off:** Slight scope increase (accepted).
- **Status:** Locked.

---

## Phase 4 — Assignments & Quizzes (D-44…D-55)

### D-44 — Versioned submissions; never overwrite
- **Decision:** Each submission creates a new version while the window is open (deadline + late config, `D-02`); prior versions preserved. Latest version is graded by default. After close, new/resubmissions rejected.
- **Why:** Academic submissions are historical records — traceability, dispute resolution, no data loss.
- **Principle:** Versioning over overwriting.
- **Trade-off:** Extra metadata (accepted; files live in Cloudinary). Edge case: a newer version submitted after grading while the window is open (see `D-64`/Phase-5 review).
- **Status:** Locked.

### D-45 — Late submissions are flagged only; no automatic penalty
- **Decision:** A late-but-in-window submission records `is_late`, `submitted_at`, human-readable `late_duration`, surfaces a "late by X" indicator. No auto penalty.
- **Why:** The system records facts; the lecturer makes the academic judgment.
- **Principle:** "the system records facts, the lecturer makes academic judgments."
- **Trade-off:** Late policy not standardized/automated across lecturers (intentional).
- **Status:** Locked.

### D-46 — Grading inputs: score required, feedback optional
- **Decision:** Grading requires a score, allows optional feedback; students view both. Applies to the active version.
- **Why:** Mandatory feedback adds workload with limited value in large classes; optional still serves exceptional cases.
- **Principle:** "required data supports the primary workflow; optional data supports exceptional cases without taxing routine ones."
- **Status:** Locked.

### D-47 — Question-bank model with random per-attempt selection
- **Decision:** A pool of N questions; each attempt randomly selects M (M ≤ N). Per-quiz: Pool Size, Max Questions, Max Grade, Shuffle, Retake Count. Each retake draws a new set; Shuffle=Yes randomizes selection + question + answer order.
- **Why:** Reduces cheating, increases variety, gives retakes meaning, one master pool.
- **Principle:** "question pools generate assessments rather than store fixed ones."
- **Trade-off:** Different students may get different combinations (all from the same approved bank).
- **Status:** Locked.

### D-48 — Two authoring modes; exact-match auto-grading
- **Decision:** (1) CSV `question,A,B,C,D,correct` — 4 choices, 1 correct. (2) Manual UI: single-choice (radio) and multi-choice (checkbox). Auto-grade: single = `selected==correct`; multi = exact set match (all-or-nothing, no partial credit).
- **Why:** Bulk import stays simple; UI handles specialized questions.
- **Principle:** "prefer simple, deterministic grading for MVP while preserving future extensibility."
- **Trade-off:** Multi-choice is all-or-nothing — partial/weighted/negative marking excluded.
- **Status:** Locked.

### D-49 — Quizzes use an availability window; no late, no timer
- **Decision:** Open At / Close At. Start/submit/retake only while open; after Close At nothing new. No late submission. Review allowed after submission (per `D-51`). No per-attempt timer.
- **Why:** Quizzes are time-bounded events, not late-policy coursework.
- **Principle:** "quiz access is governed by availability windows, not late-submission policies."
- **Trade-off:** Per-attempt timers / lockdown browser excluded.
- **Status:** Locked.

### D-50 — Official quiz score = MAX across completed attempts
- **Decision:** `official_score = MAX(attempt_scores)`; the gradebook stores the official score, full history stays available. Attempts bounded by window (`D-49`) + retake count (`D-03`).
- **Why:** Retakes support learning — highest-score rewards mastery, matches common LMS expectations.
- **Principle:** Retakes are a learning mechanism, not a penalty.
- **Trade-off:** Students may use early attempts as practice (accepted).
- **Status:** Locked (feeds the Phase-5 AUTO Quiz Average).

### D-51 — Answer-reveal policy is window-bound, not attempt-bound
- **Decision:** While open, a reviewed attempt shows score, the student's answers, per-question correct/incorrect — but NOT correct answers/explanations. After close, correct answers + per-question results become visible. Regardless of remaining retakes.
- **Why:** Revealing answers before close would let early finishers share them; integrity is tied to the window.
- **Principle:** "assessment integrity is governed by the quiz window, not individual attempt status."
- **Trade-off:** Delayed correct-answer feedback (accepted); enforced server-side off `close_at`.
- **Status:** Locked.

### D-52 — Attempt consumed on START; resumable; auto-submit on close
- **Decision:** A new attempt is created and consumes one available attempt on **start**. States: `IN_PROGRESS`, `SUBMITTED`, `AUTO_SUBMITTED`. While `IN_PROGRESS` the student may leave/return (resume) but not start another. Submit → `SUBMITTED`; window closes while in progress → `AUTO_SUBMITTED`.
- **Why:** Consuming on start prevents opening quizzes just to inspect the pool; resume protects against crashes.
- **Principle:** "opening an assessment is participation; participation creates an attempt."
- **Trade-off:** Starting consumes a retake even if few questions are answered (accepted — content was exposed).
- **Status:** Locked.

### D-53 — Notifications persist fully-rendered content at creation
- **Decision:** Each row: `recipient_id, type, title, body, resource_type?, resource_id?, link?, created_at, read_at`. Title + body rendered at creation and stored. One row per recipient; `read_at` is the read marker. Stays readable even if the resource is later archived/changed.
- **Why:** Notifications are historical events, not live projections — simpler reads, no broken notifications.
- **Principle:** "notifications are historical records that stay stable when resources change."
- **Trade-off:** Text duplicated in the DB (accepted; small). Templates/localization deferred.
- **Status:** Locked (the shared primitive Phase 5 reuses).

### D-54 — Centralized notification center (bell in header)
- **Decision:** Header bell with unread badge, a list page, mark-as-read on click, deep-link to the resource. One center aggregates all sources.
- **Why:** A single center brings users to information instead of making them search.
- **Principle:** "notifications should bring users to information, not require them to search for it."
- **Trade-off:** Out of scope: real-time push, dropdown previews, categories, preferences.
- **Status:** Locked.

### D-55 — Phase 4 notifications fire only on assignment grading
- **Decision:** Saving an assignment grade persists the grade AND creates a student notification **in the same transaction** (NOTIF-02). Quiz grading does NOT notify (scores shown inline).
- **Why:** Assignment grading is asynchronous (notify); quiz grading is synchronous (inline) → a notification would be redundant.
- **Principle:** "notify only about genuinely new information; don't duplicate information already presented directly."
- **Trade-off:** Announcement/grade-publish/request notifications arrive in Phase 5 without architecture change. (Phase-5 grade-publish DOES notify per `D-59`, contrasting with quiz's synchronous no-notify.)
- **Status:** Locked.

---

## Phase 5 — Gradebook, Announcements & Requests (D-56…D-67)

### D-56 — Hierarchical weighted-component gradebook (not flat columns)
- **Decision:** Overall = Inclass + Midterm + Final = 100%. Composite components contain sub-components summing to 100% of the parent. Each leaf is AUTO (Quiz/Assignment Average) or MANUAL (Project, Lab, Participation, Bonus, Midterm, Final).
- **Why:** Real-university-aligned; a foundation for future policies (best-N, drop-lowest) without redesign.
- **Principle:** "the gradebook computes grades, it does not merely store them."
- **Trade-off:** Server-side sum-to-100 validation at each level; advanced aggregation deferred.
- **Status:** Locked (implements `D-04`).

### D-57 — Single 0–100 scale; normalize first, aggregate second
- **Decision:** All computation on 0–100. AUTO normalizes before aggregating: Quiz Average = `avg(score/quiz_max × 100)`; Assignment Average = `avg(score/assignment_max × 100)`. MANUAL entered on 0–100 (validate 0..100).
- **Why:** Consistent scale + correct normalization across heterogeneous coursework.
- **Principle:** "normalize first, aggregate second."
- **Trade-off:** ⚠ Forces a Phase-4 schema touch — `assignments` had no `max_score`; migration `000008` adds it (`NOT NULL DEFAULT 100`) + updates the create form/handler/DTO. Configurable scale out of MVP.
- **Status:** Locked.

### D-58 — AUTO includes all eligible items; missing = zero
- **Decision:** Quiz/Assignment Average include all eligible items; a student who didn't attempt/submit gets 0 for that item and it stays in the set. Only eligible (per `D-64`) items participate.
- **Why:** Deterministic, no per-student lecturer intervention.
- **Principle:** "missing assessment is a grade of zero, not absence of data."
- **Trade-off:** Refined by `D-64` — not "missing=0" until an item is finalized & eligible.
- **Status:** Locked.

### D-59 — Component-level publication; grades hidden until published
- **Decision:** Publish each top-level component independently. Before publish: only the lecturer sees scores. After: students see that component + one grade notification per affected student. Overall visible only once all required top-level components are published. Editing a published score doesn't auto-notify unless republished.
- **Why:** Real grading releases components at different times; lecturer controls disclosure.
- **Principle:** "grades exist before students can see them; publication is a separate academic action."
- **Trade-off:** "Published coursework" (`D-58` eligibility) ≠ "published grade component" (`D-59` release) — must not be conflated.
- **Status:** Locked.

### D-60 — Announcements are first-class entities + fan-out delivery
- **Decision:** An `announcements` row (`id, course_id, author_id, title, body, audience_type, created_at`), `audience_type ∈ {ALL_STUDENTS, SPECIFIC_STUDENTS}`; SPECIFIC stores targets in a join table. Creating one fans out notification rows. Two surfaces: a per-course Announcements page + the bell.
- **Why:** Keep announcements as durable content, notifications only as delivery.
- **Principle:** "announcements are content; notifications are delivery."
- **Trade-off:** Recipients snapshotted at send time (`D-53`); later enrollment changes don't retroactively change delivered notifications.
- **Status:** Locked.

### D-61 — Announcements are immutable after sending
- **Decision:** No edit/delete/recipient-change. Corrections = a new announcement. Lifecycle is CREATED only.
- **Why:** An announcement is a notice, not a document.
- **Principle:** "an announcement is a notice, not a document."
- **Trade-off:** ⚠ Drops `updated_at` from the `D-60` sketch. No scheduled send / read receipts.
- **Status:** Locked (supersedes the `updated_at` column sketched in `D-60`).

### D-62 — Requests are directed to one student-chosen lecturer
- **Decision:** The student chooses a specific lecturer on the course; the request is visible only to the requesting student and the targeted lecturer. Other course lecturers don't see it.
- **Why:** Eliminates shared-inbox races / first-reply-wins; academic responsibility handled outside the system.
- **Principle:** "requests have a clear owner; communication is directed, not broadcast."
- **Trade-off:** Shared inbox / reassignment between co-lecturers excluded.
- **Status:** Locked.

### D-63 — Reply = required Decision + optional Note; one round-trip
- **Decision:** A request (leave-early/absence/custom, title + body) starts PENDING; the lecturer replies APPROVED or DENIED (required) + optional note, then it's closed permanently (no reopen/thread). The reply auto-generates a student notification in the same transaction (NOTIF-02).
- **Why:** A clean one-decision lifecycle; consistent with `D-46`.
- **Principle:** "a decision is required; an explanation is encouraged, not required."
- **Trade-off:** Conversation threads / reopen / multi-reply excluded.
- **Status:** Locked.

### D-64 — AUTO eligibility = when the score is FINALIZED (not window timing)
- **Decision:** A quiz is eligible at `close_at` (auto-graded). An assignment is eligible when the lecturer **finalizes** grading — independent of deadline/accept-late/threshold. Refines `D-58`: not "missing=0" until finalized & eligible. **Eligibility ≠ publication.**
- **Why:** Accept-late assignments stay open past the deadline, so keying eligibility to finalized grading prevents not-yet-due coursework from tanking averages.
- **Principle:** "an assessment contributes when its result is finalized; eligibility ≠ publication."
- **Trade-off:** ⚠ Requires an explicit per-assignment "grading finalized" marker — `000008` adds `grading_finalized_at` (set by an explicit lecturer finalize action).
- **Status:** Locked (refines `D-58`; forces a Phase-4 touch).

### D-65 — Grade scheme is immutable once created
- **Decision:** After creation, components/hierarchy/weights/structure cannot change. Lecturers may enter scores, publish, and update unpublished scores — but not add/remove components or change weights.
- **Why:** Grade weights are approved academic policy, not operational data.
- **Principle:** "academic policy is immutable."
- **Trade-off:** Typo-recovery edge resolved: whole-scheme delete + recreate is allowed **only before any score/publication exists**.
- **Status:** Locked.

### D-66 — Published components are snapshots; computation stays live
- **Decision:** The value students see is **frozen at publish** (persisted `value` + `published_at`); lecturers always see the live recomputed value. To push a new value the lecturer **republishes** — replacing the snapshot and generating a new notification.
- **Why:** Students should see a stable, deliberately-released value; lecturers need the live truth.
- **Principle:** "publication creates a snapshot; computation remains live."
- **Trade-off:** The student view reads the persisted snapshot (`grade_publications`), not compute-on-read.
- **Status:** Locked.

### D-67 — MANUAL grade CSV: one component per file, `student_id,score`
- **Decision:** Each upload targets exactly one MANUAL component with rows `student_id,score`. Follows the Phase 3 CSV discipline: whole-file validation, all-or-nothing, row-level errors, HTTP 422, no partial imports.
- **Why:** Keeps the CSV format independent of gradebook structure.
- **Principle:** "grades are imported by component; gradebook structure must not leak into the CSV format."
- **Trade-off:** One upload per component (more files) instead of a single wide file (intentional).
- **Status:** Locked (reuses `D-27` CSV discipline).

---

## Cross-cutting notes

**Supersessions & schema consequences**
- `D-61` drops the `updated_at` column sketched in `D-60`.
- `D-64` refines `D-58` (missing=0 only once finalized & eligible).
- `D-57` and `D-64` force a Phase-4 schema touch in migration `000008` (add `assignments.max_score`; add `assignments.grading_finalized_at`).
- `D-10` supersedes the layered `research/ARCHITECTURE.md`.
- `PD-I` (GitHub Flow) supersedes the former 3-branch model.
- `D-41` fulfills the `D-21` sidebar deferral.

**Recurring design themes the user reinforced across phases**
- **Preserve history over destructive cleanup** — `D-29`, `D-40`, `D-44`.
- **Immutability of historical / academic records** — `D-35`, `D-53`, `D-61`, `D-65`, `D-66`.
- **Enforce invariants close to the data** — `D-35`.
- **The system records facts; humans make academic judgments** — `D-45`, `D-46`, `D-63`.
- **Explicit business workflows over generic mechanisms** — `D-24`, `D-43`, `D-67`.
- **Notify only about genuinely new information** — `D-55`.
- **Choose the richer, real-university-aligned option over the leaner default** — `D-56`, `D-57`, `D-59`, `D-62`.

*Sources: `.planning/PROJECT.md`, `.planning/ROADMAP.md`, and the five `NN-CONTEXT.md` / `NN-DISCUSSION-LOG.md` pairs under `.planning/phases/` (read in full).*
