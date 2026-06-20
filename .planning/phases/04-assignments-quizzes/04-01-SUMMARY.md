# Phase 4 - Wave 1 Summary

## Completed Work

1. **Cloudinary Integration**
   - Implemented `cloudinary.Client` wrapper (`backend/internal/shared/cloudinary/client.go`).
   - Verified end-to-end authenticated upload and signed-download logic with `spike_test.go`. Assumption A1 is officially retired.
   - Initialized the client in `main.go` using `CLOUDINARY_URL`.

2. **Database Migrations**
   - Created migration `000006_assignments_quizzes_notifications` delivering all 8 phase tables simultaneously (D-06).
   - Applied identical identity, constraints, and audit conventions as Admin Schema 000004.
   - Added SQLC queries for assignments and submissions (`CreateAssignment`, `GetAssignmentByID`, `InsertSubmissionVersion`, `GetMaxSubmissionVersion`, `GetActiveSubmission`, `GetSubmissionByID`).

3. **Assignment Backend Module**
   - Implemented `internal/assignments` (dto, model, repository, service, handler).
   - Set up `Submit` endpoint with strict magic-byte sniffing (`gabriel-vasile/mimetype`) restricting to `.pdf` and `.zip` and blocking >10MB payloads.
   - Enforced server-side `now()` computation for `is_late` policy evaluation against deadline bounds.
   - Wired `DownloadURL` flow that only generates short-lived authenticated Cloudinary URLs upon passing ownership checks (course lecturer or submission owner).

4. **Frontend Coursework Interfaces**
   - Exposed a new `courseworkApi` client mimicking `admin-api.ts`.
   - Created `LecturerAssignments` view enabling the creation of assignments with flexible late policies.
   - Created `StudentAssignments` view facilitating file uploads and late-duration status indicators.
   - Wired views to the `/lecturer/assignments` and `/student/assignments` route namespaces.

## Next Steps

- Proceed to Phase 4 Wave 2 (`04-02-PLAN.md`) which covers manual grading and automatic grade syncing to enrollments.
