# Phase 3 — Round-2 Fix Handoff for Antigravity

**Round 2 review result:** 13 of 14 prior findings RESOLVED. The transaction refactor (CR-01/04), error-mapping (CR-02/03), SYSTEM hardening (WR-01/02), limit caps (WR-03/04), CSV type validation (WR-05), sweeper recover (WR-06), date errors (WR-07), formula strip (IN-02), sqlc regen (IN-03), flaky test (IN-05), and binary removal (CR-00) are all confirmed correct. **One regression remains** — fix it on branch `ft/phase-3-admin`.

Full report: `.planning/phases/03-admin-provisioning-course-lifecycle/03-REVIEW-R2.md`.

---

## BLOCKER — CR-R2-01: roster endpoints / frontend response-shape mismatch

**What's wrong:** IN-04 (envelope consistency) was fixed on the frontend ONLY, creating a contract drift:
- Frontend `frontend/src/lib/admin-api.ts:76-81` now reads `res.data.data` (expects `{ "data": [...] }`):
  ```ts
  listCourseStudents:  GET /admin/courses/:id/students   → return res.data.data;
  listCourseLecturers: GET /admin/courses/:id/lecturers   → return res.data.data;
  ```
- Backend `backend/internal/courses/handler.go:210` (`ListStudents`) and `:239` (`ListLecturers`) still return a **bare array**: `c.JSON(http.StatusOK, res)`.

At runtime `res.data.data` is `undefined` → the Students and Lecturers tabs on the CourseDetail page (D-42 roster) break.

**Fix (2 lines, backend — match the frontend + every other list endpoint's `{data}` envelope):**
In `backend/internal/courses/handler.go`, change both roster responses:
```go
// ListStudents (~line 210)
c.JSON(http.StatusOK, gin.H{"data": res})
// ListLecturers (~line 239)
c.JSON(http.StatusOK, gin.H{"data": res})
```
Do NOT change the frontend — the FE is already correct. Leave the other endpoints (PaginatedCourses, etc.) untouched.

**Verify:** `cd backend && go build ./...` exits 0; `cd frontend && npm run lint && npx tsc --noEmit && npm run build` exit 0; manually (or by inspection) confirm CourseDetail Students/Lecturers tabs receive the roster array.
**Commit:** `fix(03): wrap course roster responses in {data} envelope`

---

## Optional (non-blocking observation, not a finding)
- **OBS-1:** the CSV upload type check accepts `application/octet-stream` as long as the filename ends in `.csv` — fine for the app's threat model; no action needed unless you want a stricter rule.

After this fix: push, then ping Claude for round-3 re-review (`/gsd-code-review 3`) — expected to be a clean pass.
