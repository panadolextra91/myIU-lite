# Phase 2: Auth, RBAC & Forced First-Login - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-20
**Phase:** 2-Auth, RBAC & Forced First-Login
**Areas discussed:** Session & logout, Forced reset, Mật khẩu (password rules), Auth UI & routing

---

## Session & logout

### Token storage (→ D-11)
| Option | Description | Selected |
|--------|-------------|----------|
| localStorage + Authorization header (Recommended) | axios interceptor attaches Bearer; matches CLAUDE.md pattern; XSS can read token | |
| HttpOnly cookie | JS can't read token → XSS-resistant; needs CORS credentials + CSRF | ✓ |

**User's choice:** HttpOnly + Secure + SameSite=Lax cookies; axios `withCredentials:true`; credential-aware CORS. Reasoning: business app with grades/admin actions warrants XSS-resistant token storage; CSRF/CORS complexity accepted.

### Token lifetime (→ D-12)
| Option | Description | Selected |
|--------|-------------|----------|
| 1 short access token, no refresh (Recommended) | 12–24h, re-login on expiry; leanest, stateless | |
| Access + refresh token | Short access + long refresh; smoother UX, more moving parts | ✓ |

**User's choice:** Access token 15 min + refresh token 7 days (both cookies). Implies `POST /auth/refresh`.

### Logout (→ D-13)
| Option | Description | Selected |
|--------|-------------|----------|
| Client clears token (Recommended) | Stateless; token self-expires; no Redis | ✓ |
| Server revoke (blacklist) | Immediate invalidation; needs store + per-request check | |

**User's choice:** `POST /auth/logout` clears cookies, no blacklist. Added `password_changed_at` on `users`; reject JWT when `iat < password_changed_at` to invalidate old sessions statelessly. Non-goals: blacklist, Redis, session table, device mgmt, global logout.

---

## Forced reset

### Enforcement location (→ D-14)
| Option | Description | Selected |
|--------|-------------|----------|
| Live DB flag in middleware (Recommended) | Read `must_change_password` from the user row already loaded for `password_changed_at`; always live | ✓ |
| `must_change_password` in JWT claim | No extra read but stale until reissue | |

**User's choice:** Live DB flag in auth middleware. JWT = identity; mutable state = DB. Middleware order: verify JWT → load user → account status → `password_changed_at` → `must_change_password`.

### Locked-state allow-list (→ D-15)
| Option | Description | Selected |
|--------|-------------|----------|
| change-password + logout + me only (Recommended) | Strict; everything else 403; matches criterion #3 | ✓ |
| Add some read-only pages | Softer but widens attack surface | |

**User's choice:** Only `POST /auth/change-password`, `POST /auth/logout`, `GET /auth/me`; all else 403. FE calls `/auth/me`, redirects to `/change-password` when flagged.

### Post-change session (→ D-16)
| Option | Description | Selected |
|--------|-------------|----------|
| Auto-issue new cookies, continue (Recommended) | Seamless, lands on dashboard | |
| Invalidate + force re-login | Clear cookies, back to /login with new password | ✓ |

**User's choice:** Update hash + `password_changed_at` + `must_change_password=false` → clear cookies → 200 "please log in again" → FE redirect `/login`. No auto-reissue. (Overrode the recommended option — credential change should establish a new session.)

---

## Mật khẩu (password rules)

### Require current password (→ D-17)
| Option | Description | Selected |
|--------|-------------|----------|
| Require current (Recommended) | Verify current before change; blocks stolen-session change | ✓ |
| New only | Simpler, weaker | |

**User's choice:** Require `current_password`; fields `current_password` / `new_password` / `confirm_password`.

### Strength (→ D-18)
| Option | Description | Selected |
|--------|-------------|----------|
| Min length ≥8 only (Recommended) | Simple, length over composition | |
| Min ≥8 + complexity | Stronger on paper, more friction | |
| You decide | Planner picks OWASP baseline | |

**User's choice:** Min length **6**, no complexity (overrode the ≥8 recommendation) — friction-free, matches bootstrap `123456`.

### No reuse of current (→ D-19)
| Option | Description | Selected |
|--------|-------------|----------|
| New must differ from current (Recommended) | Blocks re-setting the default; bcrypt compare | ✓ |
| No restriction | Simplest, defeats forced change | |

**User's choice:** New ≠ current (blocks reusing default birthday); no password history.

---

## Auth UI & routing

### Role landing (→ D-20)
| Option | Description | Selected |
|--------|-------------|----------|
| One role-aware /dashboard (Recommended) | Single route, render by role; fewer dup shells | |
| Per-role route trees (/student, /lecturer, /admin) | Clear boundaries, separate trees | ✓ |

**User's choice:** `/student/*`, `/lecturer/*`, `/admin/*`; redirect to role root after login. Separate roles at the routing boundary, not via in-page conditionals.

### App-shell scope (→ D-21)
| Option | Description | Selected |
|--------|-------------|----------|
| Minimal: login + change-pw + empty landings (Recommended) | Auth spine + ProtectedRoute/RoleGuard; defer nav | ✓ |
| Full sidebar shell now | Pretty early but links to nonexistent pages | |

**User's choice:** Login, Change-Password, 3 role landings, `AppLayout`, `ProtectedRoute`, `RoleGuard`. No sidebar/nav/placeholder pages until real features exist.

### 401 handling (→ D-22)
| Option | Description | Selected |
|--------|-------------|----------|
| Interceptor auto-refresh + retry (Recommended) | One refresh per 401, retry, fallback logout | ✓ |
| Logout on any 401 | Simple but negates refresh token | |

**User's choice:** axios interceptor: 401 → one `/auth/refresh` → retry once; fail → clear + `/login`. Guardrails: single refresh per request, no loops, shared in-flight refresh for concurrent 401s.

---

## Claude's Discretion
- Backend role-gate middleware shape + consistent `403` JSON error envelope (machine-readable codes so FE distinguishes "change password required" from role/owner rejection).
- Ownership in Phase 2 = self-only; `user_id` always from JWT, never client-supplied (AUTH-05 pattern).
- `GET /auth/me` returns `{ id, username, role, must_change_password }`.
- Generic login errors (no user enumeration); no lockout/rate-limit in MVP.
- Auth code in `internal/auth/` (handler/service/repository/model/dto); user sqlc queries in `backend/db/queries/`.

## Deferred Ideas
- Account lockout / login rate-limiting (revisit on abuse).
- Self-service password recovery (AUTH-V2-01; admin reset in Phase 3 per D-01).
- JWT blacklist / Redis / session table / device mgmt / global logout (non-goals per D-13).
- Password history (out of scope per D-19).
- Full sidebar navigation + feature menus + placeholder pages (Phase 3+ per D-21).
