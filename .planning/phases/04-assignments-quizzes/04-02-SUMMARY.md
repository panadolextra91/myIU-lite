# Phase 4 - Wave 2 Summary

## Completed Work

1. **Notifications Backend**
   - Implemented `notifications` table queries in `backend/db/queries/notifications.sql` covering inserting, listing, unread counting, and marking read.
   - Built `internal/notifications` module including dto, handler, model, repository, and service components.
   - Wired REST endpoints (`GET /notifications`, `GET /notifications/unread-count`, `POST /notifications/:nid/read`) to `main.go`.

2. **Assignment Grading & Transactionality**
   - Modified `submissions.sql` to include `UpsertSubmissionGrade` and joined `assignment_title` and `course_id` to submission read queries.
   - Built `GradeSubmission` in `assignments/service.go`. 
   - Enforced **same-transaction consistency** where writing a grade to `submissions` and emitting a notification into `notifications` happen atomically, adhering to domain invariant D-30/32.
   - Added REST endpoint `POST /courses/:id/assignments/:aid/submissions/:sid/grade` to expose grading.

3. **Frontend Notifications & Grading Interface**
   - Exposed `listNotifications`, `unreadCount`, `markRead`, and `gradeSubmission` in `courseworkApi`.
   - Built `NotificationBell.tsx` mounted in `AppLayout.tsx`, featuring a 30s background poll with unread badge updates.
   - Built the timeline UI in `Notifications.tsx` with bold states for unread items and automatic read marking upon interaction.
   - Upgraded `LecturerAssignments.tsx` to include an embedded "Grade Submission" action, allowing lecturers to input Submission IDs and Grades directly from the assignment grid.
   - Resolved all ESLint constraints effectively across the application.

## Next Steps

- Proceed to Phase 4 Wave 3, implementing Quizzes and Quiz Submissions functionality.
