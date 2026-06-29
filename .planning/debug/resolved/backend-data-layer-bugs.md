---
status: resolved
human_verify: confirmed — LEC001 on CS101: Assignments shows "Homework 1", Quizzes shows "Quiz 1" (Playwright, 2026-06-29). Fix code-complete + go test green.
trigger: "Pre-existing frontend<->backend data-layer integration bugs in myIU. Assignments/quizzes screens fetch but render empty despite seeded data; suspected systemic raw-sqlc-serialization across handlers."
created: 2026-06-29
updated: 2026-06-29
---

# Debug: backend-data-layer-bugs

Pre-existing (NOT redesign-related) latent integration bugs surfaced during end-to-end Playwright testing of the Dark Academia UI redesign. Three were already fixed and committed on `feat/UI-enhance` (Select label, coursework-api missing `/api` prefix, requests `ListCourseLecturers` serialization). These remain.

## Symptoms

- **expected:** `GET /api/lecturer/courses/1/assignments` returns the assignment(s) for course 1 (the lecturer is assigned to the course; a row exists).
- **actual:** returns HTTP 200 with `{"data": []}` (empty) even though an assignment row exists. The frontend correctly shows the "No assignments found" empty state. Same likely affects quizzes.
- **error:** no error/exception. The handler `ListAssignments` maps to `AssignmentResponse` and returns `{data: res}` — `res` is just empty, so the empty list comes from the service/query layer (`assignments` service `ListCourseAssignments`), most likely a filter on `user_id`/`role` (lecturer vs creator vs course-membership).
- **secondary (systemic):** audit handlers that return RAW sqlc rows without json tags → JSON keys come out PascalCase (`LecturerID`, `FullName`) and `pgtype.Text`/`pgtype.Timestamptz` serialize as objects, breaking the frontend. Already found+fixed one in `requests.ListCourseLecturers`. Check the **assignments / quizzes / grades** handlers for the same pattern (each list/get endpoint: does it map to a DTO, or `c.JSON(... rawRow)`?).
- **timeline:** latent — these features were never exercised end-to-end with real data, so the bugs predate the redesign. Backend integration tests pass (they test Go directly, not the frontend contract).
- **reproduction:** see below.

## Reproduction / stack

- Postgres: Docker container `myiu-lite-postgres-1`, DB `myiu_dev`, port 5432 (already up; if down: `docker compose up -d --wait --force-recreate postgres`).
- Backend: `cd backend && go run ./cmd/api` → `:8080` (loads `backend/.env`; DATABASE_URL → myiu_dev). **Currently stopped — restart to repro.**
- Frontend: `npm --prefix frontend run dev` → `:5173`.
- Creds: admin/`123456`, **LEC001**/`01011985`, **STU001**/`student123`.
- Seeded in `myiu_dev`: course **CS101** (id 1), STU001 (user id 12) enrolled, LEC001 (user id 11) assigned, + 1 assignment (course_id 1, created_by 11), 1 quiz, grade scheme (3 components), 1 pending request.
- Fast repro without the browser: hit the API directly with a cookie, OR (cleaner) run the `ListCourseAssignments` query/service path against `myiu_dev` and compare to a raw `SELECT * FROM assignments WHERE course_id=1`.
- Tooling available: GitNexus MCP (impact/context), Playwright MCP (browser). Per CLAUDE.md, run `impact()` before editing any symbol.

## Current Focus

hypothesis: BOTH goals resolved and verified end-to-end. Awaiting user confirmation in the real frontend workflow.
test: live HTTP repro as LEC001 + go build/vet/test against Postgres (all green); seed fixture intact.
expecting: user confirms assignments + quizzes render in the redesigned UI, and create-quiz works.
next_action: await "confirmed fixed" from user, then archive session + append knowledge base.
reasoning_checkpoint:
  hypothesis: "GET /api/lecturer/courses/1/assignments returns [] because course 1 was soft-deleted by the gocron sweep (end_date 2026-05-10 < now()-1month=2026-05-29), and listCourseAssignments INNER JOINs courses with c.deleted_at IS NULL, filtering out the assignment of a deleted course."
  confirming_evidence:
    - "DB: courses.id=1 has deleted_at=2026-06-29 04:20:23 (NOT NULL) — soft-deleted today by the sweep."
    - "DB: course 1 end_date=2026-05-10; sweep cutoff now()-1month=2026-05-29; 2026-05-10 < 2026-05-29 → sweep correctly archived it."
    - "DB: assignment id=2 exists with course_id=1, created_by=11 (row IS present)."
    - "sqlc listCourseAssignments: 'WHERE a.course_id=$1 AND c.deleted_at IS NULL' — no user_id/role filter; the deleted_at on the JOINed course removes the row."
    - "Symptom is HTTP 200 + {\"data\":[]} not 403, so authz (IsLecturerAssigned) passed; the empty result comes purely from the deleted_at read filter."
  falsification_test: "If I clear courses.deleted_at for id=1 and re-run the query, the assignment row should appear. If it still returns empty, the deleted_at filter is not the cause."
  fix_rationale: "Code is correct (soft-delete read filter is required by D-06/CLAUDE.md: 'All reads filter WHERE deleted_at IS NULL'). Root cause is stale fixture data: the manually-seeded test course was given a past Spring-2026 end_date, guaranteeing the sweep archives it. Durable fix = restore the fixture to an active course (clear deleted_at, set a current/future end_date) so UAT can proceed and the sweep won't re-archive it."
  blind_spots: "No committed seed file exists, so the data fix is not durable across a DB rebuild. The CreateAssignment asymmetry (allows creating on a soft-deleted course) is a separate latent bug, not the cause of the empty list."

## Evidence

- timestamp: 2026-06-29
  checked: sqlc query `listCourseAssignments` in backend/internal/shared/db/assignments.sql.go (lines 121-160)
  found: "WHERE a.course_id = $1 AND c.deleted_at IS NULL — only filters by course_id and the JOINed course's deleted_at. No user_id/role/created_by filter."
  implication: An empty result for course 1 is NOT caused by a user/role mismatch; it can only come from (a) no row with course_id=1, or (b) the JOINed course being soft-deleted (deleted_at NOT NULL).

- timestamp: 2026-06-29
  checked: "Direct DB query against myiu_dev — courses id=1, assignments course_id=1, course_lecturers course_id=1, users 11/12."
  found: "course 1 (CS101) deleted_at = 2026-06-29 04:20:23 (soft-deleted). assignment id=2 (course_id=1, created_by=11) EXISTS. course_lecturers: LEC001 (id 11) assigned to course 1. users 11=LEC001 lecturer, 12=STU001 student."
  implication: "The assignment row exists and the lecturer is assigned, but the course is soft-deleted — so the deleted_at read filter empties the result. Confirms hypothesis branch (b)."

- timestamp: 2026-06-29
  checked: "course 1 dates vs sweep cutoff (now() - interval '1 month')."
  found: "course 1 end_date=2026-05-10; now()=2026-06-29; sweep cutoff=2026-05-29. end_date (2026-05-10) < cutoff (2026-05-29), so the daily soft-delete sweep correctly archived CS101 (deleted_at set 2026-06-29 04:20)."
  implication: "Root cause is stale fixture data (a Spring-2026 course tested in late June after its term + 1-month grace ended), not a code defect. The soft-delete sweep and the read filter both behave per spec (CLAUDE.md D-06)."

- timestamp: 2026-06-29
  checked: "Origin of the seed data (grep CS101 / 'Intro to Computer Science' / 'Homework 1' across repo)."
  found: "No committed seed/fixture file produces this data — only UI placeholder text in frontend/src/pages/admin/Courses.tsx and doc references. The myiu_dev rows were created manually via the API/UI during testing."
  implication: "There is no seed file to patch; the durable fix for goal 1 is to repair the live fixture (clear deleted_at + set a current end_date on CS101)."

- timestamp: 2026-06-29
  checked: "assignments handler ListAssignments + mapAssignment (backend/internal/assignments/handler.go)."
  found: "Maps each db.Assignment to AssignmentResponse via mapAssignment (explicit fields, pgtype unwrapped). Returns gin.H{\"data\": res}. NOT a raw sqlc row."
  implication: "The assignments list endpoint already follows the DTO pattern — no serialization bug here. Audit must focus on quizzes/grades."

- timestamp: 2026-06-29
  checked: "Serialization audit of ALL assignments handlers (handler.go): ListAssignments, ListSubmissions, SubmitAssignment, GetDownloadURL, GradeSubmission, FinalizeGrading, CreateAssignment."
  found: "Every data-returning endpoint maps to a json-tagged DTO (AssignmentResponse via mapAssignment, SubmissionResponse via mapSubmission) or returns gin.H{url|status}. No raw sqlc rows."
  implication: "assignments package is CLEAN for the serialization pattern."

- timestamp: 2026-06-29
  checked: "Serialization audit of ALL quizzes handlers (handler.go) cross-referenced with service return types."
  found: "ListQuizzes maps db.Quiz -> QuizResponse (json tags) OK. startAttempt/getAttempt return *StudentQuizAttemptView (json tags) OK. submitAttempt returns *SubmitAttemptResponse (json tags) OK. BUT CreateQuiz handler line 79 returns the raw service result `quiz` (type db.Quiz) directly: c.JSON(201, gin.H{\"data\": quiz}). db.Quiz (models.go:167) has NO json tags and pgtype fields (PoolSize pgtype.Int4, MaxGrade pgtype.Numeric, OpenAt/CloseAt pgtype.Timestamptz)."
  implication: "CONFIRMED serialization bug: POST /api/lecturer/courses/:id/quizzes returns PascalCase keys + pgtype objects (e.g. PoolSize:{Int32,Valid}, MaxGrade:{Int,Exp,Valid}). Frontend createQuiz (coursework-api.ts:162) consumes res.data.data as snake_case QuizResponse {id,title,pool_size,max_questions,max_grade,shuffle,retake_count,open_at,close_at,created_at} — contract is broken. Same systemic pattern as requests.ListCourseLecturers."

- timestamp: 2026-06-29
  checked: "Serialization audit of ALL grades handlers (handler.go) cross-referenced with service return types (service.go signatures)."
  found: "CreateScheme/GetScheme -> SchemeResponse; ComputeOverallForStudent -> OverallResponse; handleGetGrades list -> []OverallResponse; GetStudentGrades -> StudentGradesResponse. All have json tags (dto.go)."
  implication: "grades package is CLEAN for the serialization pattern."

- timestamp: 2026-06-29
  checked: "GitNexus + grep blast radius of the quizzes CreateQuiz handler edit."
  found: "Handler CreateQuiz is invoked only by the Gin router (handler.go:34). The planned mapQuiz helper is purely additive. No Go caller depends on the handler's response shape."
  implication: "Fix risk = LOW; confined to the POST /quizzes JSON response shape (the intended change)."

## Eliminated

- hypothesis: "Course-1 assignments are filtered out by a WHERE filter on user_id / role / created_by / course-membership mismatch in the assignments query (original debug-file guess)."
  evidence: "sqlc listCourseAssignments has no such filter — it filters only on course_id and the JOINed course's deleted_at. The empty result is caused by course 1 being soft-deleted, not by any user/role predicate."
  timestamp: 2026-06-29

## Resolution

root_cause: "TWO distinct issues. (1) GOAL-1 empty assignments: course 1 (CS101) was soft-deleted by the daily gocron sweep — its seeded end_date 2026-05-10 is older than the sweep cutoff now()-1month (2026-05-29). ListCourseAssignments (and ListCourseQuizzes) INNER JOIN courses with `c.deleted_at IS NULL` (a required soft-delete read filter), so the existing assignment/quiz of the archived course is correctly hidden → {\"data\":[]}. The backend code is correct; the defect is a stale manually-seeded fixture (a past-term Spring-2026 course used for late-June UAT). (2) GOAL-2 serialization: quizzes.CreateQuiz handler returned the raw sqlc struct db.Quiz (no json tags + pgtype fields) → PascalCase keys + pgtype objects, breaking the frontend QuizResponse contract. Same systemic pattern previously fixed in requests.ListCourseLecturers. assignments and grades packages audited CLEAN."
fix: "(1) GOAL-1 data fix: restored the live fixture in myiu_dev — UPDATE courses SET deleted_at = NULL, end_date = CURRENT_DATE + 90 WHERE id = 1. New end_date (2026-09-27) is past the sweep cutoff so the daily sweep will not re-archive it. NOTE: there is no committed seed file; this fix lives only in the dev DB and must be reapplied if the DB is rebuilt from scratch (seed CS101 with a future/current end_date). (2) GOAL-2 code fix: added mapQuiz(db.Quiz) QuizResponse helper in backend/internal/quizzes/handler.go and used it in CreateQuiz (was raw) and refactored ListQuizzes to reuse it (removed duplicated inline mapping)."
verification: "Live HTTP repro as LEC001 (cookie auth) against the running backend: GET /api/lecturer/courses/1/assignments now returns the assignment with clean snake_case keys (id,title,description,deadline,accept_late,late_threshold_days,max_score,grading_finalized_at,created_at). GET /api/lecturer/courses/1/quizzes returns clean snake_case. POST /api/lecturer/courses/1/quizzes (the fixed CreateQuiz) now returns {\"data\":{id,title,pool_size,max_questions,max_grade,shuffle,retake_count,open_at,close_at,created_at}} snake_case (was raw PascalCase+pgtype); verification quiz row deleted afterward. go build ./... OK; go vet on quizzes/assignments/grades OK; go test ./internal/quizzes/... ./internal/assignments/... against Postgres PASS; seed fixture confirmed intact after tests (course 1 active, 1 assignment, 1 quiz)."
files_changed:
  - "backend/internal/quizzes/handler.go (added mapQuiz helper; CreateQuiz now maps to QuizResponse; ListQuizzes refactored to reuse mapQuiz)"
  - "DB data (myiu_dev): courses.id=1 deleted_at cleared + end_date set active — not a code file, not durable across DB rebuild"

secondary_observation: "Latent (NOT fixed, out of scope, NOT the cause of the empty list): CreateAssignment and CreateQuiz authorize via ListCourseLecturers only and do NOT check courses.deleted_at, so a lecturer can create coursework on a soft-deleted course but never list it (this is how assignment id=2 was created at 14:57 after the course was swept at 04:20). Consider adding a deleted_at guard on create paths in a follow-up."
