# Phase 3: Admin Provisioning & Course Lifecycle - Pattern Map

**Mapped:** 2026-06-20
**Files analyzed:** 38 new/modified files (backend + frontend)
**Analogs found:** 36 / 38 (2 with no direct analog — append-only triggers, sweep goroutine)

This phase is overwhelmingly an *application* of Phase 1-2 patterns. Every backend feature folder mirrors `internal/auth/`; every frontend page mirrors `pages/Login.tsx`; every migration mirrors `db/migrations/000001..000003`. Two pieces are genuinely new (no in-repo analog): the Postgres append-only triggers (D-35) and the `time.Ticker` sweep goroutine (D-37) — for those, use the verbatim RESEARCH.md excerpts.

## File Classification

### Backend

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `backend/db/migrations/000004_*.up/down.sql` | migration | schema | `db/migrations/000001_init_foundation.up.sql` + `000003_*.up.sql` | exact (ALTER + CREATE TABLE) |
| `backend/db/migrations/000005_*.up/down.sql` | migration | schema | `db/migrations/000002_seed_bootstrap_admin.up.sql` (seed) | role-match (triggers = no analog) |
| `backend/db/queries/users.sql` (extend) | query | CRUD | `db/queries/users.sql` | exact |
| `backend/db/queries/courses.sql` | query | CRUD | `db/queries/users.sql` | exact |
| `backend/db/queries/enrollments.sql` | query | CRUD/batch | `db/queries/users.sql` | role-match |
| `backend/db/queries/auditlogs.sql` | query | CRUD (write + filtered list) | `db/queries/users.sql` | role-match (narg list = new) |
| `backend/internal/shared/db/*.sql.go` + `models.go` | generated | — | `internal/shared/db/users.sql.go` + `models.go` | exact (regenerate via `sqlc generate`) |
| `backend/internal/users/{handler,service,repository,model,dto}.go` | controller/service/repo | CRUD + file-I/O (CSV) | `internal/auth/*` | exact |
| `backend/internal/courses/{handler,service,repository,model,dto}.go` | controller/service/repo | CRUD | `internal/auth/*` | exact |
| `backend/internal/enrollments/{handler,service,repository,model,dto}.go` | controller/service/repo | batch + file-I/O (CSV) | `internal/auth/*` | role-match |
| `backend/internal/auditlogs/{handler,service,repository}.go` + `writeAudit` helper | controller/service/repo | request-response (read) + INSERT | `internal/auth/*` | role-match |
| `backend/internal/lifecycle/sweep.go` | service (scheduler) | event-driven (timer) | — | **no analog** (use RESEARCH Pattern 4) |
| `backend/cmd/api/main.go` (modify) | config/wiring | — | `cmd/api/main.go` | exact |
| `backend/internal/*/[*_test.go]` | test | integration/unit | `internal/auth/auth_login_test.go`, `internal/shared/db/seed_test.go` | exact |

### Frontend

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `frontend/src/pages/admin/Accounts.tsx` | component (page) | CRUD (list + mutate) | `pages/Login.tsx` (form) + TanStack pattern | role-match |
| `frontend/src/pages/admin/Courses.tsx` | component (page) | CRUD | `pages/Login.tsx` | role-match |
| `frontend/src/pages/admin/CourseDetail.tsx` | component (page) | request-response (read + remove) | `pages/Login.tsx` | partial |
| `frontend/src/pages/admin/Enrollment.tsx` | component (page) | file-I/O (CSV upload) | `pages/Login.tsx` (RHF/Zod) | partial |
| `frontend/src/pages/admin/LecturerAssignment.tsx` | component (page) | file-I/O (CSV upload) | `pages/Login.tsx` | partial |
| `frontend/src/pages/admin/AuditLogs.tsx` | component (page) | request-response (read) | `pages/Login.tsx` | partial |
| `frontend/src/pages/admin/Index.tsx` (modify → Dashboard) | component (page) | request-response (read) | `pages/admin/Index.tsx` | exact |
| `frontend/src/components/admin/AdminSidebar.tsx` | component | — | `components/AppLayout.tsx` | role-match |
| `frontend/src/components/AppLayout.tsx` (modify) | component | — | `components/AppLayout.tsx` | exact |
| `frontend/src/routes/router.tsx` (modify) | route | — | `routes/router.tsx` | exact |
| `frontend/src/components/ui/{sidebar,table,tabs,dialog,alert-dialog,select,badge,skeleton}.tsx` | config (vendored) | — | `components/ui/button.tsx` | vendored via CLI |

## Pattern Assignments

### Backend feature folders — `internal/{users,courses,enrollments,auditlogs}/` (controller/service/repo)

**Analog:** `backend/internal/auth/{handler,service,repository,dto,model}.go`

**Five-file split + `RegisterRoutes(r, pool, cfg)` wiring** (`handler.go:15-41`):
```go
type Handler struct {
	svc *Service
	cfg config.Config
}

func RegisterRoutes(r *gin.Engine, pool *pgxpool.Pool, cfg config.Config) {
	q := db.New(pool)
	repo := NewRepository(q)
	svc := NewService(repo, cfg)
	h := &Handler{svc: svc, cfg: cfg}
	// admin routes: chain AuthMiddleware + RequireRole(admin)
	g := r.Group("/admin")
	g.Use(middleware.AuthMiddleware(pool, cfg), middleware.RequireRole(db.UserRoleAdmin))
	{
		g.POST("/students/import", h.ImportStudents) // etc.
	}
}
```
> Note: existing auth uses per-route middleware (`handler.go:34-37`). For an all-admin group, apply `AuthMiddleware` + `RequireRole(db.UserRoleAdmin)` once on the group (matches CONTEXT "all admin endpoints behind RequireRole").

**`errorEnvelope` helper** — copy verbatim into each feature's `dto.go` (`auth/dto.go:21-28`):
```go
func errorEnvelope(code, message string) map[string]interface{} {
	return map[string]interface{}{
		"error": map[string]interface{}{"code": code, "message": message},
	}
}
```

**Handler validation + envelope dispatch** (`handler.go:43-63`): bind JSON, map sentinel errors to status + `errorEnvelope`, default to 500. Reuse this exact structure. For CSV import the 422 shape replaces the envelope (see below).

**Service = business + sentinel errors** (`service.go:41-71`): validate, call `bcrypt`, delegate to repo. New `model.go` declares `var ErrXxx = errors.New(...)` like `auth/model.go:5-11`.

**Repository = thin sqlc wrapper** (`repository.go:9-30`):
```go
type Repository struct{ q *db.Queries }
func NewRepository(q *db.Queries) *Repository { return &Repository{q: q} }
func (r *Repository) GetUserByID(ctx context.Context, id int64) (db.User, error) {
	return r.q.GetUserByID(ctx, id)
}
```
> For all-or-nothing CSV import the repository/service must use `pgx.Tx` + `q.WithTx(tx)` (not in the auth analog — see RESEARCH Code Examples §"All-or-nothing import"). The repo wraps `db.Queries`; the service owns `pool.Begin(ctx)` / `tx.Commit` / `defer tx.Rollback`.

### bcrypt cost=12 + DD/MM/YYYY → DDMMYYYY derivation (users service, ADMIN-03)

**Analog:** `auth/service.go:65` (`bcrypt.GenerateFromPassword([]byte(newPass), 12)`). Reuse cost=12 exactly. Derivation logic itself is new — use RESEARCH Code Examples §"CSV row → user mapping" (`time.Parse("02/01/2006")` → `t.Format("02012006")` → bcrypt 12).

### ADMIN-04 reset — reuse `password_changed_at` kill-switch

**Analog query:** `db/queries/users.sql:7-8` `UpdatePasswordAndStamp`:
```sql
UPDATE users SET password_hash = $2, password_changed_at = now(), must_change_password = false, updated_at = now() WHERE id = $1 AND deleted_at IS NULL;
```
Reset is the same shape but sets `must_change_password = true` and re-derives the hash from stored `date_of_birth`. The `password_changed_at = now()` bump is the load-bearing session-kill primitive (enforced by `middleware/auth.go:47`).

### `audit_log` write + filtered list (auditlogs feature, D-33/D-34/D-36)

**Analog:** `db/queries/users.sql` query-file style. Write helper and `narg` list query are new — use RESEARCH Pattern 1 (`WriteAuditLog :exec`, `ListAuditLogs :many` with `sqlc.narg('actor_id')` etc.). Existing `AuditLog` model (`models.go:57-64`) gains `TargetType pgtype.Text`, `TargetID pgtype.Int8`, `OperationID pgtype.UUID`, `AffectedCount pgtype.Int4` after `sqlc generate`.

### Migration `000004` (schema) — ALTER + CREATE TABLE

**Analog:** `000001_init_foundation.up.sql` (CREATE TABLE + partial unique index) and `000003_*.up.sql` (one-line ALTER ADD COLUMN).

**Existing `users` shape to extend** (`000001:3-14`): `id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY`, the `users_username_active_uq` partial index `ON users (username) WHERE deleted_at IS NULL`. New columns follow `000003`'s `ALTER TABLE users ADD COLUMN ...` style: `full_name TEXT`, `date_of_birth DATE`, `is_system BOOLEAN NOT NULL DEFAULT FALSE`.

**Existing `audit_log` shape** (`000001:16-26`): extend per RESEARCH Pattern 1 (`target_type`, `target_id`, `operation_id UUID DEFAULT gen_random_uuid()`, `affected_count`). New `courses`/`student_enrollments`/`course_lecturers` tables copy the `BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY` + `deleted_at TIMESTAMPTZ` + `created_at/updated_at DEFAULT now()` conventions from the `users` table. Membership tables add `UNIQUE (course_id, student_id)` / `(course_id, lecturer_id)` for `ON CONFLICT DO NOTHING`. Down migration drops in reverse (no analog beyond `000001.down.sql`).

### Migration `000005` (triggers + SYSTEM seed)

**SYSTEM seed analog:** `000002_seed_bootstrap_admin.up.sql:5-6` — INSERT a single user row. Adapt with sentinel `password_hash = '!'`, `is_system = TRUE`, `username = '__system__'` (RESEARCH Pattern 2):
```sql
INSERT INTO users (username, password_hash, role, must_change_password, is_system)
VALUES ('__system__', '!', 'admin', FALSE, TRUE);
```
**Append-only triggers — NO IN-REPO ANALOG.** Use RESEARCH Pattern 3 verbatim (`CREATE FUNCTION audit_log_append_only() ... RAISE EXCEPTION` + `BEFORE UPDATE`/`BEFORE DELETE` triggers; down drops triggers then function).

### Sweep goroutine — `internal/lifecycle/sweep.go` (NO IN-REPO ANALOG)

Use RESEARCH Pattern 4 verbatim: `StartSweeper(ctx, pool, systemID)` with `run()` startup catch-up + `time.NewTicker(24*time.Hour)` loop; `runSweep` does the idempotent `UPDATE courses SET deleted_at = now() WHERE deleted_at IS NULL AND end_date < now() - interval '1 month'` and conditional audit INSERT guarded by `if n > 0` (D-39).

### `cmd/api/main.go` (modify) — route registration + scheduler wiring

**Analog:** `cmd/api/main.go:29-41`. New feature routes register exactly like `auth.RegisterRoutes(router, pool, cfg)` (line 36). After route registration, resolve the SYSTEM id (`GetSystemUserID`) and call `lifecycle.StartSweeper(ctx, pool, systemID)` using the existing `ctx` (line 22) — make it cancelable for shutdown.

### Frontend admin pages — form/list/mutation

**Analog:** `pages/Login.tsx` (RHF + Zod + shadcn `Form`).

**RHF + Zod + shadcn form scaffold** (`Login.tsx:1-39, 85-115`): `useForm({ resolver: zodResolver(schema) })`, `<Form {...form}>`, `<FormField ... render={({field}) => <FormItem><FormLabel/><FormControl><Input {...field}/></FormControl><FormMessage/></FormItem>}/>`. Reuse for manual account-create and course CRUD dialogs.

**axios `api` client + error handling** (`Login.tsx:43-66`): `api.post(...)`, `axios.isAxiosError(err) && err.response?.status === ...`. For CSV import, on `422` read `err.response.data.errors` and render as a shadcn `<Table>` (RESEARCH FE example). Server state via TanStack `useMutation` + `qc.invalidateQueries` (RESEARCH FE example) — TanStack is not yet used in any page, so introduce it here; the `QueryClient` already exists at `lib/queryClient.ts`.

### Frontend routing — `routes/router.tsx` (modify)

**Analog:** `router.tsx:51-57`. The `/admin` block currently renders only `AdminIndex` as `index`. Add sibling child routes under the existing `RoleGuard allowedRoles={['admin']}` node:
```tsx
{
  path: '/admin',
  element: <RoleGuard allowedRoles={['admin']} />,
  children: [
    { index: true, element: <AdminIndex /> },
    { path: 'accounts', element: <Accounts /> },
    { path: 'courses', element: <Courses /> },
    { path: 'courses/:id', element: <CourseDetail /> },
    { path: 'enrollment', element: <Enrollment /> },
    { path: 'lecturers', element: <LecturerAssignment /> },
    { path: 'audit', element: <AuditLogs /> },
  ],
},
```
All admin routes already sit inside `<ProtectedRoute>` → `<AppLayout>` (`router.tsx:32-36`), so the sidebar belongs in `AppLayout`.

### Frontend sidebar — `components/admin/AdminSidebar.tsx` + `AppLayout.tsx` (modify)

**Analog:** `components/AppLayout.tsx` (header/logout shell + `useAuthStore` + `api.post('/auth/logout')`). Extend the layout to render the shadcn `sidebar` for admin role (D-41 sections: Dashboard; User Management→Accounts; Academic Management→Courses/Enrollment/Lecturers; System→Audit Logs). Keep the existing `handleLogout` (`AppLayout.tsx:11-18`) and `lucide-react` icon usage.

## Shared Patterns

### Auth + role gate (apply to ALL admin endpoints)
**Source:** `internal/shared/middleware/auth.go` + `role.go`
```go
// role.go:10 — gate
func RequireRole(roles ...db.UserRole) gin.HandlerFunc { ... } // RequireRole(db.UserRoleAdmin)
// auth.go:66-67 — sets c.Get("user_id") (int64) and c.Get("role") (string) for handlers
```
Admin handlers read the actor for audit via `c.Get("user_id")` (the bootstrap admin's JWT id — RESEARCH A4). SYSTEM actor is only for the sweep.

### Error envelope (apply to all controller/service files)
**Source:** `internal/auth/dto.go:21-28` (`errorEnvelope`) and the inline `gin.H{"error": gin.H{"code","message"}}` in `middleware/role.go:14`. CSV failures use the **422** override: `c.JSON(http.StatusUnprocessableEntity, gin.H{"errors": rowErrors})` (RESEARCH Handler 422 shape, D-27).

### sqlc generated-code consumption (apply to all repositories)
**Source:** `internal/shared/db/users.sql.go` + `db.go`. Pattern: `db.New(pool)` → `*db.Queries`; each query is `func (q *Queries) X(ctx, params) (T, error)`. Params structs (`UpdatePasswordAndStampParams`, `users.sql.go:58-61`) and pgtype fields (`models.go:57-76`) are generated — never hand-edit. Run `cd backend && sqlc generate` after adding `db/queries/*.sql`.

### Soft-delete + SYSTEM exclusion discipline (apply to all course/account list queries)
**Source:** `db/queries/users.sql:2,5` (`WHERE ... AND deleted_at IS NULL`). Every new `courses`/account SELECT adds `deleted_at IS NULL`; every account list also adds `AND is_system = FALSE` (RESEARCH Pitfall 2/5).

### Integration test harness (apply to all `*_test.go`)
**Source:** `internal/shared/db/seed_test.go:12-20` — `DATABASE_URL` env, `t.Skip` if unset, `pgx.Connect`, query + assert. `internal/auth/auth_login_test.go` is the handler-level analog for HTTP tests. Wave-0 test files map 1:1 to RESEARCH §"Phase Requirements → Test Map".

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `backend/db/migrations/000005_*.up.sql` (trigger portion) | migration | schema | No append-only trigger exists in repo. Use RESEARCH Pattern 3 verbatim. |
| `backend/internal/lifecycle/sweep.go` | service (scheduler) | event-driven (timer) | No scheduled/goroutine job exists in repo. Use RESEARCH Pattern 4 verbatim. |

Partial-match frontend pages (`Accounts`, `Courses`, `CourseDetail`, `Enrollment`, `LecturerAssignment`, `AuditLogs`) have a form analog (`Login.tsx`) but no list/table/TanStack analog in-repo — combine `Login.tsx`'s RHF/Zod/axios pattern with the RESEARCH FE TanStack example and newly-vendored shadcn `table`/`tabs`/`dialog`/`select` components.

## Metadata

**Analog search scope:** `backend/internal/{auth,shared}/`, `backend/db/{migrations,queries}/`, `backend/cmd/api/`, `frontend/src/{pages,components,routes,stores,lib}/`
**Files scanned:** 21 read in full
**Pattern extraction date:** 2026-06-20
