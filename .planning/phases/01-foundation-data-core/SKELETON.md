# Walking Skeleton — myIU (Lite Edition)

**Phase:** 1
**Generated:** 2026-06-19

## Capability Proven End-to-End

A developer clones the repo, runs `docker compose up -d` (Postgres only), applies migrations with `make migrate` (creating `users` + `audit_log` and seeding one bootstrap admin), runs `make run`, and `GET /healthz` returns 200 after a real DB query confirms the seeded admin exists — while the Vite + React 19 frontend stub builds and serves. The whole stack (config → pgx → migrate-applied schema → seeded data → sqlc-generated code → HTTP) is exercised by one request, and a deliberately-failing PR is proven to be merge-blocked by a required CI status check.

## Architectural Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Backend framework | Go 1.24 + Gin v1.11.0 | Committed in CLAUDE.md. Gin stable line (not experimental v1.12). Pin `go 1.24` in go.mod even though dev machine has 1.25 (drift landmine). |
| DB driver + access | pgx/v5 + sqlc v1.31.1 (`sql_package: "pgx/v5"`, `engine: "postgresql"`) | Committed. Raw-SQL ergonomics with compile-time type safety; no runtime ORM. sqlc reads the migration DDL for types. |
| Schema management | golang-migrate v4.18 (raw SQL up/down migrations) | Committed. Migrations are the single source of schema truth; incremental per-phase (D-06). sqlc only reads them. |
| Migration strategy | Incremental per-phase (D-06) | Phase 1 creates ONLY `users` + `audit_log`. Feature tables (courses/assignments/quizzes/grades/notifications/requests/enrollments) land in their owning phases. |
| Config | godotenv v1.5 + caarlos0/env v11 → typed `Config` struct with `required` tags | Committed. Fail-fast at boot on a missing secret (INFRA-03). JWT/Cloudinary slots exist now, consumed Phases 2/4. |
| Password hashing | bcrypt cost=12 (`golang.org/x/crypto/bcrypt`) | Committed. Seed admin password is a precomputed cost-12 hash literal in SQL (golang-migrate runs SQL only, cannot run Go). |
| DB runtime | Docker Compose, Postgres only (`postgres:17-alpine`) (D-08) | Committed; Postgres via Docker only. Backend (`go run`) + frontend (Vite) run natively for fast hot-reload. Migrations apply via a separate `make migrate` step, never auto-run by compose. |
| Frontend | Vite 6 + React 19 + TypeScript 5 (minimal stub) | Committed. Phase 1 is a stub that builds and serves — full shadcn/Tailwind UI starts Phase 2 (no UI hint on Phase 1). |
| CI | GitHub Actions, single job named exactly `ci`, `services: postgres:17-alpine` | Lean walking-skeleton fit over testcontainers (zero extra deps; satisfies "real Postgres service container"). Required-status-check name MUST equal job name `ci` exactly (verified landmine). |
| Merge block | GitHub branch protection requiring the `ci` status check (USER-configured in UI) + throwaway-failing-PR proof (D-09) | The workflow runs checks; branch protection is what BLOCKS the merge. Repo files (Claude) vs admin UI actions (user) cleanly separated. |
| Directory layout | Monorepo root: `backend/` (Go) + `frontend/` (React); `docker-compose.yml`, `.github/workflows/ci.yml`, `Makefile` at root or backend | Committed (INFRA-01). Two top-level folders; three branches `main`/`backend`/`frontend`. |

## Stack Touched in Phase 1

- [x] Project scaffold — `backend/` Go module (go 1.24) + `frontend/` Vite + React 19 + TS stub; lint (golangci-lint, ESLint); test runner (Go `testing` + testify)
- [x] Routing — Gin `GET /healthz` (the one real route this phase)
- [x] Database — one real read (`/healthz` runs the sqlc-generated `count(users)` query) AND one real write (the bootstrap-admin seed migration is the write; verified by `count = 1`)
- [x] UI — Vite + React 19 stub builds (`npm run build`) and serves (`npm run dev`). No backend call this phase (true stub; CORS deferred to Phase 2).
- [x] Deployment — documented local full-stack run: `docker compose up -d` → `make migrate` → `make run` → `npm run dev`

## Out of Scope (Deferred to Later Slices)

> Anything NOT in the skeleton. Explicit, to stop future phases re-litigating Phase 1's minimalism.

- Auth / login / JWT issuance / forced-reset enforcement — Phase 2 (AUTH-01..05). The seed admin is bootstrap only.
- RBAC role gating + ownership checks — Phase 2 (AUTH-05).
- Feature tables: courses, enrollments, assignments, quizzes, grades, notifications, requests — each in its owning phase (D-06).
- Real admin provisioning / CSV import / audit-log writes / soft-delete sweep — Phase 3 (ADMIN-01..08). The `audit_log` table exists now but no rows are written this phase.
- Cloudinary upload integration — Phase 4 (the `CLOUDINARY_URL` config slot exists now, consumed Phase 4).
- shadcn/ui + Tailwind + Zustand + TanStack Query + react-router + forms — Phase 2+ (frontend is a stub this phase).
- CORS middleware (gin-contrib/cors) — Phase 2, when the SPA first calls the API.
- Full-stack docker-compose (containerized backend/frontend) — rejected for Phase 1 (D-08); revisit only if a production deploy story needs it.
- testcontainers-go — `services:` Postgres is sufficient for one skeleton test; switch later if many isolated suites arrive.

## Subsequent Slice Plan

Each later phase adds one vertical slice on top of this skeleton without altering its architectural decisions (incremental migrations, typed config, `ci` job, Postgres-only compose):

- Phase 2: A user can log in (username + password → role-carrying JWT), log out, change their own password, and a `must_change_password` user is server-side restricted to change-password/logout only; routes gated by role + ownership.
- Phase 3: Admin provisions accounts/enrollment/courses from CSV or UI; every mutation writes an append-only `audit_log` row; courses auto soft-delete one month after end date.
- Phase 4: Lecturers run file-upload assignments (Cloudinary authenticated assets, magic-byte + 10MB validation, late policy) and auto-graded MCQ quizzes; students auto-notified via a shared persisted notification primitive.
- Phase 5: Weighted gradebook, announcement fan-out, and student↔lecturer requests with auto-delivered replies — all on the notification primitive.
