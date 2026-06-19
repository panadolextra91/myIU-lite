# Feature Research

**Domain:** Lightweight university student-management / LMS-lite (myIU lite)
**Researched:** 2026-06-19
**Confidence:** MEDIUM (corroborated across Canvas, Stanford CS course policies, Microsoft Forms, Postgres/Laravel soft-delete community, and audit-logging compliance guides; not vendor-official for this exact product)

## Feature Landscape

The product's stated core value is: *students and lecturers run a course end-to-end (assignments, quizzes, grades, announcements, requests) without falling back to email, and Admin provisions everything from CSV.* Features below are categorized against that value — not against a full LMS like Moodle/Canvas.

### Table Stakes (Users Expect These)

Missing any of these makes the product feel broken or untrustworthy for its three roles.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Role-based auth (Student / Lecturer / Admin) | Every action is scoped to a role; without it there is no product | MEDIUM | Authorization must be enforced server-side per endpoint, not just hidden in UI |
| Forced first-login password change | Default password = birthday `DDMMYYYY` is guessable; must rotate immediately | LOW | Block all other actions until changed; flag on the user row |
| Change password / forgot-reset | Universal account hygiene; users lock themselves out constantly | MEDIUM | "Forgot" with no email channel needs a story — likely admin-assisted reset (see PITFALLS) |
| Course roster / enrollment model | Announcements "to enrolled students", grading, and submissions all key off enrollment | MEDIUM | `enrollment(student_id, course_id)` unique pair is the spine of the whole app |
| Assignment submission (file upload) | Primary student deliverable | MEDIUM | PDF/ZIP only, hard 10MB; validate MIME + size server-side, store via Cloudinary; record server timestamp |
| MCQ quiz with auto-grade on submit | A quiz that doesn't return a grade isn't a quiz | HIGH | The grading rules are the hidden complexity — see ARCHITECTURE/PITFALLS |
| Gradebook / grade record per student | Students expect to see grades; lecturers expect to post them | MEDIUM | Grade is the join of (student, course, assignment-or-quiz) |
| Announcements (lecturer → students) | Replaces the "email blast" the product is killing | MEDIUM | Targeting: all enrolled OR specific students |
| In-app notifications (grades, replies, announcements) | The whole "no email" promise depends on users reliably *seeing* events | MEDIUM | Persist as rows so offline users see them on next login (reliability win over email) |
| Admin account creation (manual + CSV) | Without provisioning there are no users | MEDIUM | username = ID, default password = birthday; CSV is bulk path |
| Course CRUD (start/end dates) | Admin's core object | LOW | Dates drive the auto soft-delete sweep |
| Audit log of admin actions | Admin can reset others' passwords — accountability is non-negotiable | MEDIUM | Append-only; who/what/when/target (see PITFALLS for scope) |

### Differentiators (Competitive Advantage)

These are where myIU-lite earns its keep versus "just use email + a shared drive."

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| In-app student→lecturer requests (leave-early / absence / custom) | Replaces a whole category of email; structured, auditable, replyable | MEDIUM | Lecturer replies yes/no → auto-notifies student. Structured types + one freeform |
| Auto-notify on grade / reply | Tightens the feedback loop; no "did you see my email?" | LOW-MEDIUM | Triggered by the grading/reply action, idempotent on source event |
| CSV provisioning (accounts + enrollment) | Admin sets up an entire term in minutes, not hand-entry | MEDIUM-HIGH | The integrity/validation work is the real cost (see PITFALLS) |
| Auto soft-delete sweep (1 month after end date) | Removes manual cleanup; keeps history | MEDIUM | Background job; soft delete preserves records. Narrow scope = safe |
| Quiz auto-shuffle + max-questions-per-quiz | Light anti-cheat + question-bank sampling without manual effort | MEDIUM | Shuffle must preserve correct-answer mapping; max-questions = sample N from bank |

### Anti-Features (Commonly Requested, Often Problematic)

Explicitly **do not build** these for the lite edition — each adds large surface area without serving the core value.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Email as a channel | "Everyone uses email" | The entire premise is to replace it; adds SMTP, deliverability, bounce handling | In-app notifications + requests (already the design) |
| Essay / short-answer / manual-graded quizzes | "Quizzes should support everything" | Manual grading + partial credit UI is a large feature; breaks "auto-grade on submit" | MCQ only in MVP (already out of scope) |
| Discussion forums / threaded chat | "Students want to talk" | Moderation, notifications, abuse handling — a product unto itself | Announcements (one-way) + requests (one-to-one) cover MVP comms |
| Gradebook weighting / curves / GPA calc | "Grades should roll up" | Policy-heavy, error-prone, varies per institution | Store raw grades per item; defer aggregation |
| Rubrics / peer review / plagiarism detection | "Real LMSes have it" | Each is a major subsystem | Plain grade + optional feedback text |
| SCORM / content packages / video hosting | "LMS = course content" | Storage + player + standards compliance | Out of scope; this is a coursework-ops tool, not a content LMS |
| Real-time websocket push for notifications | "Notifications should be instant" | Connection management, scaling, reconnect logic | Poll-on-load / fetch unread count; persisted rows already guarantee delivery |
| Multiple file types / >10MB uploads | "Let me upload anything" | Storage + validation + virus-scan surface | PDF/ZIP, hard 10MB (already constrained) |
| Self-service "forgot password" via email link | Standard pattern | No email channel exists in this product | Admin-assisted reset (already an admin capability); see PITFALLS for the gap |
| Hard delete of courses | "Just delete it" | Loses history, breaks grade/audit references | Soft delete only (already the design) |

## Feature Dependencies

```
Auth + Roles
   └──requires──> Admin account creation (CSV/manual)
                      └──requires──> Forced first-login password change

Enrollment (student↔course, lecturer↔course)
   └──requires──> Course CRUD
   └──requires──> Accounts exist
        └──enables──> Announcements to enrolled students
        └──enables──> Assignment submission (scoped to a course)
        └──enables──> Quiz taking (scoped to a course)
        └──enables──> Grades (per student per course item)
        └──enables──> Requests (student → that course's lecturer)

Notifications (persisted rows)
   └──underpins──> Grade auto-notify
   └──underpins──> Request-reply auto-notify
   └──underpins──> Announcement delivery

Audit log
   └──wraps──> All admin actions (account CRUD, password reset, enrollment, course CRUD, soft-delete sweep)

Auto soft-delete sweep
   └──requires──> Course end dates
   └──requires──> Soft-delete column + read-time filtering everywhere
```

### Dependency Notes

- **Everything course-scoped requires Enrollment:** announcements-to-enrolled, submissions, quizzes, grades, and requests all resolve "which lecturer / which students" through the enrollment table. Build enrollment before any of them.
- **Notifications underpin three features:** grade auto-notify, request-reply auto-notify, and announcement delivery are all the same notification primitive. Build the notification model once, early, and the three features become thin.
- **Audit log wraps admin actions:** it is cross-cutting. Decide its schema before building the admin features so each admin mutation writes one log row in the same transaction.
- **Soft-delete filtering is viral:** once a course can be soft-deleted, *every* read path touching courses (and ideally their children) must filter `deleted_at IS NULL`. This is a constraint on all downstream queries, not a standalone feature.
- **CSV enrollment requires accounts AND courses to exist:** validate both foreign keys before insert; an enrollment CSV referencing a non-existent ID is the most common import failure.

## MVP Definition

### Launch With (v1)

This is essentially the full Active requirement set in PROJECT.md — the MVP is already tightly scoped. Sequenced by dependency:

- [ ] Auth + roles + forced first-login password change — gate for everything
- [ ] Admin: account creation (manual + CSV), course CRUD, enrollment (CSV) — provisioning spine
- [ ] Audit log (admin actions) — must exist as admin features land, not bolted on after
- [ ] Notification model (persisted, per-recipient, read state) — primitive for delivery
- [ ] Assignment submission (PDF/ZIP, 10MB, Cloudinary, server timestamp) + lecturer grading + auto-notify
- [ ] MCQ quizzes (create with shuffle + max-questions; auto-grade on submit; record grade)
- [ ] Announcements (all / specific enrolled) + delivery via notifications
- [ ] Student↔lecturer requests (leave-early / absence / custom) + yes/no reply + auto-notify
- [ ] Auto soft-delete sweep (course, 1 month after end date)

### Add After Validation (v1.x)

- [ ] Quiz attempts policy controls (multiple attempts, highest/last) — add when lecturers ask for retakes
- [ ] Per-assignment late policy (block vs accept-and-flag, grace window) — add once a real deadline dispute occurs
- [ ] Bulk grade entry / CSV grade import — add when manual grading at scale hurts
- [ ] Notification unread badge / digest — add when users miss things despite persistence
- [ ] Admin restore of soft-deleted course — add when an accidental sweep happens

### Future Consideration (v2+)

- [ ] Real-time push notifications — defer until polling proves insufficient
- [ ] Question bank reuse across quizzes/courses — defer until question volume justifies it
- [ ] Self-service password reset channel (SMS/email) — defer; needs a comms channel decision
- [ ] Grade aggregation / weighting — defer until institution policy is defined

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Auth + roles + forced first-login change | HIGH | MEDIUM | P1 |
| Admin account creation (CSV/manual) | HIGH | MEDIUM | P1 |
| Course CRUD + enrollment (CSV) | HIGH | MEDIUM | P1 |
| Audit log | HIGH | MEDIUM | P1 |
| Notification model (persisted) | HIGH | MEDIUM | P1 |
| Assignment submission + grading + auto-notify | HIGH | MEDIUM | P1 |
| MCQ quiz auto-grade (shuffle, max-questions) | HIGH | HIGH | P1 |
| Announcements (all/specific) | HIGH | LOW | P1 |
| Student↔lecturer requests + reply | HIGH | MEDIUM | P1 |
| Auto soft-delete sweep | MEDIUM | MEDIUM | P1 |
| Multiple quiz attempts policy | MEDIUM | MEDIUM | P2 |
| Per-assignment late policy / grace | MEDIUM | MEDIUM | P2 |
| Soft-delete restore | MEDIUM | LOW | P2 |
| Real-time push | LOW | HIGH | P3 |
| Grade aggregation / weighting | MEDIUM | HIGH | P3 |

**Priority key:** P1 = must have for launch · P2 = should have, add when possible · P3 = nice to have, future.

## Role-Specific Expected Behaviors & Commonly-Missed Edge Cases

These are the behaviors that, if unspecified, produce the most bugs and disputes. They feed PITFALLS and requirements.

### Student

- **Assignment submission:** lateness is decided by the **server** timestamp at upload, not the client clock; one minute past = late. Resubmission before deadline should overwrite (last submission graded). MUST validate file type (PDF/ZIP) and size (10MB) **server-side** — client checks are bypassable. Empty/zero-byte and corrupt-ZIP uploads are common edge cases.
- **Quiz taking:** skipped/blank answers score 0 (not null). If a quiz is shuffled, the student's selected option must map back to the canonical correct option, not a screen position. If `max-questions` samples N from a bank, two students may see different question sets — the grade must be out of the questions *they* saw. Double-submit / refresh-on-submit must not double-record or re-grade.
- **Requests:** the custom request is plain text — needs length limits and sanitization (stored/displayed to lecturer). A request must resolve to the correct course's lecturer; an unenrolled student has no lecturer to send to.
- **Receiving grades/announcements:** if the student was offline when the event fired, they must still see it on next login (persisted notification, not fire-and-forget).

### Lecturer

- **Grading:** posting a grade must auto-notify the student exactly once (idempotent on the submission/grade event — a retry or double-click must not double-notify or double-record). Grading a resubmitted file should grade the latest submission.
- **Quiz creation:** auto-shuffle must preserve the correct-answer mapping per attempt. `max-questions` greater than the bank size, or = 0, are edge cases to reject. Editing a quiz *after* students have submitted must either be blocked or trigger explicit regrade/versioning — silently changing the answer key corrupts existing grades.
- **Announcements:** "specific student(s)" must be restricted to students **enrolled in that lecturer's course** — a lecturer must not be able to message arbitrary users. "All" = all enrolled in their course, not all students in the system.
- **Request replies:** yes/no reply auto-notifies the student; a reply to an already-answered request should be prevented or treated as an update.

### Admin

- **Account creation:** username = ID must be unique — collisions on duplicate IDs (within a CSV or against existing users) are the top import failure. Birthday → `DDMMYYYY` password requires a parseable, valid date column; reject unparseable rows. Validate the **entire CSV before inserting** and return a row-level error report; decide atomic-batch (reject all on any error — simplest) vs skip-and-report.
- **Enrollment CSV:** must verify *both* the user ID and the course ID exist before inserting, and prevent duplicate `(student_id, course_id)` enrollments (unique constraint). Enrolling into a soft-deleted/ended course is an edge case to reject.
- **Password reset for others:** this is the reason the audit log exists — every reset must write an audit row (who reset whom, when). It's also the de-facto "forgot password" path given no email channel; that gap should be explicit in requirements.
- **Course soft-delete sweep:** runs 1 month after end date. Must be idempotent (re-running the sweep doesn't re-delete or error), must not sweep already-deleted courses, and must record the sweep in the audit log. Soft-deleting a course must not orphan or expose its enrollments/submissions in active queries — every course read path filters `deleted_at IS NULL`.
- **Audit log scope:** log who / what (action) / which (target resource) / when (server time), and before→after for mutations like password change or course edit. It must be **append-only**: the admin who performs actions must not be able to silently edit/delete the log (restrict to insert-only; no update/delete path). Username = the ID makes "who" unambiguous.

## Competitor Feature Analysis

| Feature | Moodle / Canvas | Google Classroom | Our Approach (myIU-lite) |
|---------|-----------------|------------------|--------------------------|
| Quiz types | Many (essay, matching, calculated, MCQ…) | Forms-based, limited | MCQ auto-grade only; shuffle + max-questions |
| Submission types | Any file, large, multiple | Drive files | PDF/ZIP, 10MB, Cloudinary |
| Comms | Forums, inbox, email, announcements | Stream, email | In-app announcements + structured requests; no email |
| Provisioning | Admin UI, SIS sync, CSV | Google Workspace sync | CSV (accounts + enrollment), username=ID, birthday password |
| Course lifecycle | Manual archive/delete | Archive | Auto soft-delete sweep 1mo after end date |
| Accountability | Extensive logs/reports | Workspace admin logs | Focused append-only admin audit log |
| Late policy | Configurable per assignment, penalties | Late flag | MVP: deadline + server timestamp (per-assignment grace deferred to v1.x) |

**Takeaway:** myIU-lite deliberately occupies the gap *below* Google Classroom — fewer features, but the request workflow and CSV/auto-lifecycle automation that small-LMS tools usually lack. Compete on workflow consolidation and admin automation, not feature count.

## Sources

- LMS feature baselines: [Canvas Classic vs New Quizzes (U. Delaware)](https://sites.udel.edu/canvas/2020/10/classic-quizzes-v-new-quizzes/), [Using Canvas Quiz Tool for Auto-Graded Quizzes (WUSTL)](https://mycanvas.wustl.edu/app/uploads/2019/02/Final_-Using-Canvas-Quiz-Tool-for-Automatically-Graded-Quizzes-2f50qng.pdf)
- Quiz auto-grading / partial credit / shuffling: [ClassMarker partial grading](https://www.classmarker.com/online-testing/blog/Partial-Grading-for-Multiple-Response-Questions), [Microsoft Forms auto-grading tips](https://techcommunity.microsoft.com/blog/educationblog/five-essential-tips-on-auto-grading-for-microsoft-forms-quizzes/2030680), [Google Forms timed-exam limitations (Qualtir)](https://qualtir.com/blog/google-forms-online-exams-timed-assessments)
- Late submission / deadline / resubmission: [Stanford CS107 late policy](https://web.stanford.edu/class/archive/cs/cs107/cs107.1186/latepolicy.html), [Stanford CS106B late policy](https://web.stanford.edu/class/archive/cs/cs106b/cs106b.1238/late)
- Audit logging: [Audit Logging Best Practices (Sonar)](https://www.sonarsource.com/resources/library/audit-logging/), [Compliance by Design: tamper-proof audit logs (Mattermost)](https://mattermost.com/blog/compliance-by-design-18-tips-to-implement-tamper-proof-audit-logs/), [Immutable / append-only audit trails (DesignGurus)](https://www.designgurus.io/answers/detail/how-do-you-enforce-immutability-and-appendonly-audit-trails)
- Soft delete pitfalls: [Soft Deletion Probably Isn't Worth It (brandur.org)](https://brandur.org/soft-deletion), [Soft Delete & Unique Constraint (Medium)](https://gusiol.medium.com/soft-delete-and-unique-constraint-da94b41cff62), [Soft Delete real-world unique constraint (ZenStack)](https://zenstack.dev/blog/soft-delete-real)

---
*Feature research for: lightweight university student-management / LMS-lite (myIU lite)*
*Researched: 2026-06-19*
