# Phase 2: Auth, RBAC & Forced First-Login - Pattern Map

**Mapped:** 2026-06-20
**Files analyzed:** 24 (16 backend, 8 frontend)
**Analogs found:** 13 with codebase analog / 24 total (11 net-new, no prior analog — first feature folder + bare Vite starter)

> This is the FIRST feature folder (`internal/auth/`). Phase 1 shipped only `internal/shared/` infra + `internal/health/`. Backend wiring/config/migration/sqlc patterns have strong analogs; the per-feature handler/service/repository split and all frontend stack files are net-new (use RESEARCH.md Code Examples as the source).

## File Classification

### Backend

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `backend/internal/auth/handler.go` | handler (route registration + HTTP transport) | request-response | `internal/shared/health/health.go` | role-match (route reg + gin.Context only) |
| `backend/internal/auth/service.go` | service (business + authz/ownership) | request-response | *none* | no analog (first service layer) |
| `backend/internal/auth/repository.go` | repository (SQL via sqlc) | CRUD | `internal/shared/health/health.go` (`db.New(pool)` usage) | partial (consumption pattern only) |
| `backend/internal/auth/model.go` | model | — | `internal/shared/db/models.go` (`User`, `UserRole`) | reuse existing types |
| `backend/internal/auth/dto.go` | dto | — | *none* | no analog (no DTOs exist yet) |
| `backend/internal/shared/auth/jwt.go` | utility (JWT mint/parse) | transform | *none* | no analog (RESEARCH Code Examples 1-2) |
| `backend/internal/shared/middleware/auth.go` | middleware (5-step chain) | request-response | *none* | no analog (first middleware) |
| `backend/internal/shared/middleware/role.go` | middleware (`RequireRole`) | request-response | *none* | no analog |
| `backend/internal/shared/middleware/cors.go` | middleware (CORS) | request-response | *none* | no analog (RESEARCH Code Example 3) |
| `backend/internal/shared/config/config.go` (MODIFY) | config | — | itself (lines 8-13) | exact (add fields to existing struct) |
| `backend/cmd/api/main.go` (MODIFY) | wiring/entrypoint | — | itself (lines 26-27) | exact (mirror `health.RegisterRoutes`) |
| `backend/db/queries/users.sql` | sqlc query input | CRUD | `db/queries/healthcheck.sql` | exact (sqlc annotation format) |
| `backend/internal/shared/db/users.sql.go` (GENERATED) | repository (generated) | CRUD | `internal/shared/db/healthcheck.sql.go` | exact (do not hand-write; run `sqlc generate`) |
| `backend/db/migrations/000003_add_password_changed_at.up.sql` | migration | — | `db/migrations/000001_init_foundation.up.sql` | role-match (naming + up convention) |
| `backend/db/migrations/000003_add_password_changed_at.down.sql` | migration | — | `db/migrations/000001_init_foundation.down.sql` | role-match (down convention) |
| `backend/internal/auth/*_test.go` | test | — | `internal/shared/health/healthz_test.go`, `internal/shared/db/seed_test.go` | exact (integration test shape) |

### Frontend (all NEW — current `frontend/` is the bare Vite + React 19 starter)

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `frontend/src/lib/api.ts` | utility (axios + interceptor) | request-response | *none* | no analog (RESEARCH Code Example 6) |
| `frontend/src/lib/queryClient.ts` | provider/config | — | *none* | no analog |
| `frontend/src/stores/auth.ts` | store (Zustand) | event-driven | *none* | no analog |
| `frontend/src/routes/{ProtectedRoute,RoleGuard,router}.tsx` | route guards | — | *none* | no analog |
| `frontend/src/pages/{Login,ChangePassword,...}.tsx` | component (pages) | request-response | `frontend/src/App.tsx` (starter, to be replaced) | baseline only |
| `frontend/src/components/AppLayout.tsx` | component | — | *none* | no analog |
| `frontend/src/main.tsx` (MODIFY) | entrypoint | — | itself (lines 1-10) | exact (wrap App in providers) |
| `frontend/package.json` / `vite.config.ts` / `tsconfig*.json` (MODIFY) | config | — | themselves | exact (extend, see below) |

## Pattern Assignments

### `backend/internal/auth/handler.go` (handler, route registration + request-response)

**Analog:** `backend/internal/shared/health/health.go`

This is the only existing example of route registration + handler in the codebase. Mirror its `RegisterRoutes(r, pool)` signature and `db.New(pool)` usage. Auth's version adds `cfg` (for JWTSecret/cookie flags) and splits public vs protected groups.

**Route registration + queries construction pattern** (health.go lines 11-25):
```go
func RegisterRoutes(r *gin.Engine, pool *pgxpool.Pool) {
	queries := db.New(pool)

	r.GET("/healthz", func(c *gin.Context) {
		count, err := queries.CountUsers(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db down"})
			return
		}
		_ = count
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}
```

**Copy for auth:** Same `func RegisterRoutes(r *gin.Engine, pool *pgxpool.Pool, cfg config.Config)` shape. Construct `db.New(pool)` → wrap in `repository` → `service` → `handler`. Register `/auth/login`, `/auth/refresh` as public; `/auth/me`, `/auth/change-password`, `/auth/logout` behind `middleware.AuthMiddleware(...)`. Use `c.Request.Context()` for DB calls (established here). Cookie set/clear and error envelope: RESEARCH Code Example 4 + the `{ "error": { "code", "message" } }` envelope from CONTEXT discretion.

**Import convention** (health.go lines 1-9) — module path is `github.com/panadolextra91/myiu-lite/backend/...`:
```go
import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
)
```

---

### `backend/internal/auth/repository.go` (repository, CRUD)

**Analog:** generated `backend/internal/shared/db/healthcheck.sql.go` (consumption) + `db.go`

No standalone repository layer exists yet. The repository wraps the sqlc-generated `*db.Queries` (constructed via `db.New(pool)` — see `db.go` lines 20-22). Repository methods delegate to generated methods (`GetUserByUsername`, `GetUserByID`, `UpdatePasswordAndStamp`).

**Generated method shape to consume** (healthcheck.sql.go lines 16-21):
```go
func (q *Queries) CountUsers(ctx context.Context) (int64, error) {
	row := q.db.QueryRow(ctx, countUsers)
	var count int64
	err := row.Scan(&count)
	return count, err
}
```

**Copy:** Repository holds `*db.Queries`; methods call `r.q.GetUserByUsername(ctx, username)` returning the reusable `db.User` struct (see model.go below). No hand-written `rows.Scan` — sqlc generates it.

---

### `backend/internal/auth/model.go` (model — reuse, do not redefine)

**Analog / reuse source:** `backend/internal/shared/db/models.go` lines 14-75

Reuse these generated types directly — do NOT redefine User/role in the auth package:
```go
type UserRole string
const (
	UserRoleStudent  UserRole = "student"
	UserRoleLecturer UserRole = "lecturer"
	UserRoleAdmin    UserRole = "admin"
)

type User struct {
	ID                 int64
	Username           string
	PasswordHash       string
	Role               UserRole
	MustChangePassword bool
	CreatedAt          pgtype.Timestamptz
	UpdatedAt          pgtype.Timestamptz
	DeletedAt          pgtype.Timestamptz
}
```
**After migration 000003 + `sqlc generate`**, `User` gains `PasswordChangedAt pgtype.Timestamptz`. Middleware compares `claims.IssuedAt.Time` against `user.PasswordChangedAt.Time`. `model.go` should only hold auth-internal types (e.g. sentinel errors `ErrCurrentPasswordWrong`, `ErrSameAsCurrent`, `ErrTooShort`) — the User/role types live in `db`.

---

### `backend/db/queries/users.sql` (sqlc query input, CRUD)

**Analog:** `backend/db/queries/healthcheck.sql`

**Exact annotation format to copy** (healthcheck.sql):
```sql
-- name: CountUsers :one
SELECT count(*) FROM users;
```

**Copy for users (per RESEARCH Code Example 7):** Same `-- name: X :one|:exec` annotation style. New file `db/queries/users.sql`:
```sql
-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1 AND deleted_at IS NULL;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdatePasswordAndStamp :exec
UPDATE users
SET password_hash = $2, password_changed_at = now(), must_change_password = false, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;
```
Note the `WHERE deleted_at IS NULL` soft-delete filter — consistent with D-10/D-08 conventions. After editing, run `sqlc generate` (config: `sqlc.yaml` → out `internal/shared/db`, package `db`, `sql_package: pgx/v5`). Generated `users.sql.go` mirrors `healthcheck.sql.go`.

---

### `backend/db/migrations/000003_add_password_changed_at.{up,down}.sql` (migration)

**Analog:** `backend/db/migrations/000001_init_foundation.up.sql` / `.down.sql`, `000002_seed_bootstrap_admin.*`

**Naming convention** (observed): `00000N_snake_case_description.{up,down}.sql` — zero-padded 6-digit sequence, paired up/down. Sequence is at `000002`; this phase appends `000003` (D-06 incremental). Raw SQL DDL, no transaction wrapper (golang-migrate handles it).

**up.sql** (per RESEARCH Code Example 5):
```sql
ALTER TABLE users ADD COLUMN password_changed_at TIMESTAMPTZ NOT NULL DEFAULT now();
```
**down.sql** (mirror the `000001` down convention — drop what up created):
```sql
ALTER TABLE users DROP COLUMN password_changed_at;
```
`000001` down drops in reverse order (`DROP TABLE audit_log; DROP TABLE users; DROP TYPE user_role;`) — single-statement down here is fine.

---

### `backend/internal/shared/config/config.go` (config — MODIFY existing struct)

**Analog:** itself, lines 8-13

**Existing struct pattern** (uses `caarlos0/env` tags + `envDefault`):
```go
type Config struct {
	DatabaseURL   string `env:"DATABASE_URL,required"`
	JWTSecret     string `env:"JWT_SECRET,required"`
	CloudinaryURL string `env:"CLOUDINARY_URL,required"`
	Port          string `env:"PORT" envDefault:"8080"`
}
```
**Add (per RESEARCH Open Questions 1):** Append two fields following the same tag convention — do NOT mark required (they have defaults):
```go
	FrontendOrigin string `env:"FRONTEND_ORIGIN" envDefault:"http://localhost:5173"`
	CookieSecure   bool   `env:"COOKIE_SECURE" envDefault:"true"`
```
`Load()` (lines 15-18) needs no change — `env.ParseAs[Config]()` picks up new fields automatically. `JWTSecret` already present and loaded; auth consumes it.

> GitNexus: run `impact({target: "Config", direction: "upstream"})` before editing this struct (CLAUDE.md mandate).

---

### `backend/cmd/api/main.go` (wiring — MODIFY)

**Analog:** itself, lines 26-27 (existing `health.RegisterRoutes`)

**Existing wiring pattern:**
```go
router := gin.Default()
health.RegisterRoutes(router, pool)
```
**Add:** CORS must be applied via `router.Use(...)` BEFORE any route registration (RESEARCH Open Question 2). Then mirror the `health.RegisterRoutes` call for auth:
```go
router := gin.Default()
router.Use(middleware.CORS(cfg.FrontendOrigin))   // before routes
health.RegisterRoutes(router, pool)
auth.RegisterRoutes(router, pool, cfg)            // mirrors health signature + adds cfg
```
pgx pool construction (lines 19-24) and `config.Load()` (lines 14-17) are already in place — reuse, do not duplicate.

> GitNexus: run `impact` on `main`/`RegisterRoutes` before editing; `detect_changes` before commit.

---

### `backend/internal/auth/*_test.go` (test, integration)

**Analog:** `backend/internal/shared/health/healthz_test.go` + `backend/internal/shared/db/seed_test.go`

**Integration test skeleton to copy** (healthz_test.go lines 18-49):
```go
func TestX_Integration(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL is not set; skipping integration test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	router := gin.Default()
	// register routes, then ServeHTTP against httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/auth/login", body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}
```
**Conventions established:** `*_test` package suffix, `os.Getenv("DATABASE_URL")` + `t.Skip` guard, `stretchr/testify/require`, `httptest.NewRecorder()` for HTTP assertions, JSON unmarshal of response body. `seed_test.go` shows the `bcrypt.CompareHashAndPassword([]byte(hash), []byte("123456"))` pattern — reuse for verifying password-change writes the bootstrap admin (`admin`/`123456`, `must_change_password=TRUE`) is the live fixture to exercise login + forced-reset.

---

### Net-new backend files (no codebase analog — use RESEARCH.md)

| File | Source pattern in RESEARCH |
|------|----------------------------|
| `internal/shared/auth/jwt.go` | Code Examples 1 (Mint) + 2 (Parse). `Claims{ Role string; jwt.RegisteredClaims }`; `WithValidMethods(["HS256"])`; module path `github.com/panadolextra91/myiu-lite/backend/internal/shared/auth`. |
| `internal/shared/middleware/auth.go` | Pattern 2 (5-step chain, D-14): verify sig+exp → load user (via `db.Queries`) → `deleted_at IS NULL` → `iat >= password_changed_at` → `must_change_password` allow-list. Sets `user_id`/`role` into `gin.Context`. |
| `internal/shared/middleware/role.go` | Pattern 3 (`RequireRole(roles ...db.UserRole)` factory reading role from context). |
| `internal/shared/middleware/cors.go` | Code Example 3 (`cors.New(cors.Config{AllowOrigins:[cfg.FrontendOrigin], AllowCredentials:true})`). |
| `internal/auth/service.go` | Code Example 4 (ChangePassword: bcrypt verify current → len>=6 → must-differ → `GenerateFromPassword(..., 12)`). `user_id` from ctx (Pitfall 4). |
| `internal/auth/dto.go` | LoginRequest, ChangePasswordRequest{current,new,confirm}, MeResponse{id,username,role,must_change_password}, error envelope. |

---

### Frontend files (no codebase analog — bare starter; use RESEARCH.md)

The current `frontend/` is the unmodified Vite + React 19 starter (`package.json` has only `react`/`react-dom`; `vite.config.ts` has only `react()`; `App.tsx` is the demo). All Phase-2 FE files are net-new. Baseline to extend:

**`frontend/src/main.tsx`** (current, lines 1-10) — wrap `<App />` with `QueryClientProvider` + `BrowserRouter` here:
```tsx
createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
```

**`frontend/vite.config.ts`** (current, lines 1-7) — add `@tailwindcss/vite` plugin + `@/*` alias (RESEARCH Pitfall 6 — shadcn on existing project):
```ts
export default defineConfig({
  plugins: [react()],
})
```

**`frontend/package.json`** (current deps: only react/react-dom) — install per RESEARCH "Installation": `react-router zustand @tanstack/react-query axios react-hook-form zod @hookform/resolvers` + `-D tailwindcss @tailwindcss/vite`, then `npx shadcn@latest init` + `add form button input card label`.

| File | Source pattern in RESEARCH |
|------|----------------------------|
| `src/lib/api.ts` | Code Example 6 (axios `withCredentials:true` + single-flight 401 refresh interceptor, D-22). |
| `src/stores/auth.ts` | Architectural Map (Zustand: `user`, `setUser`, `clear` — UI/auth flags only, not server cache). |
| `src/routes/ProtectedRoute.tsx`, `RoleGuard.tsx` | Pattern 3 note + D-20/D-21 (route-boundary role split; RoleGuard is UX-only, not the security bar). |
| `src/pages/{Login,ChangePassword}.tsx` | RHF + Zod + zodResolver + shadcn `<Form>`; mirror D-17/D-18 (min length 6) client-side. |
| Conform all pages to | `/.planning/DESIGN-SYSTEM.md` (shadcn only, 6px radius, Lucide, light+dark). |

## Shared Patterns

### Go module import path
**Source:** every backend `.go` file (e.g. `health.go` line 8)
**Apply to:** all new backend files
```go
github.com/panadolextra91/myiu-lite/backend/internal/...
```

### Route registration signature
**Source:** `backend/internal/shared/health/health.go` line 11
**Apply to:** `auth.RegisterRoutes` and every future feature folder
```go
func RegisterRoutes(r *gin.Engine, pool *pgxpool.Pool) { queries := db.New(pool); ... }
```
Auth extends with a `cfg config.Config` param (for JWTSecret + cookie/CORS flags).

### sqlc query → generated code flow
**Source:** `db/queries/healthcheck.sql` → `internal/shared/db/healthcheck.sql.go` (via `sqlc.yaml`)
**Apply to:** all DB access. Write SQL with `-- name: X :one|:exec` in `db/queries/`, run `sqlc generate`, consume the generated method through a repository. Never hand-write scan code.

### Config via caarlos0/env tags
**Source:** `backend/internal/shared/config/config.go` lines 8-13
**Apply to:** all new env vars — `env:"NAME"` + `envDefault:"..."` (or `,required`). No Viper.

### Incremental migration naming
**Source:** `backend/db/migrations/000001_*` / `000002_*`
**Apply to:** `000003_add_password_changed_at.{up,down}.sql` (D-06).

### Integration test harness
**Source:** `backend/internal/shared/health/healthz_test.go`, `db/seed_test.go`
**Apply to:** all auth tests — `_test` package, `DATABASE_URL` env + `t.Skip` guard, `testify/require`, `httptest` for HTTP, `bcrypt.CompareHashAndPassword` for credential assertions. Bootstrap admin (`admin`/`123456`, must_change=TRUE) is the live fixture.

## No Analog Found

Files with no close codebase match (planner uses RESEARCH.md Code Examples / patterns):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/auth/service.go` | service | request-response | First service layer in the project (RESEARCH Code Example 4). |
| `internal/auth/dto.go` | dto | — | No DTOs exist yet. |
| `internal/shared/auth/jwt.go` | utility | transform | First JWT code (RESEARCH Code Examples 1-2). |
| `internal/shared/middleware/{auth,role,cors}.go` | middleware | request-response | First middleware in the project (RESEARCH Patterns 2-3, Code Example 3). |
| `frontend/src/lib/api.ts` | utility | request-response | First axios/interceptor (RESEARCH Code Example 6). |
| `frontend/src/stores/auth.ts` | store | event-driven | First Zustand store. |
| `frontend/src/routes/*`, `pages/*`, `components/AppLayout.tsx` | route/component | — | Bare Vite starter; no app structure yet. Conform to DESIGN-SYSTEM.md. |

## Metadata

**Analog search scope:** `backend/` (cmd, internal/shared/{config,db,health}, db/{migrations,queries}, sqlc.yaml), `frontend/` (src, package.json, vite.config.ts)
**Files scanned:** 24 source files (full backend tree + frontend baseline)
**Pattern extraction date:** 2026-06-20
