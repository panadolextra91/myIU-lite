---
phase: 05-gradebook-announcements-requests
plan: 01
subsystem: database, api, ui
tags: [postgres, pgx, go, react, zod]

# Dependency graph
requires:
  - phase: 04-assignments-quizzes
    provides: [assignments system, authz helpers]
provides:
  - Phase 5 Gradebook base tables (grade_schemes, grade_components, grade_scores, grade_publications, announcements, announcement_recipients, requests)
  - max_score and grading_finalized_at columns for assignments
  - Finalize Grading flow for assignments
affects: [05-gradebook-announcements-requests]

# Tech tracking
tech-stack:
  added: []
  patterns: [gradebook component model, idempotent finalize, anti-theater tests]

key-files:
  created: 
    - backend/db/migrations/000008_grades_announcements_requests.up.sql
    - backend/internal/assignments/finalize_test.go
  modified: 
    - backend/internal/assignments/service.go
    - backend/internal/assignments/handler.go
    - frontend/src/pages/lecturer/Assignments.tsx

key-decisions:
  - "Implemented Gradebook Component Model tables exactly per D-56, with separation of grade_scores and grade_publications (D-66)."
  - "Added max_score to assignment creation, defaulting to 100 on FE."
  - "FinalizeGrading uses idempotent UPDATE ... WHERE grading_finalized_at IS NULL to avoid overriding timestamp."

patterns-established:
  - "Anti-theater tests: FinalizeGrading test proves RED when actual DB UPDATE is reverted."

requirements-completed: ["REQ-5"]

# Metrics
duration: 15m
completed: 2026-06-20
status: complete
---

# Phase 05 Wave 1: DB Base & Assignments Touch Summary

**Created Phase 5 base tables, added max_score and finalize grading flow for assignments.**

## Performance

- **Duration:** 15m
- **Started:** 2026-06-20T16:48:00Z
- **Completed:** 2026-06-20T17:03:00Z
- **Tasks:** 3
- **Files modified:** 13

## Accomplishments
- Implemented migration 000008 with all Phase 5 tables (Gradebook, Announcements, Requests).
- Added `max_score` and `grading_finalized_at` to Assignments.
- Updated Assignments service, handler, and UI to support creating assignments with max_score and finalizing grading.
- Wrote anti-theater integration tests for Finalize Grading.

## Task Commits

Each task was committed atomically:

1. **Phase 5 Wave 1 complete** - `0319ebb` (feat: finalize assignments and add max_score)

## Files Created/Modified
- `backend/db/migrations/000008_grades_announcements_requests.up.sql` - Phase 5 tables
- `backend/internal/assignments/service.go` - Added FinalizeGrading method
- `backend/internal/assignments/handler.go` - Added FinalizeGrading route
- `backend/internal/assignments/finalize_test.go` - Integration tests
- `frontend/src/pages/lecturer/Assignments.tsx` - Updated create assignment form and assignment table

## Decisions Made
- Implemented `max_score` during assignment creation and allowed floating point inputs.
- Handled `finalize_grading` idempotency using `WHERE grading_finalized_at IS NULL` at the SQL level.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- `security_test.go` broke due to the new NOT NULL constraint on `max_score` for assignments. Fixed by injecting a `max_score` payload in the test fixture.

## Next Phase Readiness
- Database foundation for Gradebook, Announcements, and Requests is ready.
- Assignments are fully capable of supplying components to the gradebook.

---
*Phase: 05-gradebook-announcements-requests*
*Completed: 2026-06-20*
