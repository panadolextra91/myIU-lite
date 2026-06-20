# Phase 4 Review — Assignments & Quizzes

**Reviewed:** 2026-06-20
**Reviewer:** Claude (plan + review role) — 4 parallel adversarial slice reviews + manual confirmation of the two highest-stakes findings
**Scope:** commits `10443e0..959d75d` on `ft/phase-4` (47 files, +4727 LOC)
**Verdict:** **Architecture + the 3 existential pitfalls are SOLID. Do NOT merge yet** — there is one systemic authorization gap (1 BLOCKER + 3 HIGH IDOR), one flow-breaking functional bug, several MEDIUM correctness issues, and zero feature tests for the security-critical logic.

> **For Antigravity:** Fix in severity order (BLOCKER → HIGH → MEDIUM → LOW → tests). Each finding has `file:line`, the issue, a concrete fix, and a `verify`. After fixes, `bash scripts/check.sh` must stay green AND the "Do NOT regress" controls below must still hold.

---

## DoD gate status

`bash scripts/check.sh` → **exit 0** (golangci-lint + go build + go vet + frontend eslint/tsc/vite build all pass).
⚠️ `go test` was **SKIPPED locally** (no `DATABASE_URL`). CI will run it on Postgres — but this phase added **no feature tests** (only the Cloudinary spike), so CI currently exercises almost none of the new logic. See [Test Gap](#test-gap).

---

## ✅ Do NOT regress — these controls are correct and must keep holding

The whole point of this phase was three existential pitfalls. All three hold; any fix below must not break them:

1. **Cloudinary authenticated delivery** — assets uploaded `ResourceType:"raw"` + `Type:"authenticated"`; only `PublicID`+`Format` returned (`secure_url` discarded); downloads via a freshly-minted signed URL with **5-min TTL**, minted **only after** an ownership/role check. `shared/cloudinary/client.go:25-44`, `assignments/service.go:152-173`.
2. **Upload validation** — real `http.MaxBytesReader` 10MB cap (not just a Content-Length glance) + magic-byte sniff on real bytes (`mimetype`) + extension AND sniffed-type check; ZIP never extracted. `assignments/handler.go:112-151`.
3. **Quiz answer non-leakage** — `StudentOptionView{ID,Text}` is a separate struct (no `is_correct`, no `json:"-"`); correct answers revealed **only** when `now() > close_at` (D-51), independent of attempt status; verified across all read paths. `quizzes/dto.go:46-49`, `quizzes/attempt.go:171-220`.
4. **Same-transaction grade+notify** (`assignments/service.go:203-246`), **D-55 quiz-no-notify**, and **D-44 post-grade-resubmit edge case** (grade keyed to a specific version id) all verified correct.

---

## 🔴 BLOCKER

### F1 — Quiz attempt IDOR: no enrollment check, `courseID` ignored
**`backend/internal/quizzes/attempt.go:19-68` (`StartAttempt`)** *(manually confirmed)*

`StartAttempt(ctx, courseID, quizID, studentID)` receives `courseID` but **never uses it**. `GetQuizByID(quizID)` resolves the quiz by id alone, and there is **no `student_enrollments` query anywhere** in the feature. Route is gated only by `RequireRole(Student)` (`handler.go:43`). → **Any authenticated student can start/submit a quiz in a course they are not enrolled in**, and can pass a `courseID` that doesn't even own the quiz. `GetAttempt`/`SubmitAttempt` check `attempt.StudentID == studentID` (ownership), so the hole is specifically the missing enrollment + course-binding gate at start.

**Fix:**
1. Add a repo query `IsStudentEnrolled(courseID, studentID) bool` (e.g. `SELECT EXISTS(SELECT 1 FROM student_enrollments WHERE course_id=$1 AND student_id=$2)`) — mirror how `assignments` gates Submit.
2. In `StartAttempt`, after `GetQuizByID`: reject with a not-found/forbidden error unless `quiz.CourseID == courseID` **and** `IsStudentEnrolled(courseID, studentID)` is true.
3. For defense-in-depth, have `getAttempt`/`submitAttempt` handlers also pass `courseID`/`quizID` and verify `attempt.QuizID`'s quiz `CourseID == courseID` (the ownership check already blocks cross-student, this blocks cross-course id juggling).

**Verify:** a student NOT in `student_enrollments` for the course → `POST .../quizzes/:qid/attempts` returns 403/404, no `quiz_attempts` row created. A mismatched `courseID` for a real `quizID` → 404.

---

## 🟠 HIGH

### F2 — Quiz authoring IDOR: `quizID` not bound to `courseID`
**`backend/internal/quizzes/service.go:128-132` (`ImportQuestionsCSV`), `:184-188` (`AddUIQuestion`)**

Both authoring paths check `isLecturerOfCourse(courseID, lecturerID)` but then insert into `quiz_questions.quiz_id = quizID` **without verifying `quiz.CourseID == courseID`**. → A lecturer assigned to course A can write questions into course B's quiz: `POST /lecturer/courses/{A}/quizzes/{quizB}/questions`. `GetQuizByID` already exists (`repository.go:21`) but is unused here.

**Fix:** in both methods, `q := GetQuizByID(quizID)` and reject (forbidden/not-found) unless `q.CourseID == courseID` before inserting.
**Verify:** lecturer of course A posting questions to a quiz owned by course B → 403/404, no `quiz_questions` rows written.

### F3 — `ListAssignments` has no course-membership gate
**`backend/internal/assignments/handler.go:78`, `service.go:66-68`**

`ListCourseAssignments` returns any course's assignments (titles, deadlines, late policy) for any `:id` with no enrolled-student / assigned-lecturer check — unlike Submit/Grade/Download, which all gate. Cross-course information disclosure.

**Fix:** before returning, verify the JWT caller is enrolled (student) or in `course_lecturers` (lecturer), mirroring `Submit`/`DownloadURL`.
**Verify:** a user with no membership in the course → 403 on `GET .../courses/:id/assignments`.

### F4 — `ListQuizzes` is broken AND ungated
**`backend/internal/quizzes/handler.go:81-120`** *(manually confirmed)*

Two problems in one handler:
- **Functional (flow-breaking):** it builds `res []QuizResponse` (lines 95-117) then returns `gin.H{"status":"ok"}` (line 119) — the mapped data is **discarded**. Frontend expects `res.data.data` (`coursework-api.ts`), so **the quiz list is always empty for both student and lecturer**, breaking the take-quiz entry flow. The whole mapping loop is dead code.
- **Authorization:** no membership/enrollment gate — any authenticated user can read any course's quiz config (open/close windows, max grade) by id.

**Fix:** return `c.JSON(http.StatusOK, gin.H{"data": res})`; add the same membership/enrollment gate as F3.
**Verify:** lecturer/student lists their course's quizzes and gets the array; a non-member → 403.

---

## 🟡 MEDIUM

### F5 — Concurrent-start race → two attempts / double retake consume
**`backend/internal/quizzes/attempt.go:53-67`**
The IN_PROGRESS guard is read-then-write with no DB constraint; two simultaneous starts both miss `GetInProgressAttempt`, both pass the count check, both INSERT — burning two retakes and seeding two attempts.
**Fix:** add a partial-unique index `CREATE UNIQUE INDEX ... ON quiz_attempts(quiz_id, student_id) WHERE status='IN_PROGRESS';` and handle the conflict (return the existing in-progress attempt). This needs a follow-up migration **`000007`** (the single-migration-per-phase rule was a planning constraint; a post-ship bugfix migration is fine — just don't retro-edit `000006`).
**Verify:** two concurrent start requests → exactly one attempt row, one retake consumed.

### F6 — `pool_size` never validated against the real authored question count
**`backend/internal/quizzes/service.go:47-49`**
`CreateQuiz` enforces `MaxQuestions <= PoolSize`, but `PoolSize` is a free-form number typed at create time (the bank is empty then). Nothing ever checks that the actual authored question count ≥ the number to draw — `CountQuizQuestions` exists (`repository.go:37`) but is never called. A quiz with `max_questions=10` and only 2 authored questions will silently draw 2.
**Fix:** in `StartAttempt`/`seedAttempt`, if `CountQuizQuestions(quizID) < effectiveM`, either return a clear "quiz not ready" error or clamp the draw to the real count by design — pick one and make it explicit (recommend: reject with a clear error so a half-authored quiz can't be taken).
**Verify:** starting an attempt on a quiz whose authored count < max_questions returns a clear error (or a documented clamp), not a silent short draw.

### F7 — `mark-read` returns 200 for nonexistent / another user's notification
**`backend/internal/notifications/service.go:34-46`, `repository.go:29-31`, `db/notifications.sql.go:125-128`**
`MarkRead` is a sqlc `:exec`, which never returns `pgx.ErrNoRows`, so the `errors.Is(err, pgx.ErrNoRows)` guard can't fire. Marking a nonexistent/already-read/**another user's** notification all hit zero rows and return `200`. *Not a security hole* (the `recipient_id = $2` clause still prevents the cross-user write) but the 404 contract is broken.
**Fix:** change the query to `:execrows` (or `RETURNING id`), have the repo return RowsAffected, map `0 rows → ErrNotificationNotFound` → 404.
**Verify:** marking a notification id that isn't the caller's → 404.

### F8 — Submission version race + swallowed error
**`backend/internal/assignments/service.go:112-119`**
`GetMaxSubmissionVersion` error is coerced to `maxVer=0` (silently), and two concurrent submits read the same max → one INSERT fails the `UNIQUE(assignment_id,student_id,version)` as an opaque 500.
**Fix:** don't discard the error; better, compute the version in SQL: `INSERT INTO submissions (...) SELECT ..., COALESCE(MAX(version),0)+1 FROM submissions WHERE assignment_id=$ AND student_id=$`.
**Verify:** concurrent resubmits produce sequential versions with no 500.

### F9 — Unbounded late window when `accept_late` + no threshold
**`backend/internal/assignments/service.go:99-110`, `dto.go:12`**
When `LateThresholdDays` is NULL the threshold check is skipped → submissions accepted forever after the deadline; the UI even renders "unlimited" (`lecturer/Assignments.tsx:232`). Also no server validation that `late_threshold_days >= 0`.
**Fix:** confirm the D-45 intent. If "unlimited late" is NOT intended, require a threshold when `accept_late=true`; reject `late_threshold_days < 0` in `CreateAssignment`.
**Verify:** matches the confirmed D-45 semantic; negative threshold rejected.

### F10 — Submit accepts answers for arbitrary question ids + swallowed error
**`backend/internal/quizzes/attempt.go:259-265`**
`UpdateAttemptAnswer` is called for every `qID` in the client body with its error `_ =`-ignored; a student can POST answers for questions not in their drawn set, and a failed write still proceeds to grade.
**Fix:** check the error and return it; restrict updates to questions in the attempt's persisted answer set; ignore/reject unknown question ids.
**Verify:** submitting an answer for a question id not in the attempt → rejected or no-op-with-error; DB error aborts grading.

---

## ⚪ LOW / Polish

- **Server-side score range not enforced** — assignment grade (`assignments/handler.go:215`, `service.go:216`) and quiz `max_grade` accept any value (only `binding:"required"`/`min=0`); a crafted request stores negative/absurd scores. Clamp `0 <= score <= max` server-side.
- **Ignored `pgtype.Numeric.Scan` errors** — `_ = num.Scan(fmt.Sprintf("%f", ...))` in `assignments/service.go:215`, `quizzes/service.go:62`; NaN/Inf silently store invalid. Check the error.
- **N+1 reads on attempt review** — `buildAttemptView` re-loads options/questions per question (`attempt.go:208-218`). Reuse already-loaded slices.
- **Frontend submission wiring** — `student/Assignments.tsx` keeps submissions only in component state (no list endpoint), and lecturer "download" hardcodes `submissionId=1` (`lecturer/Assignments.tsx:226`). Wire a list-submissions endpoint and pass real ids. (Server authz still holds; this is a functional gap.)
- **Notification `total`** is `len(currentPage)`, not a real count (`notifications/handler.go:78-89`) — fine for the bell, note for future pagination.
- **Bundle size** — 869KB JS chunk warning from vite (non-blocking); optional code-split later.

---

## Test Gap

Project DoD requires unit + integration tests per phase; Phase 4 added **none** beyond the Cloudinary spike. The bugs above (especially the IDORs) are exactly what tests would catch. Add integration tests (testcontainers Postgres, per the project stack) for:

- **Enrollment/authz:** non-enrolled student → 403 on start/submit; non-member lecturer → 403 on authoring/grade/list; `courseID↔quizID` mismatch → 404. *(covers F1–F4)*
- **Answer non-leakage:** take / in-progress / post-submit / during-window-review JSON contains no `is_correct`; after `close_at` it does.
- **Idempotent submit:** double-submit and submit-after-auto-submit don't change the score; resume creates no extra attempt/retake and returns the same question set.
- **Auto-grade:** single-choice exact; multi-choice exact-set all-or-nothing (a subset must score wrong).
- **CSV reject:** 3-col, 5-col, `correct=E`, empty `correct` → rejected with a proper 4xx.
- **Same-tx rollback:** a forced notification-insert failure rolls back the grade.
- **Upload:** >10MB rejected (no Cloudinary write); wrong sniffed type rejected; `.zip` extension + zip magic accepted.

---

## Suggested fix order

1. **F1** (BLOCKER) — enrollment + course-binding on quiz attempts.
2. **F2, F3, F4** (HIGH) — the rest of the authorization gap + the `ListQuizzes` return bug. Consider a shared helper (e.g. `assertCourseMember(ctx, courseID, userID, role)` and `assertQuizInCourse(quizID, courseID)`) reused across assignments + quizzes so the boundary is enforced uniformly.
3. **F5–F10** (MEDIUM).
4. LOW / polish.
5. Tests (lock the fixes so they can't regress).

After all fixes: `bash scripts/check.sh` green, and the four "Do NOT regress" controls re-verified.

---

# Re-Review Addendum (2026-06-20) — fixes verified, test quality must be fixed

**Method:** read every fix diff + ran the suite against a real `postgres:17-alpine` (all 7 migrations applied, `go test ./...`).

## Fixes — VERIFIED CLOSED
F1–F10 + polish are genuinely fixed and wired (not just claimed). Confirmed: the shared `internal/shared/authz` helper (`AssertCourseMember` + `AssertQuizInCourse`) is called in `StartAttempt`/`GetAttempt`/`SubmitAttempt` (+ handlers pass `courseID`/`quizID`), both quiz-authoring paths, `ListAssignments`, `ListQuizzes`; migration `000007` partial-unique index exists in the DB; F8 computes the version atomically in SQL; F7 uses `:execrows` + `rows==0 → 404`. The 4 "Do NOT regress" controls are intact. **Correctness is mergeable.**

## 🔴 BLOCKER for "phase is tested" — the new tests are theater

`backend/internal/{assignments,quizzes}/security_test.go` all PASS, but a fresh migrated DB has **zero** `courses`/`student_enrollments`/`course_lecturers` rows (those are created at runtime by Phase 3 admin provisioning, never seeded by migrations). So every test that uses `courseID=1`/`9999` **short-circuits at the authz gate** before reaching the code it names. They are green for the wrong reason.

| Test | Claims to test | Actually hits | Action |
|---|---|---|---|
| `quizzes: CSV Reject - Malformed CSV` | CSV format rejection | `isLecturerOfCourse(1,1)` → Forbidden, never reaches the parser. CSV data is also wrong format (7 cols; real format is `question,A,B,C,D,correct` = 6). | Rewrite with a real authorized lecturer+quiz; use the real 6-col format. |
| `quizzes: Idempotent Submit` | `validQIDs` / idempotency | `AssertCourseMember` → Forbidden before `GetAttemptByID`. (Code comment "fail at GetAttemptByID" is itself wrong.) | Rewrite with a real attempt. |
| `quizzes: Non-leakage - Answers Hidden` | **the #1 existential control** | **EMPTY body — zero assertions.** | Write a real assertion on the serialized response. |
| `assignments: Same-Tx Rollback` | grade+notify rollback | `AssertCourseMember(9999,...)` → Forbidden before the tx begins. | Rewrite to actually force a notification-insert failure. |
| `*: Enrollment Authz` (×2) | the enrollment gate | Genuinely reaches + asserts the gate. | Keep. |

## Test-fix guidance for Antigravity

**1. Build a real fixture (the missing precondition).** Add a per-test setup helper (or `testutil` in the package) that, against `DATABASE_URL`, inserts and returns ids for: a lecturer user + a student user (unique usernames via `time.Now().UnixNano()`, like the P0-1 auth-test pattern), a course, a `course_lecturers` row (lecturer↔course), a `student_enrollments` row (student↔course), a quiz (with `open_at`/`close_at` you control), and ≥2 questions each with options (one correct). Tear it all down in `t.Cleanup()` (DELETE in FK order) so `go test ./...` (parallel packages, one shared DB) doesn't pollute — this is the same isolation discipline the P0-1 fix used for auth tests. Without unique data + cleanup, these tests will flake exactly like `users/TestImportAllOrNothing` does today.

**2. Then write assertions that reach the real logic:**

- **Non-leakage (highest priority — the existential control):** with the quiz window **OPEN**, start an attempt and fetch it (and fetch a SUBMITTED attempt during the open window). Assert the **serialized JSON** (marshal the returned `StudentQuizAttemptView`) contains **no** `is_correct` and **no** `correct_options` for any question. Then move the window so `now() > close_at` (insert the quiz with a past `close_at`, or update it) and re-fetch the terminal attempt — assert `correct_options` **now appears**. This is the test that proves QUIZ-03/D-51; it must assert on bytes, not call internal helpers.
- **Idempotent submit:** start a real attempt, answer, submit → capture score. Submit again → assert the score is unchanged and no re-grade happened (e.g. `attempt.Status` already terminal, returned score equals the stored one). Also assert submitting an answer for a `questionID` **not** in the attempt's drawn set returns the `invalid question ID` error (F10).
- **Same-tx rollback:** grade a real submission but force the notification insert to fail inside the tx (e.g. a recipient_id that violates the FK, or a deliberately bad notification payload), then assert the **grade was NOT persisted** (re-read the submission → `graded_at IS NULL`). That proves the single-transaction guarantee (NOTIF-02) — the current test never enters the tx.
- **CSV reject:** with the authorized lecturer+quiz fixture so the parser is reached, table-test the malformed rows in the **real format**: 3 columns, 5 columns, `correct=E`, empty `correct`, then assert a 4xx/`ErrInvalidQuestion` per row and that a valid 6-col row succeeds.

**3. Also fix (nits):**
- `quizzes/service.go` `AddUIQuestion` — the question-type guard returns a copy-pasted wrong message ("single choice must have exactly 1 correct option" for a type check). Give it its own message.
- `internal/users/TestImportAllOrNothing` (pre-existing, NOT Phase 4) — fails under `go test ./...` due to shared-DB pollution (passes in isolation). Scope its count assertion to its own imported batch / unique usernames, mirroring the P0-2 healthz fix, so it can't flake CI.

**Definition of done for the test pass:** each of the 4 rewritten tests must FAIL if its control is reverted (delete the `StudentOptionView` mapping → non-leakage test goes red; split the tx → rollback test goes red). A test that stays green when the control is broken is still theater.
