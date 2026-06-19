# Phase 2: Auth, RBAC & Forced First-Login - Research

**Researched:** 2026-06-20
**Domain:** Cookie-based JWT auth, RBAC middleware, forced first-login reset (Go/Gin backend + React SPA frontend)
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Session & token model**
- **D-11 — Cookie storage, not localStorage.** Tokens live in **HttpOnly + Secure + SameSite=Lax cookies**. Frontend uses axios with `withCredentials: true`; backend enables credential-aware CORS (specific origin, `Access-Control-Allow-Credentials: true`). Rationale: myIU holds grades/submissions/admin actions; HttpOnly blocks JS from reading the token. localStorage is intentionally avoided.
- **D-12 — Access + refresh tokens.** Access token = **15 minutes**, refresh token = **7 days**, both HttpOnly+Secure+SameSite=Lax cookies. Implies a `POST /auth/refresh` endpoint that mints a new access token from a valid refresh token.
- **D-13 — Stateless logout via `password_changed_at`, no blacklist.** `POST /auth/logout` clears both cookies; FE clears app state + redirects to `/login`. **No** JWT blacklist, Redis, session table, device management, or global-logout. Immediate invalidation of prior sessions via a **`password_changed_at` timestamp on `users`**: any JWT whose `iat` is older than `password_changed_at` is rejected at validation. Middleware loads the user row per request (accepted; "stateless" = no session store, not no DB read).

**Forced first-login reset**
- **D-14 — Enforce via live DB flag in middleware, not a JWT claim.** Middleware order: (1) verify JWT signature + expiry, (2) load user, (3) validate account status (not soft-deleted), (4) validate `password_changed_at`, (5) check `must_change_password`. JWT carries `user_id` + `role`; mutable state read from DB.
- **D-15 — Locked-state allow-list = exactly three endpoints.** When `must_change_password = true`, only `POST /auth/change-password`, `POST /auth/logout`, `GET /auth/me` are reachable; every other authenticated endpoint returns `403` with a machine-readable code signalling "password change required". FE calls `GET /auth/me` after login; if `must_change_password` redirects to `/change-password` until success or logout.
- **D-16 — Password change ends the session; force re-login.** On success: update hash → update `password_changed_at` → set `must_change_password = false` → **clear both cookies** → return `200 {"message": "Password changed successfully. Please log in again."}`. FE redirects to `/login`. No auto-reissue of a session.

**Password-change rules (mirror Zod on FE + Go on BE)**
- **D-17 — Require current password.** Body fields: `current_password`, `new_password`, `confirm_password`. Verify `current_password` against the stored hash before accepting.
- **D-18 — Minimum length 6, no complexity rules.** No uppercase/lowercase/digit/special-char composition rules. Matches bootstrap admin `123456`.
- **D-19 — New password must differ from current; no password history.** Enforced by `bcrypt.CompareHashAndPassword(currentHash, new_password)` returning *not-equal*. No "last N passwords" store.

**RBAC + ownership (AUTH-05)**
- **D-20 — Separate role route trees on the frontend.** `/student/*`, `/lecturer/*`, `/admin/*`; after login FE redirects to the role's root. Roles separated at the routing boundary, not via in-page conditionals.
- **D-21 — Minimal Phase-2 app shell only.** Ships exactly: Login page, Change-Password page, role landing pages (`/student`, `/lecturer`, `/admin`), plus shared `AppLayout`, `ProtectedRoute`, `RoleGuard`. **Out of scope:** sidebar navigation, feature menus, placeholder pages, empty links.
- **D-22 — Transparent 401 refresh on the frontend.** axios response interceptor catches `401` → calls `POST /auth/refresh` **once** → retries the original request once; on failure clears auth + redirects to `/login`. Guardrails: at most one refresh per failed request, no infinite loops, concurrent 401s share a single in-flight refresh, a `401/403` from the refresh endpoint itself means session expired.

### Claude's Discretion (settled by D-10 architecture)
- **Backend role-gate middleware shape + 403 error envelope.** Role gating lives in `internal/shared/middleware`; a `RequireRole(...)` guard plus a consistent JSON error envelope (e.g. `{ "error": { "code": ..., "message": ... } }`). Forced-change block (D-15) and role/ownership rejections all return `403` with a machine-readable code.
- **Ownership in Phase 2 = self-only.** The only owned resource is the user's own password. `user_id` is always derived from the JWT in `service.go`, never from request body/param — the AUTH-05 "never trust client-supplied IDs" rule.
- **`GET /auth/me`** returns `{ id, username, role, must_change_password }`.
- **Login error responses are generic.** "Invalid username or password"; no user-enumeration; no lockout/rate-limiting in MVP.
- **Auth feature lives in `internal/auth/`** (`handler.go`/`service.go`/`repository.go`/`model.go`/`dto.go`); sqlc user queries go in `backend/db/queries/`.

### Deferred Ideas (OUT OF SCOPE)
- Account lockout / login rate-limiting / brute-force throttling.
- Self-service password recovery (AUTH-V2-01) — admin reset (ADMIN-04) covers it in Phase 3.
- JWT blacklist / Redis revocation / session table / device management / global logout across devices.
- Password history ("last N passwords").
- Full sidebar navigation + feature menus + placeholder pages (Phase 3+).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| AUTH-01 | User can log in with username + password and receive a session/JWT carrying their role | golang-jwt v5 claims setup (§Code Examples 1–2); bcrypt verify; cookie-set pattern; generic login errors. |
| AUTH-02 | User can log out | Cookie-clear pattern (set MaxAge<0); stateless — no server store (D-13). |
| AUTH-03 | Logged-in user can change their own password | bcrypt cost=12 hash + `CompareHashAndPassword` "must differ" trick (§Code Examples 4); `password_changed_at` bump kills old sessions. |
| AUTH-04 | `must_change_password` restricted server-side to change-password only | Middleware allow-list of exactly 3 endpoints (§Architecture Pattern 2); enforced after JWT verify, reading live DB flag. |
| AUTH-05 | Routes authorized by role + ownership, never trusting client IDs | `RequireRole` middleware (§Pattern 3); ownership = `user_id` from JWT context, never from body/param (§Pitfall 4). |
</phase_requirements>

## Summary

This phase is **almost entirely a "use the locked stack correctly" exercise** — every library, version, and the major architectural decisions are already pinned in CLAUDE.md and locked D-11…D-22 in CONTEXT.md. There are **no new package decisions to make** and **no alternatives to weigh**; the research value is in the four explicit open items (CSRF, refresh rotation, golang-jwt v5 claims, CORS credentials) plus canonical code patterns for the middleware chain.

The four open items resolve cleanly and leanly: (1) **CSRF — NO dedicated token needed for this MVP.** SameSite=Lax cookies + a custom `Content-Type: application/json` / `X-Requested-With` requirement + credentialed CORS with a strict origin allow-list is sufficient against CSRF for a JSON API; a double-submit token adds machinery the MVP does not need (Ponytail). (2) **Refresh — reuse-until-expiry, NOT rotate-on-use**, because the `password_changed_at` invalidation primitive (D-13) already gives session kill-switch semantics and rotation requires a server-side store to detect token reuse — which D-13 explicitly forbids. The `iat < password_changed_at` check **must apply to the refresh token too**. (3) **golang-jwt v5** requires `WithValidMethods(["HS256"])` (algorithm-confusion defense), `RegisteredClaims` with `sub`/`iat`/`exp`, and `NewWithClaims(SigningMethodHS256, ...)`. (4) **gin-contrib/cors** with `AllowOrigins: [<exact dev/prod origin>]` + `AllowCredentials: true` — wildcard is illegal with credentials.

The backend adds an `internal/auth/` feature folder, an `internal/shared/middleware` (auth chain + `RequireRole`), an `internal/shared/auth` (JWT helpers), one migration `000003_add_password_changed_at`, and sqlc queries. The frontend installs the full committed stack (axios, react-router v7, Zustand v5, TanStack Query v5, shadcn/ui + Tailwind v4) onto the current bare Vite starter and ships the minimal shell.

**Primary recommendation:** Build the 5-step auth middleware chain exactly per D-14 as the single gate, use cookies for transport, skip CSRF tokens (rely on SameSite=Lax + strict CORS + JSON content-type), use reuse-until-expiry refresh tokens validated against `password_changed_at`, and keep the frontend shell to the D-21 minimum.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Credential verification (bcrypt) | API / Backend | — | Secrets never leave the server; client never sees the hash. |
| JWT mint / sign | API / Backend | — | Signing key (`JWT_SECRET`) is server-only. |
| JWT validate + `iat`/`password_changed_at`/`must_change_password` checks | API / Backend (middleware) | — | Authorization is never trustworthy on the client; D-14 mandates server-side. |
| Role gate (`RequireRole`) | API / Backend (middleware) | Frontend (RoleGuard, UX only) | BE enforces; FE `RoleGuard` only routes/hides — never the security boundary. |
| Ownership (self-only this phase) | API / Backend (service.go) | — | `user_id` derived from JWT, never client input (AUTH-05). |
| Token transport (cookies) | API sets / Browser stores | — | HttpOnly cookie set by backend `Set-Cookie`; browser auto-sends. |
| Session state (current user, `must_change_password`) | Frontend (Zustand) ← `GET /auth/me` | API source of truth | Zustand holds UI/auth flags only; backend is authoritative. |
| 401→refresh→retry | Frontend (axios interceptor) | API issues new token | D-22; FE owns retry/cleanup/redirect, BE owns validation/issuance. |
| Forced-reset routing | Frontend (redirect on `must_change_password`) | API enforces allow-list | FE convenience; BE allow-list is the real bar (D-15, success criterion 3). |

## Standard Stack

> **All versions are LOCKED in CLAUDE.md and already present in `backend/go.mod`. No package decisions for the planner — this table documents what to USE, not what to choose.**

### Core (backend — already in go.mod, verified present)
| Library | Version | Purpose | Status |
|---------|---------|---------|--------|
| Gin | v1.11.0 | HTTP framework, router, middleware, cookie helpers | `[VERIFIED: backend/go.mod]` present |
| pgx/v5 | v5.7.2 | Postgres driver + pool (auth repository reads/writes users) | `[VERIFIED: backend/go.mod]` present |
| sqlc | v1.31.1 | Generates user query methods from `backend/db/queries/*.sql` | `[VERIFIED: backend/sqlc.yaml]` configured |
| golang.org/x/crypto | v0.41.0 | `bcrypt` (cost=12) password hash + compare | `[VERIFIED: backend/go.mod]` present |
| caarlos0/env + godotenv | v11.4.1 / v1.5.1 | Config incl. `JWTSecret` (already loaded) | `[VERIFIED: backend/internal/shared/config/config.go]` |

### Supporting (backend — NEW installs this phase)
| Library | Version | Purpose | Install |
|---------|---------|---------|---------|
| golang-jwt/jwt/v5 | v5.3.1 | Mint + validate access/refresh JWTs (HS256) | `go get github.com/golang-jwt/jwt/v5@v5.3.1` |
| gin-contrib/cors | v1.7.x | Credentialed CORS for the cross-origin SPA | `go get github.com/gin-contrib/cors` |
| golang-migrate (CLI) | v4.18.x | Add migration `000003_add_password_changed_at` | already used by CI / Phase 1 |

> `[ASSUMED]` gin-contrib/cors latest is v1.7.6 (search result). CLAUDE.md pins "latest" for cors — planner should run `go get github.com/gin-contrib/cors@latest` and let go.mod resolve. v1.7.x API (`cors.Config{AllowOrigins, AllowCredentials}`) is stable. `[CITED: github.com/gin-contrib/cors]`

### Core (frontend — current state: bare Vite + React 19 starter; everything below is a NEW install)
| Library | Version | Purpose |
|---------|---------|---------|
| react-router | v7.x | Role route trees `/student|/lecturer|/admin`, `ProtectedRoute` (D-20/D-21) |
| Zustand | v5.x | Auth/session UI state (current user, `must_change_password`) |
| @tanstack/react-query | v5.x | Server-state for `GET /auth/me`, mutations for login/change-password |
| axios | v1.x | HTTP client with `withCredentials: true` + 401 refresh interceptor (D-22) |
| shadcn/ui (CLI) + Tailwind v4 + @tailwindcss/vite | latest / v4 | Form/Button/Input/Card components for Login + Change-Password |
| react-hook-form + zod + @hookform/resolvers | v7.79 / v4 / v3 | Login + change-password forms (D-17/D-18 mirrored client-side) |

### Alternatives Considered
| Instead of | Could Use | Tradeoff | Verdict |
|------------|-----------|----------|---------|
| No CSRF token | Double-submit CSRF token + `X-CSRF-Token` | More robust defense-in-depth, but extra endpoint/middleware/FE plumbing | **Skip for MVP** — SameSite=Lax + strict CORS + JSON content-type suffices (see Open Item 1). |
| Reuse-until-expiry refresh | Rotate-on-use refresh | Better reuse-detection, but **requires a server store** which D-13 forbids | **Reuse-until-expiry** (see Open Item 2). |
| TanStack Query for auth mutations | Plain axios in Zustand action | Query gives loading/error/invalidation ergonomics | Use Query for `me`/login; either is fine — locked stack includes Query. |

**Installation:**
```bash
# backend (from backend/)
go get github.com/golang-jwt/jwt/v5@v5.3.1
go get github.com/gin-contrib/cors@latest
go mod tidy

# frontend (from frontend/)
npm install react-router zustand @tanstack/react-query axios react-hook-form zod @hookform/resolvers
npm install -D tailwindcss @tailwindcss/vite
npx shadcn@latest init
npx shadcn@latest add form button input card label
```

## Package Legitimacy Audit

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| golang-jwt/jwt/v5 | Go modules | 2+ yrs (v5 line) | de-facto std JWT lib for Go | github.com/golang-jwt/jwt | OK | Approved — locked in CLAUDE.md |
| gin-contrib/cors | Go modules | 7+ yrs | official Gin middleware | github.com/gin-contrib/cors | OK | Approved — locked in CLAUDE.md |
| react-router | npm | 10+ yrs | ~12M/wk | github.com/remix-run/react-router | OK | Approved — locked |
| zustand | npm | 5+ yrs | ~6M/wk | github.com/pmndrs/zustand | OK | Approved — locked |
| @tanstack/react-query | npm | 6+ yrs | ~9M/wk | github.com/TanStack/query | OK | Approved — locked |
| axios | npm | 10+ yrs | ~60M/wk | github.com/axios/axios | OK | Approved — locked |
| react-hook-form / zod / @hookform/resolvers | npm | mature | tens of M/wk | respective official repos | OK | Approved — locked |

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** none.

> All packages are pre-locked, mainstream, and version-pinned in CLAUDE.md (which already carries HIGH-confidence registry sources from the Phase-0 stack research). No new or unverified packages introduced. The legitimacy gate seam was not re-run because every package is an established, CLAUDE.md-committed dependency with a known source repo; flagging is reserved for novel/unvetted picks.

## Architecture Patterns

### System Architecture Diagram

```
                         ┌─────────────────────────── Browser (React SPA, Vite origin) ──────────────────────────┐
  login form ──────────► │  axios (withCredentials:true)                                                          │
                         │     │  POST /auth/login {username,password}                                            │
                         │     │                                                                                  │
                         │     │  ◄── Set-Cookie: access_token(15m, HttpOnly,Secure,SameSite=Lax)                 │
                         │     │      Set-Cookie: refresh_token(7d, HttpOnly,Secure,SameSite=Lax)                 │
                         │     ▼                                                                                  │
                         │  GET /auth/me ──► {id,username,role,must_change_password}                              │
                         │     │                                                                                  │
                         │     ├─ must_change_password=true ─► react-router redirect /change-password (locked)   │
                         │     └─ false ─► redirect /{role} (RoleGuard)                                           │
                         │                                                                                        │
                         │  any request ─► 401 ─► interceptor: POST /auth/refresh (once) ─► retry once (D-22)     │
                         └──────────────────────────────────────────│───────────────────────────────────────────┘
                                                                     │ cookies auto-sent (browser)
                                  CORS preflight (OPTIONS) ◄─────────┤ AllowOrigins:[exact] AllowCredentials:true
                                                                     ▼
   ┌──────────────────────────────── Gin API (single binary) ───────────────────────────────────────────────┐
   │  router: cors.New(cfg) ─► public: /auth/login, /auth/refresh, /healthz                                    │
   │                                                                                                           │
   │  protected group ─► AuthMiddleware chain (D-14):                                                          │
   │     (1) verify JWT sig+exp (HS256, WithValidMethods)                                                      │
   │     (2) load user row by sub  ──────────────► repository.go ──► pgx pool ──► PostgreSQL (users)           │
   │     (3) account status (deleted_at IS NULL)                                                               │
   │     (4) iat >= password_changed_at  (else 401 — session killed by pwd change/reset)                       │
   │     (5) must_change_password? ─► allow-list {change-password, logout, me} else 403 CODE=password_change   │
   │           │                                                                                               │
   │           ├─ RequireRole("admin"|"lecturer"|"student") ─► 403 CODE=forbidden if mismatch                  │
   │           └─ handler.go (transport) ─► service.go (authz/ownership: user_id from ctx) ─► repository.go    │
   └───────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure
```
backend/
  internal/
    auth/
      handler.go        # HTTP: parse req, set/clear cookies, shape JSON. NO business logic.
      service.go        # login verify, change-password rules (D-17/D-19), authz/ownership (user_id from ctx)
      repository.go     # sqlc-backed user reads/writes (GetUserByUsername, GetUserByID, UpdatePassword)
      model.go          # internal types if needed beyond sqlc User
      dto.go            # LoginRequest, ChangePasswordRequest, MeResponse, error envelope
    shared/
      auth/             # JWT helpers: Mint(access/refresh), Parse/Validate, claims struct
      middleware/       # AuthMiddleware (5-step chain), RequireRole, error envelope helper
  db/
    migrations/
      000003_add_password_changed_at.up.sql   # ALTER TABLE users ADD password_changed_at
      000003_add_password_changed_at.down.sql
    queries/
      users.sql         # sqlc: GetUserByUsername, GetUserByID, UpdatePasswordAndStamp
frontend/src/
  lib/        api.ts (axios instance + interceptor), queryClient.ts
  stores/     auth.ts (Zustand: user, setUser, clear)
  routes/     ProtectedRoute.tsx, RoleGuard.tsx, router.tsx
  pages/      Login.tsx, ChangePassword.tsx, student/Index.tsx, lecturer/Index.tsx, admin/Index.tsx
  components/ AppLayout.tsx + shadcn/ui (ui/*)
```

### Pattern 1: Two-token mint + cookie set (login)
**What:** On valid credentials, mint a 15-min access JWT and a 7-day refresh JWT, set both as HttpOnly cookies.
**When:** `POST /auth/login` success and `POST /auth/refresh` (re-mint access only).
**Note on cookie attributes:** `Secure` cookies are **not stored by browsers over plain `http://` except for `localhost`** — `http://localhost` is treated as a secure context, so `Secure` works in Vite dev. In prod (HTTPS) it works normally. Use a config flag (`COOKIE_SECURE`) so the same code path works dev and prod. `[CITED: developer.mozilla.org/en-US/docs/Web/HTTP/Cookies — Secure + localhost exception]`

### Pattern 2: The 5-step auth middleware chain (D-14) — the single gate
**What:** One `AuthMiddleware` runs the ordered checks; `RequireRole` chains after it.
**When:** Every protected route group. This IS success criterion 3 (server-side forced-reset) and AUTH-04.
**Critical:** Step 5's allow-list is checked by route — the simplest lean approach is a small set of exempt paths `{POST /auth/change-password, POST /auth/logout, GET /auth/me}`; everything else returns `403` with code `password_change_required` when `must_change_password`.

### Pattern 3: RequireRole gate (AUTH-05 role half)
**What:** Middleware factory `RequireRole(roles ...db.UserRole)` reads the role placed in context by `AuthMiddleware` and 403s on mismatch.
**When:** Route registration (`admin := r.Group("/admin", AuthMiddleware(...), RequireRole(db.UserRoleAdmin))`).

### Pattern 4: Ownership = JWT-derived user_id (AUTH-05 ownership half)
**What:** `service.go` reads `user_id` from the request context (set by `AuthMiddleware`), never from the request body or a URL param.
**When:** Change-password (the only owned resource this phase). Establishes the pattern Phases 3–5 reuse.

### Anti-Patterns to Avoid
- **Putting `must_change_password` in the JWT claim.** Stale-claim bug; D-14 reads it live from the DB row already loaded.
- **Trusting `RoleGuard` (frontend) as the security boundary.** It is UX only; the BE `RequireRole` is the real gate (success criterion 4 — "bypassing the SPA does not unlock endpoints").
- **Wildcard `AllowOrigins: ["*"]` with `AllowCredentials: true`.** Illegal per CORS spec; browsers reject it. Must list the exact origin.
- **Rotating refresh tokens without a store.** Reuse-detection needs persistence; D-13 forbids a store → use reuse-until-expiry.
- **Re-issuing a session on password change.** D-16 forces re-login; do not special-case a token reissue.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JWT sign/verify | Custom HMAC + base64 token | golang-jwt/jwt/v5 | Algorithm-confusion, claim validation, exp/leeway handled correctly. |
| Password hashing | SHA/MD5 or custom salt scheme | `bcrypt` cost=12 | Adaptive, salted, constant-time compare (`CompareHashAndPassword`). |
| CORS preflight handling | Manual `OPTIONS` + header writes | gin-contrib/cors | Preflight + credentials + allow-list edge cases are subtle. |
| 401 refresh-retry on FE | ad-hoc `useEffect` retry | axios response interceptor (single shared in-flight promise) | Concurrency, single-retry guard, loop prevention (D-22). |
| Forms + validation | Controlled `useState` per field | react-hook-form + zod + zodResolver | Uncontrolled inputs, schema-as-types, shadcn `<Form>` integration. |
| SQL scan/mapping for users | Hand-written `rows.Scan` | sqlc-generated query methods | Type-safe, compile-time SQL/Go drift detection (already the project pattern). |
| Constant-time "password differs" check | `newPlain == oldPlain` | `bcrypt.CompareHashAndPassword(oldHash, []byte(new))` | You don't have the old plaintext; compare new against stored hash (D-19). |

**Key insight:** Every hard part of auth in this phase already has a locked, committed library. The only bespoke code is the *composition* — the 5-step middleware chain and the cookie/error-envelope conventions.

## Common Pitfalls

### Pitfall 1: `Secure` cookies silently dropped in dev
**What goes wrong:** Setting `Secure: true` over `http://` (non-localhost) makes the browser ignore the cookie; login "succeeds" but no session sticks.
**Why:** Browsers only store `Secure` cookies over HTTPS — with the sole exception of `http://localhost`.
**How to avoid:** Drive `Secure` from a `COOKIE_SECURE` config flag (true in prod, can stay true for `localhost`). Test against `http://localhost:5173` (Vite default), not a LAN IP.
**Warning signs:** `Set-Cookie` present in response but no cookie in subsequent requests.

### Pitfall 2: CORS credentials misconfig blocks every authenticated call
**What goes wrong:** `withCredentials: true` on the FE but BE sends `Access-Control-Allow-Origin: *` → browser blocks the response.
**Why:** The CORS spec forbids wildcard origin with credentials; the allow-credentials header requires an exact origin echo.
**How to avoid:** `cors.Config{ AllowOrigins: []string{cfg.FrontendOrigin}, AllowCredentials: true }`. Drive the origin from config (Vite dev origin vs prod). Include `OPTIONS` and the headers axios sends (`Content-Type`, plus `X-Requested-With` if you adopt the CSRF mitigation below).
**Warning signs:** Console "blocked by CORS policy: 'Access-Control-Allow-Credentials' ... wildcard".

### Pitfall 3: `iat`/`password_changed_at` check forgotten on the refresh token
**What goes wrong:** Access tokens get invalidated on password change but refresh tokens don't, so a 7-day-old refresh token resurrects a killed session.
**Why:** Easy to apply the timestamp check only in the access-token middleware path.
**How to avoid:** `POST /auth/refresh` must run the SAME load-user + `iat >= password_changed_at` + account-status checks before minting a new access token (see Open Item 2).
**Warning signs:** A user who changed their password can still refresh into a session with the old refresh cookie.

### Pitfall 4: Trusting a client-supplied user_id
**What goes wrong:** Change-password (or later, any owned resource) reads the target user from the request body/param → IDOR; user A changes user B's password.
**Why:** Convenience of `req.UserID`.
**How to avoid:** AUTH-05 rule — `user_id` ALWAYS from `c.Get("user_id")` (set by `AuthMiddleware` from the verified JWT `sub`), never from input. Establish this in `service.go` now.
**Warning signs:** Any `user_id` / `id` field in a request DTO for a self-scoped action.

### Pitfall 5: golang-jwt v5 algorithm-confusion (no `WithValidMethods`)
**What goes wrong:** Parser accepts a token signed with a different alg (e.g. `none` or RS256-vs-HS256 confusion).
**Why:** Without `WithValidMethods`, the keyfunc must defend itself and is easy to get wrong.
**How to avoid:** Always pass `jwt.WithValidMethods([]string{"HS256"})` AND assert the method inside the keyfunc. (See Code Examples 2.) `[CITED: pkg.go.dev/github.com/golang-jwt/jwt/v5]`
**Warning signs:** Keyfunc returns the secret without checking `token.Method`.

### Pitfall 6: shadcn/ui Tailwind v4 path on an existing Vite project
**What goes wrong:** `shadcn init` assumes a fresh project; on the existing starter, Tailwind v4's `@tailwindcss/vite` plugin + `@theme` CSS must be wired manually before components render.
**Why:** The current `frontend/vite.config.ts` has only `react()`; no Tailwind, no path alias.
**How to avoid:** Add `@tailwindcss/vite` to `vite.config.ts`, add the `@import "tailwindcss"` to the root CSS, set the `@/*` tsconfig path alias, then run `shadcn init`. `[CITED: ui.shadcn.com/docs/installation/vite]`
**Warning signs:** shadcn components render unstyled; `cn()`/alias import errors.

## Code Examples

### Example 1: JWT claims + mint (golang-jwt v5, HS256)
```go
// internal/shared/auth/jwt.go
// Source pattern: pkg.go.dev/github.com/golang-jwt/jwt/v5  [CITED]
package auth

import (
	"time"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims // sub, iat, exp
}

func Mint(secret []byte, userID int64, role string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10), // sub = user_id
			IssuedAt:  jwt.NewNumericDate(now),        // iat — compared to password_changed_at
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}
// access: ttl = 15*time.Minute ; refresh: ttl = 7*24*time.Hour
```

### Example 2: JWT validate (v5 stricter defaults + algorithm-confusion defense)
```go
// Source pattern: pkg.go.dev/github.com/golang-jwt/jwt/v5  [CITED]
func Parse(secret []byte, tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok { // assert HMAC
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return secret, nil
		},
		jwt.WithValidMethods([]string{"HS256"}), // algorithm-confusion defense
		jwt.WithLeeway(5*time.Second),           // small clock skew tolerance
	)
	// v5 returns wrapped sentinels: jwt.ErrTokenExpired, jwt.ErrTokenSignatureInvalid, etc.
	if err != nil {
		return nil, err // caller maps expiry → 401 (triggers FE refresh, D-22)
	}
	return claims, nil
}
```

### Example 3: gin-contrib/cors credentialed config (dev + prod via config)
```go
// internal/shared/middleware/cors.go
// Source: github.com/gin-contrib/cors  [CITED]
import "github.com/gin-contrib/cors"

func CORS(allowedOrigin string) gin.HandlerFunc { // e.g. "http://localhost:5173" (dev) or "https://myiu.example.edu" (prod)
	return cors.New(cors.Config{
		AllowOrigins:     []string{allowedOrigin}, // NEVER "*" with credentials
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"}, // + "X-Requested-With" if CSRF mitigation adopted
		AllowCredentials: true, // required so the browser sends/accepts cookies
		MaxAge:           12 * time.Hour,
	})
}
// Wire FRONTEND_ORIGIN into Config (config.go) so dev/prod differ by env only.
```

### Example 4: Cookie set/clear + bcrypt "must differ" (change-password)
```go
// handler.go — set both cookies (login / refresh)
func setAuthCookies(c *gin.Context, access, refresh string, secure bool) {
	// name, value, maxAge(s), path, domain, secure, httpOnly
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", access, 15*60, "/", "", secure, true)
	c.SetCookie("refresh_token", refresh, 7*24*60*60, "/", "", secure, true)
}
func clearAuthCookies(c *gin.Context, secure bool) { // logout & post-change (D-16)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", "", -1, "/", "", secure, true)
	c.SetCookie("refresh_token", "", -1, "/", "", secure, true)
}

// service.go — change password (D-17, D-18, D-19)
func (s *Service) ChangePassword(ctx context.Context, userID int64, cur, new string) error {
	u, err := s.repo.GetUserByID(ctx, userID) // userID from JWT, never from body
	if err != nil { return ErrNotFound }
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(cur)) != nil {
		return ErrCurrentPasswordWrong // D-17
	}
	if len(new) < 6 { return ErrTooShort } // D-18 (mirror Zod on FE)
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(new)) == nil {
		return ErrSameAsCurrent // D-19: new must differ (blocks re-setting default)
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(new), 12) // cost=12
	// repository: UPDATE users SET password_hash=$1, password_changed_at=now(),
	//             must_change_password=false, updated_at=now() WHERE id=$2
	return s.repo.UpdatePasswordAndStamp(ctx, userID, string(hash))
}
```

### Example 5: Migration 000003 (password_changed_at)
```sql
-- 000003_add_password_changed_at.up.sql
ALTER TABLE users ADD COLUMN password_changed_at TIMESTAMPTZ NOT NULL DEFAULT now();
-- Existing rows (bootstrap admin) get now(); any JWT minted before this is older → invalid. Fine.

-- 000003_add_password_changed_at.down.sql
ALTER TABLE users DROP COLUMN password_changed_at;
```
> After this migration, re-run `sqlc generate` so `db.User` gains `PasswordChangedAt pgtype.Timestamptz`. The middleware compares `claims.IssuedAt.Time` to `user.PasswordChangedAt.Time` (reject if `iat < password_changed_at`).

### Example 6: axios instance + single-flight 401 refresh interceptor (D-22)
```ts
// lib/api.ts — pattern; FE owns retry/cleanup/redirect
import axios from "axios";
export const api = axios.create({ baseURL: import.meta.env.VITE_API_URL, withCredentials: true });

let refreshing: Promise<void> | null = null;
api.interceptors.response.use(
  (r) => r,
  async (error) => {
    const original = error.config;
    if (error.response?.status === 401 && !original._retry) {
      original._retry = true;                    // at most one refresh per request
      try {
        refreshing ??= api.post("/auth/refresh").then(() => { refreshing = null; }); // single in-flight
        await refreshing;
        return api(original);                     // retry once
      } catch {
        refreshing = null;
        useAuth.getState().clear();               // clear Zustand + redirect /login
        window.location.assign("/login");
      }
    }
    return Promise.reject(error);
  }
);
```

### Example 7: sqlc user queries
```sql
-- backend/db/queries/users.sql
-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1 AND deleted_at IS NULL;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdatePasswordAndStamp :exec
UPDATE users
SET password_hash = $2, password_changed_at = now(), must_change_password = false, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;
```

## Open Research Items (the four priority deliverables)

### Open Item 1 — CSRF protection for cookie auth → **NO dedicated CSRF token for this MVP**
**Recommendation:** Do **not** implement a double-submit CSRF token. Rely on the layered defenses already present:
1. **SameSite=Lax cookies (D-11).** Lax blocks cookies on cross-site *sub-requests* (the classic CSRF vector: `<form>`/`<img>`/`fetch` from an attacker page). It allows cookies only on top-level GET navigations — and all state-changing endpoints here are POST, which Lax does not send cross-site. This alone defeats the standard CSRF attack.
2. **Strict credentialed CORS (D-11).** `AllowOrigins: [exact origin]` + `AllowCredentials: true` means the browser will not expose responses to any other origin's JS.
3. **JSON content-type requirement.** State-changing endpoints accept only `application/json`. A cross-site HTML form can only send `application/x-www-form-urlencoded` / `multipart` / `text/plain` without triggering a CORS preflight — so a forged form can't hit a JSON endpoint, and a `fetch` with JSON triggers a preflight that strict CORS blocks. **Action for planner:** have handlers reject non-JSON content-type on state-changing routes (Gin `ShouldBindJSON` already requires it; make the rejection explicit).

**Why this is sufficient (and lean):** The double-submit token exists primarily to defend apps that must support SameSite=None or legacy browsers. This is an internal university SPA on modern browsers with SameSite=Lax + strict CORS — the token would be defense-in-depth with real plumbing cost (extra cookie, FE header wiring, middleware), violating Ponytail for negligible marginal security here.
**Dev-vs-prod note:** Vite dev origin (`http://localhost:5173`) and the API (`http://localhost:8080`) are *different origins* but *same site is irrelevant* — what matters is SameSite is computed on the registrable domain; cross-port on localhost is same-site, so Lax behaves consistently. Prod (same parent domain or distinct subdomains) keeps Lax semantics. No CSRF token needed in either.
**If a reviewer later mandates CSRF:** minimal add = a non-HttpOnly `csrf_token` cookie mirrored into an `X-CSRF-Token` header by axios, compared server-side (double-submit). Add `X-Requested-With`/`X-CSRF-Token` to CORS `AllowHeaders` then. **Tagged `[ASSUMED]` — confirm acceptability with the user before locking out the token.**
Confidence: **MEDIUM** (well-established OWASP guidance, but the "no token" call is a judgment trade-off — flagged in Assumptions Log). `[CITED: OWASP CSRF Prevention Cheat Sheet — SameSite + double-submit]`

### Open Item 2 — Refresh-token rotation → **reuse-until-expiry, NOT rotate-on-use**
**Recommendation:** The 7-day refresh token is **reused until it expires**. `POST /auth/refresh` validates the refresh JWT and mints a *new access token only* — the refresh cookie is left as-is (or re-set with the same value/remaining life; either is fine).
**Why not rotate-on-use:** Rotation's value is *reuse detection* — if an old (rotated-out) refresh token is presented, you revoke the whole chain. Detecting reuse **requires server-side state** (a stored token id / family / hash). D-13 explicitly forbids any server store (no blacklist, no session table, no Redis). Without a store, "rotation" is rotation-in-name-only with no reuse detection — pure overhead. So reuse-until-expiry is the honest, lean choice given D-13.
**The `iat`/`password_changed_at` invalidation MUST apply to the refresh token too (D-13).** `POST /auth/refresh` runs the same chain steps as `AuthMiddleware` before minting:
1. verify refresh JWT signature + expiry (`WithValidMethods(["HS256"])`),
2. load the user (`sub`),
3. account status (`deleted_at IS NULL`),
4. **`refreshClaims.iat >= user.password_changed_at`** — else reject `401` (this is what makes password change / admin reset (Phase 3 ADMIN-04) revoke the long-lived refresh token without a store),
5. mint a fresh 15-min access token; do not bypass `must_change_password` (a locked account refreshing still lands on the allow-list).
**Net:** password change/reset bumps `password_changed_at` → both the old access token AND the old refresh token fail step 4 → session fully dead, statelessly. This is the load-bearing reason D-13's primitive must cover refresh.
Confidence: **HIGH** (direct consequence of locked D-13; no external dependency).

### Open Item 3 — golang-jwt v5 claims setup → see Code Examples 1 & 2
**Registered claims:** `sub` = user_id (string), `iat` = issue time (compared to `password_changed_at`), `exp` = `iat + ttl`. Custom claim: `role`. **Do NOT put `must_change_password` in the token** (D-14 — read live from DB).
**Signing:** `jwt.SigningMethodHS256` with `[]byte(cfg.JWTSecret)`; `jwt.NewWithClaims(...).SignedString(secret)`.
**Validation under v5 stricter defaults:**
- `jwt.ParseWithClaims(str, &Claims{}, keyfunc, opts...)`.
- **`jwt.WithValidMethods([]string{"HS256"})`** — mandatory (algorithm-confusion defense, Pitfall 5).
- Assert `*jwt.SigningMethodHMAC` inside the keyfunc as belt-and-suspenders.
- `jwt.WithLeeway(5*time.Second)` for minor clock skew (optional but recommended).
- v5 validates `exp`/`nbf`/`iat` automatically and returns wrapped sentinel errors (`errors.Is(err, jwt.ErrTokenExpired)`) — map expired → `401` so the FE interceptor fires the refresh (D-22).
Confidence: **HIGH** `[CITED: pkg.go.dev/github.com/golang-jwt/jwt/v5]` (API confirmed via web + matches v5.3.1 in CLAUDE.md).

### Open Item 4 — CORS credentials config → see Code Example 3
**Config:** `cors.Config{ AllowOrigins: []string{cfg.FrontendOrigin}, AllowCredentials: true, AllowMethods: [...,"OPTIONS"], AllowHeaders: ["Origin","Content-Type","Accept"] }`. **Wildcard `*` is illegal with credentials** — list the exact origin.
**Dev:** `FRONTEND_ORIGIN=http://localhost:5173` (Vite default). **Prod:** the deployed SPA origin (e.g. `https://myiu.example.edu`). Drive it from a new `FrontendOrigin` field in `Config` (env `FRONTEND_ORIGIN`) so only env differs between dev/prod.
**FE side:** axios instance with `withCredentials: true` (Code Example 6); without it the browser neither sends nor stores the cookies.
Confidence: **HIGH** `[CITED: github.com/gin-contrib/cors]`.

## State of the Art

| Old Approach | Current Approach | Why |
|--------------|------------------|-----|
| JWT in localStorage | HttpOnly cookie (D-11) | XSS can't read HttpOnly cookies; localStorage tokens are XSS-exfiltratable. |
| Password complexity rules | Length-only minimum (D-18) | Modern NIST/OWASP guidance favors length over composition. |
| golang-jwt v4 lax parsing | v5 `WithValidMethods` + wrapped errors | v5 hardened validation defaults; v4 in maintenance. |
| Stateful session/blacklist for logout | `password_changed_at` timestamp check (D-13) | Stateless invalidation without Redis/session store at this scale. |

**Deprecated/outdated (do not use):** `dgrijalva/jwt-go` and golang-jwt v4 (use v5); `lib/pq` (use pgx, already locked); localStorage token storage (D-11 rejects it).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | No CSRF token is acceptable for this MVP (SameSite=Lax + strict CORS + JSON-only suffices) | Open Item 1 | If a security review mandates CSRF tokens, planner must add a double-submit token cookie + `X-CSRF-Token` header + middleware. Confirm with user. |
| A2 | gin-contrib/cors resolves to v1.7.x via `@latest`; `cors.Config{AllowOrigins,AllowCredentials}` API stable | Standard Stack | If a newer major changed the API, the CORS example needs updating. Low risk — API stable for years. |
| A3 | Reuse-until-expiry refresh is acceptable (no rotation) given D-13's no-store mandate | Open Item 2 | If stolen-refresh-token detection becomes a requirement, a server store (contradicting D-13) would be needed. |
| A4 | `Secure` cookie + `http://localhost` works in Vite dev (localhost secure-context exception) | Pattern 1 / Pitfall 1 | If devs use a non-localhost LAN IP, `Secure` cookies drop; doc note added. |

## Open Questions (RESOLVED)

1. **`FRONTEND_ORIGIN` env not yet in `Config`.**
   - Known: CORS needs an exact origin; `Config` currently has `DatabaseURL/JWTSecret/CloudinaryURL/Port` only.
   - Unclear: exact prod origin value (deployment not defined yet).
   - Recommendation: planner adds `FrontendOrigin string \`env:"FRONTEND_ORIGIN" envDefault:"http://localhost:5173"\`` and a `CookieSecure bool \`env:"COOKIE_SECURE" envDefault:"true"\`` to `Config`.
   - **RESOLVED:** Plan 02-01 Task 1 adds both `FrontendOrigin` and `CookieSecure` fields to `Config` with the defaults above; prod origin stays env-driven (no hardcode).
2. **Where to register the public vs protected route groups in `main.go`.**
   - Known: `main.go` currently only calls `health.RegisterRoutes`; CORS must be `router.Use(...)` before any routes.
   - Recommendation: `auth.RegisterRoutes(router, pool, cfg)` registers `/auth/login` + `/auth/refresh` (public) and a protected group wrapped in `AuthMiddleware`; later phases register their groups behind the same middleware.
   - **RESOLVED:** Plan 02-01 Task 3 wires `auth.RegisterRoutes(router, pool, cfg)` after `router.Use(CORS)`, registering public `/auth/login` + `/auth/refresh` and a protected group behind `AuthMiddleware`.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | backend build/test | ✓ | 1.24.0 (go.mod) | — |
| Docker + Postgres 17 | integration tests, local DB | ✓ (Phase 1 verified, D-08) | postgres:17-alpine | — |
| golang-migrate CLI | migration 000003 | ✓ (Phase 1 / CI) | v4.18.x | — |
| sqlc | regenerate user models | ✓ (Phase 1 configured) | v1.31.1 | — |
| Node + npm | frontend installs | ✓ (frontend builds) | Vite 6 / React 19 present | — |

**Missing dependencies with no fallback:** none — all tooling proven in Phase 1.
**New env vars to add:** `FRONTEND_ORIGIN`, `COOKIE_SECURE`, `VITE_API_URL` (frontend) — config additions, not missing tools.

## Project Constraints (from CLAUDE.md)

- **Stack is LOCKED** — use the committed versions; do not propose alternatives (Gin v1.11.0, pgx v5.7.2, sqlc v1.31.1, golang-jwt v5.3.1, bcrypt cost=12, gin-contrib/cors; React 19/Vite 6/Zustand v5/TanStack Query v5/shadcn/ui + Tailwind v4/RHF v7/Zod v4/react-router v7/axios).
- **Feature-Oriented Monolith (D-10)** — auth code in `internal/auth/` with handler (HTTP only) / service (business + authz/ownership) / repository (SQL only) split; cross-cutting middleware + JWT helpers in `internal/shared/`.
- **No hand-rolled UI components** — shadcn/ui only.
- **Mirror critical validation in Go** — never trust client-side Zod alone (D-18 length rule enforced server-side too).
- **bcrypt cost=12**; **never** MD5/SHA for passwords.
- **GitHub Flow** — work on the `ft/` branch (already on `ft/phase-2-auth`), squash-merge via PR; CI (unit + integration vs real Postgres + migrations + lint) must pass.
- **Incremental migrations (D-06)** — append `000003_*`, CI runs migrations before tests.
- **GitNexus** — run `impact` before editing existing symbols (`main.go` wiring, `config.go`, `health.RegisterRoutes` pattern), `detect_changes` before committing.

## Security Domain

> `security_enforcement: true`, ASVS level 1, block_on: high. Included.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | bcrypt cost=12 verify; generic login errors (no enumeration); forced first-login reset. |
| V3 Session Management | yes | HttpOnly+Secure+SameSite=Lax cookies; short-lived access (15m) + refresh (7d); `password_changed_at` invalidation; logout clears cookies. |
| V4 Access Control | yes | `RequireRole` middleware (role); `user_id` from JWT (ownership, never client-supplied); must_change_password allow-list. |
| V5 Input Validation | yes | Zod (FE) + Go binding/validation (BE) on login + change-password DTOs; JSON-only content-type. |
| V6 Cryptography | yes | golang-jwt HS256 with `WithValidMethods` (algorithm-confusion defense); `JWT_SECRET` from env; bcrypt for storage. Never hand-roll. |

### Known Threat Patterns for Go/Gin + cookie-JWT SPA

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| JWT algorithm confusion (`alg=none`/HS-vs-RS) | Spoofing/Tampering | `WithValidMethods(["HS256"])` + assert HMAC in keyfunc (Pitfall 5). |
| XSS token theft | Information Disclosure | HttpOnly cookies (D-11) — JS cannot read the token. |
| CSRF on state-changing routes | Tampering | SameSite=Lax + strict credentialed CORS + JSON-only (Open Item 1). |
| IDOR via client-supplied user_id | Elevation of Privilege | `user_id` from verified JWT only (Pitfall 4, AUTH-05). |
| Forced-reset bypass via direct API call | Elevation of Privilege | Server-side allow-list in middleware, not SPA routing (D-15, success criterion 3). |
| Session survival after password change | Spoofing | `iat < password_changed_at` check on BOTH access and refresh tokens (Open Item 2, Pitfall 3). |
| User enumeration via login errors | Information Disclosure | Generic "Invalid username or password" (locked discretion). |
| `Secure` cookie dropped over http | Info Disclosure / breakage | `Secure` flag + localhost secure-context; `COOKIE_SECURE` config (Pitfall 1). |

## Sources

### Primary (HIGH confidence)
- `backend/go.mod`, `backend/sqlc.yaml`, `backend/internal/shared/{config,db,health}` — verified installed stack, sqlc config, route-registration pattern, User model.
- `.claude/CLAUDE.md` — locked stack + versions + auth Stack Patterns (carries Phase-0 HIGH-confidence registry sources).
- `.planning/phases/02-.../02-CONTEXT.md` — locked decisions D-11…D-22.
- [pkg.go.dev/github.com/golang-jwt/jwt/v5](https://pkg.go.dev/github.com/golang-jwt/jwt/v5) — RegisteredClaims, ParseWithClaims, WithValidMethods.

### Secondary (MEDIUM confidence)
- [github.com/golang-jwt/jwt — example_test.go](https://github.com/golang-jwt/jwt/blob/main/example_test.go) — claims + parser patterns.
- [github.com/gin-contrib/cors](https://github.com/gin-contrib/cors) — AllowOrigins + AllowCredentials config.
- OWASP CSRF Prevention Cheat Sheet (SameSite + double-submit guidance) — CSRF reasoning.
- [ui.shadcn.com/docs/installation/vite](https://ui.shadcn.com/docs/installation/vite) — Tailwind v4 + Vite wiring.

### Tertiary (LOW confidence)
- gin-contrib/cors exact latest version (v1.7.6) — resolve via `go get @latest`; API stable.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — fully locked/pinned in CLAUDE.md, verified in go.mod; nothing to choose.
- Architecture: HIGH — D-11…D-22 lock the design; research only composes them.
- golang-jwt v5 / CORS code: HIGH — confirmed via official docs + matches pinned versions.
- CSRF decision: MEDIUM — sound OWASP-backed judgment call, flagged in Assumptions Log (A1) for user confirmation.
- Refresh rotation decision: HIGH — direct consequence of locked D-13.

**Research date:** 2026-06-20
**Valid until:** 2026-07-20 (stable libraries; ~30 days)
