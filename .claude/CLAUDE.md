<!-- GSD:project-start source:PROJECT.md -->

## Project

**myIU (Lite Edition)**

A lite student management platform for a university, called **myIU**. It gives three actors — Students, Lecturers, and Admins — a single place to handle coursework and course administration: assignment submission, auto-graded quizzes, announcements, grades, and student↔lecturer requests (so email is no longer needed). Admin manages accounts, course enrollment, and courses.

**Core Value:** Students and lecturers can run a course end-to-end (assignments, quizzes, grades, announcements, requests) without falling back to email — and Admin can provision everything from CSV.

### Constraints

- **Tech stack (backend)**: Go + Gin, PostgreSQL — ORM optional (raw SQL acceptable if no good PostgreSQL+Go ORM fits). Env via `.env`.
- **Tech stack (frontend)**: React + Zustand (state) + shadcn/ui (components). No hand-rolled components.
- **Storage**: Cloudinary for uploaded files, configured via environment variables.
- **Submissions**: PDF and ZIP only, hard 10MB limit.
- **Security/compliance**: forced first-login password change; audit log for all admin actions.
- **Database runtime**: PostgreSQL runs via Docker only — never natively.
- **Repo structure**: two top-level folders — `backend/` (Go source) and `frontend/` (React source). No per-folder README required.
- **Branching (GitHub Flow — agents MUST follow):** `main` is the ONLY long-lived branch (protected, always deployable). Never commit directly to `main`. Every change is made on a short-lived branch cut from latest `main`, named `ft/<slug>` | `fix/<slug>` | `chore/<slug>` | `docs/<slug>` — one branch per phase/feature (backend + frontend together). Open a PR, squash-merge into `main`, then delete the branch. (Replaces the former `main`/`backend`/`frontend` three-branch model.)
- **CI/CD**: GitHub Actions runs the `ci` job on every pull request (and on push to `main`). Merge into `main` is blocked unless `ci` passes (unit + integration tests against real Postgres, DB migrations, lint/build).
- **Testing**: every phase must pass unit tests and integration tests.

<!-- GSD:project-end -->

<!-- GSD:stack-start source:research/STACK.md -->

## Technology Stack

## Recommended Stack

### Core Technologies (committed — pinned to current versions)

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Go | 1.24.x | Backend language/runtime | Current stable; all libraries below support it. Use the latest 1.24 patch. |
| Gin | v1.11.0 (stable) | HTTP framework | Committed. Use **v1.11.0** (stable) rather than v1.12.0 — v1.12.0's headline features (HTTP/3, native context API) are flagged *experimental* and irrelevant to this app. v1.11.0 is the safe production line. |
| PostgreSQL | 17.x (Docker `postgres:17-alpine`) | Primary datastore | Committed; runs via Docker only. PG17 is current GA; alpine image keeps the dev container small. |
| React | 19.x | Frontend UI | Committed. React 19 is the current major and what shadcn/ui's current CLI targets. |
| Vite | 6.x | Frontend build/dev server | The standard React bundler in 2025; shadcn/ui's official non-Next setup path. CRA is dead — do not use it. |
| TypeScript | 5.x | Frontend language | Required for type-safe Zod+RHF+shadcn integration; shadcn components ship as TS. |

### Backend Database Layer — **sqlc** (the key decision)

| Library | Version | Purpose | Why |
|---------|---------|---------|-----|
| **sqlc** | v1.31.1 | SQL → type-safe Go code generator | **Recommended DB layer.** You write plain SQL; sqlc generates type-safe Go structs + query methods at build time. No runtime ORM overhead, no reflection, compile-time errors when SQL and Go drift. PROJECT.md says "ORM optional, raw SQL acceptable, Go ORM ecosystem is thin" — sqlc is the idiomatic answer: raw SQL ergonomics with type safety. |
| **pgx** | v5.7.x | PostgreSQL driver | The modern Postgres driver for Go (not the old `lib/pq`). sqlc generates code that targets the `pgx/v5` driver directly. Also used standalone by testcontainers. Configure sqlc with `sql_package: "pgx/v5"`. **Pinned v5.7.x (not v5.10.0): pgx v5.10.0's go.mod requires Go 1.25.0, which conflicts with the committed Go 1.24 pin; v5.7.2 is the newest pgx that builds on Go 1.24. To adopt pgx ≥v5.10, bump the toolchain to Go 1.25.** |

- **GORM** — rejected. Heavy runtime reflection, magic, and overhead; PROJECT.md explicitly notes the Go ORM ecosystem is thin and wants lean code (Ponytail principle). GORM hides SQL, which fights the audit-log/soft-delete-sweep requirements where explicit SQL is clearer.
- **pgx raw** — viable but you hand-write all scan/mapping boilerplate. Fine for a few queries, tedious for an app with courses/quizzes/grades/requests/audit tables.
- **sqlc** — best of both: you keep full SQL control (good for the soft-delete sweep `UPDATE ... WHERE end_date < now() - interval '1 month'`), but get generated type-safe accessors. Slightly faster than `database/sql` on large reads; zero runtime ORM cost. **This is the 2025 consensus pick for "raw SQL but type-safe" on Postgres.**

### Backend Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| **golang-migrate/migrate** | v4.18.x | DB schema migrations | Versioned up/down SQL migrations; strong CI/CD story (the GitHub Actions gate must run migrations against the test DB). Pairs naturally with sqlc (migrate owns schema, sqlc reads it). |
| **golang-jwt/jwt** | v5.3.1 | JWT auth tokens | Stateless access tokens for the Student/Lecturer/Admin SPA. v5 is the current major with hardened validation. |
| **golang.org/x/crypto/bcrypt** | latest x/crypto | Password hashing | **Use bcrypt cost=12** for this app. See decision note below — argon2id is theoretically stronger but bcrypt is simpler, dependency-light, and fully adequate here. |
| **cloudinary-go/v2** | v2.x (≥2.11, post-Jun-2025 security patch) | File storage SDK | Committed storage. Import `github.com/cloudinary/cloudinary-go/v2`. Use `uploader.Upload` with `ResourceType: "raw"` for PDF/ZIP (not "image"). Pin ≥ the June 2025 security release that hardened input validation. |
| **go-co-op/gocron/v2** | v2.x | Scheduled jobs | For the "soft-delete courses 1 month after end date" sweep. v2's builder API + `DurationJob`/`CronJob` are clean. Single-instance app → no need for the distributed locker. Alternatively a plain `time.Ticker` is enough for one daily sweep — see "What NOT to over-build." |
| **encoding/csv** (stdlib) | — | CSV parsing | Admin CSV account/enrollment import. **Use the standard library** — no third-party CSV dep needed for simple comma-delimited admin lists. |
| **godotenv** | v1.5.x | `.env` loading | Committed: config via `.env`. Use godotenv to load `.env` in dev, then read with `os.Getenv`. Pair with `caarlos0/env` (v11) if you want struct-based config parsing. Skip Viper — overkill for a flat `.env`. |
| **caarlos0/env** | v11.x | Env → struct parsing | Optional but recommended: parse env vars into a typed `Config` struct (DB URL, JWT secret, Cloudinary creds) with defaults + required validation. Zero-dependency, twelve-factor friendly. |
| **gin-contrib/cors** | latest | CORS middleware | The SPA (Vite dev server, separate origin) needs CORS for the Gin API. Standard Gin middleware. |

### Frontend Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| **Zustand** | v5.x | Client/UI state | Committed. Use for auth/session state, current-user, UI toggles. Keep server data OUT of Zustand — that's TanStack Query's job (see pitfall below). |
| **TanStack Query** | v5.101.x (`@tanstack/react-query`) | Server-state / data fetching | **Recommended fetch layer.** Handles caching, loading/error states, refetch, mutation invalidation against the Gin REST API. Far better than hand-rolled `useEffect+fetch`. v5 has excellent type inference. SWR is the lighter alternative but TanStack is the broader-adoption standard and pairs cleanly with REST. |
| **shadcn/ui** | CLI latest (Tailwind v4 / React 19 path) | Component library | Committed; "no hand-rolled components." Initialize via `npx shadcn@latest init`. Components are copied into your repo (you own them), built on Radix primitives — accessible by default. |
| **Tailwind CSS** | v4.x (`@tailwindcss/vite`) | Styling engine | Required by current shadcn/ui. v4 uses the Vite plugin + `@theme` directive (no more `tailwind.config.js` for the common case). |
| **React Hook Form** | v7.79.x | Form state | The form engine shadcn's `<Form>` wraps. Uncontrolled inputs = no re-render per keystroke. Needed for login, password-change, quiz-builder, request forms, CSV-upload forms. |
| **Zod** | v4.x | Schema validation | TypeScript-first validation. Define one schema per form; reuse for client validation AND as the source of truth for types. Mirror critical rules server-side in Go (never trust client validation alone). |
| **@hookform/resolvers** | v3.x (Zod v4 compatible) | RHF↔Zod bridge | `zodResolver` connects Zod schemas to React Hook Form. Required glue for the shadcn form pattern. |
| **axios** *or* native `fetch` | axios v1.x / native | HTTP client | Either works under TanStack Query. axios gives interceptors (attach JWT, handle 401→logout centrally) — recommended for the auth flow. Native fetch is fine if you want zero deps. |
| **react-router** | v7.x | Routing | Role-based routes (student/lecturer/admin areas), protected routes behind auth. Standard SPA router. |

### Development & Testing Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| **Go stdlib `testing`** | Unit tests | Base test runner. Required by the CI gate ("every phase must pass unit + integration tests"). |
| **stretchr/testify** | v1.10.x — assertions, mocks, suites | `assert`/`require` for readable assertions; `suite` for shared setup/teardown on integration tests. The de-facto Go testing toolkit. |
| **testcontainers-go** + `modules/postgres` | v0.34.x — integration test DB | **Recommended for integration tests.** Spins up a real `postgres:17` container per test suite, runs migrations, tests real SQL. Satisfies the CI "tests + DB" gate authentically (no mocking the DB). Configure with `postgres.WithSQLDriver("pgx")`. Requires Docker in CI — GitHub Actions runners have it. |
| **Vitest** | v2.x — frontend unit tests | Vite-native test runner; fast, Jest-compatible API. The standard for Vite+React. |
| **React Testing Library** | v16.x — component tests | Test components by behavior, not implementation. |
| **golangci-lint** | latest — Go linter | Satisfies the CI "syntax checks" gate for the backend. Run in GitHub Actions. |
| **ESLint + TypeScript** | flat config — frontend lint | Satisfies the "syntax" gate for the frontend. |
| **Docker Compose** | — local Postgres | PROJECT.md: Postgres via Docker only. A `docker-compose.yml` with `postgres:17-alpine` for local dev. |

## Installation

### Backend (`backend/` — Go modules)

# Core

# Storage, config, jobs

# Testing

# CLI tools (install, not go get)

### Frontend (`frontend/` — Vite + React 19 + TS)

# shadcn/ui (Tailwind v4 path) — run init then add components

# State + data + forms

# Dev/test

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| sqlc | pgx raw queries | If the schema is tiny (<5 tables) and you dislike codegen steps — but this app has enough tables that sqlc pays off. |
| sqlc | GORM | Only if the team strongly prefers active-record ergonomics and accepts runtime overhead — contradicts PROJECT.md's lean/raw-SQL stance, so not recommended here. |
| golang-migrate | pressly/goose | goose is lighter and allows Go-function migrations; fine choice if you want minimal tooling. golang-migrate chosen for its stronger CI/CD integration (matches the merge-gate requirement). |
| bcrypt | argon2id (`x/crypto/argon2`) | If a security review *requires* OWASP/RFC 9106 baseline. argon2id (m=19456,t=2,p=1) is stronger against GPU attacks but needs a hand-rolled hash/verify wrapper. For a lite university app, bcrypt cost=12 is the pragmatic, low-risk choice. |
| TanStack Query | SWR | If you want the smallest bundle and simplest API and only do basic GETs. TanStack chosen for mutation/invalidation ergonomics with the REST API. |
| godotenv (+caarlos0/env) | Viper | Use Viper only if you later need YAML/JSON config, remote config, or live-reload. Overkill for a flat `.env`. |
| gocron/v2 | `time.Ticker` (stdlib) | For a single daily sweep, a plain ticker + a `SELECT`/`UPDATE` is enough and dependency-free. gocron adds value only if jobs multiply. |
| testcontainers-go | Dockerized Postgres service in CI + plain `go test` | testcontainers manages container lifecycle in-test (cleaner). The GH Actions `services:` block is an alternative if you want the DB provisioned outside the test code. |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| GORM | Runtime reflection/overhead, hides SQL — fights the lean + explicit-SQL goals (audit log, soft-delete sweep). | sqlc + pgx |
| `lib/pq` (old driver) | Effectively in maintenance mode; pgx is the actively developed modern driver. | jackc/pgx/v5 |
| Create React App (CRA) | Deprecated/unmaintained; slow. | Vite |
| Redux / Redux Toolkit for server data | Heavy boilerplate; server-cache is a solved problem. | TanStack Query (server state) + Zustand (UI state) |
| Storing server data in Zustand | Causes stale-cache + manual-invalidation bugs. | TanStack Query owns server state; Zustand only for auth/UI. |
| Gin v1.12.0 in production | Its flagship features (HTTP/3, native context) are experimental and unused here. | Gin v1.11.0 (stable) |
| MD5/SHA-256 for passwords | Not password hashes — instantly brute-forced. | bcrypt cost=12 (or argon2id) |
| Cloudinary `ResourceType: "image"` for PDF/ZIP | PDFs/ZIPs are "raw" assets; image type breaks them. | `uploader.Upload(..., ResourceType: "raw")` |
| Trusting client-side Zod validation only | Client validation is UX, not security. | Mirror file-type/size + field validation in Go too |
| Hand-rolled UI components | Violates the explicit "no hand-rolled components" constraint. | shadcn/ui (`npx shadcn add ...`) |

## Stack Patterns by Variant

- In Gin, set `router.MaxMultipartMemory` and reject early; validate the real MIME via `http.DetectContentType` on the first 512 bytes (allow `application/pdf`, `application/zip`/`application/x-zip-compressed`), reject by extension AND sniffed type.
- Enforce the 10MB cap with `c.Request.ContentLength` check + a `http.MaxBytesReader` so a lying client can't stream past the limit.
- Only after validation passes, stream to Cloudinary `uploader.Upload(ctx, file, uploader.UploadParams{ResourceType: "raw"})`.
- bcrypt-hash passwords at account creation (default = birthday `DDMMYYYY`), set a `must_change_password` flag → force change on first login.
- Issue a short-lived JWT (golang-jwt v5) carrying `user_id` + `role` claim; Gin middleware validates the token and gates routes by role (student/lecturer/admin). Stateless = no Redis needed at this scale.
- One daily gocron job (or `time.Ticker`) running `UPDATE courses SET deleted_at = now() WHERE deleted_at IS NULL AND end_date < now() - interval '1 month'`. All reads filter `WHERE deleted_at IS NULL`. Keeps history (soft delete) per the requirement.
- Explicit `INSERT INTO audit_log (...)` in admin handlers (or a Gin middleware on admin routes). sqlc's explicit SQL makes this trivial and reviewable — another reason sqlc beats an ORM here.

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| sqlc v1.31.1 | pgx/v5, PostgreSQL 17 | Set `sql_package: "pgx/v5"` and `engine: "postgresql"` in `sqlc.yaml`. |
| Go 1.24 | pgx v5.7.x | pgx ≥v5.8 raise their `go` directive to 1.25 (`go mod tidy` under Go 1.24 resolves pgx to v5.7.2). To adopt pgx ≥v5.10, bump the toolchain to Go 1.25. |
| cloudinary-go/v2 ≥2.8 | Go 1.20–1.24 | Use ≥ the June 2025 security release. Import path includes `/v2`. |
| testcontainers-go | Docker in CI | GitHub Actions ubuntu runners include Docker — works out of the box. |
| shadcn/ui (current CLI) | React 19 + Tailwind v4 + Vite 6 | Only *new* projects default to Tailwind v4 / React 19; init via current CLI. |
| @hookform/resolvers v3 | React Hook Form v7 + Zod v4 | resolvers supports Zod 4, Zod 4-mini, and Zod 3 — pin Zod v4. |
| TanStack Query v5 | React 19 | v5 fully supports React 19. |
| golang-jwt/jwt/v5 | Go 1.24 | v5 has stricter default validation than v4 — review claims setup on upgrade. |

## Sources

- pkg.go.dev/github.com/jackc/pgx/v5 — pgx v5.7.2 pinned for Go 1.24 (v5.10.0 requires Go 1.25.0) — HIGH
- github.com/sqlc-dev/sqlc/releases — sqlc v1.31.1 (Apr 22, 2026) — HIGH
- github.com/golang-jwt/jwt/releases — golang-jwt v5.3.1 — HIGH
- gin-gonic.com / github.com/gin-gonic/gin/releases — Gin v1.11.0 stable, v1.12.0 experimental — HIGH
- github.com/TanStack/query/releases — TanStack Query v5.101.0 (Jun 2, 2026) — HIGH
- npmjs.com/package/react-hook-form — RHF v7.79.0; @hookform/resolvers supports Zod 4 — HIGH
- ui.shadcn.com/docs/tailwind-v4, /docs/installation — shadcn Vite+Tailwind v4+React 19 setup — HIGH
- cloudinary.com/documentation/go_integration; pkg.go.dev/github.com/cloudinary/cloudinary-go — v2 import path, raw resource type, Jun 2025 security patch — HIGH
- OWASP Password Storage Cheat Sheet / RFC 9106 (via reintech.io, alexedwards.net) — argon2id vs bcrypt guidance — MEDIUM
- encore.dev/resources/go-orms; brandur.org/sqlc; glukhov.org — sqlc vs GORM vs pgx consensus — MEDIUM
- github.com/golang-migrate/migrate, github.com/pressly/goose — migration tool comparison — MEDIUM
- golang.testcontainers.org/modules/postgres — testcontainers postgres module + pgx driver — MEDIUM
- github.com/go-co-op/gocron, github.com/robfig/cron — scheduler comparison — MEDIUM

<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->

## Conventions

Conventions not yet established. Will populate as patterns emerge during development.
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->

## Architecture

**Style: Feature-Oriented Monolith** (locked — PROJECT.md decision **D-10**). One Go/Gin API binary, one Postgres, one React SPA, Cloudinary for files. Organize backend code BY BUSINESS FEATURE, not by technical layer.

```
backend/
  cmd/api/main.go        # wiring + entrypoint
  internal/
    auth/                # handler.go service.go repository.go model.go dto.go
    courses/
    assignments/
    quizzes/
    grades/
    announcements/
    requests/
    auditlogs/
    shared/              # cross-cutting infra: config/ db/ cloudinary/ middleware/ auth(jwt)/ health/
  db/migrations/         # golang-migrate SQL — incremental per-phase (D-06)
  db/queries/            # sqlc input  (generated Go → internal/shared/db)
```

**Responsibility split INSIDE each feature** (clean architecture — this is the boundary that matters):
- `handler.go` — HTTP transport only (parse/validate request, shape JSON). No business logic, no SQL.
- `service.go` — business rules + authorization/ownership checks ("lecturer owns this course"). Orchestrates repos + shared.
- `repository.go` — DB access only (sqlc/pgx). No business rules.

**Rules:**
- Navigate by "which feature?" before "which layer?". Feature folder names mirror the language in REQUIREMENTS/ROADMAP.
- Cross-feature dependencies ARE allowed when business rules need them (grades read quiz attempts + submissions; announcement recipients derive from enrollments; request approval checks course ownership). No hard bounded-context enforcement.
- Do NOT add interfaces/abstractions/package boundaries purely for architectural purity (Ponytail). Add an interface only where a real second implementation or a test seam needs it.
- Each phase adds its own feature folder + migration (incremental, D-06). Phase 1 ships only `internal/shared/` infra (config, db, health) + the `users`/`audit_log` migrations; feature folders begin Phase 2.
<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->

## Project Skills

No project skills found. Add skills to any of: `.claude/skills/`, `.agents/skills/`, `.cursor/skills/`, `.github/skills/`, or `.codex/skills/` with a `SKILL.md` index file.
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->

## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:

- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->

<!-- GSD:profile-start -->

## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
