# Phase 5 Wave 5 Summary: Student-Lecturer Requests

**Goal**: Deliver student↔lecturer requests as a complete vertical slice, enabling directed typed requests and single-round-trip replies with atomic notifications.

## What was implemented

### 1. Database & Queries
- Created `requests.sql` containing queries: `InsertRequest`, `ReplyRequest`, `ListLecturerRequests`, `ListStudentRequests`, and `GetRequestByID`.
- Configured the `ReplyRequest` UPDATE with `status = 'PENDING'` and `targeted_lecturer_id = $2` to enforce a single reply from the correct lecturer (preventing reopens and IDOR).

### 2. Backend Domain (`internal/requests`)
- **Service**: 
  - `CreateRequest`: Inserts a request targeting a specific lecturer and writes a `REQUEST_CREATED` notification to the targeted lecturer in the exact same transaction.
  - `ReplyRequest`: Updates the request status (APPROVED/DENIED) with an optional note, and writes a `REQUEST_REPLIED` notification back to the student in the same transaction. Enforces `targeted_lecturer_id`.
- **Handler & Routes**:
  - `POST /api/student/courses/:id/requests`
  - `GET /api/student/requests`
  - `GET /api/student/courses/:id/lecturers`
  - `GET /api/lecturer/requests`
  - `POST /api/lecturer/requests/:id/reply`
- Registered the feature routes into `main.go`.

### 3. Frontend
- **Student Requests (`student/Requests.tsx`)**:
  - Built a composition form using `react-hook-form` to submit `LEAVE_EARLY`, `ABSENCE`, or `CUSTOM` requests.
  - Dynamically fetches course lecturers so the student can select exactly who should receive the request.
  - Read-only list showing the student's past requests, status, and any lecturer reply notes.
- **Lecturer Inbox (`lecturer/RequestInbox.tsx`)**:
  - Inbox UI displaying requests specifically targeted at the current lecturer.
  - A Dialog to reply with a required `APPROVED` or `DENIED` decision and an optional note. Once replied, the request is permanently closed.
- Wired all navigation routes into `router.tsx`.

### 4. Integration Test
- Created an **Anti-theater** integration test `request_test.go` verifying:
  - Directed routing: Requests are visible only to the targeted lecturer. Other lecturers on the same course cannot read or reply to it (`ErrNotTargeted`).
  - Atomicity: Notifications for creation (to lecturer) and reply (to student) are inserted correctly. If an error occurs, the entire action rolls back.
  - Single Round-Trip: Sending a second reply to a closed request returns `ErrAlreadyClosed`.

## Current Status
- **Phase 5 Wave 5** is **COMPLETE**.
- The `requests` vertical slice builds without errors, and all tests pass against a real PostgreSQL 17 test database.
