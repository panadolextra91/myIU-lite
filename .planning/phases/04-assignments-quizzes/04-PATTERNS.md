# Phase 04: Assignments & Quizzes - Pattern Map

**Mapped:** 2026-06-20
**Files analyzed:** 26 (3 backend feature folders × 5 files = 15, +1 shared client, +1 migration pair, +6 sqlc query files, +wiring, +frontend pages)
**Analogs found:** strong analogs for all backend feature files; from-scratch flagged for Cloudinary client, multipart upload handler, quiz state machine

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `backend/internal/assignments/handler.go` | handler | request-response + file-I/O (upload) | `internal/enrollments/handler.go` (multipart) + `internal/auth/handler.go` (wiring) | role-match (no PDF/ZIP analog) |
| `backend/internal/assignments/service.go` | service | CRUD + transactional (grade+notify) | `internal/enrollments/service.go` | exact (same-tx pattern) |
| `backend/internal/assignments/repository.go` | repository | CRUD | `internal/courses/repository.go` | exact |
| `backend/internal/assignments/model.go` | model | — | (sqlc-generated rows in `internal/shared/db`) | exact |
| `backend/internal/assignments/dto.go` | dto | — | `internal/courses/dto.go` (+ `errorEnvelope`) | exact |
| `backend/internal/quizzes/handler.go` | handler | request-response | `internal/auth/handler.go` | role-match |
| `backend/internal/quizzes/service.go` | service | event-driven (state machine) + CRUD | `internal/enrollments/service.go` (tx) | partial (state machine = new) |
| `backend/internal/quizzes/repository.go` | repository | CRUD | `internal/courses/repository.go` | exact |
| `backend/internal/quizzes/model.go` | model | — | sqlc rows + **separate student-view DTO** | partial (DTO boundary = new) |
| `backend/internal/quizzes/dto.go` | dto | — | `internal/courses/dto.go` | role-match (StudentOptionView = new) |
| `backend/internal/notifications/handler.go` | handler | request-response (list/count/mark-read) | `internal/courses/handler.go` (list/paginate) | exact |
| `backend/internal/notifications/service.go` | service | CRUD | `internal/courses/` service style | exact |
| `backend/internal/notifications/repository.go` | repository | CRUD | `internal/courses/repository.go` | exact |
| `backend/internal/notifications/model.go` | model | — | sqlc rows | exact |
| `backend/internal/notifications/dto.go` | dto | — | `internal/courses/dto.go` | exact |
| `backend/internal/shared/cloudinary/client.go` | shared-client | file-I/O (external SDK) | **NONE** | no analog (from-scratch, RESEARCH Pattern 1) |
| `backend/db/migrations/000006_*.up.sql` / `.down.sql` | migration | — | `000004_admin_schema.up/down.sql` | exact |
| `backend/db/queries/{assignments,submissions,quizzes,quiz_questions,quiz_attempts,notifications}.sql` | sqlc-query | CRUD | `db/queries/courses.sql` | exact |
| `backend/cmd/api/main.go` (modified) | wiring | — | existing `RegisterRoutes` calls | exact |
| `frontend/src/lib/coursework-api.ts` (new) | api-client | request-response | `frontend/src/lib/admin-api.ts` | exact |
| `frontend/src/pages/lecturer|student/*` (assignment/quiz pages) | React-page | request-response | `frontend/src/pages/admin/Courses.tsx` | exact |
| `frontend/src/components/NotificationBell.tsx` (new) | React-component | polling read | header in `AppLayout.tsx` (mount point) | partial (bell = new) |
| `frontend/src/routes/router.tsx` (modified) | route | — | existing role subtrees | exact |

## Pattern Assignments

### `backend/internal/*/handler.go` (all three features) — wiring + RBAC

**Analog:** `backend/internal/auth/handler.go` lines 15-41 and `internal/enrollments/handler.go` lines 21-35.

Mirror the exact constructor + `RegisterRoutes(r *gin.Engine, pool *pgxpool.Pool, cfg config.Config)` signature, build `q := db.New(pool)` → `repo := NewRepository(q)` → `svc := NewService(...)` → `h := &Handler{svc: svc}`, then a route group guarded by middleware. Coursework groups register behind both middleware functions exactly as enrollments does (`internal/enrollments/handler.go:27-28`):

```go
g := r.Group("/courses/:id/assignments")
g.Use(middleware.AuthMiddleware(pool, cfg), middleware.RequireRole(db.UserRoleLecturer))
```

Use `db.UserRoleLecturer` for authoring/grading routes, `db.UserRoleStudent` for submit/take routes. `actorID := c.GetInt64("user_id")` (enrollments/handler.go:68) — never read user/student IDs from the body.

**Error envelope** — copy `errorEnvelope` verbatim into each feature's `dto.go` (it is duplicated per-feature today, see `courses/dto.go:42-46`, `enrollments/dto.go:3`). Always shape errors as `c.JSON(status, errorEnvelope("code", "message"))`.

---

### `backend/internal/assignments/handler.go` — file upload (ASMT-03)

**Analog (partial):** `internal/enrollments/handler.go:45-66` shows the `MaxBytesReader` + `c.Request.FormFile("file")` + extension-check shape — but it only checks extension/Content-Type for CSV. The PDF/ZIP **magic-byte** path has NO analog; build it from RESEARCH "Code Examples → Magic-byte + size validation" (04-RESEARCH.md lines 345-377). Reuse this proven scaffold from enrollments:

```go
c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20) // 10MiB (enrollments uses 5<<20)
file, header, err := c.Request.FormFile("file")
if err != nil { c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_FILE", "...")); return }
defer file.Close()
ext := strings.ToLower(filepath.Ext(header.Filename))
```

Then add the NEW magic-byte block (`mimetype.Detect(head[:n])` on first 512B, reject by extension AND sniff). Flagged below as no-analog.

---

### `backend/internal/assignments/service.go` — grade + notify in ONE transaction (NOTIF-02, D-55)

**Analog:** `backend/internal/enrollments/service.go:70-117` — the PROVEN `pool.Begin` / `q.WithTx(tx)` pattern. Copy this exact skeleton:

```go
tx, err := s.pool.Begin(ctx)
if err != nil { return err }
defer func() { _ = tx.Rollback(ctx) }()   // note the deferred-ignore form (errcheck-clean)
qtx := s.q.WithTx(tx)
// ... qtx.UpsertSubmissionGrade(...) then qtx.InsertNotification(...) ...
return tx.Commit(ctx)
```

Service struct mirrors `enrollments/service.go:12-20`: `{ pool *pgxpool.Pool; repo *Repository; q *db.Queries }`, `NewService(pool, repo)` sets `q: db.New(pool)`. The notification title/body are rendered NOW and passed as plain strings (D-53) — see 04-RESEARCH.md lines 248-257.

**Do NOT** call `auditlogs.WriteAudit` here — Phase 4 lecturer actions are audit-OFF (CONTEXT discretion, RESEARCH A3). The enrollments analog calls it; the assignments service deliberately omits it.

---

### `backend/internal/*/repository.go` — thin sqlc passthrough

**Analog:** `backend/internal/courses/repository.go` (whole file, 47 lines). Mirror exactly: `type Repository struct { q *db.Queries }`, `NewRepository(q *db.Queries) *Repository`, one method per sqlc query delegating to `r.q.Xxx(ctx, arg)`. No business logic. The `WithTx` transactional variant uses `NewRepository(qtx)` (enrollments/service.go:77).

---

### `backend/internal/*/dto.go` — request/response shapes

**Analog:** `backend/internal/courses/dto.go` (whole file). Request structs use `json` + `binding:"required"` tags (lines 5-19); response structs use `json` tags with `time.Time` for timestamps; paginated lists use `{ Data []T; Total int64 }` (lines 31-34). Copy `errorEnvelope` (lines 42-46) into each new `dto.go`.

---

### `backend/internal/quizzes/dto.go` + `model.go` — answer non-leakage (QUIZ-03, D-51)

**Analog:** none for the security boundary; use RESEARCH Pattern 3 (04-RESEARCH.md lines 262-278). The DB-row struct lives with `IsCorrect bool`; the take-quiz serializer is a **separate** `StudentOptionView` struct with NO `IsCorrect` field. Do NOT use `json:"-"`. The reveal of correct answers is gated server-side on `now() > quiz.close_at`, independent of attempt status. Flagged below as no-analog.

---

### `backend/db/queries/*.sql` — sqlc inputs

**Analog:** `backend/db/queries/courses.sql` (whole file). Mirror the `-- name: X :one|:many|:exec|:execrows` annotation style. Soft-delete-aware reads filter active courses (`WHERE ... AND deleted_at IS NULL`, courses.sql:7). Membership reads to derive scope/recipients: copy `ListCourseStudents` / `ListCourseLecturers` (courses.sql:31-41) for notification recipients and ownership checks. Use `:execrows` for the idempotent guarded update (RESEARCH lines 380-387) and the mark-read query (RESEARCH lines 389-395). `sqlc.yaml` already targets `sql_package: "pgx/v5"`, schema `db/migrations`, out `internal/shared/db` — no config change.

---

### `backend/db/migrations/000006_*.up.sql` / `.down.sql`

**Analog:** `backend/db/migrations/000004_admin_schema.up.sql` (+ `.down.sql`). Mirror table conventions exactly:
- PK: `id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY` (000004:15)
- FKs: `course_id BIGINT NOT NULL REFERENCES courses(id)`, `student_id/recipient_id BIGINT NOT NULL REFERENCES users(id)` (000004:28-29)
- Timestamps: `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`; soft-delete dependents inherit hiding via their active-course gate (no own `deleted_at` needed per D-40 no-cascade)
- Uniqueness: `UNIQUE (course_id, student_id)` style for natural keys (000004:31)
- `.down.sql` drops tables in reverse-FK order with `DROP TABLE IF EXISTS` (000004 down).

8 tables per CONTEXT D-10: `assignments`, `submissions`, `quizzes`, `quiz_questions`, `quiz_question_options`, `quiz_attempts`, `quiz_attempt_answers`, `notifications`. Stable option IDs (QUIZ-04) = the `quiz_question_options.id` identity column.

---

### `backend/cmd/api/main.go` (modified) — route registration

**Analog:** `backend/cmd/api/main.go:42-46`. Add three lines mirroring the existing calls:

```go
assignments.RegisterRoutes(router, pool, cfg)
quizzes.RegisterRoutes(router, pool, cfg)
notifications.RegisterRoutes(router, pool, cfg)
```

The cloudinary client should be constructed once from `cfg.CloudinaryURL` (config.go:11, already loaded/required) and injected into the assignments `RegisterRoutes` — this is the one deviation from the parameter-less `RegisterRoutes(r, pool, cfg)` convention (the client is the new dependency). Lazy quiz AUTO_SUBMIT needs no scheduler (RESEARCH chooses lazy eval); the existing `lifecycle.StartSweeper` (main.go:48-53) is the optional belt-and-suspenders model if a sweep is ever wanted.

---

### Frontend pages — `frontend/src/pages/{lecturer,student}/*`

**Analog:** `frontend/src/pages/admin/Courses.tsx` (whole file, 300 lines) is the canonical feature-page pattern. Mirror:
- TanStack Query for reads (`useQuery({ queryKey, queryFn })`, lines 39-47) and mutations (`useMutation` with `onSuccess` → `queryClient.invalidateQueries` + `toast.success`, `onError` → `isAxiosError` → `toast.error`, lines 54-69)
- RHF + Zod via `zodResolver` (lines 49-52) with shadcn `<Form>/<FormField>` (lines 141-199) — coursework create/grade/quiz-author forms follow this exactly
- shadcn-only components (Table, Dialog, AlertDialog) — no hand-rolled UI (DESIGN-SYSTEM D-05)
- Server data NEVER in Zustand (CLAUDE.md pitfall) — TanStack Query owns notifications/attempts/submissions cache.

File upload form: copy `adminApi.importAccounts` FormData pattern (`admin-api.ts:42-50`) — `new FormData(); fd.append('file', file)` + `headers: {'Content-Type':'multipart/form-data'}`.

---

### `frontend/src/lib/coursework-api.ts` (new)

**Analog:** `frontend/src/lib/admin-api.ts` (lines 1-50). Single exported object of async methods wrapping `api.get/post`; TS interfaces for request/response; shares the `api` axios instance (cookies + 401/403 interceptor already handled in `lib/api.ts`).

---

### `frontend/src/components/NotificationBell.tsx` (new) + router

**Mount point:** `frontend/src/components/AppLayout.tsx:30-40` — the header's right cluster (next to username/Logout). The bell is a NEW component (no analog) but uses the established stack: TanStack Query poll for `CountUnread`, Lucide `Bell` icon (already importing Lucide, AppLayout.tsx:4), shadcn badge. Mark-read-on-click invalidates the count query. Router: add notification list + coursework routes to the existing role subtrees in `router.tsx:43-69` (same `{ path, element }` child shape).

## Shared Patterns

### Authentication / RBAC
**Source:** `backend/internal/shared/middleware/auth.go` (AuthMiddleware) + `role.go:10-34` (RequireRole)
**Apply to:** every Phase 4 route group.
```go
g.Use(middleware.AuthMiddleware(pool, cfg), middleware.RequireRole(db.UserRoleLecturer))
// inside handler: actorID := c.GetInt64("user_id")  // JWT-derived, never from body
```
Ownership beyond role (lecturer-assigned-to-course / student-enrolled) is enforced in `service.go` via the membership queries (`ListCourseLecturers`/`ListCourseStudents`), not in middleware.

### Same-transaction mutation+notification
**Source:** `backend/internal/enrollments/service.go:70-117`
**Apply to:** assignment grading only (D-55).
```go
tx, err := s.pool.Begin(ctx); if err != nil { return err }
defer func() { _ = tx.Rollback(ctx) }()
qtx := s.q.WithTx(tx)
// guarded mutation + InsertNotification on qtx
return tx.Commit(ctx)
```

### Error envelope
**Source:** `backend/internal/courses/dto.go:42-46` (duplicated per feature)
**Apply to:** all handlers.
```go
func errorEnvelope(code, message string) map[string]interface{} {
    return map[string]interface{}{"error": map[string]interface{}{"code": code, "message": message}}
}
```

### Soft-delete-aware reads
**Source:** `backend/db/queries/courses.sql:7` (`AND deleted_at IS NULL`)
**Apply to:** all coursework reads scoped by course — gate on the active course (D-29/D-40).

### Idempotent guarded UPDATE
**Source:** RESEARCH lines 380-387 (no in-repo analog; `:execrows` annotation exists in sqlc)
**Apply to:** quiz submit (`UPDATE ... WHERE status='IN_PROGRESS'`, check rows-affected) and `MarkRead`.

## No Analog Found

Files/patterns with no close codebase match — planner must design from RESEARCH.md:

| File / Pattern | Role | Data Flow | Reason | Grounding |
|----------------|------|-----------|--------|-----------|
| `backend/internal/shared/cloudinary/client.go` | shared-client | external file-I/O | First Cloudinary integration; no SDK wrapper exists | 04-RESEARCH.md Pattern 1 (lines 200-233): `NewFromURL`, `Upload(raw,authenticated)`, `PrivateDownloadURL` w/ 5min `ExpiresAt`. Wave-1 spike needed (OQ1/A1). |
| Magic-byte upload block in `assignments/handler.go` | handler | file-I/O validation | enrollments only sniffs CSV by extension; no PDF/ZIP magic-byte path | 04-RESEARCH.md Code Examples (lines 345-377): `mimetype.Detect` first 512B + extension∩sniff whitelist; reject 415. |
| `quizzes` student-view DTO (`StudentOptionView`) | dto | serialization | No existing answer-key-safety boundary | 04-RESEARCH.md Pattern 3 (lines 262-278): separate struct with no `IsCorrect`; reveal gated on `close_at`. |
| Quiz attempt state machine in `quizzes/service.go` | service | event-driven | No state-machine analog (IN_PROGRESS/SUBMITTED/AUTO_SUBMITTED, consume-on-start, resume, lazy auto-submit) | 04-RESEARCH.md D-52 + Pitfalls 3-5 (lines 325-341); M-of-N draw via `math/rand/v2`. |
| `NotificationBell.tsx` | React-component | polling read | No header widget with badge exists | DESIGN-SYSTEM D-05 + AppLayout header mount (AppLayout.tsx:30-40); TanStack Query `CountUnread` poll. |

## Metadata

**Analog search scope:** `backend/internal/{auth,enrollments,courses,auditlogs,lifecycle,shared}`, `backend/db/{queries,migrations}`, `backend/cmd/api`, `frontend/src/{lib,stores,routes,components,pages}`
**Files scanned:** ~18 read in full + directory listings
**Pattern extraction date:** 2026-06-20
