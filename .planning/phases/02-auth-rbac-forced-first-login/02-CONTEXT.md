# Phase 2: Auth, RBAC & Forced First-Login - Context

**Gathered:** 2026-06-20
**Status:** Ready for planning

<domain>
## Phase Boundary

The authentication + authorization spine every later route hangs off. Delivers AUTH-01 → AUTH-05:

- Username + password login that issues a role-carrying JWT, and logout.
- A logged-in user can change their own password.
- A user flagged `must_change_password` is **server-side** restricted to change-password / logout / who-am-I until they reset — bypassing the SPA does not unlock other endpoints.
- Every route is gated by **role** (student / lecturer / admin) and by **ownership** (records the caller owns), never trusting client-supplied IDs → `403` otherwise.

This phase is the auth/RBAC foundation only. No admin provisioning / CSV / audit-log writing (Phase 3), no feature tables or pages (Phases 3–5). The `users` table + bootstrap admin already exist from Phase 1; this phase adds the auth feature folder, the auth middleware, one migration (`password_changed_at`), and the minimal frontend auth shell.

</domain>

<decisions>
## Implementation Decisions

Decision IDs continue the project sequence (PROJECT.md ends at D-10; Phase 1 CONTEXT used D-06…D-09). D-11…D-22 are new and owned by this phase.

### Session & token model
- **D-11 — Cookie storage, not localStorage.** Tokens live in **HttpOnly + Secure + SameSite=Lax cookies**. Frontend uses axios with `withCredentials: true`; backend enables credential-aware CORS (specific origin, `Access-Control-Allow-Credentials: true`). Rationale: myIU holds grades/submissions/admin actions; HttpOnly blocks JS from reading the token, shrinking XSS impact. localStorage is intentionally avoided.
- **D-12 — Access + refresh tokens.** **Access token = 15 minutes**, **refresh token = 7 days**, both HttpOnly+Secure+SameSite=Lax cookies. Implies a `POST /auth/refresh` endpoint that mints a new access token from a valid refresh token.
- **D-13 — Stateless logout via `password_changed_at`, no blacklist.** `POST /auth/logout` clears both cookies; FE clears app state + redirects to `/login`. **No** JWT blacklist, Redis revocation store, session table, device management, or global-logout-across-devices. Immediate invalidation of old sessions is achieved by a **`password_changed_at` timestamp on `users`**: any JWT whose `iat` is older than `password_changed_at` is rejected at validation. So password change / admin reset / forced reset all kill prior sessions without a server-side store. (NB: validating `iat < password_changed_at` requires the auth middleware to load the user row per request — accepted; "stateless" here means no session store, not no DB read.)

### Forced first-login reset (ROADMAP existential pitfall #1)
- **D-14 — Enforce via live DB flag in middleware, not a JWT claim.** The auth middleware already loads the user row (for account-status + `password_changed_at` validation), so it reads `must_change_password` from that same row — no extra query, always live, no stale-claim/reissue problems. JWT carries identity (`user_id`, `role`); mutable account state is read from the DB. Middleware order: (1) verify JWT signature + expiry, (2) load user, (3) validate account status (not soft-deleted), (4) validate `password_changed_at`, (5) check `must_change_password`.
- **D-15 — Locked-state allow-list = exactly three endpoints.** When `must_change_password = true`, only `POST /auth/change-password`, `POST /auth/logout`, `GET /auth/me` are reachable; **every other authenticated endpoint returns `403`** with an application error code signalling "password change required". FE calls `GET /auth/me` after login; if `must_change_password = true` it immediately redirects to `/change-password` and keeps the user there until success or logout. A default-password account cannot reach courses/assignments/quizzes/grades/announcements/requests/admin until it changes its password.
- **D-16 — Password change ends the session; force re-login.** On success: update password hash → update `password_changed_at` → set `must_change_password = false` → **clear both cookies** → return `200 {"message": "Password changed successfully. Please log in again."}`. FE redirects to `/login`. The system does **not** auto-issue a new session (no special-case token reissuance); a new credential establishes a new session. Acceptable because the forced change happens once at onboarding.

### Password-change rules (mirror Zod on FE + Go on BE)
- **D-17 — Require current password.** Body fields: `current_password`, `new_password`, `confirm_password`. Verify `current_password` against the stored hash before accepting the change. Possession of a session is not enough — the user must prove knowledge of the current credential. Works naturally with forced first-login (the user just logged in with that password).
- **D-18 — Minimum length 6, no complexity rules.** No required uppercase/lowercase/digit/special-char composition. Friction-free for an internal university lite app with admin-assisted recovery; also consistent with the bootstrap admin password `123456`. (Modern guidance favours length over composition.)
- **D-19 — New password must differ from current; no password history.** Enforced by `bcrypt.CompareHashAndPassword(currentHash, new_password)` returning *not-equal*. This is what prevents re-setting the default birthday password (at forced change, current *is* the default). No "last N passwords" history store (out of scope for MVP).

### RBAC + ownership (AUTH-05)
- **D-20 — Separate role route trees on the frontend.** `/student/*`, `/lecturer/*`, `/admin/*`; after login the FE redirects to the role's root. Each tree owns its own sidebar/nav/feature pages (added in later phases); shared shadcn/ui primitives stay reusable. Roles are separated at the routing boundary instead of branching on role inside pages.
- **D-21 — Minimal Phase-2 app shell only.** Phase 2 ships exactly: Login page, Change-Password page, role landing pages (`/student`, `/lecturer`, `/admin`), plus shared `AppLayout` foundation, `ProtectedRoute`, and `RoleGuard`. **Out of scope this phase:** sidebar navigation, feature menus, placeholder feature pages, empty links to future functionality. Navigation arrives when real feature pages exist (Phase 3+).
- **D-22 — Transparent 401 refresh on the frontend.** An axios response interceptor catches `401` → calls `POST /auth/refresh` **once** → on success retries the original request once; on failure clears auth state + redirects to `/login`. Guardrails: at most one refresh per failed request, no infinite refresh loops, concurrent 401s share a single in-flight refresh, and a `401/403` from the refresh endpoint itself means the session is expired. Backend owns token validation/issuance/expiry; frontend owns the interceptor/retry/cleanup/redirect.

### Claude's Discretion (settled without a user question — locked by D-10 architecture + decisions above)
- **Backend role-gate middleware shape + 403 error envelope.** Role gating lives in `internal/shared/middleware` (per D-10); a `RequireRole(...)` style guard plus a consistent JSON error envelope (e.g. `{ "error": { "code": ..., "message": ... } }`). Planner picks the exact shape; the forced-change block (D-15) and role/ownership rejections (AUTH-05) all return `403` with a machine-readable code so the FE can distinguish "change password required" from "wrong role/owner".
- **Ownership in Phase 2 = self-only.** The only owned resource this phase is the user's own password. `user_id` is always derived from the JWT in `service.go`, never from a request body/param — this is the AUTH-05 "never trust client-supplied IDs" rule, established now as the pattern later phases reuse.
- **`GET /auth/me`** returns `{ id, username, role, must_change_password }` (minimal identity + the forced-change flag the FE routes on).
- **Login error responses are generic.** "Invalid username or password" with no user-enumeration signal; no account lockout / login rate-limiting in the MVP (see Deferred).
- **Auth feature lives in `internal/auth/`** (`handler.go` / `service.go` / `repository.go` / `model.go` / `dto.go`) per D-10; sqlc queries for users go in `backend/db/queries/`.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project-level (locked stack, constraints & architecture)
- `/.claude/CLAUDE.md` — committed stack + versions and the auth-relevant **Stack Patterns**: short-lived JWT (golang-jwt v5) carrying `user_id` + `role`, Gin middleware gating routes by role; bcrypt **cost=12**; gin-contrib/cors for the SPA; sqlc v1.31.1 + pgx v5.7.x; golang-migrate v4.18 owns schema. Authoritative for all library choices this phase.
- `/.planning/PROJECT.md` — vision, constraints, Key Decisions table (incl. **D-01** no self-service forgot-password — admin resets in Phase 3; **D-10** Feature-Oriented Monolith layout + handler/service/repository split).
- `/.planning/REQUIREMENTS.md` §"Authentication & Accounts (AUTH)" — AUTH-01 → AUTH-05 acceptance wording.
- `/.planning/ROADMAP.md` §"Phase 2: Auth, RBAC & Forced First-Login" — goal + 4 success criteria (incl. the "bypassing the SPA does not unlock other endpoints" enforcement bar).

### Design (frontend)
- `/.planning/DESIGN-SYSTEM.md` (D-05) — global UI ruleset: shadcn/ui only (no hand-rolled components), light+dark themes, 6px radius, Lucide icons, Skeleton loaders, WCAG AA, expandable sidebar (sidebar itself is **deferred** per D-21, but Login/Change-Password/landing pages conform to this file).

### Prior phase context
- `/.planning/phases/01-foundation-data-core/01-CONTEXT.md` — Phase 1 decisions: **D-06** incremental per-phase migrations (Phase 2 appends its own), **D-07** bootstrap admin `admin/123456` seeded bcrypt-hashed with `must_change_password=TRUE` (so admin hits forced reset on first login), **D-08** Docker = Postgres-only.

### Existing code to read (Phase 1 output)
- `/backend/db/migrations/000001_init_foundation.up.sql` — current `users` schema (`id`, `username`, `password_hash`, `role` enum, `must_change_password`, `created_at`, `updated_at`, `deleted_at`) + `audit_log`. **Phase 2 adds migration `000003` for `password_changed_at TIMESTAMPTZ` on `users`.**
- `/backend/db/migrations/000002_seed_bootstrap_admin.up.sql` — seeded admin row.
- `/backend/internal/shared/db/models.go` — sqlc-generated `User` / `UserRole` types to reuse.
- `/backend/internal/shared/config/config.go` — `Config` struct (`JWTSecret` slot already present; consumed this phase).
- `/backend/cmd/api/main.go` — wiring/entrypoint where the auth routes + middleware get registered (currently only `health.RegisterRoutes`).

No external ADRs beyond the above — auth requirements are fully captured in the decisions here + the locked stack in CLAUDE.md.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`users` + `audit_log` tables already exist** (Phase 1 migration `000001`). No new auth table needed — only the `password_changed_at` column add.
- **Bootstrap admin already seeded** (`admin` / `123456`, `must_change_password=TRUE`) — gives login + forced-reset something to exercise immediately, no provisioning needed.
- **sqlc `User`/`UserRole` models** in `internal/shared/db/models.go` — reuse directly; add user queries under `backend/db/queries/`.
- **Config + pgx pool wiring** in `config.go` / `main.go` — `JWTSecret` already loaded from `.env`; auth handlers register onto the existing Gin router + pgx pool.
- **Frontend is still the Vite starter** (`frontend/src/App.tsx` is the demo) with **no** Zustand / TanStack Query / react-router / shadcn / axios installed yet — this phase installs and wires them per CLAUDE.md's frontend stack.

### Established Patterns
- **Feature-Oriented Monolith (D-10):** new code goes in `internal/auth/` (handler=HTTP only, service=business+authz/ownership, repository=SQL only); cross-cutting auth middleware + JWT helpers + CORS go in `internal/shared/middleware` / `internal/shared/auth`.
- **Incremental migrations (D-06):** append `000003_*.up/down.sql`; CI runs migrations before tests.
- **Stack patterns pre-locked in CLAUDE.md:** golang-jwt v5 claims, bcrypt cost=12, gin-contrib/cors.

### Integration Points
- Auth middleware becomes the gate **every later phase's routes** register behind (role + ownership).
- `password_changed_at` column + the `iat`-vs-`password_changed_at` check is the session-invalidation primitive Phase 3's admin password-reset (ADMIN-04) will rely on.
- The `403` + machine-readable error envelope established here is the contract the FE (and later phases) decode.
- Frontend `ProtectedRoute` / `RoleGuard` / `AppLayout` are the shells later feature pages mount inside.

</code_context>

<specifics>
## Specific Ideas

- Cookie auth chosen *because* the app carries grades/admin actions (D-11) — a deliberate security-over-simplicity call by the user, accepting CORS-credentials + CSRF complexity.
- The `password_changed_at` "invalidate-by-timestamp" trick (D-13) is the user's idea to keep logout/reset stateless without Redis or a blacklist — preserve it; it is load-bearing for D-16 and Phase 3 admin resets.
- Min-length **6** (D-18), not 8 — explicitly to match the bootstrap `123456` and keep onboarding friction-free.
- Roles split at the **routing boundary** (D-20), not via in-page conditionals — a structural preference to carry into later phases.

</specifics>

<deferred>
## Deferred Ideas

- **Account lockout / login rate-limiting / brute-force throttling** — not in MVP; revisit if abuse appears. (Login errors stay generic to avoid user enumeration in the meantime.)
- **Self-service password recovery** (AUTH-V2-01) — out of scope by D-01; admin reset (ADMIN-04) covers it in Phase 3. No email/SMS channel exists.
- **JWT blacklist / Redis revocation / session table / device management / global logout across devices** — explicitly non-goals (D-13).
- **Password history ("last N passwords")** — out of scope (D-19).
- **Full sidebar navigation + feature menus + placeholder pages** — deferred to Phase 3+ when real feature pages exist (D-21).

None of these are scope creep into Phase 2 — discussion stayed within the auth/RBAC boundary.

## Research items for gsd-phase-researcher
- **CSRF protection for cookie auth** — with SameSite=Lax + cross-origin SPA, decide whether a CSRF token (double-submit / `X-CSRF-Token`) is needed for state-changing requests, and how it interacts with the Vite dev origin vs prod.
- **Refresh-token rotation** — decide rotate-on-use vs reuse-until-expiry for the 7-day refresh token, and how the `iat`-vs-`password_changed_at` check applies to refresh tokens too.
- **golang-jwt v5 claims setup** — exact registered claims (`sub`/`iat`/`exp`), signing method (HS256 with `JWT_SECRET`), and validation config under v5's stricter defaults.
- **CORS credentials config** — gin-contrib/cors with explicit allowed origin + `AllowCredentials: true` (wildcard origin is incompatible with credentials).

</deferred>

---

*Phase: 2-Auth, RBAC & Forced First-Login*
*Context gathered: 2026-06-20*
