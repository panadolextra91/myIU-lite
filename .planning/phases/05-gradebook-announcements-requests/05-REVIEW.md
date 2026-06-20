# Phase 5 Review — Gradebook, Announcements & Requests

**Reviewed:** 2026-06-21 · **Reviewer:** Claude (plan+review role) · **Implementer:** Antigravity
**Method:** 3 parallel code reviewers (gradebook / notify+announce+requests / cross-phase+scope+FE) + ran `scripts/check.sh` on real Postgres + adversarial re-verification of the top findings.

## Verdict: **NOT DONE — FIX-FIRST**

The phase's own Definition of Done (`bash scripts/check.sh` exits 0) is **RED**. On top of that, two confirmed CRITICAL bugs make the announcements + requests features non-functional / insecure, despite the SUMMARY files claiming completion. The gradebook computation core is genuinely good (correct math, real anti-theater tests) — the failures are concentrated in authz wiring, FE API paths, test hygiene, and a few logic edges.

### DoD gate (`scripts/check.sh`) — actual result: **FAIL**
- ✗ `golangci-lint` — errcheck on unchecked `tx.Rollback(ctx)` (requests/service.go:51,112; announcements/service.go:65) and unchecked `pool.Exec(...)` cleanups in fanout_test.go (41,47,51,55,59) and request_test.go (40,45,49,53).
- ✗ `go test` — `TestAnnouncementFanout` and `TestRequestsIntegration` FAIL on real Postgres with `duplicate key ... users_username_active_uq`. Tests use **hardcoded usernames** + best-effort `defer DELETE` cleanup that FK-fails and leaks rows → not residue-tolerant, fail on re-run / populated DB.
- ✓ go build, go vet, grades & assignments & all other packages' tests, frontend lint+build pass.

---

## CRITICAL (must fix before merge)

### C1 — Role context-key mismatch defeats the D-62 IDOR guard
`requests/handler.go:144`, `announcements/handler.go:158`, `grades/handler.go:80` read `c.GetString("user_role")`, but the auth middleware sets the key as `"role"` (`shared/middleware/auth.go:67`). So `role` is always `""`. Earlier phases correctly read `"role"` (assignments/handler.go:90, quizzes/handler.go:90).
**Impact:** In `requests/service.go GetByID`, the per-row ownership filter (`role==student && r.StudentID!=caller` / `role==lecturer && r.TargetedLecturerID!=caller`) is **dead code** when role is `""` — any authenticated caller can read any request by ID (any student reads another student's request; any lecturer reads a request targeted at a different lecturer). Same defeats the SPECIFIC-announcement visibility check in `announcements GetByID`. Directly violates D-62 ("visible ONLY to the requesting student and the targeted lecturer").
**Fix:** Read `c.GetString("role")` in all three Phase-5 handlers. Add a test that asserts a non-party caller gets 404/403 (the existing tests call the service directly with explicit IDs, so they never exercise this glue — which is why it shipped).

### C2 — Frontend announcements + requests API clients omit the `/api` prefix → 404 end-to-end
`frontend/src/lib/announcements-api.ts` and `requests-api.ts` call `/lecturer/...` and `/student/...`; backend mounts under `/api/lecturer` / `/api/student` and axios `baseURL` is `http://localhost:8080` (no `/api`). `grades-api.ts` correctly uses `/api/...`, proving the convention.
**Impact:** Every announcements + requests call 404s; both features are non-functional in the UI.
**Fix:** Prefix all 10 paths in those two files with `/api`.

### C3 — DoD gate red (lint + 2 failing tests) — see "DoD gate" above
**Fix:** (a) check the deferred `tx.Rollback`/`pool.Exec` returns (e.g. `defer func(){ _ = tx.Rollback(ctx) }()` or assign to `_`); (b) make the announcements/requests tests residue-tolerant like the rest of the suite — unique/randomized usernames or robust teardown that deletes dependents first (notifications/recipients/requests → then users), so a re-run / populated DB stays green under `-count=1`.

### C4 — `DeleteScheme` deletes the scheme before its components (FK violation) — D-65 typo-escape always 500s
`grades/service.go:163` calls `DeleteSchemeIfEmpty` (DELETE FROM grade_schemes) **before** `DeleteSchemeComponents` (:177). `grade_components.scheme_id REFERENCES grade_schemes(id)` has **no ON DELETE CASCADE** (000008 up.sql:17). Deleting the parent while children reference it raises an FK violation → the whole delete errors. The source even contains comments admitting the bug ("Wait, Postgres will block DeleteSchemeIfEmpty if components exist").
**Impact:** D-65's "delete+recreate a mistyped scheme before any score/publication exists" is impossible — returns 500 every time. **No test covers `DeleteScheme`**, which is why it shipped.
**Fix:** Delete components first, then the scheme (keep an emptiness guard that doesn't depend on component rows), OR add `ON DELETE CASCADE` and drop `DeleteSchemeComponents`. Add a delete-scheme test.

---

## HIGH

### H1 — Grade writes are not enrollment-scoped (`EnterScore` + CSV)
`grades/service.go EnterScore` (~:188) upserts a `grade_scores` row for `req.StudentID` taken from JSON with no membership check. The CSV path resolves usernames via `GetUserIDsByRole` (global, role-only, **not** joined to `student_enrollments`), so any student username passes even though the row error claims "not enrolled student".
**Impact:** A lecturer can write grades keyed to arbitrary users / students in other courses; they leak into this course's compute/publish set. Violates D-67 and basic course-scoping.
**Fix:** Validate `studentID` ∈ `ListCourseStudents(courseID)` in both paths before upsert; otherwise `ErrValidation`/`ErrNotFound`.

### H2 — Lecturer Gradebook hardcodes course 1, ignores the route param
`frontend/src/pages/lecturer/Gradebook.tsx:32` `useState(1)` ("hardcoded for demo") + a manual course-ID `<Input>`; route is `/lecturer/courses/:id/gradebook` but the page never reads `useParams().id`.
**Impact:** A lecturer on `/lecturer/courses/42/gradebook` edits course 1's gradebook. (Student `Grades.tsx` reads the param correctly — mirror it.)
**Fix:** `const { id } = useParams(); const courseId = Number(id)`; remove the manual input.

### H3 — `Score binding:"required"` makes a legitimate score of 0 unenterable
`grades/dto.go:32` — Go's `required` treats numeric 0 as "missing", so `ShouldBindJSON` 400s on a score of exactly 0. The service correctly allows `0 ≤ score ≤ 100`, but it's unreachable for 0 via this endpoint (CSV path accepts 0).
**Impact:** Can't record an explicit zero — and missing-vs-zero is a core Phase-5 distinction (D-58).
**Fix:** Drop `required` on `Score`; keep the service-side range check.

---

## MEDIUM

- **M1 — AUTO assignment average reads the latest submission *version*, not the graded one** (`grades.sql` ORDER BY s.version DESC LIMIT 1, COALESCE(...,0)). If a student resubmits after the lecturer finalized, compute reads the ungraded new version → grade silently flips to 0. Fix: select latest **graded** version (`AND s.score IS NOT NULL`), or block resubmit after `grading_finalized_at`. (D-58/D-64)
- **M2 — Scheme structural validation incomplete.** `validateWeights` checks per-parent sum-to-100 but not: composite has ≥1 child, leaf has a source_type, AUTO⇒auto_kind / MANUAL⇒no auto_kind, depth ≤ 2 (migration says "depth enforced in Go" — it isn't). An AUTO leaf with no `auto_kind` silently computes 0. Fix: add the structural assertions in `CreateScheme`. (D-56)
- **M3 — Hand-rolled FE components** violate the "no hand-rolled components" constraint: raw `<input type="file">` (Gradebook.tsx) and raw `<textarea>` (Announcements.tsx, RequestInbox.tsx, student/Requests.tsx). Fix: shadcn `Input`/`Textarea` (`npx shadcn add textarea`).
- **M4 — No server-side max-length** on announcement/request title/body/note (unbounded TEXT + only `required`). A multi-MB body fans out into one notification row per recipient (amplification). Fix: `binding:"max=..."` + sane column limits.
- **M5 — `GetRequestByID` doesn't filter soft-deleted courses** (announcements join `courses ... deleted_at IS NULL`; requests don't). Requests on a soft-deleted course stay readable/replyable. Fix: mirror the announcements join.
- **M6 (product decision, not a hard bug)** — an AUTO leaf with an *empty* eligible set computes 0 (vs "n/a"), pulling Overall down early in term; and ALL_STUDENTS announcement *list* visibility is live (late-enrollers see past announcements) while notification delivery is correctly snapshotted. Confirm intended semantics for both.

## LOW
- Dead generated queries `CountSchemeScores` / `CountSchemePublications` (+ repo wrappers) — unused; delete.
- `ReplyRequest` re-reads status on the pool before `Begin` (TOCTOU) — harmless because the SQL `WHERE status='PENDING'` is authoritative; add a comment so it isn't "simplified" away.
- announcements/requests services call `db.New(s.pool)` directly, leaving most `Repository` wrappers unused — pre-existing inconsistency, not worth churning now.

---

## What's correct (keep — credit where due)
- **Same-transaction notify atomicity: CLEAN** across all four flows (announcement fan-out, request create, request reply, grade publish/republish) — `pool.Begin → defer Rollback → qtx → Insert + InsertNotification → Commit`; no notification escapes its tx; `InsertNotification` reused, not redefined. (NOTIF-02 / D-59 / D-63)
- **Cross-phase assignments touch: SAFE & well-tested.** `max_score NUMERIC NOT NULL DEFAULT 100` (backfill safe), `grading_finalized_at TIMESTAMPTZ NULL` (existing rows correctly not-finalized), finalize endpoint authz-gated to the course lecturer + idempotent (SQL `WHERE grading_finalized_at IS NULL`), `max_score` validated `gt=0`, fully wired dto→handler→service→repo→sqlc. `finalize_test.go` is real & red-when-reverted. (D-57/D-64, A2/A3)
- **Gradebook computation core: correct & genuinely tested.** Normalize-first-then-aggregate, missing-finalized=0, eligibility-on-finalize (quiz close_at / assignment grading_finalized_at), weighted tree, NULL/zero-max divide guards in SQL. `compute_test.go` asserts exact composed numbers (54.5 pre-finalize → 64.5 post-finalize) and `TestPublishComponent` verifies the D-66 snapshot-freeze (live 90→95, student stays 90 until republish) — both red-when-reverted. (D-56/57/58/64/66)
- **Scope discipline: CLEAN.** No deferred features built (no configurable scale, best-N/drop-lowest, grade export, request threads/reopen, announcement edit/delete, shared inbox). Lecturer actions correctly NOT audit-logged (audit_log stays admin-only).

## Test authenticity
Existing tests are mostly **authentic** (real seeded fixtures, red-when-reverted): compute, csv (all-or-nothing+422), publish-snapshot, fan-out targeting, request routing/lifecycle. But coverage has **gaps in exactly the buggy spots**: no `DeleteScheme` test (→C4), no `GetByID`/non-party-caller test (→C1), no enrollment-scoping test (→H1), no score=0 test (→H3); and the announcements/requests tests are not residue-tolerant (→C3).

---

## Recommended fix order (hand back to Antigravity)
1. **C3** — green the DoD gate (errcheck + residue-tolerant tests) so the suite is trustworthy again.
2. **C1** — role key in 3 handlers + a non-party-caller IDOR test (security).
3. **C2** — `/api` prefix in 2 FE clients.
4. **C4, H1, H3** — DeleteScheme order (+test), enrollment scoping (+test), score=0 binding.
5. **H2** + **M1–M5** — gradebook route param, resubmit-zeroes-grade, scheme structural validation, shadcn components, text limits, soft-delete filter.
6. Re-run `bash scripts/check.sh` until green, then re-review C1/C2/C4/H1 specifically.
