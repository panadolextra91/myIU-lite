# Phase 1: Foundation & Data Core - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-19
**Phase:** 1-Foundation & Data Core
**Areas discussed:** Schema scope, Docker bring-up, CI proof

---

## Schema scope

| Option | Description | Selected |
|--------|-------------|----------|
| Incremental per-phase | Phase 1 creates only minimal foundational tables; each later phase adds its own migration. Lean, less rework. | ✓ |
| Foundation tables only | Create the multi-phase "core" tables (users, courses, enrollments, audit_log) now; feature-specific tables later. | |
| Big-bang full schema | Create ALL tables now in one migration batch. | |

**User's choice:** Incremental per-phase (D-06) — "we only create the very foundational tables for the DB. After each phase, we will add migration, the DB will grow alongside with the codebase."
**Notes:** Working assumption for the Phase 1 foundational set is `users` + `audit_log`; planner/researcher confirm the exact minimal set.

### Bootstrap admin (sub-question)

| Option | Description | Selected |
|--------|-------------|----------|
| Seed 1 admin | Migration seeds one default admin so Phase 2 login + Phase 3 provisioning have a starting account. | ✓ |
| Defer to Phase 3 | No seed; all accounts created in Phase 3; Phase 2 uses test fixtures. | |

**User's choice:** Seed 1 admin (D-07) — username `admin`, password `123456`.
**Notes:** Claude added the non-negotiable security note (not re-asked): bcrypt-hash the seeded password (cost=12), set `must_change_password` so Phase 2 forced reset applies.

---

## Docker bring-up

| Option | Description | Selected |
|--------|-------------|----------|
| Postgres only | Compose runs postgres:17-alpine; backend + frontend run native (fast hot-reload). | ✓ |
| Postgres + backend | Compose runs DB + Go backend (auto-migrate); frontend native. | |
| Full stack | One command brings up DB + backend + frontend. | |

**User's choice:** Postgres only (D-08) — "Docker only run DB, frontend and backend will run via npm."
**Notes:** "One Docker command" in success criteria #1 deliberately scoped down to Postgres-only.

---

## CI proof

| Option | Description | Selected |
|--------|-------------|----------|
| Throwaway failing PR | Open a PR that deliberately fails; capture required-status-check blocking merge as evidence; close PR. | ✓ |
| Local-only | Show CI red in logs only, without testing real branch protection. | |

**User's choice:** Throwaway failing PR (D-09) — "Throwaway failing PR with proper log for evidence. I need guidance to setup branch protection on GitHub."

| GitHub access | Description | Selected |
|--------|-------------|----------|
| Have admin | Repo on GitHub, user has admin to set branch protection. | ✓ |
| Not yet / unsure | Repo not pushed / unclear permissions. | |
| Claude handles config | Agent handles GitHub config, guides manual clicks. | |

**User's choice:** Have admin — repo is on GitHub, user has admin rights.
**Notes:** Plan must include a step-by-step branch-protection setup guide the user performs.

---

## Claude's Discretion

- Frontend scaffold depth — minimal Vite + React 19 + TS stub for Phase 1; real UI starts Phase 2.
- CI integration-test DB — planner picks testcontainers-go vs GitHub Actions `services:` Postgres container.
- Exact foundational table set beyond `users` + `audit_log`, and migration file layout.

## Deferred Ideas

- Full-stack docker-compose (containerized backend + frontend) — rejected for Phase 1; revisit if a deploy story needs it.
- Feature tables (assignments, quizzes, grades, notifications, requests, courses, enrollments) — deferred to owning phases per incremental strategy.
- Real admin provisioning / CSV import — Phase 3.
