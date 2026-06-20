# Roadmap: myIU (Lite Edition)

## Overview

myIU lite is built bottom-up along its dependency arrows: a proven-blocking CI gate and the relational data core come first, then the auth + RBAC + forced-first-login spine that every other route hangs off, then the admin provisioning that creates the users/courses/enrollments the whole app reads (plus the course-lifecycle sweep). Only then do the course-scoped features land — assignment submission and auto-graded quizzes (the first Cloudinary + answer-key-safety work), followed by the weighted gradebook, announcements, and student↔lecturer requests, all sharing one persisted-fan-out notification primitive. Five existential security/correctness pitfalls (server-enforced forced reset, authenticated Cloudinary delivery, magic-byte upload validation, quiz answer non-leakage, append-only audit + soft-delete discipline) are designed into the *first* implementation of their feature, never deferred to a hardening pass.

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Foundation & Data Core** - Monorepo, Docker Postgres, migrations, .env, and a CI gate proven to block un-mergeable PRs
- [x] **Phase 2: Auth, RBAC & Forced First-Login** - Login/logout, server-enforced password change, role + ownership authorization
- [x] **Phase 3: Admin Provisioning & Course Lifecycle** - CSV accounts/enrollment, course CRUD, append-only audit log, auto soft-delete sweep
- [x] **Phase 4: Assignments & Quizzes** - File-upload submission + grading and auto-graded MCQ quizzes, on a shared notification primitive
- [ ] **Phase 5: Gradebook, Announcements & Requests** - Weighted grades, announcement fan-out, and student↔lecturer requests with auto-delivered replies

## Phase Details

### Phase 1: Foundation & Data Core

**Goal**: A running monorepo skeleton where Postgres comes up via Docker, schema is migration-managed, config loads from `.env`, and the CI gate is proven to block a deliberately-failing PR.
**Mode:** mvp
**Depends on**: Nothing (first phase)
**Requirements**: INFRA-01, INFRA-02, INFRA-03, INFRA-04, INFRA-05, INFRA-06, INFRA-07
**Success Criteria** (what must be TRUE):

  1. A developer can clone the repo, run one Docker command, and have `backend/` + `frontend/` plus a Postgres database running with all migrations applied.
  2. Backend reads DB, JWT secret, and Cloudinary credentials from `.env` with no hardcoded secrets.
  3. Pushing to `main`/`backend`/`frontend` triggers GitHub Actions that runs unit + integration tests against a real Postgres service container.
  4. A pull request that deliberately fails a test or build is blocked from merging by a required status check (verified, not just configured).

**Plans**: 3 plans
**Wave 1**

- [x] 01-01-PLAN.md — Walking skeleton: Postgres-only compose, migrations + bootstrap-admin seed, `.env` config, sqlc, Gin `/healthz`, frontend stub (INFRA-01/02/03/04)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 01-02-PLAN.md — GitHub Actions `ci` workflow: services Postgres, migrate-before-test, lint + frontend build (INFRA-05/06)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 01-03-PLAN.md — Merge-block proof: user branch-protection setup + throwaway failing-PR evidence (INFRA-07)

### Phase 2: Auth, RBAC & Forced First-Login

**Goal**: Any user can log in and receive a role-carrying JWT, must change a default password before doing anything else, and every route is gated by role and ownership.
**Mode:** mvp
**Depends on**: Phase 1
**Requirements**: AUTH-01, AUTH-02, AUTH-03, AUTH-04, AUTH-05
**Success Criteria** (what must be TRUE):

  1. A user can log in with username + password, receive a JWT carrying their role, and log out.
  2. A logged-in user can change their own password.
  3. A user flagged `must_change_password` is server-side restricted to only change-password/logout until they reset it — bypassing the SPA does not unlock other endpoints.
  4. Requests to endpoints outside a user's role, or against records they don't own, are rejected with 403 (role gate + ownership check, never trusting client-supplied IDs).

**Plans**: 3 plans
**Wave 1**

- [x] 02-01-PLAN.md — Login slice: migration 000003 (password_changed_at), sqlc user queries, JWT helpers, CORS, AuthMiddleware skeleton, config; /auth/login + /auth/me + /auth/logout; FE stack bootstrap + Login + role landing pages (AUTH-01/02)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 02-02-PLAN.md — Forced first-login slice: AuthMiddleware step-5 allow-list (D-15), /auth/change-password (D-17/18/19), session-kill on change; FE ChangePassword + post-login redirect (AUTH-03/04)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 02-03-PLAN.md — RBAC slice: RequireRole (403 role_forbidden), /auth/refresh with password_changed_at kill-switch, self-ownership pattern; FE 401 refresh interceptor + ProtectedRoute + RoleGuard + AppLayout + role trees (AUTH-05/01)

**UI hint**: yes

### Phase 3: Admin Provisioning & Course Lifecycle

**Goal**: Admin can provision the entire term — accounts, enrollment, courses — from CSV or UI, every mutation is audit-logged, and stale courses auto-soft-delete.
**Mode:** mvp
**Depends on**: Phase 2
**Requirements**: ADMIN-01, ADMIN-02, ADMIN-03, ADMIN-04, ADMIN-05, ADMIN-06, ADMIN-07, ADMIN-08
**Success Criteria** (what must be TRUE):

  1. Admin can create student/lecturer accounts manually or by uploading a CSV that is validated whole-file, all-or-nothing, with a per-row error report and no partial inserts.
  2. New accounts default to username = ID and password = birthday `DDMMYYYY` with the forced-change flag set; admin can reset any user back to that default.
  3. Admin can CRUD courses (start/end dates) and assign students + lecturers to a course from a CSV list (idempotent enrollment).
  4. Every admin mutation writes an append-only audit row (actor, action, target, timestamp) that cannot be edited or deleted.
  5. Courses are automatically soft-deleted one month after their end date with no manual action, and each sweep is itself audit-logged.

**Plans**: 4 plans

**Wave 1**

- [x] 03-01-PLAN.md — Foundation slice: migrations 000004 (users cols, courses, membership tables, audit_log cols) + 000005 (append-only triggers + SYSTEM seed), writeAudit helper, read-only audit viewer, admin sidebar shell (ADMIN-08; D-33/34/35/36/38/41)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 03-02-PLAN.md — Accounts slice: manual create + all-or-nothing CSV import (422 per-row errors) + admin reset to default DDMMYYYY, Accounts admin page (ADMIN-01/02/03/04; D-24/25/26/27)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 03-03-PLAN.md — Courses + sweep slice: course CRUD (soft-delete only), read-only roster page (Overview/Students/Lecturers), in-process daily+startup sweep audit-logged under SYSTEM (ADMIN-05/07; D-28/29/37/39/40/42)

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 03-04-PLAN.md — Enrollment & assignment slice: idempotent all-or-nothing per-course student/lecturer CSV import + UI-only individual removal from roster (ADMIN-06; D-30/31/32/43)

**UI hint**: yes

### Phase 4: Assignments & Quizzes

**Goal**: Lecturers can run graded coursework — file-upload assignments with late policy and auto-graded MCQ quizzes — and students are auto-notified of results via a shared persisted notification primitive.
**Mode:** mvp
**Depends on**: Phase 3
**Requirements**: ASMT-01, ASMT-02, ASMT-03, ASMT-04, ASMT-05, ASMT-06, QUIZ-01, QUIZ-02, QUIZ-03, QUIZ-04, QUIZ-05, QUIZ-06, NOTIF-01, NOTIF-02
**Success Criteria** (what must be TRUE):

  1. A lecturer can create an assignment with a deadline and an accept-late/threshold policy; a student can submit a single PDF or ZIP (≤10MB, magic-byte validated server-side), and the server enforces the late policy by its own timestamp.
  2. Submitted files are stored on Cloudinary as authenticated (non-public) assets and are only downloadable through backend-generated short-lived signed URLs gated by role/ownership.
  3. A lecturer can grade a submission, and saving the grade auto-notifies the student in the same transaction.
  4. A lecturer can create an MCQ quiz (CSV or UI questions; configurable max questions, max grade, shuffle, retake count); the take-quiz API never exposes which option is correct, and shuffle preserves the correct-answer mapping by stable option ID.
  5. The system auto-grades a quiz on submit (idempotent per attempt), records the score, and enforces the configured retake limit with attempts tracked distinctly.

**Plans**: 4 plans

**Wave 1**

- [x] 04-01-PLAN.md — Cloudinary client + Wave-1 spike (retires A1) + migration 000006 (all 8 tables) + assignment create/submit (magic-byte, 10MB, late policy, versioned, authenticated) + signed-URL download (ASMT-01/02/03/04/05; D-44/45)

**Wave 2** *(blocked on Wave 1)*

- [x] 04-02-PLAN.md — Assignment grade + notification primitive: same-transaction grade+notify, one-row-per-recipient notifications, NotificationBell + list page (ASMT-06, NOTIF-01/02; D-46/53/54/55)

**Wave 3** *(blocked on Wave 1; sequenced after Wave 2 — shared wiring files)*

- [x] 04-03-PLAN.md — Quiz authoring: config (pool/max/shuffle/retake/window) + CSV + UI questions, stable option IDs, StudentOptionView DTO boundary (QUIZ-01/02/04; D-47/48/49)

**Wave 4** *(blocked on Wave 3)*

- [x] 04-04-PLAN.md — Quiz take/attempt state machine: consume-on-start/resume, M-of-N shuffle, idempotent auto-grade, window-bound reveal, lazy AUTO_SUBMITTED, retake limit, MAX score (QUIZ-03/04/05/06; D-50/51/52)

**UI hint**: yes

### Phase 5: Gradebook, Announcements & Requests

**Goal**: A course can be run to completion without email — lecturers compute weighted final grades, broadcast or target announcements, and answer student requests, all auto-delivered to the right students.
**Mode:** mvp
**Depends on**: Phase 4
**Requirements**: GRADE-01, GRADE-02, GRADE-03, GRADE-04, GRADE-05, ANNC-01, ANNC-02, ANNC-03, REQ-01, REQ-02, REQ-03
**Success Criteria** (what must be TRUE):

  1. A lecturer can configure a course grade scheme as Inclass + Midterm + Final summing to 100%, with Inclass sub-components summing to 100% of Inclass, and enter Midterm/Final manually.
  2. The system computes each student's weighted overall grade, and a student can view their grades for a course with availability auto-notified.
  3. A lecturer can send an announcement to all enrolled students or to specific ones, and the targeted students see it persisted on next login (no email).
  4. A student can send a leave-early / absence / custom request to their course's lecturer, and the lecturer's yes/no reply is auto-delivered back to that student.

**Plans**: TBD
**UI hint**: yes

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation & Data Core | 3/3 | Completed | 2026-06-20 |
| 2. Auth, RBAC & Forced First-Login | 3/3 | Completed | 2026-06-20 |
| 3. Admin Provisioning & Course Lifecycle | 4/4 | Completed | 2026-06-20 |
| 4. Assignments & Quizzes | 4/4 | Completed | 2026-06-20 |
| 5. Gradebook, Announcements & Requests | 0/TBD | Not started | - |
