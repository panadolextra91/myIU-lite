# Phase 5: Gradebook, Announcements & Requests - Pattern Map

**Mapped:** 2026-06-20
**Files analyzed:** 30 (3 backend feature folders × 5 files + 3 query files + 1 migration up/down + assignments touch + 6 frontend pages/api/router)
**Analogs found:** 30 / 30 (every new file has a strong in-repo analog — Phase 5 is ~80% reuse)

All backend code uses the module path `github.com/panadolextra91/myiu-lite/backend/...`, the `errorEnvelope(code, message)` → `{"error":{"code","message"}}` envelope, ownership-from-JWT via `c.GetInt64("user_id")` / `c.GetString("role")`, and the Feature-Oriented Monolith handler/service/repository split. Mirror these everywhere.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `backend/internal/grades/handler.go` | handler | request-response + CSV | `backend/internal/assignments/handler.go` + `enrollments/handler.go` (CSV) | exact |
| `backend/internal/grades/service.go` | service | CRUD + compute + same-tx notify | `backend/internal/assignments/service.go` (notify) + `enrollments/service.go` (CSV tx) | exact |
| `backend/internal/grades/repository.go` | repository | CRUD | `backend/internal/assignments/repository.go` | exact |
| `backend/internal/grades/model.go` | model | — | `backend/internal/enrollments/model.go` (errors + RowError) | exact |
| `backend/internal/grades/dto.go` | dto | — | `backend/internal/assignments/dto.go` (+ `errorEnvelope`) | exact |
| `backend/internal/grades/{compute,publish,csv}_test.go` | test | integration | `backend/internal/enrollments/import_test.go` | exact |
| `backend/internal/announcements/handler.go` | handler | request-response | `backend/internal/assignments/handler.go` | exact |
| `backend/internal/announcements/service.go` | service | CRUD + fan-out notify | `backend/internal/assignments/service.go:202` + `enrollments/service.go` loop-in-tx | exact |
| `backend/internal/announcements/repository.go` | repository | CRUD | `backend/internal/assignments/repository.go` | exact |
| `backend/internal/announcements/{model,dto}.go` | model/dto | — | `enrollments/model.go` + `assignments/dto.go` | exact |
| `backend/internal/announcements/fanout_test.go` | test | integration | `enrollments/import_test.go` | exact |
| `backend/internal/requests/handler.go` | handler | request-response | `backend/internal/assignments/handler.go` | exact |
| `backend/internal/requests/service.go` | service | CRUD + same-tx notify (both directions) | `backend/internal/assignments/service.go:202` | exact |
| `backend/internal/requests/repository.go` | repository | CRUD | `backend/internal/assignments/repository.go` | exact |
| `backend/internal/requests/{model,dto}.go` | model/dto | — | `enrollments/model.go` + `assignments/dto.go` | exact |
| `backend/internal/requests/request_test.go` | test | integration | `enrollments/import_test.go` | exact |
| `backend/db/migrations/000008_*.up.sql` / `.down.sql` | migration | DDL | `backend/db/migrations/000006_*.up.sql` / `.down.sql` | exact |
| `backend/db/queries/grades.sql` | query | CRUD + aggregate reads | `assignments.sql` + `courses.sql` (`ListCourseStudents`) + RESEARCH compute SQL | role-match |
| `backend/db/queries/announcements.sql` | query | CRUD | `assignments.sql` + `notifications.sql` (`InsertNotification`) | exact |
| `backend/db/queries/requests.sql` | query | CRUD | `assignments.sql` | exact |
| `backend/db/queries/assignments.sql` (modify) | query | CRUD | self — add `max_score`/`grading_finalized_at` to `CreateAssignment` + new `FinalizeAssignmentGrading` | self |
| `backend/internal/assignments/{dto,service,handler}.go` (modify) | mixed | — | self — carry `max_score`, add finalize action | self |
| `backend/cmd/api/main.go` (modify) | config | wiring | `main.go:50-58` RegisterRoutes block | self |
| `frontend/src/lib/coursework-api.ts` (modify, or new `grades-api.ts`) | service | request-response | `coursework-api.ts:110-189` | exact |
| `frontend/src/pages/student/Grades.tsx` | component | request-response | `frontend/src/pages/Notifications.tsx` (read-only list) | role-match |
| `frontend/src/pages/lecturer/Gradebook.tsx` | component | CRUD + form + CSV | `frontend/src/pages/lecturer/Assignments.tsx` | exact |
| `frontend/src/pages/.../Announcements.tsx` | component | CRUD + form | `frontend/src/pages/lecturer/Assignments.tsx` (compose) + `Notifications.tsx` (browse) | exact |
| `frontend/src/pages/lecturer/RequestInbox.tsx` | component | list + reply form | `frontend/src/pages/lecturer/Assignments.tsx` (table + dialog form) | exact |
| `frontend/src/pages/student/Requests.tsx` | component | compose form | `frontend/src/pages/lecturer/Assignments.tsx` (create form) | exact |
| `frontend/src/routes/router.tsx` (modify) | route | — | self — add routes under existing role trees | self |

## Pattern Assignments

### `backend/internal/<feature>/handler.go` (handler, request-response)

**Analog:** `backend/internal/assignments/handler.go`

**RegisterRoutes wiring** (lines 23-48) — constructs repo→service→handler, mounts auth + role groups. `grades`/`requests` take `(r, pool, cfg)` like `enrollments`; only `assignments` takes the extra `cld` (no Cloudinary this phase — grade CSV is in-memory, see RESEARCH §Environment). Copy this shape:
```go
func RegisterRoutes(r *gin.Engine, pool *pgxpool.Pool, cfg config.Config) {
    repo := NewRepository(db.New(pool))
    service := NewService(pool, repo)
    handler := &Handler{service: service}
    api := r.Group("/api"); api.Use(middleware.AuthMiddleware(pool, cfg))
    lecturer := api.Group("/lecturer"); lecturer.Use(middleware.RequireRole(db.UserRoleLecturer))
    { /* lecturer.POST("/courses/:id/grade-scheme", ...) etc */ }
    student := api.Group("/student"); student.Use(middleware.RequireRole(db.UserRoleStudent))
    { /* student.GET("/courses/:id/grades", ...) etc */ }
}
```

**Param parse + error mapping** (lines 54-79) — every handler: `strconv.ParseInt(c.Param("id"), 10, 64)` → `errorEnvelope("invalid_id", ...)` on fail; `ShouldBindJSON(&req)` → `errorEnvelope("invalid_request", ...)`; pull actor with `c.GetInt64("user_id")` / `c.GetString("role")`; map sentinel errors to status codes (`ErrForbidden`→403, `ErrNotFound`→404, else 500). List responses wrap in `gin.H{"data": res}`; creates return `http.StatusCreated`.

**CSV handler** (grades MANUAL import) — copy `enrollments/handler.go:45-89` `handleImport`: `http.MaxBytesReader(c.Writer, c.Request.Body, 5<<20)`, `c.Request.FormFile("file")`, extension/content-type check (`.csv`/`text/csv`/`application/vnd.ms-excel`), then on `errors.Is(err, ErrValidation)` emit **`c.JSON(http.StatusUnprocessableEntity, gin.H{"errors": rowErrs})`**. Component id comes from the URL/body, never the CSV (D-67).

---

### `backend/internal/<feature>/service.go` (service, CRUD + compute + notify)

**Analog:** `backend/internal/assignments/service.go`

**Struct + constructor** (lines 23-32): `Service{ pool *pgxpool.Pool; repo *Repository; q *db.Queries }`, `NewService(pool, repo)` sets `q: db.New(pool)`. Sentinel errors declared at top (`ErrForbidden`, `ErrNotFound`, add `ErrValidation`, `ErrSchemeExists`, `ErrSchemeImmutable`, etc).

**Ownership/authz** (lines 73-78): gate every mutation with `authz.AssertCourseMember(ctx, s.pool, courseID, userID, db.UserRoleLecturer)` (or `...Student`). For requests, after membership also filter `targeted_lecturer_id == JWT user_id` (D-62). Never trust client-supplied course/student IDs.

**Same-transaction mutation + notify (NOTIF-02)** (lines 202-253) — THE canonical pattern for grade-publish, republish, announcement fan-out, request-create, request-reply:
```go
tx, err := s.pool.Begin(ctx)
if err != nil { return err }
defer func() { _ = tx.Rollback(ctx) }()
qtx := s.q.WithTx(tx)
// ...mutation via qtx (upsert snapshot / insert announcement / insert/update request)...
_, err = qtx.InsertNotification(ctx, db.InsertNotificationParams{
    RecipientID:  recipientID,
    Type:         "GRADE_PUBLISHED",                 // planner picks enum string (A6)
    Title:        "Grades available",
    Body:         fmt.Sprintf("Your %s grade is available.", componentName),
    ResourceType: pgtype.Text{String: "course", Valid: true},
    ResourceID:   pgtype.Int8{Int64: courseID, Valid: true},
    Link:         pgtype.Text{String: fmt.Sprintf("/courses/%d/grades", courseID), Valid: true},
})
if err != nil { return err }
return tx.Commit(ctx)
```
**Fan-out variant** (publish to N students, announcement to roster): resolve recipients via `s.q.ListCourseStudents(ctx, courseID)`, then loop `qtx.Insert...` + `qtx.InsertNotification` **inside the one tx** — mirrors `enrollments/service.go:80-101`. All-or-nothing.

**Numeric handling** (lines 220-223): `var num pgtype.Numeric; num.Scan(fmt.Sprintf("%f", score))` for writes; clamp `0 ≤ score ≤ 100` (lines 210-214). Do compute aggregates in Go `float64`, store snapshot as NUMERIC (RESEARCH Pitfall 4).

**CSV all-or-nothing tx** (grades) — copy `enrollments/service.go:22-119`: ParseCSV → if `len(rowErrs)>0` return `ErrValidation` (no tx opened); validate enrolled+`0≤score≤100` appending to `rowErrs`; re-check; only then `pool.Begin` → loop upserts → `Commit`.

---

### `backend/internal/<feature>/repository.go` (repository, CRUD)

**Analog:** `backend/internal/assignments/repository.go` — thin pass-throughs to `r.q.<Query>(ctx, arg)`. `Repository{ q *db.Queries }`, `NewRepository(q *db.Queries)`. One method per sqlc query. No business logic, no tx (tx lives in service via `s.q.WithTx(tx)`).

### `backend/internal/<feature>/model.go` (errors + RowError)

**Analog:** `backend/internal/enrollments/model.go` — sentinel `errors.New(...)` block + `RowError{Row int; Field string; Message string}` (json-tagged) for the grade CSV 422 list. Reuse `RowError` shape verbatim in `grades`.

### `backend/internal/<feature>/dto.go`

**Analog:** `backend/internal/assignments/dto.go` — request/response structs with `binding:"required"` on required fields; **copy `errorEnvelope(code, message)`** (lines 36-40) into each feature package (it is per-package, not shared).

### `backend/internal/<feature>/*_test.go` (integration)

**Analog:** `backend/internal/enrollments/import_test.go` — `t.Skip` if `DATABASE_URL` unset; `pgxpool.New(ctx, dbURL)`; inline `INSERT ... RETURNING id` seeds for courses/users/quizzes/assignments; `defer pool.Exec(DELETE...)` cleanup; `testify` `require`/`assert`. **Anti-theater (project memory):** red-when-reverted, real seeded fixtures, assert the atomic rollback property (force mid-loop failure → assert ZERO publications AND ZERO notifications), not just happy path. See RESEARCH §Validation Architecture for the per-requirement test specs.

---

### `backend/db/queries/grades.sql` / `announcements.sql` / `requests.sql`

**Analog:** `backend/db/queries/assignments.sql` (`-- name: X :one|:many|:exec`, `RETURNING *`, courses-deleted-filter `JOIN courses c ON ... WHERE c.deleted_at IS NULL`), `notifications.sql` (`InsertNotification` reused verbatim — do NOT redefine), `courses.sql` `ListCourseStudents`/`ListCourseLecturers` (roster fan-out + lecturer selection — reuse, don't rebuild).

- **grades.sql** new queries: `CreateGradeScheme`/`InsertGradeComponent`/`ListSchemeComponents`/`UpsertGradeScore`/`UpsertGradePublication`/`ListPublicationsForStudent`; the AUTO compute `ComputeQuizAverage`/`ComputeAssignmentAverage` are spelled out in RESEARCH §Code Examples (mirror `GetMaxScore` at `quiz_attempts.sql:30` for the MAX-over-SUBMITTED inner select). Defensive eligibility filters: `q.close_at IS NOT NULL AND q.close_at <= now() AND q.max_grade > 0`; `a.grading_finalized_at IS NOT NULL AND a.max_score > 0` (RESEARCH Pitfall 1/2).
- **announcements.sql**: `InsertAnnouncement` (no `updated_at` — D-61), `InsertAnnouncementRecipient`, `ListCourseAnnouncements`, `GetAnnouncementByID`.
- **requests.sql**: `InsertRequest`, `ReplyRequest` (status PENDING→APPROVED/DENIED + note), `ListLecturerRequests` (filter `targeted_lecturer_id`), `ListStudentRequests`, `GetRequestByID`.

### `backend/db/migrations/000008_*.up.sql` / `.down.sql`

**Analog:** `backend/db/migrations/000006_assignments_quizzes_notifications.up.sql` (raw `CREATE TABLE ... BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY`, `REFERENCES`, `TIMESTAMPTZ NOT NULL DEFAULT now()`, explicit indexes) and its `.down.sql` (DROP in reverse-dependency order, `IF EXISTS`). New tables per RESEARCH §Patterns 1-2: `grade_schemes` (UNIQUE course_id), `grade_components` (self-ref `parent_id`), `grade_scores` (UNIQUE component_id,student_id), `grade_publications` (UNIQUE component_id,student_id), `announcements`, `announcement_recipients`, `requests`. **Plus the Phase-4 touch:** `ALTER TABLE assignments ADD COLUMN max_score NUMERIC NOT NULL DEFAULT 100;` and `ADD COLUMN grading_finalized_at TIMESTAMPTZ NULL;` (down: `DROP COLUMN`). Sum-to-100 validated in Go service, NOT a DB trigger (RESEARCH Anti-Patterns).

### Phase-4 `assignments` modifications (self-analog + impact mandate)

`assignments.sql` `CreateAssignment` (add `max_score` param), `dto.go` `CreateAssignmentRequest`/`AssignmentResponse` + FE `coursework-api.ts` `CreateAssignmentRequest`, `handler.go`/`service.go` `CreateAssignment` to carry `max_score`; add a `FinalizeAssignmentGrading` lecturer action + query (sets `grading_finalized_at`). **CLAUDE.md mandate: run `impact({target:"CreateAssignment", direction:"upstream"})` before editing** (FE form + handler + service + query + `security_test.go` all touch it).

### `backend/cmd/api/main.go` (wiring)

**Analog:** `main.go:50-58` — append `grades.RegisterRoutes(router, pool, cfg)`, `announcements.RegisterRoutes(router, pool, cfg)`, `requests.RegisterRoutes(router, pool, cfg)` after `quizzes`. (No `cld` arg — no uploads.)

---

### Frontend pages

**API layer** — `frontend/src/lib/coursework-api.ts:110-189`: typed `interface`s + a `const xApi = { fn: async () => { const res = await api.get<{data:T[]}>(url); return res.data.data } }` object using the shared `api` axios client. Add grades/announcements/requests methods (or a parallel `grades-api.ts`). List endpoints return `{data:[]}`; mutations POST JSON; CSV uses `FormData` + `multipart/form-data` header (lines 159-167).

**Lecturer editor pages (Gradebook, Announcements compose, RequestInbox reply)** — `frontend/src/pages/lecturer/Assignments.tsx`: `useQuery`/`useMutation`/`useQueryClient`, `react-hook-form` + `zodResolver(schema)` + Zod schema, shadcn `Form/FormField/FormItem/FormControl/FormLabel/FormMessage`, `Input`, `Button`, `Table*`, `Dialog*`, `toast` from `sonner`. The grade-score schema reuses `z.coerce.number().min(0).max(100)` (lines 25-29). Reply form = a Dialog with a required Decision (Select/RadioGroup) + optional note Textarea, `invalidateQueries` on success.

**Student read pages (Grades view, Announcements browse, request list)** — `frontend/src/pages/Notifications.tsx`: read-only `useQuery` + `Card`/`CardHeader`/`CardContent`, `Skeleton` loading state (`isLoading` → 3 skeletons), empty state `text-muted-foreground`, `date-fns formatDistanceToNow`. Student Grades shows the frozen snapshot values (D-66) — overall only when all top-level components published.

**Routing** — `frontend/src/routes/router.tsx:52-82`: add children under existing `/student` and `/lecturer` `RoleGuard` trees, e.g. `{ path: 'courses/:id/grades', element: <StudentGrades /> }`, `{ path: 'courses/:id/gradebook', element: <Gradebook /> }`, `{ path: 'courses/:id/announcements', ... }`, `{ path: 'courses/:id/requests', ... }`. Mirror the existing `lecturer/courses/:id/quizzes` pattern (line 67).

## Shared Patterns

### Same-transaction notification write (NOTIF-02)
**Source:** `backend/internal/assignments/service.go:202-253`
**Apply to:** grades publish/republish, announcement fan-out, request create + reply — every student/lecturer-facing delivery.
`pool.Begin → defer Rollback → q.WithTx(tx) → mutation → InsertNotification(per recipient) → Commit`. `InsertNotificationParams` fields: `RecipientID, Type, Title, Body, ResourceType (pgtype.Text), ResourceID (pgtype.Int8), Link (pgtype.Text)`.

### Authorization / ownership
**Source:** `backend/internal/shared/authz/*.go` `AssertCourseMember(ctx, pool, courseID, userID, role)` + `middleware.RequireRole(db.UserRole...)` route groups.
**Apply to:** every handler (role gate) + every service mutation (membership gate). Requests add a `targeted_lecturer_id == user_id` filter (D-62 IDOR guard). Student grade reads scoped to `student_id == user_id`.

### Error envelope
**Source:** `errorEnvelope(code, message)` in `assignments/dto.go:36-40` → `{"error":{"code","message"}}`.
**Apply to:** all handler error responses. CSV validation is the one exception: `gin.H{"errors": rowErrs}` at HTTP 422.

### CSV import discipline (D-67)
**Source:** `enrollments/csv.go` (ParseCSV: collect ALL row errors, formula-injection sanitize `strings.TrimLeft(v, "=+-@\t\r ")`, dup detection) + `enrollments/service.go:22-119` (validate-all-then-tx, all-or-nothing) + `enrollments/handler.go:45-89` (MaxBytesReader 5<<20, 422 on ErrValidation).
**Apply to:** grade MANUAL CSV. Header `student_id,score`; per-row validate enrolled + numeric + `0≤score≤100`; one component per file (component_id from URL, not CSV).

### Roster / lecturer reads (reuse, don't rebuild)
**Source:** `backend/db/queries/courses.sql` `ListCourseStudents` (line 31), `ListCourseLecturers` (line 37) — both filter `u.deleted_at IS NULL`.
**Apply to:** gradebook roster, announcement ALL_STUDENTS fan-out, request lecturer-selection.

### Integration test harness
**Source:** `enrollments/import_test.go` — `DATABASE_URL` skip, `pgxpool.New`, inline seed + `defer DELETE`, testify. Anti-theater: assert atomic rollback explicitly.

## No Analog Found

None. Every Phase 5 file maps to an existing analog. The only **net-new logic** (no copy-from analog, build per RESEARCH §Code Examples) is:
- The normalize→aggregate grade computation (`ComputeQuizAverage`/`ComputeAssignmentAverage` SQL + Go weighted aggregation, missing=0-after-eligible).
- The 2-level `parent_id` component tree + sum-to-100 service validation (`validateWeights`, RESEARCH §Code Examples).
- The published-snapshot persistence model (`grade_publications` keyed `(component_id, student_id)`; student reads frozen value, lecturer recomputes live).

These have schema/algorithm sketches in RESEARCH but no existing code analog — the planner should reference RESEARCH §Patterns 1-2 and §Code Examples for them.

## Metadata

**Analog search scope:** `backend/internal/{assignments,enrollments,notifications,courses}/`, `backend/db/{queries,migrations}/`, `backend/internal/shared/{authz,middleware}/`, `backend/cmd/api/main.go`, `frontend/src/{pages,lib,routes}/`
**Files scanned:** ~25 read in full or targeted
**Pattern extraction date:** 2026-06-20
