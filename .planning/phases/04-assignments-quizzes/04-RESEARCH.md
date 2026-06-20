# Phase 4: Assignments & Quizzes - Research

**Researched:** 2026-06-20
**Domain:** Go/Gin file-upload + Cloudinary authenticated delivery, MCQ quiz engine (answer-key safety), persisted notification primitive
**Confidence:** HIGH (stack is locked + verified; Cloudinary API surface confirmed against pkg.go.dev; schema/patterns grounded in existing Phase 1-3 code)

## Summary

Phase 4 is a backend-heavy phase with one genuinely new external integration (Cloudinary) and three existential correctness pitfalls baked into the success criteria: authenticated Cloudinary delivery, magic-byte upload validation, and quiz answer non-leakage. Every library is already pinned in `.claude/CLAUDE.md` and present in `backend/go.mod` — **do not re-litigate the stack.** The research effort is entirely about *how to use the locked libraries correctly* and *which schema/serializer shapes prevent the three pitfalls*.

Three findings dominate planning. (1) Cloudinary: upload PDF/ZIP with `ResourceType:"raw"` + `Type:"authenticated"`, and generate downloads with `uploader.PrivateDownloadURL(...)` carrying `ExpiresAt` — confirmed available in cloudinary-go/v2 v2.16.0. The backend gates this behind an ownership check before minting the URL. (2) Magic-byte validation reads the first 512 bytes; `gabriel-vasile/mimetype` (already an indirect dep, v1.4.12) is more reliable than stdlib `http.DetectContentType` for distinguishing PDF/ZIP and is the recommended tool — but the planner must know **`.docx`/`.xlsx`/`.pptx` sniff as ZIP**, so a ZIP whitelist necessarily admits Office files (accepted: out-of-scope to inspect ZIP contents). (3) The quiz take-API must serialize options from a DTO that *structurally cannot* carry `is_correct` — the safest design is a separate "student view" struct, not field omission on the DB row.

**Primary recommendation:** Build three feature folders (`internal/assignments`, `internal/quizzes`, `internal/notifications`) + one shared client (`internal/shared/cloudinary`), add migration `000006`, and reuse the already-proven `pool.Begin(ctx)` + `q.WithTx(tx)` transaction pattern (from `internal/enrollments/service.go`) for the same-transaction grade+notification write. Use **lazy evaluation** (not the daily sweeper) for quiz `AUTO_SUBMITTED` on window close.

## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-44 — Versioned submissions; never overwrite.** Multiple submissions allowed while window open; each creates a **new version record**; prior versions preserved/viewable. Window = deadline + late config (D-02) + threshold; while open: first-submit/resubmit/unlimited; once closed: reject all. **Most recent version = active version** = what lecturers grade by default.
- **D-45 — Late submissions flagged only; no automatic penalty.** Record `is_late`, `submitted_at`, human-readable `late_duration` ("5 minutes", "2 days", "6 days 12 hours"); surface "late by X" to lecturer. System records facts; lecturer judges penalty. Late resubmissions still create versions while window open.
- **D-46 — Grading inputs: score required, feedback optional.** Form requires a **score**, allows **optional feedback text**; students view both. Grade applies to active version (D-44).
- **D-47 — Question-bank model with random per-attempt selection.** Pool of N questions; each attempt randomly selects M (M≤N). Per-quiz config: Pool Size N, Max Questions M, Max Grade, Shuffle (y/n), Retake Count. Each retake gets a newly generated set. Shuffle=Yes → randomize question selection + question order + answer order; Shuffle=No → fixed configured order.
- **D-48 — Two authoring modes; exact-match auto-grading.** (1) CSV import fixed format: 4 choices A-D, exactly 1 correct: `question,A,B,C,D,correct` (e.g. `What is 2 + 2?,1,2,3,4,D`). (2) Manual UI: single-choice (1 correct, radio) AND multi-choice (multiple correct, checkbox). Grading: single-choice correct when `selected == correct`; multi-choice correct **only** on exact set match `selected_set == correct_set` (**all-or-nothing**, no partial credit).
- **D-49 — Quizzes use availability window (Open At / Close At); no late, no timer.** Students start/submit only while open; retakes only while open (subject to count). After Close At: no new attempts/retakes/submissions. No late submission for quizzes. Students may review completed attempts after submission (subject to D-51 reveal policy).
- **D-50 — Official quiz score = MAX across completed attempts.** `official_score = MAX(attempt_scores)`. Gradebook (Phase 5) stores official only; full history stays available. All attempts within window (D-49) + within retake count (D-03).
- **D-51 — Answer-reveal policy is window-bound, not attempt-bound.** While window open: student reviewing a completed attempt sees final score + their submitted answers + per-question correct/incorrect status — but **NOT** correct answers or explanations. After window closes: correct answers + per-question results become visible. Applies **regardless of remaining/completed retakes**. This is the concrete enforcement of QUIZ-03 answer-non-leakage.
- **D-52 — Attempt consumed on START; resumable; auto-submit on window close.** New attempt consumes one available attempt when student **starts** (not on submit). States: `IN_PROGRESS`, `SUBMITTED`, `AUTO_SUBMITTED`. While `IN_PROGRESS` student may leave/return (resume same attempt — no new attempt, no extra retake) but may not start another. On submit → `SUBMITTED` + `submitted_at`. If window closes while `IN_PROGRESS` → `AUTO_SUBMITTED`. New attempt may start only when previous is `SUBMITTED`/`AUTO_SUBMITTED`, window still open, attempts remain.
- **D-53 — Notifications persist fully-rendered content at creation.** Row: `recipient_id, type, title, body, resource_type (opt), resource_id (opt), link (opt), created_at, read_at`. **Title + body rendered at creation, stored directly.** One row per recipient with `read_at` as read marker. Stays readable even if resource later archived/soft-deleted/modified.
- **D-54 — Centralized notification center (bell in header).** Bell icon + unread badge count, notification list page, mark-as-read on click, deep-link navigation to resource. One center aggregates all sources. **MVP scope:** bell + unread badge + list page + mark-read-on-click + deep-link. **Out:** real-time push, dropdown previews, categories, preferences.
- **D-55 — Phase 4 notifications fire ONLY on assignment grading.** Lecturer saves assignment grade → grade persisted **AND** notification created for student in **same transaction** (NOTIF-02), per D-53, delivered via D-54. **Quiz grading does NOT generate notifications** (scores shown inline post-submission).

### Claude's Discretion

- **Feature-folder layout (D-10).** `internal/assignments/`, `internal/quizzes/`, `internal/notifications/`, each with `handler.go / service.go / repository.go / model.go / dto.go`; sqlc queries under `backend/db/queries/`. New `internal/shared/cloudinary/` client. Migration(s) `000006+` add: `assignments`, `submissions` (versioned), `quizzes`, `quiz_questions`, `quiz_question_options`, `quiz_attempts`, `quiz_attempt_answers`, `notifications`.
- **Ownership/authorization (AUTH-05 pattern).** All coursework endpoints behind `RequireRole(...)` + membership-derived scope: only a lecturer in `course_lecturers` may author/grade; only a student in `student_enrollments` may submit/take. `user_id` always from JWT, never client-supplied. Reads filter soft-deleted courses. Errors use `{error:{code,message}}` envelope.
- **Audit-logging of lecturer actions — default OFF.** `audit_log` is admin-only (ADMIN-08); Phase 4 lecturer actions NOT audit-logged unless planning surfaces a reason. **(Researcher confirmation below — keep OFF.)**
- **Notification read-marker UX.** Mark-as-read on click (D-54); "mark all read" affordance discretionary. Exact `type` enum strings, `link` URL shapes, badge-count query are planner's call.

### Deferred Ideas (OUT OF SCOPE)

- Per-attempt quiz timers / duration limits / lockdown-browser (availability windows only).
- Partial-credit / weighted-selection / negative-marking quiz grading (exact-match all-or-nothing only).
- Notification real-time push / dropdown previews / categories / preferences.
- Notification templates / localization.
- Announcement / assignment-creation / enrollment / gradebook-publication notifications (Phase 5).
- Late submission for quizzes (time-bounded events only).
- **Server-side ZIP extraction (REQUIREMENTS Out of Scope — zip-bomb risk; files stored, never unpacked).**

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| ASMT-01 | Lecturer creates assignment for a course with deadline (date+time) | `assignments` table (course_id FK, deadline TIMESTAMPTZ) + lecturer-ownership check via `course_lecturers` |
| ASMT-02 | accept-late=y/n; if y, threshold X days or "no threshold" (D-02) | `accept_late BOOL`, `late_threshold_days INT NULL` (NULL = no threshold = accept until course soft-delete) |
| ASMT-03 | Student submits single PDF/ZIP ≤10MB, magic-byte validated server-side | `mimetype.Detect` (first 512B) + extension whitelist + `http.MaxBytesReader` (Magic-Byte Validation section) |
| ASMT-04 | System enforces late policy using server timestamp | Window computation in `service.go` from `now()` vs deadline+threshold (Submission Versioning section) |
| ASMT-05 | Files stored as Cloudinary authenticated assets; downloads via backend short-lived signed URLs gated by role/ownership | `Type:"authenticated"` + `ResourceType:"raw"` upload, `uploader.PrivateDownloadURL` w/ `ExpiresAt` behind ownership gate (Cloudinary section) |
| ASMT-06 | Lecturer views/grades submission; saving grade auto-notifies student | Grade on active version + same-tx notification insert (Notification + Transaction sections) |
| QUIZ-01 | Lecturer creates quiz: max questions, max grade, shuffle y/n, retake count (D-03) | `quizzes` config columns (Quiz Data Model section) |
| QUIZ-02 | Quiz questions via CSV upload or UI | CSV fixed format `question,A,B,C,D,correct`; UI single+multi-choice (D-48) |
| QUIZ-03 | Take-quiz API never exposes which option is correct | Separate student-view DTO that cannot carry `is_correct` (Answer Non-Leakage section) |
| QUIZ-04 | Shuffle preserves correct-answer mapping by stable option ID | `quiz_question_options.id` is the stable key; shuffle reorders presentation only (Quiz Data Model) |
| QUIZ-05 | Auto-grade on submit against max grade; idempotent per attempt | Grade computed in submit handler, guarded by attempt status transition (Idempotent Auto-Grade section) |
| QUIZ-06 | Enforce retake limit; retakes tracked as distinct attempts | `quiz_attempts` rows; attempt-count check consumed-on-start (D-52) |
| NOTIF-01 | Single persisted notification primitive, one row per recipient, read marker | `notifications` table per D-53 (Notification Schema section) |
| NOTIF-02 | Notifications accompanying a mutation written in same transaction | `pool.Begin` + `q.WithTx(tx)` (Same-Transaction Write section) |

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Assignment CRUD + late policy config | API / Backend | DB | Business rules (ownership, deadline math) live in service.go; persisted in Postgres |
| File upload validation (magic byte, size) | API / Backend | — | Security control — MUST be server-side; client validation is UX only |
| File storage (raw asset) | External (Cloudinary) | API / Backend | Cloudinary holds bytes; backend holds metadata + gates access |
| Signed download URL minting | API / Backend | External (Cloudinary) | Ownership check + TTL must be backend-enforced before URL is generated |
| Late-policy enforcement | API / Backend | — | Server timestamp is authoritative (ASMT-04); never trust client clock |
| Quiz authoring (CSV/UI) | API / Backend | Browser/Client | Parsing + validation backend; form UX on client |
| Quiz attempt generation (M-of-N draw, shuffle) | API / Backend | — | Random draw + correct-answer mapping MUST be server-side (answer-key safety) |
| Take-quiz serialization (hide correct option) | API / Backend | — | Existential: the DTO boundary is the security control |
| Auto-grade on submit | API / Backend | — | Authoritative scoring; idempotent per attempt |
| AUTO_SUBMITTED on window close | API / Backend | — | Lazy evaluation in read/write path; no client involvement |
| Notification persistence + same-tx write | API / Backend + DB | — | Atomicity (NOTIF-02) requires one DB transaction |
| Notification bell + badge + list | Browser/Client | API / Backend | Pull-based UI reads backend count/list endpoints |

## Standard Stack

All libraries are **already locked in `.claude/CLAUDE.md` and present in `backend/go.mod`** — this phase adds ONE new direct dependency (`cloudinary-go/v2`) and promotes ONE indirect dep (`mimetype`) to direct.

### Core (already present — no install)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Gin | v1.11.0 | HTTP, multipart upload | Locked. `c.Request.MultipartForm` / `c.FormFile` for upload; `router.MaxMultipartMemory` tuning |
| pgx/v5 | v5.7.2 | Postgres driver + `pgx.Tx` | Locked. `pool.Begin(ctx)` → `tx` is the same-tx mechanism for NOTIF-02 [VERIFIED: backend/go.mod] |
| sqlc | v1.31.1 (CLI) | SQL→Go codegen + `WithTx` | Locked. Generated `*Queries` has `.WithTx(tx)` — already used in `enrollments/service.go` [VERIFIED: backend/internal/enrollments/service.go] |
| golang-migrate | v4.18.x (CLI) | `000006` migration | Locked. CI runs migrations before tests |
| golang-jwt/jwt/v5 | v5.3.1 | role+user_id from JWT | Locked; ownership-from-JWT already in middleware [VERIFIED: backend/go.mod] |

### Supporting (new or promoted)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| **cloudinary-go/v2** | **v2.16.0** (use ≥ Jun-2025 security release; v2.16.0 released 2026-05-28 satisfies this) | File storage SDK | NEW direct dep. `cloudinary.NewFromURL(cfg.CloudinaryURL)`, `uploader.Upload(..., Type:"authenticated", ResourceType:"raw")`, `uploader.PrivateDownloadURL(...)` [VERIFIED: go list -m -versions returns up to v2.16.0; pkg.go.dev confirms API] |
| **gabriel-vasile/mimetype** | v1.4.12 (already indirect) | Magic-byte sniffing | Promote to direct. `mimetype.Detect(buf)` more reliable than stdlib for PDF/ZIP discrimination [VERIFIED: backend/go.mod indirect] |
| encoding/csv (stdlib) | — | Quiz CSV import | No new dep; mirrors Phase 3 enrollment CSV pattern |
| math/rand/v2 (stdlib) | — | M-of-N draw + shuffle | No new dep. **Use crypto-free `math/rand/v2`** — shuffle is fairness, not security; the security control is the DTO boundary, not RNG unpredictability |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `mimetype.Detect` | stdlib `http.DetectContentType` | CLAUDE.md mentions `http.DetectContentType`; it works but returns coarse `application/zip` only sometimes and is weaker on PDF edge cases. `mimetype` is already vendored and gives a precise extension + MIME. Either is acceptable; **mimetype recommended** because it's already present and more precise. |
| `uploader.PrivateDownloadURL` | Cloudinary `AuthToken` signed delivery URL | AuthToken (cookie/token-based) suits streaming many assets; `PrivateDownloadURL` is the simpler fit for "one short-lived download link per submission file" and directly takes `ExpiresAt`. **PrivateDownloadURL recommended.** |
| Lazy AUTO_SUBMITTED eval | Reuse Phase 3 daily sweeper | A daily sweeper would leave attempts `IN_PROGRESS` up to 24h after close. Lazy eval (resolve on next access after `close_at`) is correct-by-construction and idempotent. **Lazy recommended; optional sweeper as belt-and-suspenders only.** |

**Installation:**
```bash
cd backend
go get github.com/cloudinary/cloudinary-go/v2@v2.16.0
# mimetype: already indirect; `go mod tidy` after first direct import promotes it
```

**Version verification (done this session):**
- cloudinary-go/v2: `go list -m -versions` → latest **v2.16.0** [VERIFIED: Go module proxy]
- mimetype: `go list -m` → **v1.4.12** present as indirect [VERIFIED: backend/go.mod]

## Package Legitimacy Audit

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| cloudinary-go/v2 | Go module proxy | mature (v2 line since 2021) | official Cloudinary SDK | github.com/cloudinary/cloudinary-go | OK | Approved — official vendor SDK, version confirmed via `go list -m -versions` |
| gabriel-vasile/mimetype | Go module proxy | mature (8+ yrs) | widely used | github.com/gabriel-vasile/mimetype | OK | Approved — already a transitive dep of Gin's validator chain |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

*Note: the `package-legitimacy check` seam targets npm/PyPI/crates, not the Go module proxy; the npm `@cloudinary` result it returned is irrelevant (this is a Go module). Go packages here were verified directly via `go list -m -versions` against the Go module proxy, which is the authoritative registry for Go.*

## Architecture Patterns

### System Architecture Diagram

```
ASSIGNMENT SUBMISSION (student)
  Browser ──multipart POST /courses/:id/assignments/:aid/submissions──▶ Gin handler
                                                                          │
                          ┌── MaxBytesReader(10MB) + ContentLength check ─┤ (reject early, 413)
                          │                                               ▼
                          │   read first 512B ─▶ mimetype.Detect ─▶ extension∩sniff whitelist
                          │                                               │ (reject → 415)
                          ▼                                               ▼
                   service.go: ownership (enrolled?) + window open? (now vs deadline+threshold)
                          │ (closed → 422 window_closed)                 │
                          ▼                                               ▼
              cloudinary.Upload(raw, authenticated) ◀───────────── stream file bytes
                          │ returns public_id, format
                          ▼
              repository: INSERT submissions (new version, is_late, late_duration, cloudinary_public_id)

ASSIGNMENT DOWNLOAD (lecturer or owning student)
  Browser ──GET .../submissions/:sid/download-url──▶ handler ─▶ service ownership gate
                                                                     │
                                                                     ▼
                                       uploader.PrivateDownloadURL(public_id, ExpiresAt=now+5m)
                                                                     │
                                          {url} ◀── 200 ── browser follows signed URL ─▶ Cloudinary

GRADE + NOTIFY (lecturer)  [SAME TRANSACTION — NOTIF-02]
  POST .../submissions/:sid/grade ─▶ handler ─▶ service ownership gate
        tx := pool.Begin(); qtx := q.WithTx(tx)
            qtx.UpsertGrade(active_version, score, feedback)
            qtx.InsertNotification(student_id, "ASSIGNMENT_GRADED", title, body, link)   ← rendered now (D-53)
        tx.Commit()   (both or neither)

QUIZ TAKE (student)
  POST .../quizzes/:qid/attempts (start) ─▶ window open? attempts remain? prev SUBMITTED/AUTO?
        └─ draw M-of-N, shuffle if configured, INSERT attempt(IN_PROGRESS) + snapshot question set
  GET  .../attempts/:id ─▶ STUDENT-VIEW DTO (no is_correct field exists on the struct)  ← QUIZ-03 control
  POST .../attempts/:id/submit ─▶ idempotent: if already SUBMITTED return stored score
        └─ grade exact-match (single ==, multi set-equality), store score, status=SUBMITTED

QUIZ AUTO-SUBMIT (lazy)
  any read/start touching an attempt where status=IN_PROGRESS AND now > close_at
        └─ grade-as-is, status=AUTO_SUBMITTED, submitted_at=close_at  (idempotent)
```

### Recommended Project Structure
```
backend/internal/
├── assignments/          # handler service repository model dto
├── quizzes/              # handler service repository model dto
├── notifications/        # handler service repository model dto (shared primitive)
└── shared/
    └── cloudinary/       # NewFromURL client wrapper: Upload(raw,authenticated) + SignedDownloadURL
backend/db/
├── migrations/000006_assignments_quizzes_notifications.up.sql (+ .down.sql)
└── queries/
    ├── assignments.sql submissions.sql
    ├── quizzes.sql quiz_questions.sql quiz_attempts.sql
    └── notifications.sql
```

### Pattern 1: Cloudinary authenticated upload + signed download (ASMT-05)
**What:** Store raw asset as authenticated; mint short-lived signed download URL behind an ownership check.
**When to use:** Every submission upload and every download request.
```go
// internal/shared/cloudinary/client.go
// Source: pkg.go.dev/github.com/cloudinary/cloudinary-go/v2 (NewFromURL),
//         .../api/uploader (Upload, PrivateDownloadURL) — verified 2026-06-20
import (
    "github.com/cloudinary/cloudinary-go/v2"
    "github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

func New(cloudinaryURL string) (*cloudinary.Cloudinary, error) {
    return cloudinary.NewFromURL(cloudinaryURL) // reads CLOUDINARY_URL string already in config
}

// Upload: raw + authenticated (NOT image, NOT public)
res, err := cld.Upload.Upload(ctx, fileReader, uploader.UploadParams{
    ResourceType: "raw",          // PDF/ZIP are raw assets
    Type:         "authenticated", // only reachable via signed URL
    Folder:       "submissions",
    // Cloudinary returns res.PublicID and res.Format — persist both
})

// Download: short-lived signed URL, gated AFTER ownership check
exp := time.Now().Add(5 * time.Minute) // sane TTL: 5 min one-shot link
url, err := cld.Upload.PrivateDownloadURL(uploader.PrivateDownloadURLParams{
    PublicID:     submission.CloudinaryPublicID,
    Format:       submission.CloudinaryFormat,   // e.g. "pdf" / "zip"
    DeliveryType: "authenticated",
    ExpiresAt:    &exp,
})
```
**Landmine:** `PrivateDownloadURL` needs `Format` — persist the format Cloudinary returns at upload, don't reconstruct it from the original filename. The TTL default is 1h; **explicitly set 5 min** for tighter exposure. Never store/return the raw delivery URL; always mint fresh per request.

### Pattern 2: Same-transaction grade + notification (NOTIF-02)
**What:** Persist the grade and the student notification atomically.
**When to use:** Assignment grading only (D-55 — NOT quiz grading).
```go
// Mirrors internal/enrollments/service.go (proven pattern in this repo)
// Source: backend/internal/enrollments/service.go:70-76 (pool.Begin + q.WithTx)
tx, err := s.pool.Begin(ctx)
if err != nil { return err }
defer tx.Rollback(ctx) // no-op after Commit

qtx := s.q.WithTx(tx)
if err := qtx.UpsertSubmissionGrade(ctx, gradeParams); err != nil { return err }

// D-53: render title+body NOW, store rendered text
title := "Assignment Graded"
body  := fmt.Sprintf("Your assignment %q has been graded. Score: %s.", asmtTitle, scoreStr)
if err := qtx.InsertNotification(ctx, db.InsertNotificationParams{
    RecipientID:  studentID, Type: "ASSIGNMENT_GRADED",
    Title: title, Body: body,
    ResourceType: pgtype.Text{String: "assignment", Valid: true},
    ResourceID:   pgtype.Int8{Int64: assignmentID, Valid: true},
    Link:         pgtype.Text{String: fmt.Sprintf("/courses/%d/assignments/%d", courseID, assignmentID), Valid: true},
}); err != nil { return err }

return tx.Commit(ctx)
```

### Pattern 3: Student-view DTO that structurally hides correct answers (QUIZ-03)
**What:** A serializer type that has no `is_correct` field, so it is *impossible* to leak it.
**When to use:** Every take-quiz / in-progress / during-window review response.
```go
// model.go (DB row — has the secret)
type QuizOption struct { ID int64; QuestionID int64; Text string; IsCorrect bool }

// dto.go — separate type; no IsCorrect field EXISTS here
type StudentOptionView struct {
    ID   int64  `json:"id"`   // stable option ID (QUIZ-04 mapping key)
    Text string `json:"text"`
}
// During window (D-51): per-question correct/incorrect status of THEIR answer is allowed,
// but the correct OPTION is not — so even the "review" DTO omits which option is correct
// until now() > close_at.
```
**Landmine:** Do NOT reuse the DB struct with `json:"-"` on `IsCorrect` — a future refactor can flip the tag. A distinct struct makes leakage a compile error, not a tag mistake.

### Anti-Patterns to Avoid
- **Trusting `Content-Type` header or file extension for upload validation** → spoofable; sniff bytes server-side AND check extension.
- **Extracting/inspecting ZIP contents server-side** → zip-bomb risk; explicitly out of scope (REQUIREMENTS). Store the ZIP opaque.
- **Storing server data (notifications/attempts) in Zustand on the frontend** → use TanStack Query; Zustand only for auth/UI (CLAUDE.md pitfall).
- **Computing official quiz score on read** → store per-attempt scores; compute `MAX` via SQL `MAX(score)` over completed attempts (D-50) at gradebook read time, but cache the attempt scores at submit.
- **Returning the Cloudinary delivery URL directly** → always proxy through a backend endpoint that ownership-checks then mints a fresh signed URL.
- **`json:"-"` to hide correct answers** → use a separate DTO (Pattern 3).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Magic-byte detection | Custom byte-signature table | `gabriel-vasile/mimetype` (already vendored) | Handles hundreds of signatures + PDF/ZIP edge cases; stdlib `http.DetectContentType` as fallback |
| 10MB enforcement | Manual byte counting | `http.MaxBytesReader` + `ContentLength` check | Stops a lying client from streaming past the cap; stdlib |
| Signed download URLs | HMAC URL signing by hand | `uploader.PrivateDownloadURL` | Cloudinary owns the signing scheme; rolling your own breaks on their format changes |
| Transaction + same-tx writes | Manual SQL `BEGIN`/`COMMIT` strings | `pool.Begin(ctx)` + `q.WithTx(tx)` | Already proven in `enrollments/service.go`; sqlc-native |
| CSV parsing | Custom comma splitter | `encoding/csv` (stdlib) | Handles quoting/escaping; mirrors Phase 3 enrollment import |
| M-of-N shuffle | Custom Fisher-Yates | `math/rand/v2` `Shuffle` / `Perm` | Stdlib; correctness over crypto here (security is the DTO boundary) |

**Key insight:** Two of the three existential pitfalls (upload validation, signed delivery) are *fully solved by libraries already in or trivially added to the dep tree.* The third (answer non-leakage) is solved by a **type-system boundary**, not a library — a separate student-view DTO.

## Runtime State Inventory

> Phase 4 is greenfield feature work (new tables, new code) — not a rename/refactor/migration of existing runtime state. No existing stored data, live-service config, OS-registered state, secrets, or build artifacts carry strings this phase renames.

- **Stored data:** None — new tables only (`assignments`, `submissions`, `quizzes`, `quiz_questions`, `quiz_question_options`, `quiz_attempts`, `quiz_attempt_answers`, `notifications`). Verified: migrations stop at `000005`, no Phase 4 entities exist yet.
- **Live service config:** New — Cloudinary account must exist and `CLOUDINARY_URL` must be set (already a `required` config field — `backend/internal/shared/config/config.go`). Assets created at runtime; nothing to migrate.
- **OS-registered state:** None.
- **Secrets/env vars:** `CLOUDINARY_URL` already loaded as a required env field (Phase 1). No new secret names. Verified in `config.go`.
- **Build artifacts:** None — adding one Go dependency (`cloudinary-go/v2`) via `go get`; `go mod tidy` updates `go.sum`.

## Common Pitfalls

### Pitfall 1: ZIP whitelist silently admits Office documents
**What goes wrong:** `.docx`, `.xlsx`, `.pptx` are ZIP containers and sniff as `application/zip`. A PDF+ZIP whitelist therefore accepts Office files.
**Why it happens:** OOXML is literally a ZIP under the hood; magic bytes `50 4B 03 04` (`PK\x03\x04`) are identical.
**How to avoid:** Accept it as intended behavior (out of scope to inspect ZIP contents per REQUIREMENTS). If strictness is desired later, check the file extension is exactly `.zip`/`.pdf` AND sniff — but per spec, an Office file uploaded as `.zip` is allowed. Document this so reviewers don't flag it as a bug.
**Warning signs:** QA uploads a `.docx` and it succeeds — that is correct given the spec, not a defect.

### Pitfall 2: Late-policy clock skew / client timestamp trust
**What goes wrong:** Using a client-supplied submission time lets students backdate.
**Why it happens:** Forgetting that `submitted_at` must be the server's `now()`.
**How to avoid:** Compute `is_late` and `late_duration` from `now()` in `service.go`; never accept a timestamp from the request body (ASMT-04). Compute the window boundary as `deadline` (block) or `deadline + late_threshold_days` (accept-flag), with NULL threshold meaning "until course soft-delete".
**Warning signs:** Any DTO field named `submitted_at` on the inbound request.

### Pitfall 3: Non-idempotent quiz auto-grade double-scores an attempt
**What goes wrong:** A retried submit (network retry, double-click) re-grades and could mutate the stored score.
**Why it happens:** Grading not guarded by a status transition.
**How to avoid:** Gate on attempt status: only `IN_PROGRESS → SUBMITTED` grades; if already `SUBMITTED`/`AUTO_SUBMITTED`, return the stored score unchanged (QUIZ-05 "idempotent per attempt"). Use a conditional `UPDATE ... WHERE status='IN_PROGRESS'` and check rows-affected.
**Warning signs:** A submit handler that computes score before checking current status.

### Pitfall 4: Resume creates a second attempt (D-52 violation)
**What goes wrong:** Returning to an in-progress quiz starts a new attempt, consuming an extra retake.
**Why it happens:** "start" endpoint always inserts.
**How to avoid:** "start" must first look for an existing `IN_PROGRESS` attempt for (quiz, student) and resume it; only insert when none exists AND prior attempts are terminal AND attempts remain AND window open.
**Warning signs:** No `SELECT ... WHERE status='IN_PROGRESS'` guard before insert.

### Pitfall 5: Window-bound reveal leaks correct answers to early finishers (D-51)
**What goes wrong:** Revealing correct answers right after a student submits lets them share with peers still in the window.
**Why it happens:** Reveal logic keyed to attempt completion instead of `close_at`.
**How to avoid:** Gate correct-answer visibility on `now() > quiz.close_at` server-side, independent of attempt status. The during-window review DTO shows score + own answers + per-question right/wrong, but never the correct option.
**Warning signs:** Reveal condition references `attempt.status` rather than `quiz.close_at`.

## Code Examples

### Magic-byte + size validation (ASMT-03)
```go
// Source: stdlib http.MaxBytesReader + gabriel-vasile/mimetype (vendored v1.4.12)
const maxUpload = 10 << 20 // 10 MiB

func (h *Handler) Submit(c *gin.Context) {
    // 1. cheap pre-check on declared length
    if c.Request.ContentLength > maxUpload {
        c.JSON(413, errorEnvelope("file_too_large", "max 10MB")); return
    }
    // 2. hard cap so a lying client can't stream past it
    c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUpload)

    fh, err := c.FormFile("file")
    if err != nil { c.JSON(400, errorEnvelope("missing_file", "file required")); return }

    f, _ := fh.Open(); defer f.Close()
    head := make([]byte, 512)
    n, _ := io.ReadFull(f, head)
    mtype := mimetype.Detect(head[:n])       // precise MIME + extension
    _, _ = f.Seek(0, io.SeekStart)           // rewind for upload

    ext := strings.ToLower(filepath.Ext(fh.Filename))
    okExt := ext == ".pdf" || ext == ".zip"
    okSniff := mtype.Is("application/pdf") ||
               mtype.Is("application/zip") ||
               mtype.Is("application/x-zip-compressed")
    if !okExt || !okSniff {                   // reject by extension AND sniffed type
        c.JSON(415, errorEnvelope("invalid_file_type", "PDF or ZIP only")); return
    }
    // ... ownership + window check, then cloudinary upload
}
```

### Idempotent quiz submit (QUIZ-05)
```sql
-- queries/quiz_attempts.sql
-- name: MarkAttemptSubmitted :execrows
UPDATE quiz_attempts
SET status = 'SUBMITTED', score = $2, submitted_at = now()
WHERE id = $1 AND status = 'IN_PROGRESS';
-- handler: if rows-affected == 0, attempt already terminal → return stored score, do not regrade
```

### Unread badge count + mark-read (D-54)
```sql
-- name: CountUnread :one
SELECT count(*) FROM notifications WHERE recipient_id = $1 AND read_at IS NULL;
-- name: MarkRead :exec
UPDATE notifications SET read_at = now() WHERE id = $1 AND recipient_id = $2 AND read_at IS NULL;
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `http.DetectContentType` only | `mimetype.Detect` (richer signatures) | ongoing | More precise PDF/ZIP discrimination; already vendored |
| Public Cloudinary uploads + obscure URL | `Type:"authenticated"` + signed `PrivateDownloadURL` | — | Required by ASMT-05; obscurity is not access control |
| `math/rand` (global, seeded) | `math/rand/v2` | Go 1.22+ | Cleaner API; auto-seeded; fine for non-crypto shuffle |

**Deprecated/outdated:**
- `lib/pq`, GORM, public-by-default Cloudinary delivery, client-trusted MIME — all explicitly rejected in CLAUDE.md.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `PrivateDownloadURL` works for `ResourceType:"raw"` authenticated assets (docs/examples mostly show images) | Cloudinary | LOW-MED — if raw needs a different path, fall back to `AuthToken`-signed delivery URL; both documented. Verify with a one-shot integration spike during Wave 1. |
| A2 | 5-minute TTL is an acceptable signed-URL lifetime | Cloudinary | LOW — TTL is a tunable; planner/user may prefer 1-15 min. Not a correctness risk. |
| A3 | Audit-logging Phase 4 lecturer actions stays OFF | Discretion | LOW — CONFIRMED below: keep OFF. `audit_log` is admin-only (ADMIN-08); lecturer create/grade are not admin mutations. No reason found to log. |
| A4 | `Folder` upload param is supported in v2.16.0 UploadParams | Cloudinary | LOW — cosmetic; drop `Folder` if absent. Core upload works without it. |

**Audit-OFF confirmation (A3):** Research found **no reason to flip audit-logging ON** for Phase 4 lecturer actions. The append-only `audit_log` is scoped to admin mutations (ADMIN-08, `000005`), and lecturer create-assignment/create-quiz/grade are course-scoped academic actions, not account/course-provisioning mutations. Keep the discretion default **OFF**. The notification primitive (D-53) already provides the student-facing record of grading events.

## Open Questions (RESOLVED)

1. **Cloudinary `PrivateDownloadURL` for raw authenticated assets — confirm in a live spike.**
   - What we know: method exists in cloudinary-go/v2, takes `PublicID/Format/DeliveryType/ExpiresAt`, returns a time-limited URL.
   - What's unclear: whether raw `authenticated` assets deliver correctly through this exact method vs needing AuthToken-signed URLs (most docs examples are images).
   - Recommendation: Wave 1 first task = a 20-line integration spike that uploads a tiny PDF as raw/authenticated and round-trips a download through `PrivateDownloadURL`. Resolve A1 before building the submission flow on it.
   - **RESOLVED:** retired by the Wave-1 spike in 04-01 Task 1 (env-gated `spike_test.go` round-trips a raw/authenticated PDF through `PrivateDownloadURL`); the plan documents the AuthToken fallback to switch to if `PrivateDownloadURL` fails for raw authenticated assets.

2. **`late_duration` format & storage — computed vs stored.**
   - What we know: D-45 wants a human string ("6 days 12 hours").
   - What's unclear: store the rendered string vs compute on read from `submitted_at − deadline`.
   - Recommendation: store `is_late BOOL` + `submitted_at`; compute `late_duration` on read (single source of truth). Planner's call.
   - **RESOLVED:** resolved to compute-on-read in 04-01 (migration 000006 stores `is_late BOOL` + `submitted_at` only; `late_duration` is computed on read from `submitted_at − deadline` as the single source of truth — no rendered string is persisted).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | all backend | ✓ | 1.24.0 (`go.mod`) | — |
| PostgreSQL (Docker) | all persistence | ✓ (Docker-only per D-08) | postgres:17 | — |
| Cloudinary account + `CLOUDINARY_URL` | ASMT-05 upload/delivery | env-dependent | — | **None — blocking for upload/download tests.** Must be a real Cloudinary account; no local stand-in for authenticated delivery. |
| cloudinary-go/v2 | ASMT-05 | ✓ via `go get` | v2.16.0 | — |
| gabriel-vasile/mimetype | ASMT-03 | ✓ (already indirect) | v1.4.12 | stdlib `http.DetectContentType` |

**Missing dependencies with no fallback:**
- A working `CLOUDINARY_URL` (account credentials) is required for any real upload/download test. Integration tests touching Cloudinary need it in CI secrets, or those tests must be skip-gated like the existing `DATABASE_URL` skip pattern in `lifecycle/sweep_test.go`.

**Missing dependencies with fallback:**
- mimetype → stdlib `http.DetectContentType` if ever removed (won't be; it's a Gin transitive dep).

## Validation Architecture

> `workflow.nyquist_validation` is **false** in `.planning/config.json` — section omitted per instructions.

## Security Domain

> `security_enforcement: true`, `security_asvs_level: 1`, `block_on: high`. This phase carries 3 existential security pitfalls — security is central, not optional.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | Existing JWT spine (Phase 2); all routes behind `AuthMiddleware` |
| V3 Session Management | yes (inherited) | Cookie JWT + refresh from Phase 2; no new session surface |
| V4 Access Control | **yes (critical)** | `RequireRole` + membership-derived ownership (`course_lecturers`/`student_enrollments`); `user_id` from JWT only; signed-download URL minted ONLY after ownership check; quiz answer-key gated by `close_at` |
| V5 Input Validation | **yes (critical)** | Magic-byte + extension + size validation on upload; Zod (FE) mirrored by Go server-side validation; CSV row validation |
| V6 Cryptography | yes (delegated) | Cloudinary owns signed-URL crypto via `PrivateDownloadURL` — never hand-roll URL signing; bcrypt (passwords) unchanged |
| V12 File & Resources | **yes (critical)** | PDF/ZIP whitelist, 10MB `MaxBytesReader`, no server-side ZIP extraction, authenticated (non-public) storage |

### Known Threat Patterns for Go/Gin + Cloudinary + Postgres

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malicious/oversized upload (zip bomb, huge file) | Denial of Service | `MaxBytesReader` 10MB cap; never unpack ZIP server-side |
| Content-type spoofing (exe renamed `.pdf`) | Tampering | Magic-byte sniff (`mimetype.Detect`) AND extension whitelist |
| Public asset URL leak | Information Disclosure | `Type:"authenticated"` storage; short-lived signed `PrivateDownloadURL` only |
| IDOR on submission download / cross-course access | Elevation of Privilege | Ownership check (owning student or assigned lecturer) BEFORE minting URL; `user_id` from JWT |
| Quiz answer-key leakage via API | Information Disclosure | Separate student-view DTO with no `is_correct` field; reveal gated on `close_at` (D-51) |
| Backdated late submission | Tampering | Server `now()` only; reject any client-supplied timestamp |
| Double-submit re-grade / score tamper | Tampering | Idempotent status-guarded `UPDATE ... WHERE status='IN_PROGRESS'` |
| SQL injection | Tampering | sqlc parameterized queries only (no string-built SQL) |

## Sources

### Primary (HIGH confidence)
- `go list -m -versions github.com/cloudinary/cloudinary-go/v2` → latest **v2.16.0** (Go module proxy, authoritative for Go) — VERIFIED this session
- pkg.go.dev/github.com/cloudinary/cloudinary-go/v2 — `NewFromURL`, client construction — VERIFIED
- pkg.go.dev/github.com/cloudinary/cloudinary-go/v2/api/uploader — `UploadParams{ResourceType, Type(DeliveryType), AccessMode}`, `Upload(ctx,file,params)`, `PrivateDownloadURL(params)` + `PrivateDownloadURLParams{PublicID,Format,DeliveryType,Attachment,ExpiresAt,ResourceType}` — VERIFIED
- `backend/go.mod` — Go 1.24.0, gin v1.11.0, pgx v5.7.2, mimetype v1.4.12 (indirect) — VERIFIED
- `backend/internal/enrollments/service.go` — `pool.Begin(ctx)` + `q.WithTx(tx)` transaction pattern — VERIFIED (in-repo)
- `backend/cmd/api/main.go` + `backend/internal/lifecycle/sweep.go` — in-process scheduler/ticker pattern (D-37) — VERIFIED (in-repo)
- `backend/db/migrations/000004_admin_schema.up.sql` — `courses`/`student_enrollments`/`course_lecturers` FK targets — VERIFIED (in-repo)

### Secondary (MEDIUM confidence)
- cloudinary.com/documentation/control_access_to_media — authenticated vs private vs upload delivery types; signed-URL access control — CITED
- cloudinary.com/documentation/upload_parameters — `resource_type`, `type`, `access_mode` semantics — CITED
- cloudinary.com/documentation/go_integration — Go SDK overview — CITED

### Tertiary (LOW confidence)
- WebSearch summaries on `PrivateDownloadURL` raw-asset behavior (examples skew to images) — drives Open Question 1 / Assumption A1; resolve via Wave-1 spike

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every lib pinned in CLAUDE.md + present in go.mod; cloudinary-go version confirmed via Go module proxy
- Architecture: HIGH — schema/patterns grounded in existing Phase 1-3 code; transaction pattern already proven in-repo
- Cloudinary API surface: HIGH — exact structs/methods verified on pkg.go.dev
- Cloudinary raw-authenticated delivery behavior: MEDIUM — needs a Wave-1 integration spike (A1/OQ1)
- Pitfalls: HIGH — derived from locked decisions + REQUIREMENTS out-of-scope + ASVS

**Research date:** 2026-06-20
**Valid until:** 2026-07-20 (stable stack; re-check cloudinary-go version only if upgrading)
