# Phase 5 Wave 3 Summary: Grade Publication & Student View

## What was Accomplished

### 1. Publication Snapshot & Same-Tx Notify (Backend)
- Added `UpsertGradePublication` and `ListPublicationsForStudent` to `grades.sql` and generated code with `sqlc`.
- Implemented `PublishComponent` in `backend/internal/grades/service.go`. This loops through enrolled students, calculates the "live" value for the targeted component, and atomically inserts into `grade_publications` and `notifications` (type `GRADE_PUBLISHED`) inside a single transaction. This guarantees the snapshot and notifications are never decoupled.
- Implemented `GetStudentGrades` which strictly reads from `ListPublicationsForStudent` (frozen snapshot), computing the overall grade only if all top-level components are fully published.

### 2. Student Grades Page & Lecturer Publish (Frontend)
- **Lecturer Gradebook**: Added a "Publish" button for top-level components in `Gradebook.tsx`. This triggers the publish API and refreshes the live view.
- **Student Grades Page**: Created `Grades.tsx` to display published grades. Unpublished overall grades are gracefully shown as "Pending". Removed unused imports and fixed shadcn components (`Card` instead of `Alert`).
- **Routing**: Added `/student/courses/:id/grades` to `router.tsx` protected by the student `RoleGuard`.

### 3. Anti-Theater Integration Test
- Added `TestPublishComponent` to `backend/internal/grades/compute_test.go`.
- Validated:
  - Lecturer updating live score does NOT leak to student view until published.
  - Snapshot explicitly freezes the score.
  - Notifications fan out alongside publication correctly.
  - `check.sh` passes completely across lint, build, and tests for both frontend and backend.
