# myIU (Lite Edition)

## What This Is

A lite student management platform for a university, called **myIU**. It gives three actors — Students, Lecturers, and Admins — a single place to handle coursework and course administration: assignment submission, auto-graded quizzes, announcements, grades, and student↔lecturer requests (so email is no longer needed). Admin manages accounts, course enrollment, and courses.

## Core Value

Students and lecturers can run a course end-to-end (assignments, quizzes, grades, announcements, requests) without falling back to email — and Admin can provision everything from CSV.

## Requirements

### Validated

(None yet — ship to validate)

### Active

**Student**
- [ ] Change password
- [ ] Forgot/reset password
- [ ] Submit assignment via file upload — PDF and ZIP only, max 10MB
- [ ] Take multiple-choice quizzes; system auto-grades on submit and records the grade
- [ ] Receive announcements from the course's lecturer
- [ ] Receive grades from the course's lecturer
- [ ] Send requests to the lecturer: leave-early, absence, and custom (title + plain-text body)

**Lecturer**
- [ ] Change password
- [ ] Forgot/reset password
- [ ] Grade assignments; grade is sent to the student automatically when done
- [ ] Create quizzes with auto-shuffle and max-questions-per-quiz settings
- [ ] Send announcements to all students, or to specific student(s) enrolled in their course
- [ ] Reply yes/no to a student request; reply is sent to the student automatically

**Admin**
- [ ] Create student/lecturer accounts manually or from a CSV list
- [ ] Default credentials: username = student ID / lecturer ID; password = birthday in `DDMMYYYY`; force password change on first login
- [ ] Assign students and lecturers to a course from a CSV list
- [ ] CRUD courses with start date and end date
- [ ] Auto soft-delete (sweep) courses 1 month after their end date passes
- [ ] Support password changes for lecturers/students
- [ ] Audit log recording all admin actions

### Out of Scope

- Self-coded UI components — using shadcn/ui instead (user is not confident with visual design)
- Email as a communication channel — replaced by in-app announcements and requests
- File types beyond PDF/ZIP for submissions, and files over 10MB — keeps storage/validation simple for the lite edition
- Non-multiple-choice quiz types (essay, etc.) — only MCQ auto-grading in MVP

## Context

- Greenfield build. Repo currently holds only README + GitNexus/agent context files.
- This project is also a testbed for combining GSD (planning), Ponytail (minimal code), and GitNexus (impact analysis) — keep implementations lean.

## Constraints

- **Tech stack (backend)**: Go + Gin, PostgreSQL — ORM optional (raw SQL acceptable if no good PostgreSQL+Go ORM fits). Env via `.env`.
- **Tech stack (frontend)**: React + Zustand (state) + shadcn/ui (components). No hand-rolled components.
- **Storage**: Cloudinary for uploaded files, configured via environment variables.
- **Submissions**: PDF and ZIP only, hard 10MB limit.
- **Security/compliance**: forced first-login password change; audit log for all admin actions.
- **Database runtime**: PostgreSQL runs via Docker only — never natively.
- **Repo structure**: two top-level folders — `backend/` (Go source) and `frontend/` (React source). No per-folder README required.
- **Branching**: exactly three branches — `main`, `backend`, `frontend`.
- **CI/CD**: GitHub Actions workflow auto-triggers on push to `backend`, `frontend`, and `main`. Merge is blocked unless tests pass and DB + syntax checks succeed.
- **Testing**: every phase must pass unit tests and integration tests.

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go + Gin + PostgreSQL backend | User's chosen stack; Gin is a lightweight, common Go web framework | — Pending |
| ORM optional, raw SQL allowed | Go ORM ecosystem is thin; avoid forcing a poor fit | — Pending |
| React + Zustand + shadcn/ui frontend | User wants ready-made, good-looking components | — Pending |
| Cloudinary for file storage | Offload upload/storage; configured via `.env` | — Pending |
| Default password = birthday `DDMMYYYY`, forced change | Simple admin provisioning from CSV, secured by first-login reset | — Pending |
| Auto soft-delete courses 1 month after end date | Removes manual cleanup for admin; soft delete keeps history | — Pending |
| Audit log for admin actions | Admin can change others' passwords — needs accountability | — Pending |
| PostgreSQL via Docker only | Reproducible DB env, no native install drift | — Pending |
| Monorepo: `backend/` + `frontend/` folders, 3 branches (main/backend/frontend) | Clear separation, simple branch-per-stack CI triggers | — Pending |
| GitHub Actions CI gate on tests + DB + syntax | No broken code merges to protected branches | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-06-19 after initialization*
