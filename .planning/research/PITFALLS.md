# Pitfalls Research

**Domain:** University student-management platform (myIU lite) — Go/Gin/PostgreSQL backend, React/Zustand/shadcn frontend, Cloudinary storage
**Researched:** 2026-06-19
**Confidence:** HIGH (security mechanics verified against current Cloudinary/GitHub/OWASP docs; domain logic pitfalls from established practice)

## Critical Pitfalls

### Pitfall 1: Predictable default passwords with leaky reset enforcement

**What goes wrong:**
Default password = birthday `DDMMYYYY` and username = student/lecturer ID means every account starts with a credential an attacker can fully enumerate. `DDMMYYYY` has only ~36,500 valid values (≈100 years × 365 days), and student IDs are sequential/guessable. The "force change on first login" gate is then usually only enforced in the *UI* — the API still accepts requests with the default password, so an attacker who calls the backend directly never hits the forced-change wall. Result: trivially brute-forceable accounts that bypass the one control meant to protect them.

**Why it happens:**
Forced reset is implemented as a frontend redirect ("if must_change_password, route to /change-password") instead of a server-side authorization rule. Every other endpoint keeps working because the JWT/session was already issued at login.

**How to avoid:**
- Store a `must_change_password` (and `password_changed_at`) flag on the user. After login while this flag is true, issue a **restricted token/session** that is authorized for *only* the change-password and logout endpoints; every other handler rejects it with 403. Enforce this in middleware, not in the SPA.
- Never log or return the default password; generate it server-side at provisioning, never echo it in CSV-import responses.
- Rate-limit + lock login attempts per account (e.g., 5 failures → temporary lock) so the small `DDMMYYYY` keyspace can't be swept.
- Hash with bcrypt/argon2 even for default passwords (never store plaintext "so admin can read it").

**Warning signs:**
Integration test: log in with default password, then call a normal endpoint (e.g., list courses) *without* changing the password — if it returns 200, the gate is cosmetic. Also: any code path that emails/displays the generated password.

**Phase to address:**
Auth & account-provisioning phase (foundational). Must land before any feature endpoints exist, or the restricted-token middleware gets retrofitted incorrectly.

---

### Pitfall 2: CSV import that half-succeeds and corrupts state

**What goes wrong:**
Bulk import of accounts/enrollments processes rows in a loop, inserting as it goes. Row 50 has a duplicate ID or bad date → the import dies mid-way. Now 49 accounts exist, 51 don't, and the admin re-uploads the *same* file → the first 49 collide and either error out again or create duplicates. Common sub-failures: duplicate IDs within the same file, IDs that already exist in the DB, malformed `DDMMYYYY` birthdays producing impossible default passwords, UTF-8 BOM / Excel Windows-1252 encoding mangling Vietnamese names (đ, ô, ư), trailing empty rows, and silent column-order assumptions.

**Why it happens:**
Naive `for row in csv { INSERT }` with no transaction and no pre-validation pass. Encoding is assumed UTF-8 because the dev's test file was UTF-8.

**How to avoid:**
- **Two-pass, all-or-nothing:** Pass 1 validates *every* row (schema, ID uniqueness within file, ID not already in DB, parseable birthday, required columns present) and returns a per-row error report **without** writing anything. Pass 2 runs only if pass 1 is clean, inside a **single DB transaction** — any error rolls back the whole import.
- Detect and strip UTF-8 BOM; decode explicitly (try UTF-8, surface a clear error rather than silently mojibake). Document "save as CSV UTF-8" for admins.
- Match columns by header name, not position. Reject files with missing/extra required headers.
- Make enrollment import idempotent: `ON CONFLICT (student_id, course_id) DO NOTHING` so re-running a partially-applied file is safe.
- Cap row count and file size to prevent a giant upload locking a table.

**Warning signs:**
Import that returns "success" with no count of rows processed; no dry-run/preview step; tests only cover the happy-path file; any duplicate-ID test that isn't in the suite.

**Phase to address:**
Admin provisioning phase (right after auth). The two-pass + transaction pattern is hard to bolt on later.

---

### Pitfall 3: File upload trusting extension/MIME instead of content (and unbounded ZIP expansion)

**What goes wrong:**
Submission accepts "PDF and ZIP only, ≤10MB" by checking the filename extension or the browser-supplied `Content-Type` header — both are attacker-controlled and trivially spoofed. A renamed `.exe`/`.html`/`.svg` passes as `.pdf`; a polyglot file is valid as two types at once. Worse for ZIP: validating the *compressed* size (10MB) says nothing about the *uncompressed* size — a 10MB zip bomb can expand to terabytes, exhausting disk/RAM if the server ever extracts or inspects it. Size enforcement is also often done only client-side, so a direct API call uploads a 2GB file.

**Why it happens:**
Extension/MIME checks are one line and "look done." Devs forget ZIP is a compression container, not a flat blob. Size limit lives in the React dropzone, not the Gin handler.

**How to avoid:**
- Validate by **magic bytes** server-side: PDF must start with `%PDF-` (and reject if header isn't within the first bytes — note PDF spec allows offset, so bound it); ZIP must start with `PK\x03\x04`. Reject on mismatch regardless of extension/Content-Type.
- Enforce the 10MB limit in the **Gin handler** via `http.MaxBytesReader` / `MaxMultipartMemory`, before reading the whole body into memory — not just in the SPA.
- For ZIP: **do not extract on the server.** Store the archive as an opaque blob (lecturer downloads and opens it). If you must inspect it, read `zip.File.UncompressedSize64`, refuse if the *declared* uncompressed total or per-file ratio exceeds a cap (e.g., aggregate > 100MB or ratio > 100:1), and stream-decompress with a hard byte limit (`io.LimitReader`) rather than trusting the header.
- Store with a random server-generated filename (never the user's filename — path traversal / overwrite). Strip path components.

**Warning signs:**
No magic-byte check in the upload handler; size validation only in frontend; any `unzip`/`archive/zip` extraction of user uploads; uploads named with the original filename.

**Phase to address:**
Assignment-submission phase. Magic-byte + server-side size cap must be in the first version of the upload handler.

---

### Pitfall 4: Cloudinary uploads publicly readable by URL (broken access control)

**What goes wrong:**
Cloudinary's **default delivery type is `upload` = public**. Anyone with (or who can guess/iterate) the asset URL can fetch a student's submitted assignment — no auth. Even "private"/"authenticated" types fail if implemented wrong: a *private* derived asset, once generated, is reachable without the signature; and storing signed URLs in the DB long-term defeats the point since they're meant to expire. Public IDs are often predictable (`submissions/CS101/student123`), so URL enumeration leaks the whole class's work.

**Why it happens:**
The Cloudinary quickstart uploads as public; access control is opt-in and under-documented for the "students shouldn't see each other's files" case. Devs treat Cloudinary URLs like a CDN, not like protected storage.

**How to avoid:**
- Upload submissions with **`type: 'authenticated'`** (original *and* derivations require a valid signature) — not `private`, not the default `upload`.
- Serve files only through **short-lived signed URLs** generated on-demand by the backend *after* it checks RBAC (is this caller the owning student, the course's lecturer, or an admin?). Default signed-URL expiry is ~1 hour; keep it short.
- Use unguessable public IDs (random suffix), not `course/studentID`.
- Never embed the Cloudinary API secret in the frontend; all signing happens server-side via env-configured credentials.

**Warning signs:**
Upload code with no `type` parameter (defaults to public); Cloudinary URLs stored and served directly to the browser; ability to open another student's submission by editing the URL; API secret present in any client bundle.

**Phase to address:**
Assignment-submission phase (storage integration). Access-control decision must be made at integration time — changing delivery type later requires re-uploading assets.

---

### Pitfall 5: Quiz answer leakage and shuffle/grading correctness bugs

**What goes wrong:**
Several distinct failures cluster here:
- **Answer leakage:** the quiz-fetch endpoint returns the full question objects *including* `is_correct` flags (or a `correct_answer` field). Students read answers from the network tab / API response before submitting. This is the single most common LMS quiz bug.
- **Shuffle corruptness:** options are shuffled for display, but the submission maps the *shuffled index* back to the original answer incorrectly, so grading marks right answers wrong (or vice-versa). Or questions are shuffled per-render so a student's answers don't line up with the questions on submit.
- **max-questions / question-pool:** "max questions per quiz" draws a random subset, but grading assumes the full bank, so denominators are wrong, or two students get different question counts with no score normalization.
- **Regrade:** a lecturer fixes a wrong answer key, but already-submitted attempts keep the old (wrong) grade — no recompute path.
- **Concurrent / repeat attempts:** no constraint preventing a student from submitting the same quiz twice (double-submit, or open in two tabs) → duplicate grades, or a race that records two rows.

**Why it happens:**
The question model is reused verbatim for both authoring and student-facing fetch, so correct-answer data rides along. Shuffle is done with display indices instead of stable option IDs. Auto-grade is computed once on submit with no idempotency key.

**How to avoid:**
- **Separate the student-facing DTO** from the authoring model: the take-quiz endpoint returns questions with stable option **IDs** but *no* correctness data. Grade by comparing submitted option IDs against the answer key **server-side only**.
- Shuffle by reordering option IDs (stable), not array positions; the client submits IDs, so display order is irrelevant to grading.
- For a question pool, persist *which* questions/options each attempt received (store the attempt's question set) so grading and review are deterministic and the denominator is correct.
- Enforce one active attempt: unique constraint `(student_id, quiz_id)` (or attempt-number column) + check "already submitted" in a transaction; idempotent submit (re-submit returns the existing grade, doesn't create a second).
- Provide a regrade action that recomputes affected attempts when the answer key changes; record both old and new scores in the audit/grade history.

**Warning signs:**
Take-quiz API response contains any correctness/`correct` field (inspect it directly); grading logic references array index rather than option ID; no unique constraint on attempts; no test for "submit twice"; no test that a shuffled quiz still grades correctly.

**Phase to address:**
Quiz phase. Answer-leakage and the student-DTO split must be in the first implementation; regrade can be a follow-up but the data model (stored attempt question set, attempt uniqueness) must exist from day one.

---

### Pitfall 6: Soft-delete and enrollment referential integrity gaps

**What goes wrong:**
"Auto soft-delete courses 1 month after end date" plus enrollment relationships create silent data-integrity holes:
- Soft-deleted courses (`deleted_at` set) still show up because most queries forget `WHERE deleted_at IS NULL`. Conversely, hard joins break when a soft-deleted course is filtered out but enrollments/grades reference it.
- The sweep job runs on a schedule; if it double-fires or runs during a partial outage it can re-process or, worse, a careless version hard-deletes and orphans submissions/grades/quiz attempts.
- Enrollment to a soft-deleted course, or to a non-existent student/course (no FK), leaves dangling rows. Unenrolling a student often deletes the enrollment row *and* orphans their grades.
- Sweep uses server-local time / no timezone handling → courses deleted a day early/late around DST or UTC boundaries.

**Why it happens:**
Soft-delete is added as a column with no global query discipline; the sweep is a quick cron with no idempotency; FKs are skipped "for speed" with raw SQL.

**How to avoid:**
- Enforce FKs at the DB level (`enrollments.course_id REFERENCES courses(id)`, `student_id REFERENCES users(id)`). Decide ON DELETE behavior explicitly (RESTRICT for courses that have grades).
- Centralize the `deleted_at IS NULL` filter (a base query/scope or a view) so no read path forgets it. Soft-delete should cascade *logically* (a soft-deleted course hides its enrollments) without destroying grade history.
- Make the sweep **idempotent and time-explicit:** `UPDATE courses SET deleted_at = now() WHERE end_date < now() - interval '1 month' AND deleted_at IS NULL`. Run in UTC; the `deleted_at IS NULL` guard makes re-runs safe.
- Never hard-delete records that have dependent grades/submissions; keep history (the whole point of soft delete).
- Log every sweep run (count of affected courses) into the audit log.

**Warning signs:**
Any list query missing the `deleted_at` filter; no FK constraints in the schema; the sweep job has no `AND deleted_at IS NULL` guard; deleting an enrollment with no consideration of its grades; tests that don't cover "course soft-deleted → student can no longer access it but lecturer can still see archived grades."

**Phase to address:**
Course/enrollment data-model phase (schema + FKs early), with the sweep job in the course-lifecycle phase.

---

### Pitfall 7: Audit log that is incomplete, bypassable, or tamper-able

**What goes wrong:**
The requirement is "audit log recording all admin actions" — especially because admin can change other users' passwords. Common failures: the log is written from the handler, so any code path that mutates data *without going through that handler* (a bulk SQL update, a missed endpoint) leaves no trace. Logs record "admin did something" without *who/what/when/before-after*. The log lives in the same DB the admin can write to, with no append-only protection, so a malicious/compromised admin can edit or delete their own trail. Sensitive values (new passwords) get logged in plaintext.

**Why it happens:**
Audit logging is treated as an afterthought sprinkled into a few handlers, not a cross-cutting concern. "Append-only" is assumed but never enforced.

**How to avoid:**
- Define the audit event up front: `actor_id`, `actor_role`, `action`, `target_type`, `target_id`, `timestamp (UTC)`, `ip`, and a *redacted* before/after summary (never the actual password — log "password reset", not the value).
- Write audit entries in the **same DB transaction** as the mutating action (so you can't have the action without the log, and a rollback rolls back both). Centralize via middleware or a service wrapper so no admin action can skip it.
- Make the table append-only at the app layer (no UPDATE/DELETE endpoints; ideally a DB role/grant that can INSERT but not UPDATE/DELETE on `audit_log`). For stronger tamper-evidence, hash-chain entries (each row stores hash of previous) so deletion/edit is detectable.
- Cover **every** admin mutation: account create/import, password reset, enrollment changes, course CRUD, *and* the automated sweep.

**Warning signs:**
Audit writes scattered across handlers (grep for inconsistency); any admin endpoint with no corresponding audit insert; audit table with UPDATE/DELETE access; plaintext passwords/secrets in log rows; audit row missing actor or target.

**Phase to address:**
Cross-cutting — establish the audit service/middleware in the admin phase, and add "emits audit event" to the success criteria of every subsequent admin-facing feature.

---

### Pitfall 8: RBAC authorization holes (role checked but ownership not)

**What goes wrong:**
The three roles (Student/Lecturer/Admin) are checked at a coarse level ("is this a lecturer?") but **object-level ownership** is not. So *any* lecturer can grade *any* course's assignments, view another lecturer's quizzes, or message students they don't teach; *any* student can fetch another student's submission/grade by changing an ID in the URL (IDOR). Also: authorization enforced only in the frontend (menu hidden) while the API is open; and JWT role claims trusted without re-checking on each request.

**Why it happens:**
Middleware checks the role but the handler doesn't verify "does this lecturer own this course?" / "is this the student's own record?". Frontend route guards create a false sense of security.

**How to avoid:**
- Enforce authorization **server-side on every endpoint**, two layers: (1) role (is caller a lecturer?) and (2) **ownership/scope** (is this lecturer assigned to this course? is this submission the requesting student's own?). Never authorize by hidden UI.
- For every read/mutation of a course-scoped resource, join through enrollment/assignment to confirm the caller is in scope. Treat any user-supplied ID as hostile (prevent IDOR).
- Default-deny: a new endpoint with no explicit authorization should reject, not allow.
- Keep role in a server-validated session/JWT but re-derive course scope from the DB, not from client input.

**Warning signs:**
Handlers that check role but take a `course_id`/`submission_id` from the URL without verifying the caller's relationship to it; any "the frontend hides this so it's fine"; tests that only cover the authorized user, never "lecturer B accessing lecturer A's course."

**Phase to address:**
Foundational auth phase establishes the pattern (role + ownership middleware/helpers); every feature phase must add ownership checks and an "unauthorized actor is rejected" test.

---

### Pitfall 9: Notifications that silently drop on grading / request-reply

**What goes wrong:**
Grades, announcements, and request replies are supposed to be delivered "automatically." If the notification is created in a separate step *after* the grade is committed (or in a fire-and-forget goroutine), a crash/panic between the two means the grade exists but the student is never notified — and there's no retry. Or the same grade fires duplicate notifications on a retry/double-submit. Since email is explicitly out of scope, in-app notifications are the *only* channel, so a dropped one is invisible to everyone.

**Why it happens:**
"Send notification" is bolted on after the main write, outside the transaction, with no idempotency or delivery tracking. Goroutine errors are swallowed.

**How to avoid:**
- Create the notification record in the **same transaction** as the triggering event (grade saved → notification row inserted atomically). In-app notifications are just DB rows; there's no reason to send them out-of-band.
- Make creation idempotent (tie to the event ID) so retries don't duplicate.
- Track read/unread state; don't lose unread notifications on logout.
- Don't swallow errors in background goroutines — for a lite app, just write synchronously in the transaction and avoid goroutines entirely.

**Warning signs:**
Notification insert outside the DB transaction of the triggering action; `go func(){ ... }()` around notification sends with ignored errors; no idempotency tying notification to event; a grade with no corresponding notification row in tests.

**Phase to address:**
Built into each feature that triggers a notification (grading, announcements, requests). Define the "event + notification in one transaction" pattern in the first such phase.

---

### Pitfall 10: CI/CD guardrails that look enforced but don't actually block merges

**What goes wrong:**
The constraint says "merge is blocked unless tests pass." Three common ways this is silently false:
1. The workflow runs tests, but the check is **not added to branch protection "required status checks,"** so a red build is still mergeable. Branch protection config is *separate* from the workflow file.
2. **Path filtering self-defeat:** the project uses `backend`/`frontend` branches with path-triggered workflows. A skipped workflow (due to `paths`/`paths-ignore`) either leaves the required check **stuck "pending"** (blocking forever) or reports a **skipped job as "success"** — letting changes through untested.
3. Tests "pass" because they never ran the real thing: integration tests need PostgreSQL, but CI has no DB service, so the suite is skipped/mocked and green is meaningless. Or coverage is collected but no threshold gates it.

**Why it happens:**
Devs assume "workflow exists = merge protected." The branch-protection ↔ required-check linkage and the skipped-job-reports-success behavior are non-obvious.

**How to avoid:**
- Explicitly mark the test job as a **required status check** in branch protection for `main`/`backend`/`frontend`. Verify by opening a PR that fails a test and confirming the merge button is blocked.
- Provide PostgreSQL as a **service container** in the workflow (matches the "Docker-only Postgres" constraint) and run real integration tests against it; fail the job on any test failure and on missing DB connectivity.
- Avoid relying on path filters for required checks; if used, add a "dummy success" job for skipped paths or use a single always-run gate job. Use job-level conditionals instead of workflow-level `paths` for required checks.
- Make CI fail closed: non-zero exit on test failure, lint/syntax failure, and migration/DB-check failure.

**Warning signs:**
A merged PR whose CI was red or skipped; required-status-check list empty in branch protection; no `services: postgres` in the workflow; integration tests that pass locally but are skipped in CI; "tests pass" with zero assertions run.

**Phase to address:**
Phase 0 / infrastructure (CI scaffolding). Validate the gate *actually blocks* before building features on top of it.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Forced password change enforced only in frontend | Fast to ship | Full auth bypass via direct API; rewrite of session model | Never |
| CSV import row-by-row, no transaction | Simple loop | Partial state, duplicate accounts, painful re-import | Never (use two-pass + txn) |
| Validate upload by extension/MIME only | One line | Arbitrary file upload, malware/polyglot, zip bombs | Never |
| Cloudinary default public upload | Works in demo | Cross-student data leak via URL guessing | Never for submissions |
| Returning full question objects (with answers) to take-quiz | Reuse one model | Trivial answer leakage; rebuild DTO + retest | Never |
| Skipping FK constraints with raw SQL | Faster writes | Orphaned grades/enrollments, integrity drift | Only for throwaway tables, never for core relations |
| Audit logging in a few handlers | Quick | Incomplete trail; admin actions untraceable | Never (centralize) |
| Notification send outside the event transaction | Decoupled-feeling | Silent drops, duplicates; the only channel fails | Never (in-app = DB row, keep it in txn) |
| Coarse role-only authz (no ownership) | Less code | IDOR across students/courses; security review failure | Never |
| Trusting workflow existence as the merge gate | No config work | Broken code merges to protected branches | Never (wire required checks) |
| Mocked integration tests because CI has no DB | Green builds | False confidence; real bugs ship | Only as a temporary stopgap with a tracked task to add the Postgres service |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Cloudinary | Default `type: upload` (public); serving stored URLs directly | `type: 'authenticated'`; backend-generated short-lived signed URLs after RBAC check; API secret server-side only |
| Cloudinary | Predictable public IDs (`course/studentID`) | Random public IDs; never enumerable |
| PostgreSQL (Docker) | Tests assume native DB / no service container in CI | `services: postgres` in workflow; same connection config local & CI |
| PostgreSQL | No FKs / no migrations, ad-hoc schema | Versioned migrations; FK constraints on enrollments/grades/courses |
| Gin file upload | Size limit only client-side; whole body read into RAM | `http.MaxBytesReader` / `MaxMultipartMemory` in handler before reading body |
| GitHub Actions | Required checks not linked in branch protection; skipped jobs report success | Explicitly require the test job; avoid workflow-level `paths` for required gates; verify a failing PR is blocked |
| JWT/session | Trusting role/scope claims from client without re-check | Re-derive course scope from DB each request; default-deny |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Loading whole upload into memory before size check | Memory spikes, OOM on large/malicious uploads | `MaxBytesReader` caps the stream before buffering | First multi-hundred-MB upload (or one attacker) |
| ZIP "inspection" that decompresses fully | CPU/disk exhaustion, DoS | Don't extract; if you must, cap with `io.LimitReader` + ratio check | First zip bomb |
| N+1 queries listing courses/enrollments/grades | Slow dashboards as enrollment grows | Joins / batched queries; index FK columns | Hundreds of enrollments per course |
| Sweep job scanning all courses unindexed | Slow nightly job, lock contention | Index `end_date`, `deleted_at`; bounded `UPDATE ... WHERE` | Thousands of historical courses |
| Quiz attempt list without index on `(student_id, quiz_id)` | Slow grade lookups, no fast dup-check | Unique index doubles as the dedupe guard | Many attempts per quiz |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Predictable default password + small `DDMMYYYY` keyspace | Mass account takeover by enumeration | Restricted post-login token until change; account lockout/rate-limit; hash all passwords |
| Forced reset enforced in UI only | Auth bypass via direct API | Server-side restricted session authorized only for change-password/logout |
| IDOR on submissions/grades (ID in URL) | Students read each other's work/grades | Ownership check on every object access; treat IDs as hostile |
| Public Cloudinary assets | Cross-student file leak | Authenticated delivery + signed URLs gated by RBAC |
| Quiz answers in take-quiz response | Students see answers pre-submit | Student DTO with no correctness data; grade server-side |
| Audit log writable/deletable by admin | Cover-up of malicious admin action | Append-only (no UPDATE/DELETE), txn-coupled, optional hash chain |
| Plaintext passwords in audit/logs | Credential leak | Log "password reset" event, never the value |
| Path traversal via original filename | Overwrite/escape storage path | Random server-generated filenames; strip path components |
| API secret (Cloudinary) in frontend bundle | Full account compromise | All signing server-side; secrets in backend `.env` only |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| CSV import fails with no per-row feedback | Admin can't tell which rows are bad | Per-row validation report before committing |
| Silent encoding mangling of Vietnamese names | Garbled student records | Detect BOM/encoding; clear "save as UTF-8" guidance |
| Forced-change screen with no clear path / dead-ends user | New users locked out on first login | Clear, unskippable but completable change-password flow |
| Quiz double-submit shows error / loses answers | Student panic, support load | Idempotent submit returns existing grade gracefully |
| Dropped notifications (no delivery) | Student misses grade/announcement entirely | Txn-coupled in-app notifications; unread badge |
| Soft-deleted course vanishes with no archive view | Lecturer loses access to past grades | Hidden from active lists but archived/read-only view retained |

## "Looks Done But Isn't" Checklist

- [ ] **Forced password change:** verify the *API* (not just UI) rejects normal calls until the password is changed.
- [ ] **CSV import:** verify duplicate-ID, already-exists, bad-birthday, and non-UTF-8 files are caught with per-row errors and that a mid-file failure rolls back entirely.
- [ ] **File upload:** verify a renamed non-PDF is rejected (magic bytes), the 10MB limit is enforced server-side, and a zip bomb cannot exhaust resources.
- [ ] **Cloudinary:** verify another student's submission URL returns 401/403 (not the file).
- [ ] **Quiz:** verify the take-quiz response contains no correct-answer data, a shuffled quiz still grades correctly, and a second submit doesn't create a second grade.
- [ ] **Soft-delete:** verify deleted courses disappear from active lists everywhere, grade history survives, and the sweep is safe to run twice.
- [ ] **Audit log:** verify every admin action (incl. CSV import and the sweep) produces a row with actor/target/timestamp, and that rows can't be updated/deleted.
- [ ] **RBAC:** verify lecturer B cannot access lecturer A's course and student A cannot fetch student B's record.
- [ ] **Notifications:** verify every grade/reply produces exactly one notification row (no drops, no dupes).
- [ ] **CI gate:** verify a PR with a failing test is actually un-mergeable and that integration tests ran against a real Postgres.

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Default-password / UI-only forced reset shipped | HIGH | Add restricted-session middleware; force-expire all sessions; require all unchanged-password accounts to reset |
| Partial CSV import corrupted accounts | MEDIUM | Identify created rows from audit/timestamps; delete the partial batch; re-run with two-pass+txn version |
| Public Cloudinary submissions leaked | HIGH | Switch delivery to authenticated; re-upload affected assets with new random IDs; invalidate old URLs; treat as data breach |
| Quiz answers were exposed in API | MEDIUM | Patch DTO; assume answers leaked → consider regrade/new question pool for affected quizzes |
| Audit log found editable/incomplete | HIGH | Add append-only enforcement; backfill is impossible — accept gap, document, prevent forward |
| Soft-delete orphaned grades | MEDIUM | Reconcile via FK audit query; restore from history; add FKs to prevent recurrence |
| CI let red builds merge | LOW | Add required status checks; re-run CI on recent merges to find regressions |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| CI gate doesn't block merges (#10) | Phase 0 — infra/CI | Failing-test PR is un-mergeable; Postgres service runs integration tests |
| Default-password / forced-reset bypass (#1) | Auth & provisioning | API rejects normal calls pre-change; lockout works |
| RBAC ownership holes (#8) | Auth (pattern) + every feature | Cross-actor access tests fail closed |
| CSV import partial failure (#2) | Admin provisioning | Two-pass validation report; mid-file failure rolls back |
| Audit log incompleteness/tamper (#7) | Admin (audit service) + all admin features | Every admin action logged; rows immutable |
| Soft-delete / enrollment integrity (#6) | Course/enrollment data model + lifecycle | FKs enforced; sweep idempotent; grade history survives |
| File upload content/zip-bomb (#3) | Assignment submission | Magic-byte reject; server-side size cap; bomb-safe |
| Cloudinary public access (#4) | Assignment submission (storage) | Other students get 403 on submission URL |
| Quiz answer leakage / grading (#5) | Quiz | No answers in DTO; shuffle grades right; no double-submit |
| Notification drops (#9) | Each notification-triggering feature | One notification per event, in-transaction |

## Sources

- [Cloudinary — Media Access Control and Authentication](https://cloudinary.com/documentation/control_access_to_media) (HIGH)
- [Cloudinary — Generating delivery URL signatures](https://cloudinary.com/documentation/delivery_url_signatures) (HIGH)
- [Cloudinary — Authenticated/private images still accessible by URL (support thread)](https://support.cloudinary.com/hc/en-us/community/posts/115001874511-Authenticated-or-private-images-still-accessible-to-anyone-with-access-to-the-URL) (HIGH)
- [GitHub Docs — About protected branches / required status checks](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches) (HIGH)
- [GitHub Docs — Troubleshooting required status checks (skipped jobs report success; path-filter pending)](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/collaborating-on-repositories-with-code-quality-features/troubleshooting-required-status-checks) (HIGH)
- [OWASP — File Upload Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/File_Upload_Cheat_Sheet.html) (HIGH)
- [Sourcery — File Upload Content-Type / MIME Type Bypass](https://www.sourcery.ai/vulnerabilities/file-upload-content-type-bypass) (MEDIUM)
- [UBOS — Understanding Zip Bombs: Construction, Risks, Mitigation](https://ubos.tech/news/understanding-zip-bombs-construction-risks-and-mitigation-2/) (MEDIUM)
- Established LMS/edtech practice for quiz integrity, RBAC ownership/IDOR, soft-delete discipline, and audit-log design (HIGH — domain knowledge)

---
*Pitfalls research for: university student-management platform (myIU lite)*
*Researched: 2026-06-19*
