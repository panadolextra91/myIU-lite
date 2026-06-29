# myIU (Lite Edition)

A lite student-management platform for a university. **myIU** gives three actors — **Students**, **Lecturers**, and **Admins** — a single place to run coursework end-to-end without falling back to email: assignment submission, auto-graded quizzes, a weighted gradebook, announcements, and student↔lecturer requests. Admins provision everything (accounts, enrollment, courses) from CSV.

> **Core value:** a course can be run to completion — assignments, quizzes, grades, announcements, requests — entirely inside the app. There is **no email anywhere** in the system; cross-actor communication is the persisted notification center and the request round-trip.

---

## Screenshots

Redesigned in a **"Dark Academia"** visual language — editorial typography (Cormorant Garamond · Inter · JetBrains Mono), a warm paper/walnut palette, and first-class **light + dark** themes.

| | |
|---|---|
| ![Login](docs/screenshots/login.png) | ![Role dashboard](docs/screenshots/dashboard.png) |
| **Login** — forced first-login password change | **Role dashboard** — editorial table-of-contents landing |
| ![Quizzes](docs/screenshots/quizzes.png) | ![Courses — dark mode](docs/screenshots/courses-dark.png) |
| **Quizzes** — question-bank authoring & availability windows | **Courses (dark mode)** — admin sidebar + editorial data table |

---

## Features

| Area | What it does | Actors |
|------|--------------|--------|
| **Auth & RBAC** | Cookie-based JWT (access + refresh), role-gated routes, forced password change on first login | All |
| **Admin provisioning** | CSV import of student/lecturer accounts, course CRUD, CSV enrollment / lecturer assignment, append-only audit log | Admin |
| **Course lifecycle** | Soft-delete courses; auto-sweep one month after end date | Admin / System |
| **Assignments** | Versioned PDF/ZIP submissions (Cloudinary, 10 MB), per-assignment late policy, grading + feedback | Lecturer / Student |
| **Quizzes** | Question-bank with random per-attempt draw, CSV/UI authoring, exact-match auto-grading, availability windows, MAX-of-attempts scoring | Lecturer / Student |
| **Gradebook** | Hierarchical weighted scheme (Inclass + Midterm + Final = 100%), AUTO (normalized quiz/assignment averages) + MANUAL leaves, component-level publication with frozen snapshots | Lecturer / Student |
| **Announcements** | Immutable, fan-out to all or specific students, persisted browse + bell delivery | Lecturer / Student |
| **Requests** | Directed student→lecturer requests (leave-early / absence / custom), single approve/deny reply, both directions notified | Student / Lecturer |
| **Notifications** | One persisted, fully-rendered row per recipient; header bell + list page; written in the same transaction as the event | All |

A full record of every design decision and its rationale lives in [`DECISION_LOGS.md`](./DECISION_LOGS.md); the technical design is in [`ARCHITECTURE.md`](./ARCHITECTURE.md).

---

## Tech stack

| Layer | Stack |
|-------|-------|
| **Backend** | Go 1.24 · Gin · **sqlc + pgx v5** (raw SQL, type-safe — no ORM) · golang-migrate · golang-jwt v5 · bcrypt · Cloudinary |
| **Frontend** | React 19 · Vite 6 · TypeScript · Zustand (UI/auth state) · TanStack Query (server state) · shadcn/ui + Tailwind v4 · React Hook Form + Zod · axios · react-router |
| **Database** | PostgreSQL 17 (Docker only) |
| **CI / gate** | GitHub Actions + `scripts/check.sh` (golangci-lint, build, vet, real-Postgres tests, frontend lint/build) |

See [`ARCHITECTURE.md`](./ARCHITECTURE.md#tech-stack--versions) for pinned versions and the reasoning (and [`.claude/CLAUDE.md`](./.claude/CLAUDE.md) for the full stack rationale).

---

## Repository layout

```
backend/                Go API (one Gin binary)
  cmd/api/main.go        wiring + entrypoint
  internal/
    auth/ courses/ enrollments/ assignments/ quizzes/
    grades/ announcements/ requests/ auditlogs/ notifications/ lifecycle/
                         one feature folder each: handler.go / service.go / repository.go / model.go / dto.go
    shared/              cross-cutting infra: config, db (sqlc output), middleware, auth (JWT), authz, cloudinary, health
  db/migrations/         golang-migrate SQL (incremental per phase, 000001–000008)
  db/queries/            sqlc input (generated → internal/shared/db)
frontend/                React SPA (Vite)
  src/routes/ stores/ lib/ pages/{admin,lecturer,student} components/
.planning/               product docs: PROJECT.md, ROADMAP.md, REQUIREMENTS.md, DESIGN-SYSTEM.md, per-phase context
scripts/check.sh         local CI-parity gate (the Definition of Done)
```

The codebase is organized **by business feature, not by technical layer** (Feature-Oriented Monolith — decision [D-10](./DECISION_LOGS.md)). Inside each feature: `handler` = HTTP only, `service` = business rules + authorization, `repository` = SQL only.

---

## Getting started

### Prerequisites

- **Docker** (Postgres runs in Docker only — never natively)
- **Go 1.24.x**
- **Node 20.x** (frontend)
- CLI tools: [`golang-migrate`](https://github.com/golang-migrate/migrate) and (only if you change SQL) [`sqlc`](https://sqlc.dev)
  ```bash
  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
  ```

### 1. Start Postgres

```bash
docker compose up -d postgres          # postgres:17-alpine on localhost:5432 (db: myiu_dev, user/pass: myiu/myiu)
```

### 2. Backend

```bash
cd backend
cp .env.example .env                   # DATABASE_URL, JWT_SECRET, CLOUDINARY_URL, PORT, FRONTEND_ORIGIN, COOKIE_SECURE
make migrate                           # apply migrations 000001–000008 to myiu_dev
make run                               # go run ./cmd/api  →  http://localhost:8080
```

> For local HTTP dev, set `COOKIE_SECURE=false` in `.env` if your browser refuses the auth cookie (it defaults to `true` for production).

`.env` keys (all of `DATABASE_URL`, `JWT_SECRET`, `CLOUDINARY_URL` are **required**):

| Key | Example | Notes |
|-----|---------|-------|
| `DATABASE_URL` | `postgres://myiu:myiu@localhost:5432/myiu_dev?sslmode=disable` | required |
| `JWT_SECRET` | any non-empty secret | required; signs HS256 tokens |
| `CLOUDINARY_URL` | `cloudinary://key:secret@cloud_name` | required; PDF/ZIP uploads (`ResourceType: raw`) |
| `PORT` | `8080` | default `8080` |
| `FRONTEND_ORIGIN` | `http://localhost:5173` | CORS allow-origin (credentials) |
| `COOKIE_SECURE` | `true` | set `false` for local HTTP |

### 3. Frontend

```bash
cd frontend
cp .env.example .env                   # VITE_API_URL=http://localhost:8080
npm install
npm run dev                            # Vite  →  http://localhost:5173
```

### 4. First login

A bootstrap admin is seeded by migration:

- **username:** `admin`
- **password:** `123456`

You will be **forced to change the password on first login** (every account ships with `must_change_password = true`; changing it ends the session and you log in again). Admins then provision students, lecturers, courses and enrollments via CSV.

---

## Testing & the pre-push gate

**Definition of Done:** a change is not done until `bash scripts/check.sh` exits `0`. It mirrors the GitHub `ci` job locally so failures surface before you push.

`scripts/check.sh` runs:

1. `golangci-lint` (pinned **v1.64.8** — match it: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8`)
2. `go build ./...` and `go vet ./...`
3. **`migrate up` + `go test -count=1 -p 1 ./...` against a real Postgres** (see note below)
4. frontend `eslint` + `tsc` + `vite build`

### Running the Go tests (real Postgres, not mocks)

The integration tests run against a **real Postgres** and **fail loudly** (not skip) when `DATABASE_URL` is unset — a green run with no DB would be a false "passed". Use a dedicated test DB (the suite mutates data):

```bash
docker exec myiu-lite-postgres-1 psql -U myiu -d myiu_dev -c "CREATE DATABASE myiu_test;"
export DATABASE_URL="postgres://myiu:myiu@localhost:5432/myiu_test?sslmode=disable"
cd backend && go test -count=1 -p 1 ./...
```

Or drop those values into `scripts/check.env` (gitignored) and `scripts/check.sh` picks them up automatically.

> **`-p 1` matters:** the suite shares one Postgres and one test briefly renames the `notifications` table to prove same-transaction rollback. Serializing package binaries (`-p 1`) keeps that window from racing other packages' inserts. CI and `check.sh` both pass `-p 1` — keep them in sync.

### Pre-push hook

A committed hook runs the gate on every `git push` once you enable it (one-time per clone):

```bash
git config core.hooksPath .githooks
```

Emergency bypass only: `git push --no-verify`.

---

## Contributing (GitHub Flow)

`main` is the **only** long-lived branch — protected and always deployable. Never commit to `main` directly.

1. Cut a short-lived branch from latest `main`: `ft/<slug>` · `fix/<slug>` · `chore/<slug>` · `docs/<slug>` (backend + frontend together).
2. Make `bash scripts/check.sh` green.
3. Open a PR — the GitHub Actions `ci` job (unit + integration tests on real Postgres, migrations, lint/build) gates the merge.
4. Squash-merge into `main`, then delete the branch.

---

## Documentation

- [`ARCHITECTURE.md`](./ARCHITECTURE.md) — system design, data model (ERD), request lifecycle, cross-cutting subsystems, sequence diagrams.
- [`DECISION_LOGS.md`](./DECISION_LOGS.md) — every product/engineering decision (76 of them) with rationale, design principle, and trade-offs — the record of how the product evolved.
- [`.planning/`](./.planning) — PROJECT.md (vision), ROADMAP.md (phases), REQUIREMENTS.md, DESIGN-SYSTEM.md.
