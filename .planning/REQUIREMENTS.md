# Requirements: myIU (Lite Edition)

**Defined:** 2026-06-19
**Core Value:** Students and lecturers can run a course end-to-end (assignments, quizzes, grades, announcements, requests) without email — and Admin can provision everything from CSV.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Infrastructure (INFRA)

- [ ] **INFRA-01**: Monorepo with `backend/` (Go) and `frontend/` (React) folders
- [ ] **INFRA-02**: PostgreSQL runs via Docker (compose), never natively
- [ ] **INFRA-03**: Backend config loaded from `.env` (DB, JWT secret, Cloudinary creds)
- [ ] **INFRA-04**: DB schema managed by versioned migrations
- [ ] **INFRA-05**: GitHub Actions CI triggers on push to `main`, `backend`, `frontend`
- [ ] **INFRA-06**: CI runs unit + integration tests against a real Postgres service/container
- [ ] **INFRA-07**: Merge to a protected branch is blocked unless tests, DB checks, and syntax/build pass (verified to actually block)

### Authentication & Accounts (AUTH)

- [ ] **AUTH-01**: User can log in with username + password and receive a session/JWT carrying their role
- [ ] **AUTH-02**: User can log out
- [ ] **AUTH-03**: Logged-in user can change their own password
- [ ] **AUTH-04**: A user flagged `must_change_password` is restricted to the change-password action until they reset it (enforced server-side, not just UI)
- [ ] **AUTH-05**: Routes are authorized by role (Student / Lecturer / Admin) and by ownership where applicable

### Admin — Provisioning & Courses (ADMIN)

- [ ] **ADMIN-01**: Admin can create a student or lecturer account manually
- [ ] **ADMIN-02**: Admin can bulk-create student/lecturer accounts from a CSV file (whole-file validation before insert; duplicate/invalid IDs reported)
- [ ] **ADMIN-03**: New accounts default to username = student/lecturer ID, password = birthday `DDMMYYYY`, with `must_change_password` set
- [ ] **ADMIN-04**: Admin can reset any user's password back to the default `DDMMYYYY`, re-setting the forced-change flag (D-01)
- [ ] **ADMIN-05**: Admin can create, read, update, and delete courses with a start date and end date
- [ ] **ADMIN-06**: Admin can assign students and lecturers to a course from a CSV list
- [ ] **ADMIN-07**: System auto soft-deletes courses 1 month after their end date passes, without manual action
- [ ] **ADMIN-08**: Every admin mutation (account create, password reset, course CRUD, enrollment) writes an append-only audit log entry (actor, action, target, timestamp)

### Assignments & Submissions (ASMT)

- [ ] **ASMT-01**: Lecturer can create an assignment for a course with a deadline (date + time)
- [ ] **ASMT-02**: When creating an assignment, lecturer sets accept-late = yes/no; if yes, sets a late threshold of X days or "no threshold" (accept until the course is soft-deleted); if no, no threshold is collected (D-02)
- [ ] **ASMT-03**: Student can submit an assignment by uploading a single PDF or ZIP file, max 10MB, validated server-side by magic bytes (not just extension/MIME)
- [ ] **ASMT-04**: System enforces the assignment's late policy using the server timestamp (block after deadline, or accept-and-flag-late within threshold)
- [ ] **ASMT-05**: Uploaded files are stored on Cloudinary as non-public (authenticated) assets; downloads go through backend-generated short-lived signed URLs gated by role/ownership
- [ ] **ASMT-06**: Lecturer can view and grade a student's submission; saving the grade auto-notifies the student

### Quizzes (QUIZ)

- [ ] **QUIZ-01**: Lecturer can create a quiz for a course, configuring: max number of questions, max grade, shuffle yes/no, and retake count (0 = single attempt, N = N retakes) (D-03)
- [ ] **QUIZ-02**: Lecturer can supply quiz questions/answers either by uploading a CSV, or by entering them directly in the UI
- [ ] **QUIZ-03**: Student can take a multiple-choice quiz; the take-quiz API never exposes which option is correct
- [ ] **QUIZ-04**: When shuffle is on, options are presented in randomized order while preserving the correct-answer mapping by stable option ID
- [ ] **QUIZ-05**: System auto-grades the quiz on submission against the configured max grade and records the student's score (idempotent per attempt)
- [ ] **QUIZ-06**: System enforces the configured retake limit; retakes are tracked as attempts distinct from the original submission

### Gradebook (GRADE)

- [ ] **GRADE-01**: Lecturer configures a course's grade scheme as three weighted columns — Inclass, Midterm, Final — whose percentages sum to 100% of the overall grade (D-04)
- [ ] **GRADE-02**: Lecturer configures Inclass sub-components and their weights (e.g. project / quizzes / laboratory / bonus) summing to 100% of Inclass
- [ ] **GRADE-03**: Lecturer enters/uploads Midterm and Final grades manually (offline exams)
- [ ] **GRADE-04**: System computes each student's overall course grade from the configured weights
- [ ] **GRADE-05**: Student can view their grades for a course; grade availability is auto-notified

### Announcements (ANNC)

- [ ] **ANNC-01**: Lecturer can send an announcement to all students enrolled in their course
- [ ] **ANNC-02**: Lecturer can send an announcement to one or more specific enrolled students
- [ ] **ANNC-03**: Student receives announcements for their enrolled courses (persisted, visible on next login — no email)

### Requests (REQ)

- [ ] **REQ-01**: Student can send a request to the course's lecturer of type leave-early, absence, or custom (title + plain-text body)
- [ ] **REQ-02**: Lecturer can reply yes or no to a student request
- [ ] **REQ-03**: The lecturer's reply is auto-delivered to the student

### Notifications (NOTIF)

- [ ] **NOTIF-01**: A single persisted notification primitive backs grade delivery, request replies, and announcements (one row per recipient, with a read marker)
- [ ] **NOTIF-02**: Notifications that accompany a mutation (grade saved, reply sent) are written in the same transaction as the mutation

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Auth

- **AUTH-V2-01**: Self-service password recovery (security questions or email channel) — currently admin-assisted only

### Quizzes

- **QUIZ-V2-01**: Non-multiple-choice question types (essay, short answer)

### Gradebook

- **GRADE-V2-01**: Grade export (CSV/PDF transcripts)

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Email / SMS notifications | Replaced by in-app persisted notifications — a core product goal |
| Self-coded UI components | Using shadcn/ui; user prefers ready-made components |
| Files other than PDF/ZIP, or >10MB | Keeps upload validation and storage simple for the lite edition |
| Server-side ZIP extraction | Zip-bomb risk; submissions are stored, never unpacked |
| Essay / free-text auto-grading | Only MCQ auto-grading in MVP |
| Real-time chat / websockets | Pull-based persisted delivery is sufficient |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| (populated during roadmap creation) | — | Pending |

**Coverage:**
- v1 requirements: 41 total
- Mapped to phases: 0 (pending roadmap)
- Unmapped: 41 ⚠️

---
*Requirements defined: 2026-06-19*
*Last updated: 2026-06-19 after initial definition*
