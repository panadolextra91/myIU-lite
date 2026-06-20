# Phase 3 Wave 3 Summary: Course Lifecycle (Plan 03-03)

## Overview
Successfully implemented the Course Lifecycle workflow for the admin, fulfilling the requirements specified in **ADMIN-05 and ADMIN-07**. The system now allows admins to fully CRUD courses, ensuring strong active-course disciplines via soft deletes and an automated background sweep.

## Work Completed

### Backend (`internal/courses` and `internal/lifecycle`)
- **Courses CRUD (`handler.go`, `service.go`, `repository.go`):** 
  - Standardized REST APIs for Create, Read, Update, and Soft-Delete courses. 
  - Every API endpoint is guarded via `RequireRole(admin)`. 
  - Every mutation produces exactly one `COURSE_CREATE`, `COURSE_UPDATE`, or `COURSE_DELETE` audit log.
  - Roster APIs (`GET /:id/students` and `GET /:id/lecturers`) implemented with joins filtering out soft-deleted users `u.deleted_at IS NULL`.
- **Soft-Delete Discipline (`courses.sql`):** 
  - All read and list queries mandate `deleted_at IS NULL`. 
  - The "Delete" operation performs an `UPDATE courses SET deleted_at = now()`, fully preventing physical deletion (D-29).
- **Automated Lifecycle Sweep (`sweep.go`):** 
  - Idempotent `StartSweeper` runs an immediate catch-up sweep at system boot and launches a daily background `time.Ticker` process (D-37). 
  - Any course past 1 month of its `end_date` is marked as soft-deleted. 
  - The sweep writes precisely one `COURSE_SWEEP` audit log under the `SYSTEM` actor, **only if** one or more rows were affected (D-39). No-op days skip logging to keep audit logs clean.
  - Successfully verified via integrated testing (`sweep_test.go`).

### Frontend (`src/pages/admin/`)
- **Courses Listing (`Courses.tsx`):** Built out the TanStack Table displaying active courses (Pagination + Search + Term Filtering). Handled Create/Edit with `Dialog` logic and native `<input type="date">` inputs enforcing `end_date >= start_date`. Soft-delete flows through an `AlertDialog` confirming action.
- **Roster Overview (`CourseDetail.tsx`):** Implemented a detailed Course read-only view utilizing `shadcn` Tabs (Overview, Students, Lecturers) mapping to the roster APIs to deliver a clean dashboard of active class sizes (D-42).

## Verification
- **Codebase Tests:** Backend logic (`go test ./internal/courses ./internal/lifecycle -count=1`) passes. Automated tests specifically prove the sweeper logic works idempotently and prevents extraneous audit logs. Frontend checks (`npm run lint`, `tsc --noEmit`, `vite build`) completed cleanly.
- **Threat Mitigation:** `deleted_at IS NULL` rules successfully block zombie resurrection. Background sweep operates flawlessly independent of user contexts by resolving the fixed `SYSTEM` actor ID.
