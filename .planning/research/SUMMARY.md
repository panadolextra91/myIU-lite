# Project Research Summary

**Project:** myIU lite
**Domain:** LMS-lite / university student-management platform (Go+Gin+Postgres backend, React+Zustand+shadcn/ui frontend, Cloudinary storage)
**Researched:** 2026-06-19
**Confidence:** HIGH

## Executive Summary

myIU lite is a lightweight, single-deployment LMS that lets students and lecturers run a course end-to-end (assignments, quizzes, grades, announcements, requests) without falling back to email, while Admin provisions everything from CSV. The stack is already committed (Go+Gin, PostgreSQL via Docker, Cloudinary, React+Zustand+shadcn/ui, GitHub Actions, monorepo), so research focused on best practices *inside* those constraints. Experts build this exact shape as a **layered Go monolith** — one Gin API binary (handler -> service -> repository), one Postgres DB, one React SPA, with the scheduled course-cleanup sweep running in-process. No microservices, queues, or separate workers are warranted at this scale; the layering exists for testability, not distribution.

The recommended approach is opinionated and lean: use **sqlc + pgx** for the data layer (raw SQL ergonomics with compile-time type safety — the idiomatic answer to PROJECT.md's "raw SQL acceptable, Go ORM ecosystem is thin"), **golang-migrate** for versioned schema, **JWT + bcrypt(cost=12)** for stateless role-based auth, **backend-proxy uploads to Cloudinary** (so the server enforces size/type/ownership before storing), and **persisted fan-out rows** for all "automatic" notifications (no websockets, no email). On the frontend, **TanStack Query owns server state** while **Zustand holds only auth/UI state** — keeping server data out of Zustand is a hard rule. Integration tests run against a **real Postgres via testcontainers**, and the CI gate must be wired as a *required status check* (not merely present).

The dominant risk in this product is **security/correctness in the data-handling paths**, not technology choice. Five pitfalls are existential and must be designed in from the first version of their feature: (1) the forced-password-change gate must be enforced server-side via a restricted token, not in the SPA; (2) CSV import must be two-pass, all-or-nothing inside one transaction; (3) file uploads must be validated by magic bytes with a server-side size cap and must never be extracted (zip-bomb safe); (4) Cloudinary submissions must use `type: authenticated` with short-lived signed URLs gated by RBAC — never public; (5) quizzes must never ship answer keys to the client and must grade server-side by stable option ID. Layered over all of these: RBAC must check object-level ownership (not just role), the audit log must be append-only and transaction-coupled, and notifications must be written in the same transaction as their triggering event. These are not "phase 2 hardening" — they are the correct first implementation, and retrofitting any of them is HIGH-cost.

## Key Findings

### Recommended Stack

The stack is committed; research pins specific libraries and versions *within* it (see [STACK.md](STACK.md)). The headline decision is the data layer: **sqlc** (SQL -> type-safe Go codegen) over **pgx/v5**, rejecting GORM (runtime reflection, hides SQL) and raw-pgx-everywhere (too much hand-written boilerplate for an app this size). On the frontend, the key discipline is the **TanStack Query (server state) + Zustand (UI/auth state)** split.

**Core technologies:**
- **Go 1.24 + Gin v1.11.0** (stable, not the experimental v1.12.0): HTTP API framework — committed, production-safe line.
- **sqlc v1.31.1 + pgx/v5**: type-safe DB layer — raw SQL control with compile-time safety; ideal for the explicit soft-delete sweep and audit inserts.
- **PostgreSQL 17 (Docker `postgres:17-alpine`)**: datastore — committed, Docker-only.
- **golang-migrate v4.18**: versioned migrations — strong CI/CD story for the merge gate.
- **golang-jwt/jwt v5 + bcrypt(cost=12)**: stateless auth — no session store needed at this scale.
- **cloudinary-go/v2 (>=2.11)**: file storage — `ResourceType: "raw"` for PDF/ZIP; pin post-Jun-2025 security patch.
- **React 19 + Vite 6 + TypeScript 5**: SPA — shadcn/ui's current target path (Tailwind v4).
- **TanStack Query v5 + Zustand v5**: server-state + UI-state (kept strictly separate).
- **React Hook Form v7 + Zod v4 + @hookform/resolvers**: the shadcn form pattern.
- **testcontainers-go + testify**: real-Postgres integration tests satisfying the CI gate authentically.

### Expected Features

The MVP is already tightly scoped — essentially the full Active requirement set in PROJECT.md (see [FEATURES.md](FEATURES.md)). myIU-lite deliberately sits *below* Google Classroom on feature count but *above* it on the request workflow and CSV/auto-lifecycle automation that small tools lack.

**Must have (table stakes):**
- Role-based auth (Student/Lecturer/Admin) enforced server-side per endpoint — users expect this.
- Forced first-login password change (default pw = birthday `DDMMYYYY`) — must rotate immediately.
- Course roster / enrollment model — the spine every course-scoped feature keys off.
- Assignment submission (PDF/ZIP, 10MB, Cloudinary, server timestamp) + lecturer grading.
- MCQ quiz with auto-grade on submit (shuffle + max-questions) — users expect this.
- Gradebook record per student, announcements (lecturer -> enrolled), persisted in-app notifications.
- Admin provisioning (manual + CSV accounts, enrollment CSV, course CRUD) + append-only audit log.

**Should have (competitive differentiators):**
- In-app student->lecturer requests (leave-early / absence / custom) with yes/no reply + auto-notify — replaces a whole category of email.
- CSV provisioning for accounts + enrollment — set up a term in minutes.
- Auto soft-delete sweep (1 month after course end date) — removes manual cleanup, keeps history.

**Defer (v1.x / v2+):**
- Quiz multiple-attempt policy, per-assignment late/grace policy, soft-delete restore (v1.x).
- Real-time push, grade aggregation/weighting, self-service forgot-password channel, question-bank reuse (v2+).
- **Explicit anti-features (do NOT build):** email channel, essay/manual-graded quizzes, discussion forums, rubrics/plagiarism, SCORM/video, websockets, hard-delete of courses.

### Architecture Approach

A **single-deployment layered monolith** (see [ARCHITECTURE.md](ARCHITECTURE.md)): Gin API with strict handler -> service -> repository layering wired via constructor-injected interfaces in `main()`; only repositories touch SQL; ownership checks live in services. Backend-proxy file uploads, in-process cron sweep, and pull-based persisted-fan-out notifications keep the system free of queues, workers, and websockets. The relational core is `users --< enrollments >-- courses` with assignments/submissions, quizzes/questions/options/attempts, announcements/recipients, requests, and an append-only `audit_log` hanging off it; grades are best projected from `submissions`/`quiz_attempts` rather than duplicated into a separate table.

**Major components:**
1. **Handlers** — parse/validate HTTP, call one service, shape JSON. No business logic, no SQL.
2. **Services** — business rules (auto-grade, sweep, CSV provisioning, RBAC ownership checks).
3. **Repositories** — all DB access via sqlc-generated SQL; one method = a few statements.
4. **Middleware chain** — CORS -> logger -> JWT auth -> RBAC -> audit write.
5. **Cron sweep + Cloudinary adapter** — in-process daily soft-delete job; thin storage wrapper.
6. **Frontend** — role-gated React Router tree (`pages/{student,lecturer,admin}/`), single axios client with JWT interceptor, Zustand auth/UI stores.

### Critical Pitfalls

The top pitfalls are security/correctness traps that are HIGH-cost to retrofit and must be in the *first* implementation of their feature (see [PITFALLS.md](PITFALLS.md)).

1. **Forced password change enforced only in UI** — issue a *restricted token* after login while `must_change_password` is true; middleware rejects all non-(change-password/logout) endpoints with 403. Add login rate-limit/lockout for the small `DDMMYYYY` keyspace.
2. **CSV import that half-succeeds** — two-pass, all-or-nothing: Pass 1 validates every row (uniqueness, FK existence, parseable birthday, UTF-8/BOM, header-name matching) and returns a per-row report writing nothing; Pass 2 runs only if clean, inside one transaction. Make enrollment import idempotent (`ON CONFLICT DO NOTHING`).
3. **File upload trusting extension/MIME + zip bombs** — validate by magic bytes (`%PDF-`, `PK\x03\x04`), enforce 10MB via `http.MaxBytesReader` in the handler, never extract ZIPs on the server, store with random server-generated filenames.
4. **Cloudinary public-by-default leak** — upload submissions as `type: authenticated`, serve only via short-lived backend-generated signed URLs *after* an RBAC ownership check, use unguessable public IDs, keep the API secret server-side only.
5. **Quiz answer leakage + shuffle/grading bugs** — student-facing DTO carries stable option IDs but *no* correctness data; grade server-side by ID; persist each attempt's question set; unique `(student_id, quiz_id)` + idempotent submit.

Cross-cutting (apply in every relevant phase): **RBAC ownership checks** (prevent IDOR — never trust client-supplied IDs), **append-only transaction-coupled audit log** for every admin mutation incl. the sweep, **notifications written in the same transaction** as their event, and a **CI gate wired as a required status check** with a real Postgres service container.

## Implications for Roadmap

Based on combined research, the dependency arrows are unambiguous: foundation/CI -> data model -> auth -> admin provisioning (creates the data everything reads) -> course features -> cross-cutting sweep. Suggested phase structure:

### Phase 0: Scaffolding & CI Gate
**Rationale:** Blocks everything; the CI "merge blocked unless tests pass" guarantee is silently false unless wired correctly (Pitfall #10), so it must be *proven to block* before any feature sits on top of it.
**Delivers:** Monorepo `backend/`+`frontend/`, Docker Postgres + `docker-compose.yml`, `.env` config, golang-migrate, GitHub Actions with a Postgres **service container** and a **required status check** verified against a deliberately-failing PR.
**Uses:** Docker, golang-migrate, golangci-lint/ESLint, testcontainers (STACK.md).
**Avoids:** Pitfall #10 (CI guardrails that look enforced but don't block merges).

### Phase 1: Data Model & Migrations
**Rationale:** Blocks all repositories; FKs and soft-delete discipline must exist before any feature writes to these tables (Pitfall #6 is hard to bolt on).
**Delivers:** `users`, `courses`, `enrollments`, `audit_log` migrations first (rest follow per feature), with FK constraints, `deleted_at` columns, and indexes on FK/sweep columns.
**Implements:** the relational core (ARCHITECTURE.md data model).
**Avoids:** Pitfall #6 (soft-delete/enrollment referential-integrity gaps).

### Phase 2: Auth + RBAC + First-Login
**Rationale:** Everything authenticated depends on this; the restricted-token gate and the role+ownership pattern must be established here so feature phases inherit it correctly.
**Delivers:** JWT issue/verify, bcrypt hashing, login rate-limit/lockout, restricted-token middleware for `must_change_password`, role gate + ownership helpers, audit middleware; frontend auth store, axios JWT interceptor, login/change-password routes + role guards.
**Addresses:** role-based auth, forced first-login password change (FEATURES.md).
**Avoids:** Pitfalls #1 (forced-reset bypass) and #8 (RBAC ownership/IDOR holes).

### Phase 3: Admin Provisioning + Audit
**Rationale:** Produces the users/courses/enrollments the rest of the app reads; the two-pass-CSV and append-only-audit patterns are foundational and hard to retrofit.
**Delivers:** CSV + manual account import, enrollment CSV, course CRUD, audit-log view; two-pass transactional import with per-row error report; audit service writing in-transaction.
**Addresses:** admin account creation, CSV provisioning, course CRUD, audit log (FEATURES.md).
**Avoids:** Pitfalls #2 (CSV half-success) and #7 (incomplete/tamperable audit log).

### Phase 4: Course Core (student/lecturer views)
**Rationale:** Thin layer that depends on enrollments existing; unblocks all course-scoped features.
**Delivers:** view enrolled/taught courses, course pages, course-scoped navigation with ownership enforcement.
**Implements:** role-gated pages + course store (ARCHITECTURE.md frontend).

### Phase 5: Assignments + Submissions (Cloudinary)
**Rationale:** First external-storage integration; the upload-security and Cloudinary-access decisions must be correct on first write (re-uploading assets later is HIGH-cost).
**Delivers:** assignment CRUD (lecturer), backend-proxy upload flow (magic-byte + size validation), grading + in-transaction notification.
**Addresses:** assignment submission + grading + auto-notify (FEATURES.md).
**Avoids:** Pitfalls #3 (upload content/zip-bomb), #4 (public Cloudinary access), #9 (dropped notifications).

### Phase 6: Quizzes (auto-grade)
**Rationale:** Independent of assignments (can parallel Phase 5); the student-DTO split and attempt model must be correct from day one.
**Delivers:** quiz create (shuffle, max-questions), take + server-side auto-grade by stable option ID, persisted per-attempt question set, idempotent single-attempt submit.
**Addresses:** MCQ quiz auto-grade (FEATURES.md).
**Avoids:** Pitfall #5 (answer leakage, shuffle/grading correctness, double-submit).

### Phase 7: Announcements + Requests
**Rationale:** Depends on enrollments; can parallel Phases 5/6. Reuses the persisted-fan-out + in-transaction-notification primitive.
**Delivers:** announcement fan-out with read receipts; student requests (leave-early/absence/custom) + lecturer yes/no reply + auto-notify, with ownership-scoped targeting.
**Addresses:** announcements, student<->lecturer requests (FEATURES.md).
**Avoids:** Pitfalls #8 (lecturer messaging students they don't teach) and #9 (notification drops).

### Phase 8: Stale-Course Sweep
**Rationale:** Small, isolated, depends only on courses; build last. In-process idempotent cron.
**Delivers:** daily `time.Ticker`/cron job soft-deleting courses 1 month past end date (UTC, `deleted_at IS NULL` guarded), each run audit-logged.
**Addresses:** auto soft-delete sweep (FEATURES.md).
**Avoids:** Pitfall #6 (idempotency, timezone, history preservation).

### Phase Ordering Rationale

- **Dependency-driven:** CI/data model/auth/provisioning are strictly sequential because each produces what the next consumes (enrollment is the spine; provisioning creates the rows every feature reads).
- **Parallelizable middle:** Phases 5, 6, 7 are independent of each other and can be split across the `backend`/`frontend` branches once 0-4 land.
- **Pitfall-driven placement:** the five existential pitfalls each map to the *first* version of their feature phase, not a later hardening pass — research is explicit that retrofitting any of them is HIGH-cost.
- **Cross-cutting threads:** RBAC ownership, audit-on-mutation, and in-transaction notifications are added as success criteria to every relevant phase rather than living in one phase.

### Research Flags

Phases likely needing deeper research during planning (`/gsd-plan-phase --research-phase <N>`):
- **Phase 5 (Assignments/Cloudinary):** Cloudinary `authenticated`-delivery + signed-URL-on-demand flow and zip-bomb-safe validation are nuanced and security-critical — verify exact SDK calls and expiry config.
- **Phase 6 (Quizzes):** shuffle-by-stable-ID, per-attempt question-set persistence, and idempotent grading have several subtle correctness traps; the data model needs careful design.

Phases with standard patterns (skip research-phase):
- **Phase 0/1 (Scaffolding, Data Model):** well-documented Go-Gin clean layout, golang-migrate, GitHub Actions service containers.
- **Phase 2 (Auth):** standard JWT/bcrypt/RBAC patterns — the *discipline* (restricted token, ownership) matters more than novel research.
- **Phase 4 (Course Core), Phase 7 (Announcements/Requests), Phase 8 (Sweep):** established CRUD, persisted-fan-out, and idempotent-cron patterns already detailed in ARCHITECTURE.md.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Versions verified against pkg.go.dev / npm / GitHub releases; choices committed by user, research is best-practice within them. |
| Features | MEDIUM | Corroborated across Canvas, Stanford CS policies, MS Forms, soft-delete/audit community; not vendor-official for this exact product. |
| Architecture | HIGH | Stack is fixed; patterns are standard, well-documented Go-Gin clean architecture and pull-based delivery. |
| Pitfalls | HIGH | Security mechanics verified against current Cloudinary/GitHub/OWASP docs; domain-logic pitfalls from established practice. |

**Overall confidence:** HIGH

### Gaps to Address

- **"Forgot password" with no email channel:** the de-facto path is admin-assisted reset (which the audit log already covers). Make this explicit in requirements so users aren't left stranded; defer any self-service channel to v2+.
- **`grades` table vs projection:** decide during Phase 1/5 planning whether grades are projected from `submissions`/`quiz_attempts` (recommended, avoids dual source of truth) or whether a `grades` table is needed *only* for manual ad-hoc lecturer grades.
- **Quiz edit-after-submission policy:** editing a quiz's answer key after attempts exist must be blocked or trigger explicit regrade/versioning — decide the policy in Phase 6 planning (regrade can be a follow-up, but the data model must support it from day one).
- **Atomic-batch vs skip-and-report for CSV:** research recommends atomic all-or-nothing (simplest, safest); confirm with the user during Phase 3 discussion that rejecting the whole file on any error is acceptable UX.
- **Feature confidence is MEDIUM:** validate the exact request types and announcement-targeting rules against the real PROJECT.md requirements during phase discussion.

## Sources

### Primary (HIGH confidence)
- pkg.go.dev (pgx v5.10.0), github.com/sqlc-dev/sqlc/releases (v1.31.1), golang-jwt v5.3.1, Gin v1.11.0 — version verification.
- ui.shadcn.com (Vite+Tailwind v4+React 19 setup), TanStack Query v5 releases, react-hook-form/@hookform/resolvers — frontend stack.
- Cloudinary docs — Media Access Control, delivery URL signatures, authenticated/private asset access thread.
- GitHub Docs — protected branches / required status checks / troubleshooting skipped-job-reports-success.
- OWASP File Upload Cheat Sheet — magic-byte validation, zip-bomb mitigation.
- robfig/cron/v3 + Cloudinary Go upload docs — architecture patterns.

### Secondary (MEDIUM confidence)
- encore.dev / brandur.org / glukhov.org — sqlc vs GORM vs pgx consensus.
- Canvas/WUSTL/ClassMarker/MS Forms — quiz auto-grading and shuffle baselines.
- Stanford CS107/CS106B late policies — submission/deadline/resubmission rules.
- brandur.org / ZenStack / Medium — soft-delete + unique-constraint pitfalls.
- go-gin-clean-starter, Go Backend Clean Architecture — layered monolith reference.

### Tertiary (LOW confidence)
- OWASP/RFC 9106 via reintech.io/alexedwards.net — argon2id vs bcrypt guidance (bcrypt chosen pragmatically; revisit if a security review requires the OWASP baseline).

---
*Research completed: 2026-06-19*
*Ready for roadmap: yes*
