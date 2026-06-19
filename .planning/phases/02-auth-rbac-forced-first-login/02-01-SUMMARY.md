# Phase 2 Wave 1 Execute Summary

## Work Completed
- **Database/Migrations:**
  - Added `password_changed_at` (timestamptz) to `users` table via migration `000003`.
  - Added Auth/Login queries in sqlc (`GetUserByUsername`, `GetUserByID`, `UpdatePasswordAndStamp`).
- **Backend Setup (`internal/auth`):**
  - Created reusable JWT utilities for Minting and Parsing (`HS256`).
  - Added `CORS` and `AuthMiddleware` checking cookies (`access_token`).
  - Registered `/auth/login`, `/auth/me`, `/auth/logout` endpoints.
  - Successfully verified auth integration via `auth_login_test.go`.
- **Frontend Bootstrap:**
  - Configured Vite + React 19 + Tailwind CSS v4 stack.
  - Bootstrapped Shadcn UI (`button`, `card`, `input`, `label`, `form`).
  - Implemented `react-router` and basic `zustand` state management.
  - Created Login page and configured redirection to role-specific landing pages (`/student`, `/lecturer`, `/admin`).

## Deviations / Discoveries
- Upgraded Go to 1.25.0 temporarily during `go get`, but reverted back to standard `1.24.0` in `go.mod` per project constraints.
- `shadcn` initialization encountered issues locating `form.tsx`, required manual insertion and fixing dependency paths. Fixed and built successfully.
- Added `golang-migrate` to workflow to execute tests locally.

## Next Steps
Proceeding to Wave 2 to handle Forced First-Login Reset Enforcement as planned.
