# Phase 5 Wave 4 Summary: Announcements

**Goal**: Deliver the Announcements vertical slice, enabling lecturers to broadcast immutable messages to all or specific students with atomic notification fan-out.

## What was implemented

### 1. Database & Queries
- Created `announcements` and `announcement_recipients` tables via goose migrations.
- Implemented SQLC queries (`announcements.sql`) for creating announcements, inserting specific recipients, listing course announcements for lecturers, and scoped listing for students.

### 2. Backend Domain (`internal/announcements`)
- **Service**: Implemented `CreateAnnouncement` which handles the same-transaction fan-out logic. It creates the announcement, inserts any `announcement_recipients`, and simultaneously loops through target students to insert `notifications` (A6). All wrapped in `pool.Begin()` -> `tx.Commit()` to guarantee atomicity.
- **Repository**: Pass-through methods for SQLC queries.
- **Handler**: REST endpoints:
  - `POST /api/lecturer/courses/:id/announcements`
  - `GET /api/lecturer/courses/:id/announcements`
  - `GET /api/student/courses/:id/announcements`
  - `GET /api/student/announcements/:id`
- Wired to `main.go`.
- Exposed `/api/lecturer/courses/:id/students` in `internal/courses/handler.go` so lecturers can fetch the enrolled students roster.

### 3. Frontend
- **Lecturer Announcements Page**: Integrated `react-hook-form` to compose announcements. Dynamically fetches course students for the `SPECIFIC_STUDENTS` multi-select checkbox list. Shows history of sent announcements.
- **Student Announcements Page**: Read-only view for students to browse announcements. Includes auto-scrolling to a specific announcement when navigated from the notification bell (`/courses/:id/announcements/:announcementId`).
- **Routing**: Added the corresponding routes in `router.tsx`.

### 4. Anti-Theater Integration Test
- Wrote `fanout_test.go` checking:
  1. `ALL_STUDENTS` correctly inserts notifications for all enrolled students.
  2. `SPECIFIC_STUDENTS` correctly inserts notifications *only* for selected students and ignores unenrolled or unselected ones.
  3. Atomicity guarantee (if fan-out fails, the announcement creation rolls back).
- Tests run against real Postgres 17 via `check.sh`.

## Current Status
- **Phase 5 Wave 4** is **COMPLETE**.
- The `announcements` feature is fully working, atomic, and immutable.
