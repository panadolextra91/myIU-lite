# Architecture Research

**Domain:** LMS-lite / university student-management platform (myIU lite)
**Researched:** 2026-06-19
**Confidence:** HIGH (stack is fixed; patterns are standard and well-documented)

## Standard Architecture

myIU lite is a **single-deployment monolith**: one Go/Gin API binary, one PostgreSQL database (Docker), one React SPA, with Cloudinary as the only external service. There is no need for microservices, message queues, or a separate worker — the scheduled sweep runs in-process. This keeps the build lean (a stated project goal) while the layered structure inside the binary keeps it testable.

### System Overview

```
┌──────────────────────────────────────────────────────────────────┐
│                    FRONTEND (React SPA, frontend/)                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐   role-gated routes     │
│  │ Student  │  │ Lecturer │  │  Admin   │   (React Router)        │
│  │  pages   │  │  pages   │  │  pages   │                         │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘                         │
│       └─────────────┼─────────────┘                                │
│                ┌────┴─────┐   ┌──────────────┐                     │
│                │ Zustand  │   │  API client  │  (axios + JWT       │
│                │  stores  │←→ │  (interceptor)│   interceptor)     │
│                └──────────┘   └──────┬───────┘                     │
└──────────────────────────────────────┼────────────────────────────┘
                                        │  HTTPS / JSON  (Bearer JWT)
┌───────────────────────────────────────┼───────────────────────────┐
│                  BACKEND (Go + Gin, backend/)                      │
│  ┌─────────────────────────────────────────────────────────────┐  │
│  │ Middleware: CORS → Logger → JWT auth → RBAC → audit          │  │
│  └─────────────────────────────────────────────────────────────┘  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐           │
│  │ Handlers │  │ Handlers │  │ Handlers │  │ Handlers │ (HTTP I/O)│
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘           │
│       └─────────────┴──────┬──────┴─────────────┘                  │
│  ┌────────────────────────┴────────────────────────────────────┐  │
│  │ Services (business logic: grading, sweep, provisioning, RBAC)│  │
│  └────────────────────────┬────────────────────────────────────┘  │
│  ┌────────────────────────┴────────────────────────────────────┐  │
│  │ Repositories (SQL queries — the only layer touching the DB)  │  │
│  └────────────────────────┬────────────────────────────────────┘  │
│  ┌──────────────┐   ┌──────┴───────┐                               │
│  │ Cron sweep   │   │ Cloudinary   │ ──→ Cloudinary (file storage) │
│  │ (goroutine)  │   │ client       │                               │
│  └──────────────┘   └──────┬───────┘                               │
└────────────────────────────┼───────────────────────────────────────┘
                       ┌──────┴───────┐
                       │ PostgreSQL   │  (Docker container)
                       └──────────────┘
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| Handlers | Parse/validate HTTP request, call one service, shape JSON response. No business logic, no SQL. | Gin handler funcs grouped by domain (e.g. `assignment_handler.go`) |
| Services | Business rules: auto-grade quiz, run sweep, provision from CSV, enforce "lecturer owns this course". Orchestrate repos + Cloudinary. | Plain Go structs holding repo interfaces |
| Repositories | All DB access; one method = one or a few SQL statements. Return domain structs. | Structs wrapping `*pgxpool.Pool` (or `database/sql`); raw SQL |
| Middleware | Cross-cutting: CORS, request logging, JWT verify, RBAC role check, admin-action audit write. | Gin middleware in chain order |
| Cloudinary client | Upload submission files, return secure URL + public_id. | `cloudinary-go` SDK, configured from `.env` |
| Cron sweep | Once/day: soft-delete courses 1 month past end date; write audit rows. | `robfig/cron/v3` job started in `main()` |
| Frontend stores | Hold auth/session + per-domain UI state; call API client. | Zustand slices |
| API client | One axios instance; attaches JWT, handles 401 → logout, normalizes errors. | `frontend/src/lib/api.ts` |

## Recommended Project Structure

```
myIU-lite/
├── backend/
│   ├── cmd/
│   │   └── api/main.go          # wire deps, start cron, run Gin
│   ├── internal/
│   │   ├── config/              # load .env (DB, JWT secret, Cloudinary)
│   │   ├── db/
│   │   │   ├── db.go            # pgx pool
│   │   │   └── migrations/      # *.sql, run via golang-migrate
│   │   ├── domain/              # structs: User, Course, Assignment… (no logic)
│   │   ├── middleware/          # auth.go, rbac.go, audit.go, cors.go
│   │   ├── handler/             # *_handler.go per domain
│   │   ├── service/             # *_service.go (business logic) + interfaces
│   │   ├── repository/          # *_repo.go (SQL) + interfaces
│   │   ├── cron/                # sweep.go
│   │   └── storage/             # cloudinary.go wrapper
│   ├── go.mod
│   └── .env.example
├── frontend/
│   ├── src/
│   │   ├── routes/             # route tree + role guards
│   │   ├── pages/
│   │   │   ├── student/
│   │   │   ├── lecturer/
│   │   │   └── admin/
│   │   ├── components/ui/      # shadcn/ui (generated, not hand-rolled)
│   │   ├── stores/             # auth.ts, course.ts, … (Zustand)
│   │   ├── lib/api.ts          # axios client + interceptors
│   │   └── App.tsx
│   ├── package.json
│   └── .env.example
└── .github/workflows/ci.yml
```

### Structure Rationale

- **`internal/` per-layer folders:** Standard Go-Gin clean layout (handler → service → repository). Layers talk through interfaces so services are unit-testable with mock repos; only repositories know SQL, so a query change never leaks into business logic.
- **Interfaces defined in the consuming layer:** the service package declares the repo interface it needs; the repository package implements it. This keeps the dependency arrow pointing inward (handlers → services → repos) and makes mocking trivial.
- **`cron/` and `storage/` as siblings of services:** the sweep is invoked as a service call, and Cloudinary is a thin adapter, so both stay out of business logic.
- **Frontend `pages/{role}/`:** mirrors the three actors and the role-gated route tree, so a route guard maps 1:1 to a folder.
- **`components/ui/` is generated:** shadcn/ui components are copied in by its CLI; per the project's "no hand-rolled components" rule, treat this folder as vendor-ish.

## Data Model

The relational core. PostgreSQL; soft-delete via `deleted_at TIMESTAMPTZ NULL` on courses (and anywhere history must survive).

```
users ──< enrollments >── courses ──< assignments ──< submissions
  │           (role per           │
  │            enrollment)        ├──< quizzes ──< questions ──< options
  │                               │       │
  │                               │       └──< quiz_attempts ──< attempt_answers
  │                               │
  │                               ├──< announcements ──< announcement_recipients
  │                               └──< requests
  │
grades (per student × gradable item)
audit_log (admin actions)
```

### Key Tables & Relationships

| Table | Key columns | Relationships / notes |
|-------|-------------|------------------------|
| `users` | id, username (=student/lecturer ID), password_hash, role, must_change_password, birthday | role ∈ {student, lecturer, admin}. Default pw = birthday `DDMMYYYY`, hashed; `must_change_password=true` on create. |
| `courses` | id, code, title, start_date, end_date, deleted_at | Sweep sets `deleted_at` 1mo after `end_date`. |
| `enrollments` | id, user_id, course_id, role_in_course | Join table; a lecturer and students both enroll. UNIQUE(user_id, course_id). |
| `assignments` | id, course_id, title, description, due_date | Owned by course → lecturer-scoped via enrollment. |
| `submissions` | id, assignment_id, student_id, file_url, file_public_id, grade, graded_at | `file_url`/`public_id` from Cloudinary. UNIQUE(assignment_id, student_id) for resubmit-overwrite. |
| `quizzes` | id, course_id, title, shuffle, max_questions | Auto-grade MCQ only. |
| `questions` | id, quiz_id, text | |
| `options` | id, question_id, text, is_correct | `is_correct` never sent to student client. |
| `quiz_attempts` | id, quiz_id, student_id, score, submitted_at | Score computed server-side at submit. |
| `attempt_answers` | id, attempt_id, question_id, option_id | Chosen options. |
| `announcements` | id, course_id, sender_id, title, body, audience | audience ∈ {all, specific}. |
| `announcement_recipients` | id, announcement_id, student_id, read_at | Materializes per-student delivery; `read_at` = read receipt. For `audience=all`, fan out one row per enrolled student. |
| `requests` | id, course_id, student_id, type, title, body, status, reply | type ∈ {leave_early, absence, custom}; status ∈ {pending, yes, no}. |
| `grades` | id, student_id, course_id, item_type, item_id, score | Unified grade record (assignment or quiz), or fold into submissions/attempts — see note. |
| `audit_log` | id, admin_id, action, target_type, target_id, detail, created_at | Append-only; written by audit middleware/service on every admin mutation. |

**Design note on `grades`:** because assignment grades live on `submissions.grade` and quiz scores on `quiz_attempts.score`, a separate `grades` table risks duplication. Recommended: make `grades` a **read-oriented view/projection** (or skip the table and query the two sources) unless lecturers must issue manual ad-hoc grades — in which case keep `grades` as the manual-entry sink only.

## File-Upload Flow (Cloudinary)

**Chosen pattern: backend proxy upload** (file passes through the Go API), not signed direct browser upload. Rationale: the server must enforce the 10MB cap, PDF/ZIP-only rule, and "this student is enrolled in this assignment's course" authorization *before* anything is stored; the `api_secret` stays in `.env`; and the audit/grade records are written in the same transaction. The bandwidth savings of direct upload don't matter at lite scale.

```
Student picks PDF/ZIP
    ↓ multipart/form-data
[Submission handler]
    ├─ validate size ≤ 10MB (Gin MaxMultipartMemory + check)
    ├─ validate MIME/extension ∈ {pdf, zip}
    ↓
[Submission service]
    ├─ check enrollment + assignment open
    ├─ storage.Upload(file) ──→ Cloudinary ──→ {secure_url, public_id}
    ├─ repo.UpsertSubmission(file_url, public_id)   (overwrite prior)
    ↓
[201 + submission JSON]
```

On resubmit, overwrite the row and optionally delete the old Cloudinary `public_id`. Lecturers download via the stored `file_url`.

## Scheduled Stale-Course Sweep (where it runs)

**Runs in-process** inside the API binary as a `robfig/cron/v3` job (or a `time.Ticker` goroutine — both fine; cron is more self-documenting). Started in `main()` after the DB pool is ready. No separate worker/container — appropriate for a monolith and the "keep it lean" goal.

```
main() → cron.New() → AddFunc("@daily", sweepService.Run) → cron.Start()

sweepService.Run():
    UPDATE courses
       SET deleted_at = now()
     WHERE deleted_at IS NULL
       AND end_date < now() - interval '1 month';
    → for each swept course, INSERT audit_log (action='auto_sweep_course', admin_id=SYSTEM)
```

Make the sweep **idempotent** (the `deleted_at IS NULL` guard ensures re-runs are safe) so a missed day self-heals on the next tick. If horizontal scaling ever happens, move to an external scheduler or add a DB advisory lock so only one instance runs it — not needed now.

## Notification / Announcement Delivery Model

**Pull-based, persisted fan-out** — no websockets, no email (email is explicitly out of scope). Delivery = rows the recipient polls/fetches on login or page load.

- **Announcements:** lecturer creates one `announcements` row; for `audience=all` the service fans out one `announcement_recipients` row per enrolled student; for `specific`, one row per selected student. Student "inbox" = query recipients joined to announcements; `read_at` marks read.
- **Auto-delivered events** (grade posted, request replied) use the same idea: the act of writing `submissions.grade` / `requests.reply` *is* the delivery — the student sees it next time they fetch grades/requests. A lightweight unread badge = `COUNT(*) WHERE read_at IS NULL` across recipients + ungraded-now-graded items.

This avoids real-time infrastructure entirely; "automatic" in the requirements means "appears without the lecturer separately messaging," satisfied by the recipient rows.

## Frontend Structure

### Routing per role

One route tree with a `<RequireRole roles={[...]}>` guard reading the auth store. After login, redirect by role to `/student`, `/lecturer`, or `/admin`. A `must_change_password` flag forces redirect to a change-password page before any other route renders.

```
/login, /forgot-password, /reset-password   (public)
/change-password                            (any authed; forced if must_change_password)
/student/*   → courses, assignments, quizzes, grades, announcements, requests
/lecturer/*  → courses, grade-assignments, quizzes (create), announcements, requests
/admin/*     → users (CSV import), enrollments (CSV), courses CRUD, audit log
```

### Zustand stores

Keep stores thin and domain-sliced; data-fetching state can also live in TanStack Query if added later, but Zustand alone is sufficient here.

| Store | Holds |
|-------|-------|
| `authStore` | current user, role, JWT, `mustChangePassword`; `login/logout` |
| `courseStore` | enrolled/taught courses, selected course |
| `announcementStore` | inbox items, unread count |
| `requestStore` | student's requests / lecturer's queue |
| (per-page) | quiz-taking and grading screens can keep transient state local |

### API client

Single axios instance in `lib/api.ts`: base URL from `.env`, request interceptor attaches `Authorization: Bearer <jwt>` from `authStore`, response interceptor maps 401 → `authStore.logout()` + redirect to `/login`, and normalizes the backend error shape. All stores call this client — no `fetch` scattered in components.

## Auth & RBAC (3 roles)

- **AuthN:** username/password → bcrypt verify → issue JWT (claims: `sub`, `role`, `must_change_password`). Stateless; no server session store needed at this scale.
- **First-login:** if `must_change_password`, the API rejects all non-(change-password) mutations and the SPA forces the change-password route.
- **AuthZ (RBAC):** two layers.
  1. **Role gate (middleware):** `RequireRole("admin")` etc. on route groups — coarse.
  2. **Ownership check (service):** "is this lecturer enrolled-as-lecturer in the course that owns this assignment?" / "is this student enrolled?" — fine-grained, enforced in services because it needs DB lookups. Never trust the client's claimed course/student IDs.
- **Audit:** an `audit.go` middleware (or a service helper) writes an `audit_log` row for every admin mutation (account create, password reset, enrollment, course CRUD, and the auto-sweep with a SYSTEM actor).

## Architectural Patterns

### Pattern 1: Handler → Service → Repository (interface-segregated)

**What:** HTTP concerns in handlers, business rules in services, SQL in repositories; layers wired in `main()` via constructor injection of interfaces.
**When to use:** the whole backend — it's the default Gin clean-architecture shape.
**Trade-offs:** a little boilerplate (interface + impl) per domain, bought back as full unit-testability and a clean blast radius when SQL or rules change.

```go
type SubmissionRepo interface {
    Upsert(ctx context.Context, s domain.Submission) error
}
type SubmissionService struct { repo SubmissionRepo; store Storage }
func (s *SubmissionService) Submit(ctx context.Context, in SubmitInput) (domain.Submission, error) {
    // validate enrollment, upload, persist — all rules live here
}
```

### Pattern 2: Persisted fan-out for "delivery"

**What:** Turn a one-to-many notification into recipient rows at write time; readers just query their rows.
**When to use:** announcements, grade/request delivery.
**Trade-offs:** more rows vs. real-time push; massively simpler (no websockets), and read receipts come free via `read_at`.

### Pattern 3: Idempotent in-process scheduled job

**What:** A cron goroutine whose SQL is safe to run repeatedly (guard clauses), co-located with the API.
**When to use:** the daily course sweep.
**Trade-offs:** can't horizontally scale the API without a lock — irrelevant now, and avoids running/operating a separate worker.

## Data Flow

### Request Flow (quiz auto-grade example)

```
Student submits answers
    ↓
[Quiz handler] → [Quiz service] → [Quiz repo]
   parse JSON      score = compare      load correct options
                   chosen vs correct    insert attempt + answers
    ↓
[200 + score]  ← service returns computed score (client never saw is_correct)
```

### State Management

```
authStore ──(subscribe)──> route guards + API client (reads JWT)
components ──(call)──> store actions ──> api.ts ──> backend ──> set store state
```

### Key Data Flows

1. **CSV provisioning:** Admin uploads CSV → handler parses → service creates `users` (default pw = birthday, `must_change_password=true`) / `enrollments` in a transaction → audit rows. Reject the whole batch on any malformed line.
2. **Submission:** see file-upload flow above.
3. **Grade delivery:** lecturer sets `submissions.grade` → student's next grades fetch shows it (no separate send).
4. **Sweep:** daily cron → soft-delete + audit, idempotent.

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| 0–1k users (target) | Single API instance + one Postgres; in-process cron; backend-proxy uploads. No changes needed. |
| 1k–100k users | Add DB indexes on FKs (`enrollments(course_id)`, `submissions(assignment_id)`, `announcement_recipients(student_id, read_at)`); put API behind ≥2 instances → guard the cron with a DB advisory lock; consider signed direct Cloudinary uploads to offload bandwidth. |
| 100k+ users | Split read replicas; move announcement fan-out to a queue/worker; cache hot course/announcement reads. Unlikely for a university-course tool. |

### Scaling Priorities

1. **First bottleneck:** announcement fan-out and recipient queries — fix with indexes before anything fancier.
2. **Second bottleneck:** the cron running on multiple API replicas — add an advisory lock or external scheduler.

## Anti-Patterns

### Anti-Pattern 1: SQL in handlers

**What people do:** query the DB directly from Gin handlers to "save a layer."
**Why it's wrong:** business rules and ownership checks leak into HTTP code; nothing is unit-testable; a query change ripples everywhere.
**Do this instead:** handlers call one service; only repositories run SQL.

### Anti-Pattern 2: Trusting client-supplied IDs for authorization

**What people do:** grade/announce based on `courseId` from the request body without checking the actor owns it.
**Why it's wrong:** any lecturer/student can act on courses they aren't in.
**Do this instead:** enforce enrollment/ownership in the service against the JWT `sub`.

### Anti-Pattern 3: Sending quiz answer keys to the client

**What people do:** include `is_correct` in the quiz-fetch payload, grade in the browser.
**Why it's wrong:** trivially cheatable; defeats auto-grading.
**Do this instead:** strip `is_correct` on read; compute the score server-side at submit.

### Anti-Pattern 4: A separate `grades` table duplicating submission/attempt scores

**What people do:** copy every score into `grades`, then drift.
**Why it's wrong:** two sources of truth.
**Do this instead:** project grades from `submissions`/`quiz_attempts`; reserve a `grades` table only for manual lecturer entries if needed.

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| Cloudinary | Backend-proxy upload via `cloudinary-go` SDK; creds in `.env` | Server validates size/type/auth first; store `secure_url` + `public_id`; delete old `public_id` on resubmit. |
| PostgreSQL | `pgx`/`database/sql` pool, Docker container; `golang-migrate` for schema | DB runs only via Docker (constraint); never embed credentials — use `.env`. |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| Frontend ↔ Backend | REST/JSON over HTTPS, Bearer JWT | Single axios client; CORS middleware. |
| Handler ↔ Service ↔ Repo | In-process Go interface calls | Constructor-injected in `main()`. |
| Cron ↔ Service | Direct call from goroutine | Reuses the same course/audit services. |

## Suggested Build Order (dependency-driven)

Order follows the dependency arrows: foundation → auth → admin provisioning (creates the data everything else needs) → course features → cross-cutting.

1. **Scaffolding & CI** — monorepo `backend/`+`frontend/`, 3 branches, Docker Postgres, `.env`, GitHub Actions (tests+DB+syntax gate), `golang-migrate`. *Blocks everything.*
2. **Data model & migrations** — `users`, `courses`, `enrollments`, `audit_log` first; rest can follow per feature. *Blocks all repos.*
3. **Auth + RBAC + first-login** — JWT, bcrypt, middleware (role gate, audit), forced password change. Frontend: auth store, API client, login/forgot/change-password, route guards. *Everything authenticated depends on this.*
4. **Admin: provisioning** — CSV user import, CSV enrollment, course CRUD, audit log view. *Produces the users/courses/enrollments the rest of the app reads.*
5. **Course core for student/lecturer** — view enrolled/taught courses, course pages. *Depends on enrollments existing.*
6. **Assignments + submissions (Cloudinary)** — assignment CRUD (lecturer), upload flow (student), grading + auto-delivery. *Depends on courses + storage wrapper.*
7. **Quizzes** — create (lecturer, shuffle/max-questions), take + auto-grade (student). *Independent of assignments; can parallel #6.*
8. **Announcements + requests** — fan-out delivery, read receipts; student requests + lecturer yes/no reply. *Depends on enrollments; can parallel #6/#7.*
9. **Stale-course sweep** — cron job + audit. *Depends only on courses; build last, independent.*

Phases 6–8 can be parallelized across the `backend`/`frontend` branches once 1–5 land. Phase 9 is small and isolated.

## Sources

- [go-gin-clean-starter (Controller-Service-Repository + DI)](https://github.com/Caknoooo/go-gin-clean-starter) — MEDIUM
- [Go Backend Clean Architecture](https://outcomeschool.com/blog/go-backend-clean-architecture) — MEDIUM
- [robfig/cron/v3 package docs](https://pkg.go.dev/github.com/robfig/cron/v3) — HIGH (official)
- [Building Scheduled Task Runner in Go (Ticker vs cron)](https://oneuptime.com/blog/post/2026-01-30-how-to-build-scheduled-task-runner-in-go/view) — MEDIUM
- [Cloudinary Go image and video upload docs](https://cloudinary.com/documentation/go_image_and_video_upload) — HIGH (official)
- [Cloudinary client-side vs backend uploading](https://cloudinary.com/documentation/client_side_uploading) — HIGH (official)

---
*Architecture research for: LMS-lite university student-management platform*
*Researched: 2026-06-19*
