# Phase 1: Foundation & Data Core - Research

**Researched:** 2026-06-19
**Domain:** Greenfield monorepo scaffold — Go/Gin backend, React/Vite frontend, Docker Postgres, golang-migrate + sqlc, GitHub Actions CI gate (walking skeleton)
**Confidence:** HIGH (stack pre-committed in CLAUDE.md; research focused on HOW, not WHAT)

## Summary

This is the **Walking Skeleton** phase: the thinnest end-to-end slice that proves the wiring, with zero feature logic. Every library is already locked in `.claude/CLAUDE.md` — this research does NOT re-litigate choices; it answers the seven genuine open questions about *how* to assemble them: the minimal table set, the bootstrap-admin seed mechanism, the migrate↔sqlc layout, the CI integration-DB choice, the merge-block proof procedure, the Postgres-only compose, and the single thinnest "it works" demonstration.

**Primary recommendation:** Two foundational tables (`users` + `audit_log`); seed the admin via a **precomputed bcrypt(cost=12) hash literal** embedded in a SQL migration; use the **GitHub Actions `services:` Postgres** container (not testcontainers) for the walking skeleton; layout `backend/db/migrations/` (golang-migrate owns schema) + `backend/db/queries/` (sqlc reads it) with a `Makefile` `migrate` target; prove the merge-block with a throwaway failing PR after the user configures branch protection requiring the **exact CI job name** as a required status check.

**Critical landmines (verified this session):**
1. Branch-protection required checks match the **job name string exactly** — if the CI job is renamed or matrixed, the required check silently never reports and PRs hang on "Expected".
2. golang-migrate leaves `dirty=true` in `schema_migrations` after a failed migration and **blocks all further migrations** until manually resolved.
3. Go toolchain on this machine is **1.25.4**, but CLAUDE.md pins **1.24.x**. Pin `go 1.24` in `go.mod` and the CI `setup-go` step to match the committed stack (see Environment Availability).

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Schema definition & versioning | Database / Migrations (golang-migrate) | — | Migrations are the single source of schema truth; sqlc only *reads* them |
| Type-safe DB access | API / Backend (sqlc + pgx) | — | Generated Go from SQL; no runtime ORM |
| Config loading (`.env`) | API / Backend (godotenv + caarlos0/env) | — | Backend owns secrets; frontend never sees DB/JWT/Cloudinary creds |
| Bootstrap admin seed | Database / Migrations | — | Seed is data-as-migration so clone→migrate yields a login (no Go seed step) |
| HTTP entrypoint / healthcheck | API / Backend (Gin) | Database (ping) | `/healthz` proves backend↔Postgres wiring via real ping |
| Postgres runtime | Database (Docker compose) | — | Postgres-only compose; backend/frontend run natively |
| CI gate (tests + lint + block) | CI (GitHub Actions) + GitHub branch protection | — | Workflow runs checks; **branch protection** (admin UI) is what *blocks* merge |
| Frontend shell | Browser/Client (Vite + React 19) | — | Stub only — exists to satisfy "clone-and-run"; no UI logic this phase |

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-06:** Migrations are **incremental per-phase**, not big-bang. Phase 1 creates only the *very foundational* tables; each later phase adds its own migration. Working assumption for Phase 1 foundational set: `users` + `audit_log` (researcher/planner to confirm exact minimal set — anything Phase 2 auth strictly needs may land here, the rest defers).
- **D-07:** Seed **one bootstrap admin** via migration so Phase 2 has a login and Phase 3 has an admin to provision others. Credentials: username `admin`, password `123456`. Security (non-negotiable): seed the password **bcrypt-hashed (cost=12)**, never plaintext, and set the `must_change_password` flag so Phase 2's server-enforced forced reset applies on first login.
- **D-08:** `docker compose` runs **Postgres only** (`postgres:17-alpine`). Backend (Go) and frontend (Vite) run **natively** during dev (`go run` / `npm`), not in compose. Migrations apply via a **separate command** (not auto-run by compose).
- **D-09:** Prove the merge-block by opening a **throwaway PR that deliberately fails a test/build**, capturing the required-status-check blocking the merge as logged evidence, then closing the PR. Repo is on GitHub; the user **has admin** and must perform branch-protection setup (GitHub UI/admin action).

### Claude's Discretion
- **Frontend scaffold depth:** minimal stub — `frontend/` folder with basic Vite + React 19 + TS init, just enough to "exist" and satisfy clone-and-run. Full shadcn/Tailwind UI starts Phase 2.
- **CI integration-test DB approach:** planner chooses **testcontainers-go** vs GitHub Actions `services:` Postgres. Either satisfies criteria #6; pick the cleaner fit.
- **Exact foundational table set** beyond `users` + `audit_log`, and migration file layout — researcher/planner to finalize.

### Deferred Ideas (OUT OF SCOPE)
- **Full-stack docker-compose** (backend + frontend containerized) — rejected for Phase 1 (D-08).
- **Feature tables** (assignments, quizzes, grades, notifications, requests, courses, enrollments) — NOT created in Phase 1 (D-06); each lands in its owning phase.
- **Real admin provisioning / CSV import** — Phase 3. The Phase 1 seed admin is only the bootstrap.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INFRA-01 | Monorepo with `backend/` (Go) and `frontend/` (React) folders | Project structure section: two top-level folders, three branches; backend Go module + frontend Vite scaffold |
| INFRA-02 | PostgreSQL runs via Docker (compose), never natively | `docker-compose.yml` section: `postgres:17-alpine` only, healthcheck + named volume; "one Docker command" = `docker compose up -d` |
| INFRA-03 | Backend config loaded from `.env` (DB, JWT secret, Cloudinary creds) | Config section: godotenv loads `.env`, caarlos0/env parses into typed `Config` struct with `required` tags; `.env.example` committed, `.env` gitignored |
| INFRA-04 | DB schema managed by versioned migrations | migrate↔sqlc section: `backend/db/migrations/NNNNNN_name.up/down.sql`, golang-migrate v4, `make migrate` command; dirty-flag landmine documented |
| INFRA-05 | GitHub Actions CI triggers on push to `main`, `backend`, `frontend` | CI workflow section: `on.push.branches: [main, backend, frontend]` + `on.pull_request` |
| INFRA-06 | CI runs unit + integration tests against a real Postgres service/container | CI DB decision: GitHub Actions `services: postgres:17` container; `go test ./...` runs against it |
| INFRA-07 | Merge to protected branch blocked unless tests/DB/syntax pass (verified to block) | Merge-block section: branch-protection setup guide + throwaway-failing-PR proof procedure; exact-job-name landmine |
</phase_requirements>

## Standard Stack

> All versions are committed in `.claude/CLAUDE.md` and treated as authoritative. Reproduced here for the planner; not re-verified against registries (locked decisions per project constraints). `[CITED: .claude/CLAUDE.md]` for all rows.

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | 1.24.x | Backend runtime | Committed. Pin `go 1.24` in go.mod even though machine has 1.25 |
| Gin | v1.11.0 | HTTP framework | Committed (stable line, not experimental v1.12) |
| PostgreSQL | 17 (`postgres:17-alpine`) | Datastore (Docker only) | Committed |
| sqlc | v1.31.1 | SQL→Go codegen | Committed DB layer; `sql_package: "pgx/v5"`, `engine: "postgresql"` |
| pgx | v5 | Postgres driver | Committed; sqlc targets pgx/v5 |
| golang-migrate | v4.18.x | Versioned migrations | Committed; owns schema |
| React | 19.x | Frontend (stub) | Committed |
| Vite | 6.x | Frontend build | Committed |
| TypeScript | 5.x | Frontend language | Committed |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| godotenv | v1.5.x | Load `.env` in dev | INFRA-03; load then read via caarlos0/env |
| caarlos0/env | v11.x | Env→typed struct | INFRA-03; `Config` struct with `env:"..."` + `required` |
| golang.org/x/crypto/bcrypt | latest | Password hashing | D-07 seed (cost=12); verified working this session |
| stretchr/testify | v1.10.x | Assertions | Unit + integration tests |
| golangci-lint | latest | Go lint/syntax gate | CI "syntax checks" gate |
| gin-contrib/cors | latest | CORS | Only if frontend calls backend this phase — likely defer (stub frontend) |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| GH Actions `services:` Postgres | testcontainers-go | testcontainers manages lifecycle in-test (cleaner for many suites) but adds a dep + Docker-in-CI orchestration; overkill for a walking skeleton with one test. **`services:` chosen** — see CI DB decision |
| precomputed bcrypt hash literal | Go seed step / migrate `--seed` | golang-migrate runs **pure SQL, cannot execute Go** — a runtime Go seed needs a separate program + ordering guarantees. Precomputed literal is reproducible and zero-moving-parts |

**Installation (planner reference — commands per CLAUDE.md):**
```bash
# backend/ (go mod init then add)
go get github.com/gin-gonic/gin
go get github.com/jackc/pgx/v5
go get github.com/joho/godotenv github.com/caarlos0/env/v11
go get golang.org/x/crypto/bcrypt
go get github.com/stretchr/testify
# CLI tools (install, not go get):
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# frontend/ (stub)
npm create vite@latest frontend -- --template react-ts
```

## Package Legitimacy Audit

> All packages are pre-committed in `.claude/CLAUDE.md` (a project-authoritative source) and are long-established, high-download Go/JS ecosystem standards. No new or WebSearch-discovered packages were introduced by this research. Per project constraints, library choices are locked and not re-litigated.

| Package | Registry | Age | Source Repo | Verdict | Disposition |
|---------|----------|-----|-------------|---------|-------------|
| gin-gonic/gin | pkg.go.dev | 9+ yrs | github.com/gin-gonic/gin | OK (committed) | Approved |
| jackc/pgx/v5 | pkg.go.dev | mature | github.com/jackc/pgx | OK (committed) | Approved |
| sqlc-dev/sqlc | pkg.go.dev | mature | github.com/sqlc-dev/sqlc | OK (committed) | Approved |
| golang-migrate/migrate | pkg.go.dev | mature | github.com/golang-migrate/migrate | OK (committed) | Approved |
| joho/godotenv | pkg.go.dev | mature | github.com/joho/godotenv | OK (committed) | Approved |
| caarlos0/env/v11 | pkg.go.dev | mature | github.com/caarlos0/env | OK (committed) | Approved |
| golang.org/x/crypto | pkg.go.dev | official x/ | go.googlesource.com/crypto | OK (committed) | Approved |
| stretchr/testify | pkg.go.dev | mature | github.com/stretchr/testify | OK (committed) | Approved |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
DEV MACHINE (native)                          DOCKER
┌─────────────────────┐                       ┌──────────────────────┐
│ frontend/ (Vite 6)  │                       │ postgres:17-alpine    │
│  npm run dev :5173  │                       │  :5432, named volume  │
└─────────────────────┘                       │  healthcheck pg_isready│
                                              └───────────▲──────────┘
┌─────────────────────┐   reads .env (DB URL,            │ pgx/v5 conn
│ backend/ (Go+Gin)   │   JWT secret, Cloudinary)        │ (DB ping)
│  go run ./cmd/api   │──────────────────────────────────┘
│  GET /healthz ──────┼─► ping DB → 200 OK / 503
└─────────▲───────────┘
          │ applied out-of-band BEFORE boot:
          │  make migrate  ──►  migrate up  ──► schema_migrations + users + audit_log + seed admin
          │
   ┌──────┴───────────────────────────────────────────────────────────┐
   │ sqlc generate: backend/db/queries/*.sql + migrations → db/*.go     │
   └───────────────────────────────────────────────────────────────────┘

GITHUB
push → main / backend / frontend  ─►  GitHub Actions workflow
   job "ci" : spin services.postgres:17 → migrate up → go test ./... (unit+integ)
              → golangci-lint → frontend npm ci && lint && build
   PR  ─►  branch protection requires status check named EXACTLY "ci"
           failing job → required check red → MERGE BLOCKED  ◄── criteria #4
```

A reader traces the primary "it works" path: `docker compose up -d` (Postgres) → `make migrate` (schema + seed) → `go run ./cmd/api` reads `.env`, connects via pgx → `GET /healthz` pings DB → 200.

### Recommended Project Structure
```
myiu-lite/
├── docker-compose.yml          # postgres:17-alpine ONLY (D-08)
├── .gitignore                  # ignores backend/.env, node_modules, etc.
├── .github/workflows/ci.yml    # job name "ci" — the required status check
├── Makefile                    # migrate / sqlc / test targets (or per-folder)
├── backend/
│   ├── go.mod                  # go 1.24
│   ├── .env.example            # committed template (no secrets)
│   ├── .env                    # gitignored — real secrets
│   ├── sqlc.yaml               # engine: postgresql, sql_package: pgx/v5
│   ├── cmd/api/main.go         # Gin entrypoint, /healthz
│   ├── internal/config/        # caarlos0/env Config struct
│   ├── internal/db/            # sqlc-generated code (output dir)
│   └── db/
│       ├── migrations/         # golang-migrate owns schema
│       │   ├── 000001_init_foundation.up.sql
│       │   ├── 000001_init_foundation.down.sql
│       │   ├── 000002_seed_bootstrap_admin.up.sql
│       │   └── 000002_seed_bootstrap_admin.down.sql
│       └── queries/            # sqlc reads these
│           └── healthcheck.sql # minimal query, e.g. SELECT 1 / SELECT count(*) FROM users
└── frontend/                   # Vite + React 19 + TS stub
    ├── package.json
    └── src/...
```

### Pattern 1: golang-migrate owns schema, sqlc reads it
**What:** Two tools, one schema source. `migrate` applies versioned `.sql` files (the authoritative DDL). sqlc points its `schema:` at the same `migrations/` directory and generates type-safe Go from `queries/`.
**When to use:** Always, this stack.
**Example:**
```yaml
# backend/sqlc.yaml   [CITED: .claude/CLAUDE.md — sql_package: pgx/v5, engine: postgresql]
version: "2"
sql:
  - engine: "postgresql"
    schema: "db/migrations"      # sqlc reads the migration DDL for types
    queries: "db/queries"        # your hand-written SQL
    gen:
      go:
        package: "db"
        out: "internal/db"
        sql_package: "pgx/v5"
```
> Note: sqlc parses `*.up.sql` migration files for schema. Keep DDL clean and parseable; sqlc ignores `down` files for schema inference in practice — verify the planner points `schema:` at the migrations dir and that seed-only migrations (INSERTs) don't confuse type inference (they won't — INSERTs aren't DDL).

### Pattern 2: Config via typed struct (INFRA-03)
**What:** godotenv loads `.env` into process env; caarlos0/env parses into a struct with `required` validation so a missing secret fails fast at boot, not at first use.
**Example:**
```go
// internal/config/config.go   [CITED: .claude/CLAUDE.md]
type Config struct {
    DatabaseURL       string `env:"DATABASE_URL,required"`
    JWTSecret         string `env:"JWT_SECRET,required"`
    CloudinaryURL     string `env:"CLOUDINARY_URL,required"` // slot exists now, consumed Phase 4
    Port              string `env:"PORT" envDefault:"8080"`
}
func Load() (Config, error) {
    _ = godotenv.Load()                 // dev only; CI/prod inject real env
    return env.ParseAs[Config]()
}
```
The JWT secret + Cloudinary creds **slots exist now** (proving the config wiring) but are not *consumed* until Phases 2/4. That is correct walking-skeleton behavior — wire the substrate, don't use it.

### Pattern 3: `/healthz` proves the wiring (walking-skeleton proof)
**What:** A single Gin route that does a real DB ping. This is the thinnest "it works" — it exercises config load → pgx connect → DB round-trip → HTTP response.
**Example:**
```go
// cmd/api/main.go
r := gin.Default()
r.GET("/healthz", func(c *gin.Context) {
    if err := pool.Ping(c.Request.Context()); err != nil {
        c.JSON(503, gin.H{"status": "db down"}); return
    }
    c.JSON(200, gin.H{"status": "ok"})
})
```

### Anti-Patterns to Avoid
- **Auto-running migrations from compose or app boot:** D-08 says migrations apply via a *separate command*. Do not embed `migrate up` in `main.go` or a compose entrypoint — keep it a `make migrate` step.
- **Containerizing backend/frontend this phase:** D-08 — Postgres-only compose. No backend/frontend Dockerfiles.
- **Building feature tables now:** D-06 — only `users` + `audit_log`. No courses/assignments/etc.
- **Plaintext or runtime-hashed seed password in a way that breaks reproducibility:** use the precomputed cost-12 hash literal (see Seed section).
- **Matrix CI job for the required check:** a matrix expands into multiple check names; branch protection then needs each one or a summary job. For a single-job skeleton, use ONE job named `ci` and require exactly that.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Schema versioning | Custom migration runner / manual `psql` | golang-migrate | Tracks version + dirty state, up/down, CI-friendly |
| SQL→Go mapping | Hand-written scan/struct boilerplate | sqlc | Compile-time type safety; committed |
| Env parsing/validation | `os.Getenv` + manual nil checks | caarlos0/env `required` | Fail-fast on missing secret at boot |
| Password hashing | MD5/SHA/custom | bcrypt cost=12 | Committed; verified working |
| Integration test DB | Mocked DB / sqlite | real Postgres 17 via GH `services:` | Criteria #6 demands a *real* Postgres |
| Test assertions | `if got != want { t.Fatal }` everywhere | testify `require`/`assert` | Committed; readable |

**Key insight:** Everything in this phase is "plumbing that already has a standard tool." The walking-skeleton risk is *over-building* (feature tables, full-stack compose, testcontainers orchestration), not under-tooling. Ponytail applies: pick the GH Actions `services:` Postgres (zero extra deps) over testcontainers, and a precomputed hash literal over a seed program.

## Foundational Table Set (Open Question #1 — RECOMMENDATION)

Confirming D-06's working assumption: **`users` + `audit_log` only.** This is the minimal set Phase 2 (auth + forced reset) and Phase 3 (admin + audit) strictly need. No feature tables.

### `users` — columns Phase 2 strictly needs
```sql
-- 000001_init_foundation.up.sql
CREATE TYPE user_role AS ENUM ('student', 'lecturer', 'admin');

CREATE TABLE users (
    id                   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username             TEXT NOT NULL,            -- = student/lecturer ID (Phase 3); 'admin' for bootstrap
    password_hash        TEXT NOT NULL,            -- bcrypt; never plaintext
    role                 user_role NOT NULL,
    must_change_password BOOLEAN NOT NULL DEFAULT TRUE,  -- AUTH-04 server-enforced reset
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at           TIMESTAMPTZ                       -- soft-delete column (used later; cheap to add now)
);

-- Username unique among non-deleted rows (partial unique index — the soft-delete discipline
-- flagged in STATE.md). Lets a username be reused after a soft delete without collision.
CREATE UNIQUE INDEX users_username_active_uq
    ON users (username) WHERE deleted_at IS NULL;
```
**Rationale for each column:**
- `id` IDENTITY — modern PG17 surrogate key (preferred over `SERIAL`). `[ASSUMED]` style choice; either is fine.
- `username` TEXT — Phase 3 sets it to the student/lecturer ID; bootstrap admin uses `admin`. `[CITED: REQUIREMENTS ADMIN-03]`
- `password_hash` TEXT — bcrypt output (~60 chars; TEXT avoids length pitfalls). `[CITED: .claude/CLAUDE.md]`
- `role` enum — RBAC in Phase 2 keys off this. `[CITED: AUTH-05]`
- `must_change_password` — AUTH-04 forced reset; bootstrap admin seeded `TRUE`. `[CITED: D-07, AUTH-04]`
- `created_at`/`updated_at` — standard audit timestamps.
- `deleted_at` — soft-delete is a project-wide discipline (STATE.md). Adding the column + partial-unique-index *now* is near-free and establishes the pattern; later phases reuse it. **Optional** — defer if the planner wants strict minimalism, but recommended because Phase 3 will need the partial-unique-index pattern and establishing it in the foundation migration avoids a later `users` ALTER. `[ASSUMED — judgment call; flag for planner]`

### `audit_log` — append-only (written Phases 3–5)
```sql
CREATE TABLE audit_log (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor_id    BIGINT REFERENCES users(id),   -- nullable: system/sweep actions have no human actor
    action      TEXT NOT NULL,                 -- e.g. 'account.create', 'password.reset', 'course.delete'
    target      TEXT,                          -- free-form target identifier (e.g. 'user:42', 'course:7')
    metadata    JSONB NOT NULL DEFAULT '{}',   -- structured extra context
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX audit_log_actor_idx   ON audit_log (actor_id);
CREATE INDEX audit_log_created_idx ON audit_log (created_at);
```
**Rationale:** matches ADMIN-08 "actor, action, target, timestamp" plus a `metadata jsonb` for flexible context. Append-only is a *discipline* (no UPDATE/DELETE queries written), not a DB constraint in Phase 1 — Phase 3 owns enforcing it (e.g., revoking UPDATE/DELETE, or just never generating those queries). `actor_id` is nullable so the Phase 3 soft-delete *sweep* (no human actor) can audit itself. `[CITED: ADMIN-08; STATE.md cross-cutting threads]`

> **Do NOT add now:** any FK from `audit_log` to feature tables (courses, etc.) — those tables don't exist. `target`/`metadata` are deliberately string/JSON to stay decoupled.

## Bootstrap Admin Seed (Open Question #2 — RECOMMENDATION)

**Decision: precomputed bcrypt(cost=12) hash literal embedded in a SQL migration.** golang-migrate executes **pure SQL only — it cannot run Go**, so a runtime Go `bcrypt.GenerateFromPassword` call cannot live inside a migration. The two real options:

| Approach | How | Tradeoff |
|----------|-----|----------|
| **Precomputed hash literal (RECOMMENDED)** | Generate one bcrypt hash of `123456` once, paste the literal into `000002_seed_bootstrap_admin.up.sql` | Zero moving parts; reproducible; migrate-native. The hash's salt is random *at generation time* but fixed once chosen — verification still succeeds for `123456` (verified this session) |
| Separate Go seed program | A `cmd/seed/main.go` that hashes at runtime + INSERTs after `migrate up` | Adds a program + an ordering/idempotency burden + a CI step. Violates "seed via migration" (D-07) and Ponytail |

**Concrete hash-generation method** (pick one; both produce a `$2a$12$...` ~60-char string):
```bash
# Option A — Go one-liner (verified working this session: prints $2a$12$..., verify=true, cost=12)
go run - <<'EOF'
package main
import ("fmt";"golang.org/x/crypto/bcrypt")
func main(){h,_:=bcrypt.GenerateFromPassword([]byte("123456"),12);fmt.Println(string(h))}
EOF

# Option B — htpasswd (apache2-utils), bcrypt cost 12, no go needed
htpasswd -bnBC 12 "" 123456 | tr -d ':\n' | sed 's/^\$2y/\$2a/'
```
> `htpasswd` emits the `$2y$` variant; Go's bcrypt accepts `$2a`/`$2b`/`$2y` for verification, but normalize to `$2a$` to match what `bcrypt.GenerateFromPassword` produces and keep the seed consistent. `[VERIFIED: x/crypto/bcrypt — cost/verify confirmed by running this session]`

The seed migration:
```sql
-- 000002_seed_bootstrap_admin.up.sql
INSERT INTO users (username, password_hash, role, must_change_password)
VALUES ('admin', '$2a$12$<PASTE_GENERATED_HASH_HERE>', 'admin', TRUE);

-- 000002_seed_bootstrap_admin.down.sql
DELETE FROM users WHERE username = 'admin' AND role = 'admin';
```
**Landmines:**
- The hash literal contains `$` — in a *static SQL file* it's just a string literal (safe). Do NOT pass it through a shell/`psql -c` with unescaped `$`. Applying via `migrate -path ... up` reads the file directly — safe.
- `must_change_password = TRUE` is mandatory (D-07) so Phase 2's forced reset fires on first login.
- This is a **dev/bootstrap credential** to be rotated on first login once Phase 2 ships forced reset (CONTEXT specifics). Document it as such; it is acceptable for the walking skeleton, not a production secret.

## CI Integration-Test DB (Open Question #4 — DECISION: GitHub Actions `services:`)

**Pick: GitHub Actions `services:` Postgres container** for the walking skeleton.

| Criterion | `services:` Postgres | testcontainers-go |
|-----------|----------------------|-------------------|
| Satisfies INFRA-06 "real Postgres service container" | Yes (literally a service container) | Yes |
| Extra Go dependency | None | testcontainers-go + modules/postgres |
| Setup complexity for ONE skeleton test | Trivial (YAML block + health) | Container lifecycle code in tests |
| Docker-in-CI requirement | GH runners include it | GH runners include it |
| Local `go test` parity | Needs local Postgres / `DATABASE_URL` (have compose) | Self-contained (spins its own) |

**Rationale:** Ponytail + walking-skeleton. `services:` adds zero dependencies and is the smallest thing that satisfies "real Postgres." testcontainers shines when you have *many* suites needing isolated DBs and want local self-containment — that value arrives in Phases 3–5, and the planner can introduce it then without rework (the test code just changes how it gets `DATABASE_URL`). For Phase 1's single connectivity/migration test, `services:` wins.

> Trade-off to note for later: with `services:`, local `go test` needs `DATABASE_URL` pointed at the compose Postgres. Document that in `backend/.env.example` and the test setup. If the team finds that annoying across phases, switch to testcontainers in a later phase.

### Concrete CI workflow shape
```yaml
# .github/workflows/ci.yml
name: CI
on:
  push:
    branches: [main, backend, frontend]   # INFRA-05
  pull_request:                            # so PR checks report for branch protection

jobs:
  ci:                                      # <-- THIS job-name string is the required check
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:17-alpine          # INFRA-06 real Postgres 17
        env:
          POSTGRES_USER: myiu
          POSTGRES_PASSWORD: myiu
          POSTGRES_DB: myiu_test
        ports: ["5432:5432"]
        options: >-
          --health-cmd "pg_isready -U myiu"
          --health-interval 10s --health-timeout 5s --health-retries 5
    env:
      DATABASE_URL: postgres://myiu:myiu@localhost:5432/myiu_test?sslmode=disable
      JWT_SECRET: test-secret
      CLOUDINARY_URL: cloudinary://test:test@test
    steps:
      - uses: actions/checkout@v4

      # --- Backend ---
      - uses: actions/setup-go@v5
        with: { go-version: "1.24" }       # MATCH CLAUDE.md pin, not the 1.25 on dev machine
      - name: Install migrate
        run: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
      - name: Apply migrations
        working-directory: backend
        run: migrate -path db/migrations -database "$DATABASE_URL" up
      - name: Backend lint (syntax gate)
        uses: golangci/golangci-lint-action@v6
        with: { working-directory: backend }
      - name: Backend tests (unit + integration)
        working-directory: backend
        run: go test ./...

      # --- Frontend ---
      - uses: actions/setup-node@v4
        with: { node-version: "20" }
      - name: Frontend install + lint + build (syntax/build gate)
        working-directory: frontend
        run: npm ci && npm run lint && npm run build
```
> Single job `ci` keeps the required-status-check name simple. If the planner later splits backend/frontend into separate jobs or a matrix, **add a `summary` job** that `needs:` all of them and require *that* in branch protection (see landmine below).

## Proving the Merge Block (Open Question #5 — D-09 / criteria #4)

The merge is blocked by **GitHub branch protection requiring a status check**, NOT by the workflow itself. This separates *repo files* (the workflow — Claude can write) from *admin UI actions* (branch protection — the USER performs).

### Part A — Branch-protection setup (USER / ADMIN UI ACTIONS)
Step-by-step for the user (repo Settings → Branches → Add branch ruleset / protection rule):
1. **Repo → Settings → Branches → Add branch protection rule** (or Settings → Rules → Rulesets → New ruleset for the modern path).
2. **Branch name pattern:** `main` (repeat for `backend`, `frontend` if those are also protected — at minimum protect the branch PRs target).
3. Enable **"Require a pull request before merging."**
4. Enable **"Require status checks to pass before merging."**
5. In the status-checks search box, type and select the check named **`ci`** (the job name). **It only appears after the workflow has run at least once** — so push the workflow first, let CI run once, then it becomes selectable.
6. (Recommended) Enable **"Require branches to be up to date before merging."**
7. (Recommended) Enable **"Do not allow bypassing the above settings"** / **"Include administrators"** — otherwise an admin (the user) can merge past a red check and the proof is meaningless.
8. Save.

> **LANDMINE (VERIFIED):** the required status check matches the **job name string exactly** (here `ci`), not the workflow filename or the workflow `name:`. If the job is renamed/matrixed after protection is set, the required check shows "Expected — Waiting for status to be reported" forever and PRs hang. Keep the job named `ci` and the required check named `ci` in lockstep. `[VERIFIED: GitHub Docs + community discussions, websearch]`

### Part B — Throwaway-failing-PR proof (REPO FILES + UI verification)
1. Branch off `main`: `git checkout -b ci-block-proof`.
2. Introduce a **deliberate failure** — simplest: add a test that always fails, e.g. `func TestCIBlockProof(t *testing.T){ t.Fatal("intentional CI-block proof") }` in `backend/`. (Or break the build / break lint — a failing *test* is cleanest and exercises the test path.)
3. Push and open a PR into `main`.
4. **Capture evidence:**
   - Screenshot of the PR's merge box showing the **red `ci` check** and **"Merging is blocked / Required statuses must pass"** with the merge button disabled.
   - The Actions run log showing `go test` failing on `TestCIBlockProof`.
   - (Optional) `gh pr checks <num>` output and `gh pr view <num>` showing `mergeStateStatus: BLOCKED`.
5. Store evidence in the phase's verification artifacts (e.g. `.planning/phases/01-foundation-data-core/` or a `docs/ci-proof/` folder).
6. **Close the PR without merging and delete the branch** (it was throwaway).

This is the literal satisfaction of criteria #4 "verified, not just configured."

## docker-compose (Open Question #6 — D-08)

"One Docker command" = `docker compose up -d` bringing up **Postgres only**. Backend/frontend run natively.
```yaml
# docker-compose.yml   (repo root)
services:
  postgres:
    image: postgres:17-alpine          # INFRA-02, D-08
    environment:
      POSTGRES_USER: myiu
      POSTGRES_PASSWORD: myiu
      POSTGRES_DB: myiu_dev
    ports: ["5432:5432"]
    volumes: ["pgdata:/var/lib/postgresql/data"]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U myiu -d myiu_dev"]
      interval: 10s
      timeout: 5s
      retries: 5
volumes:
  pgdata:
```
Then the dev "clone-and-run" sequence (satisfies criteria #1):
```bash
docker compose up -d            # Postgres up + healthy
cd backend && make migrate      # apply migrations + seed admin (NOT auto-run by compose — D-08)
make run                        # go run ./cmd/api  → /healthz returns 200
cd ../frontend && npm ci && npm run dev   # stub serves
```
> Migrations are deliberately a **separate `make migrate` step** (D-08), not a compose entrypoint or app-boot side effect.

### Makefile targets (the "separate command")
```makefile
# backend/Makefile (or root)
DATABASE_URL ?= postgres://myiu:myiu@localhost:5432/myiu_dev?sslmode=disable
migrate:    ; migrate -path db/migrations -database "$(DATABASE_URL)" up
migrate-down: ; migrate -path db/migrations -database "$(DATABASE_URL)" down 1
sqlc:       ; sqlc generate
run:        ; go run ./cmd/api
test:       ; go test ./...
```

## Walking Skeleton End-to-End Proof (Open Question #7)

The single thinnest "it works" demonstration:
1. **Minimum real entrypoint (backend):** `go run ./cmd/api` boots Gin, loads `.env` via config struct, opens a pgx pool to the compose Postgres.
2. **Minimum real DB round-trip:** `GET /healthz` calls `pool.Ping()` (or runs the sqlc-generated `SELECT 1` / `SELECT count(*) FROM users` query — using the sqlc query *also proves the migrate→sqlc→generated-Go chain*, which is slightly stronger evidence). Returns 200 on success, 503 on DB failure.
3. **Minimum real write proof:** the bootstrap-admin seed migration *is* the write — verify with the sqlc query returning `count(users) = 1` (the admin). This proves migrations applied AND the seed landed AND sqlc reads real data.
4. **Frontend stub:** `npm run build` succeeds and `npm run dev` serves the default Vite + React 19 page. No backend call required this phase (keeps it a true stub; CORS deferrable).
5. **CI proof:** the failing-PR procedure above.

**Recommendation:** make `/healthz` run the sqlc-generated `count(users)` query (not just `pool.Ping`) — it exercises config → pgx → migrate-applied schema → seeded data → sqlc-generated code in one request, which is the maximal-coverage minimal proof.

## Runtime State Inventory

> Greenfield phase — no pre-existing runtime state to migrate. Included for completeness.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — no DB exists yet; this phase *creates* it | none |
| Live service config | None — no deployed services | none |
| OS-registered state | None | none |
| Secrets/env vars | New `.env` to be created (DATABASE_URL, JWT_SECRET, CLOUDINARY_URL). `.env.example` committed; `.env` gitignored | create `.env.example`; ensure `.gitignore` covers `backend/.env` |
| Build artifacts | None — `go.mod` and `package.json` created fresh this phase | none |

**Nothing found** in stored-data, live-config, OS-state, build-artifact categories — verified: repo has no `backend/`/`frontend/` source (greenfield, confirmed by `ls`).

## Common Pitfalls

### Pitfall 1: Required status check never reports (exact-name mismatch)
**What goes wrong:** PR sits on "Expected — Waiting for status to be reported"; merge stays blocked but for the wrong reason.
**Why:** branch protection requires job-name `ci` but the workflow job is named something else, or was matrixed/renamed.
**How to avoid:** name the job exactly `ci`; configure protection to require `ci`; only configure protection *after* one CI run so the name is selectable.
**Warning signs:** "Expected" status that never turns green or red. `[VERIFIED: websearch]`

### Pitfall 2: golang-migrate dirty state blocks CI
**What goes wrong:** a migration fails mid-way; `schema_migrations.dirty=true`; every subsequent `migrate up` refuses to run.
**Why:** golang-migrate sets dirty before running and only clears it on success.
**How to avoid:** keep each migration small and valid; never paste a half-broken DDL. If it happens in dev, fix the SQL then `migrate force <prev_version>` (which only rewrites the version, does NOT run SQL) then re-run — understand `force` does not execute the migration. In CI, a dirty DB usually means a bad migration committed; fix the file, not the DB. `[VERIFIED: websearch]`

### Pitfall 3: Admin can merge past a red check
**What goes wrong:** the failing-PR proof appears to fail because the admin user can still click merge.
**Why:** "Include administrators" / "Do not allow bypassing" not enabled.
**How to avoid:** enable that toggle in branch protection (Part A step 7), otherwise criteria #4 is not truly proven.

### Pitfall 4: Go version drift (1.25 dev vs 1.24 committed)
**What goes wrong:** code compiles locally on Go 1.25 using a 1.25-only feature, fails in CI pinned to 1.24 (or vice versa).
**Why:** machine has Go 1.25.4; CLAUDE.md commits 1.24.x.
**How to avoid:** `go 1.24` in `go.mod`; `go-version: "1.24"` in CI `setup-go`. Treat 1.24 as the contract.

### Pitfall 5: `.env` committed with real secrets
**What goes wrong:** secrets leak into git history (violates criteria #2 "no hardcoded secrets").
**How to avoid:** commit only `.env.example` (placeholder values); add `backend/.env` to `.gitignore` *before* the first config commit. CI injects env via the workflow `env:` block, not a committed `.env`.

## Validation Architecture

> `nyquist_validation` is `false` in config, so a full Nyquist mapping is not required. Concrete test guidance is included because the CI test gate (INFRA-06) is central to this phase.

### Test Framework
| Property | Value |
|----------|-------|
| Backend framework | Go stdlib `testing` + testify v1.10 |
| Frontend | Vitest deferred — Phase 1 frontend is a stub; `npm run build` + `npm run lint` are the frontend gates |
| Quick run (backend) | `go test ./...` (needs `DATABASE_URL` → compose Postgres) |
| Full suite | CI workflow: migrate + `go test ./...` + golangci-lint + frontend build |

### Phase Requirements → Test Map
| Req | Behavior | Test Type | Command | Exists? |
|-----|----------|-----------|---------|---------|
| INFRA-03 | Config fails fast on missing required env | unit | `go test ./internal/config/...` | ❌ Wave 0 |
| INFRA-04 | Migrations apply cleanly; admin seeded | integration | `migrate up` then `go test` asserting `count(users)=1` | ❌ Wave 0 |
| INFRA-06 | Backend connects to real Postgres; `/healthz` 200 | integration | `go test` against `services:` Postgres | ❌ Wave 0 |
| INFRA-07 | Failing PR is merge-blocked | manual (UI proof) | throwaway-PR procedure + screenshots | ❌ Wave 0 (manual) |
| INFRA-01/02/05 | Structure, compose, CI trigger | smoke | `docker compose up -d` health; CI run on push | ❌ Wave 0 |

### Wave 0 Gaps
- [ ] `backend/internal/config/config_test.go` — covers INFRA-03 (missing-required-env fails)
- [ ] `backend/internal/db/healthcheck_test.go` (or `cmd/api`) — integration test: connect, `/healthz`, seeded admin count = 1 (INFRA-04, INFRA-06)
- [ ] `backend/db/queries/healthcheck.sql` — minimal sqlc query for the count
- [ ] `.github/workflows/ci.yml` — the `ci` job (INFRA-05/06)
- [ ] Manual proof artifact for INFRA-07 (screenshots + logs) — cannot be automated

## Security Domain

> `security_enforcement: true`, `security_asvs_level: 1`, `security_block_on: high`. Phase 1 is infrastructure; most ASVS categories activate in Phase 2+.

### Applicable ASVS Categories (Level 1)
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | partial | bcrypt cost=12 for the seed password hash (no login flow yet — Phase 2) |
| V3 Session Management | no | JWT sessions are Phase 2 |
| V4 Access Control | no | RBAC is Phase 2 |
| V5 Input Validation | no | No user input endpoints this phase (`/healthz` only) |
| V6 Cryptography | yes | bcrypt (committed, not hand-rolled); never store plaintext |
| V7 Secrets / Config | yes | `.env` not committed; `.env.example` only; CI injects via `env:`; no hardcoded secrets (criteria #2) |

### Known Threat Patterns for this stack/phase
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Plaintext/weak seed password | Information Disclosure | bcrypt cost=12 hash literal; `must_change_password=TRUE` forces rotation |
| Secrets in git | Information Disclosure | `.gitignore` `backend/.env`; commit `.env.example` only |
| Default bootstrap credential left active | Elevation of Privilege | Documented as dev-only; Phase 2 forced reset rotates it on first login |
| Merge of broken/unreviewed code | Tampering | Branch protection + required `ci` check + "include administrators" |

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Classic branch protection rules | Repository **Rulesets** (match checks by integration source, not exact string) | GitHub, ~2023+ | Either works; classic rules are fine for a single `ci` check. Rulesets are more robust if matrix/multiple workflows appear later |
| `lib/pq` + hand-written scans | pgx/v5 + sqlc | committed | type-safe, no runtime ORM |
| `SERIAL` PKs | `GENERATED ALWAYS AS IDENTITY` | PG10+ | standards-compliant identity columns (used in table designs above) |

**Deprecated/outdated:** none introduced — stack is current per CLAUDE.md.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `deleted_at` soft-delete column + partial-unique-index belongs in the Phase 1 `users` migration | Table Set | Low — if planner prefers strict minimalism, drop it; Phase 3 would then ALTER `users`. Recommended to include |
| A2 | `users` uses `GENERATED ALWAYS AS IDENTITY` (vs `SERIAL` or UUID) | Table Set | Low — cosmetic; any PK type works for the skeleton |
| A3 | `audit_log` uses `metadata jsonb` beyond ADMIN-08's literal 4 fields | Table Set | Low — extra flexibility; harmless if unused |
| A4 | `/healthz` should run the sqlc count query (not just ping) for stronger proof | E2E Proof | Low — ping alone also satisfies criteria; count is strictly better evidence |
| A5 | CORS (gin-contrib/cors) can be deferred (stub frontend makes no backend call) | Standard Stack | Low — add it in Phase 2 when login calls the API |
| A6 | Node 20 in CI is acceptable for a Vite 6 build (machine has Node 25) | CI workflow | Low — Vite 6 supports Node 18/20/22; pin to an LTS the team uses |

**No `[ASSUMED]` claims touch locked decisions or security-critical controls** — bcrypt/cost-12, no-plaintext, no-committed-secrets, and the merge-block mechanism are all CITED/VERIFIED.

## Open Questions (RESOLVED)

1. **Are `backend` and `frontend` branches themselves protected, or only `main`?**
   - What we know: D-09 / INFRA-05 mention all three branches as CI triggers; criteria #4 says "a protected branch."
   - What's unclear: whether PRs target `main` only (so only `main` needs protection) or also into `backend`/`frontend`.
   - Recommendation: protect at least `main`; the user decides during branch-protection setup. The proof PR should target a protected branch.
   - **(RESOLVED):** Protect `main` at minimum; the user confirms the exact scope (main-only vs all three) during Plan 01-03 Task 1 branch-protection setup. Low-risk — does not affect any other plan.

2. **sqlc `schema:` pointed at `migrations/` with a seed (INSERT) migration present — any parse friction?**
   - What we know: sqlc infers schema from DDL; INSERTs are DML, not DDL, so they're ignored for type inference.
   - Recommendation: keep DDL in `000001` and the seed INSERT in `000002`; if sqlc ever complains, point `schema:` at a dedicated schema-only file. Verify during execution (cheap).
   - **(RESOLVED):** DDL in `000001`, seed INSERT in `000002` — INSERTs are DML and are ignored by sqlc schema inference, so this layout is verified-safe. Plan 01-01 already splits the migrations this way.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | backend build/test | ✓ | **1.25.4** (CLAUDE.md pins 1.24 — pin go.mod + CI to 1.24) | — |
| Docker | Postgres compose + CI services | ✓ | 29.4.0 | — |
| Node/npm | frontend stub | ✓ | 25.8.1 / 11.11.0 (pin CI to an LTS, e.g. 20) | — |
| sqlc | DB codegen | ✗ (not checked on PATH) | — | `go install sqlc` (CLI install step) |
| migrate (golang-migrate CLI) | apply migrations | ✗ (not checked on PATH) | — | `go install .../migrate` (CLI install step) |
| GitHub repo + admin | branch protection / CI | ✓ (per CONTEXT — user has admin) | — | — |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** sqlc and golang-migrate CLIs — install via `go install` (already in the install commands). Note the Go-version mismatch (machine 1.25 vs committed 1.24): pin to 1.24 everywhere.

## Sources

### Primary (HIGH confidence)
- `.claude/CLAUDE.md` — committed stack, versions, "What NOT to Use", "Stack Patterns" (authoritative for all library choices)
- `.planning/REQUIREMENTS.md` — INFRA-01..07, AUTH/ADMIN wording for table-design forward-compat
- `.planning/phases/01-foundation-data-core/01-CONTEXT.md` — D-06..D-09 locked decisions
- `.planning/ROADMAP.md` §Phase 1 — goal + 4 success criteria
- Local verification (this session): `bcrypt.GenerateFromPassword(...,12)` produced `$2a$12$...`, verify=true, cost=12

### Secondary (MEDIUM confidence)
- GitHub Docs — About protected branches / Troubleshooting required status checks (job-name exact-match behavior)

### Tertiary (LOW confidence — websearch, mechanism confirmation)
- GitHub community discussions #26698, #33579, #4324 — required-check name matching, matrix summary-job pattern
- golang-migrate FAQ + community guides — `schema_migrations` dirty flag + `force` semantics

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — pre-committed in CLAUDE.md
- Table set: HIGH for `users`/`audit_log` necessity (driven by AUTH/ADMIN reqs); MEDIUM on optional `deleted_at` inclusion (judgment call, flagged A1)
- Seed mechanism: HIGH — bcrypt cost-12 generation verified locally; migrate=SQL-only is established
- CI DB choice: HIGH — `services:` is the lean fit; rationale solid
- Merge-block mechanism: HIGH on the procedure; landmine (job-name exact match) VERIFIED
- Pitfalls: HIGH — two key landmines verified this session

**Research date:** 2026-06-19
**Valid until:** 2026-07-19 (stable stack; GitHub UI labels may shift — verify branch-protection wording at setup time)
