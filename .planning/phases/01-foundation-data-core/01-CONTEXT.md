# Phase 1: Foundation & Data Core - Context

**Gathered:** 2026-06-19
**Status:** Ready for planning

<domain>
## Phase Boundary

A running monorepo skeleton: `backend/` (Go) + `frontend/` (React) exist, Postgres comes up via Docker, schema is migration-managed (foundational tables only), config loads from `.env`, and the GitHub Actions CI gate is **proven** to block a deliberately-failing PR.

Delivers INFRA-01 → INFRA-07. This phase is infrastructure + the relational foundation only — no auth logic (Phase 2), no admin provisioning (Phase 3), no feature tables (Phases 4–5).

</domain>

<decisions>
## Implementation Decisions

### Schema strategy
- **D-06:** Migrations are **incremental per-phase**, not big-bang. Phase 1 creates only the *very foundational* tables; each later phase adds its own migration so the DB grows alongside the codebase. Working assumption for Phase 1 foundational set: `users` + `audit_log` (planner/researcher to confirm exact minimal set — anything Phase 2 auth strictly needs may land here, the rest defers).
- **D-07:** Seed **one bootstrap admin** via migration so Phase 2 has a login and Phase 3 has an admin to provision others (avoids the chicken-and-egg problem). Credentials: username `admin`, password `123456`.
  - Security (non-negotiable, not a re-ask): seed the password **bcrypt-hashed** (cost=12), never plaintext, and set the `must_change_password` flag so Phase 2's server-enforced forced reset applies on first login.

### Docker scope
- **D-08:** `docker compose` runs **Postgres only** (`postgres:17-alpine`). Backend (Go) and frontend (Vite) run **natively** during dev (`go run` / `npm`), not in compose — keeps hot-reload fast and avoids extra Dockerfiles. Migrations apply via a separate command (not auto-run by compose).

### CI gate proof
- **D-09:** Prove the merge-block by opening a **throwaway PR that deliberately fails a test/build**, capturing the required-status-check blocking the merge as logged evidence, then closing the PR. This satisfies criteria #4 ("verified, not just configured").
  - Repo is already on GitHub and the user **has admin** rights. The plan must include a **step-by-step guide for the user to set up branch protection** (required status checks) — some of this is GitHub UI/admin action the user performs.

### Claude's Discretion
- **Frontend scaffold depth:** minimal stub — `frontend/` folder with a basic Vite + React 19 + TS init, just enough to "exist" and satisfy "clone-and-run". Full shadcn/Tailwind UI work starts in Phase 2 (no UI hint on Phase 1).
- **CI integration-test DB approach:** planner chooses between **testcontainers-go** vs GitHub Actions `services:` Postgres container. Criteria #6 wording ("real Postgres service container") is satisfied by either; pick the cleaner fit.
- **Exact foundational table set** beyond `users` + `audit_log`, and migration file layout — researcher/planner to finalize.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project-level (locked stack & constraints)
- `.claude/CLAUDE.md` — committed tech stack + versions (Go 1.24 / Gin v1.11 / Postgres 17 / React 19 / Vite 6 / TS 5), DB layer (sqlc v1.31.1 + pgx v5), migrations (golang-migrate v4.18), config (godotenv + caarlos0/env), CI/testing (testcontainers, golangci-lint, GitHub Actions), and the "What NOT to Use" + "Stack Patterns" guidance. **Authoritative for all library choices in this phase.**
- `.planning/PROJECT.md` — project vision, constraints, Key Decisions table (D-01…D-05).
- `.planning/REQUIREMENTS.md` — INFRA-01 → INFRA-07 acceptance wording.
- `.planning/ROADMAP.md` §"Phase 1: Foundation & Data Core" — goal + 4 success criteria.

### Design
- `.planning/DESIGN-SYSTEM.md` — global design system (not exercised this phase; frontend is a stub, but referenced so the Phase 1 Vite init aligns with later UI work).

No external ADRs beyond the above — requirements fully captured in the decisions above and the locked stack in CLAUDE.md.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- None — greenfield. No `backend/` or `frontend/` source exists yet; this phase scaffolds them.

### Established Patterns
- Stack patterns are pre-locked in `.claude/CLAUDE.md` (sqlc config `sql_package: "pgx/v5"`, `engine: "postgresql"`; migrate owns schema, sqlc reads it; `.env` via godotenv → `os.Getenv` / caarlos0/env struct).

### Integration Points
- This phase establishes the integration substrate every later phase hangs off: the migration chain (Phase 2+ append migrations), the `.env` config struct (JWT secret + Cloudinary creds slots exist now, consumed Phases 2/4), the `audit_log` table (written Phases 3–5), and the CI gate (every later phase's tests run through it).

</code_context>

<specifics>
## Specific Ideas

- Bootstrap admin is concrete: username `admin`, password `123456` (hashed + forced-change). This is a dev/bootstrap credential, intended to be rotated on first login once Phase 2 ships forced reset.
- "One Docker command" is interpreted as **Postgres-only** compose (D-08), not full-stack — the success-criteria phrasing was deliberately scoped down by the user.

</specifics>

<deferred>
## Deferred Ideas

- **Full-stack docker-compose** (backend + frontend containerized) — considered, rejected for Phase 1 (D-08). Could revisit if a production deployment story needs it later.
- **Feature tables** (assignments, quizzes, grades, notifications, requests, courses, enrollments) — deliberately NOT created in Phase 1 per incremental strategy (D-06); each lands in its owning phase.
- **Real admin provisioning / CSV import** — Phase 3 (ADMIN-01…08). The Phase 1 seed admin is only the bootstrap.

None of these are scope creep into Phase 1 — discussion stayed within the foundation boundary.

</deferred>

---

*Phase: 1-Foundation & Data Core*
*Context gathered: 2026-06-19*
