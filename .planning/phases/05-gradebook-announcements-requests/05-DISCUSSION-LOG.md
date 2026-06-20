# Phase 5: Gradebook, Announcements & Requests - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered and the user's full decision records.

**Date:** 2026-06-20
**Phase:** 5-gradebook-announcements-requests
**Areas discussed:** Inclass sourcing, Grade visibility, Announcement model, Request lifecycle, Eligible coursework, Scheme lock, AUTO recompute timing, MANUAL grade CSV

---

## Inclass sourcing — component model (D-56)

| Option | Description | Selected |
|--------|-------------|----------|
| Manual columns (lean) | Lecturer defines named Inclass columns + weights, types each student's score | |
| Auto-pull / linked | Each sub-component links to a Phase 4 source; scores flow automatically | |
| Hybrid (linked OR manual) | Each sub-component is either coursework-linked (auto) or manual | ✓ |

**User's choice:** Authored **D-56 — Gradebook Component Model**: a hierarchical weighted-component model (not flat columns). Overall = Inclass + Midterm + Final = 100%; composite components hold sub-components whose weights sum to 100% of the parent. Each leaf is **AUTO** (Quiz Average, Assignment Average — system aggregates/recalculates from quiz attempts + assignment grades) or **MANUAL** (Project, Laboratory, Midterm, Final, Participation, Bonus — lecturer entry or CSV).
**Notes:** Design principle — "Gradebook computes grades. Gradebook does not merely store grades." Future extensions named: best-N quizzes, drop-lowest, weighted quiz groups, alternative aggregation.

---

## Grade scale & AUTO normalization (D-57)

| Option | Description | Selected |
|--------|-------------|----------|
| 0–10, normalize AUTO | Whole gradebook on 0–10 (VN convention); AUTO normalized to it | |
| 0–100, normalize AUTO | Gradebook on 0–100; AUTO converted to %, MANUAL entered 0–100 | ✓ |
| Lecturer picks scale/course | Per-course configurable scale | |

**User's choice:** Authored **D-57 — Gradebook Scale & AUTO Score Normalization**: 0–100 scale for all computations (HCMIU practice). Quiz Average = avg(student_score / quiz_max × 100); Assignment Average = avg(student_score / assignment_max × 100) — **assignments SHALL define a maximum score**. MANUAL entered on 0–100, validate 0 ≤ score ≤ 100. Per-institution configurable scale is out of MVP scope.
**Notes:** Design principle — "Normalize first. Aggregate second."

---

## AUTO aggregation & missing scores (D-58)

| Option | Description | Selected |
|--------|-------------|----------|
| Missing = 0, include all | All course quizzes/assignments included; not done = 0 in the mean | ✓ |
| Exclude missing items | Average only over items that have a score | |
| Lecturer chooses / You decide | Per-component configurable rule | |

**User's choice:** Authored **D-58 — AUTO Component Aggregation & Missing Scores**: AUTO includes all eligible items; missing (not attempted/submitted) = 0 and stays in the aggregation set. Only **published** coursework participates (draft/deleted/unpublished excluded).
**Notes:** Design principle — "Missing assessment is a grade of zero. Missing assessment is not absence of data."

---

## Grade visibility & publication (D-59)

| Option | Description | Selected |
|--------|-------------|----------|
| Publish per component/column | Lecturer publishes each component independently; notify on each | ✓ |
| Publish whole gradebook once | Single release of the whole gradebook + one notification | |
| Live, always visible | Students see evolving grade; notify per component entry | |

**User's choice:** Authored **D-59 — Grade Publication Model**: component-level publication. Scores hidden from students until the lecturer publishes each top-level component (Midterm first, Inclass later, Final last). After publish → student sees that component + receives a notification. Overall visible only once all top-level components are published. Editing a published score does not auto-notify unless republished.
**Notes:** Design principle — "Grades exist before students can see them. Publication is a separate academic action."

---

## Announcement model (D-60)

| Option | Description | Selected |
|--------|-------------|----------|
| Entity + fan-out | `announcements` table (ALL/SPECIFIC audience) + fan-out notifications; course board + bell | ✓ |
| Notification-only fan-out | No entity; announcement = N notification rows; bell only | |

**User's choice:** Authored **D-60 — Announcement Model**: first-class entities (id, course_id, author_id, title, body, audience_type, created_at; SPECIFIC_STUDENTS uses a join table). Creating one generates notification rows for targeted recipients. Two surfaces: per-course Announcements page (browsable history) + bell. Lecturers can view previously sent announcements + audience/scope.
**Notes:** Design principle — "Announcements are content. Notifications are delivery." Reuses the notification system without making it the source of truth.

---

## Announcement edit/delete (D-61)

| Option | Description | Selected |
|--------|-------------|----------|
| Edit no re-notify; delete = soft | Edit updates board, no resend; delete soft-hides, notifications persist | |
| Edit re-notify; delete removes notif | Edit resends; delete removes related notifications | |
| Send is final (no edit/delete) | No edit/delete after sending | ✓ |

**User's choice:** Authored **D-61 — Announcement Immutability**: immutable after sending — content cannot be edited, deleted, or have recipients changed. Corrections = create a new announcement. **Schema consequence: remove `updated_at`** — lifecycle is CREATED only. No re-notification / notification updates.
**Notes:** Design principle — "An announcement is a notice, not a document. Corrections create new notices."

---

## Request routing (D-62)

| Option | Description | Selected |
|--------|-------------|----------|
| All lecturers, any can reply | Fan-out to all course lecturers; first-reply-wins | |
| Student picks one lecturer | Student chooses a specific course lecturer; only that lecturer sees/replies | ✓ |

**User's choice:** Authored **D-62 — Request Routing**: addressed to a specific lecturer chosen by the student; visible only to the requesting student + targeted lecturer. Other lecturers don't see it. Selected lecturer must be assigned to the course; no main/assistant distinction. Removes race conditions / first-reply-wins.
**Notes:** Design principle — "Requests have a clear owner. Communication is directed, not broadcast."

---

## Request reply model & lifecycle (D-63)

| Option | Description | Selected |
|--------|-------------|----------|
| Yes/No + optional note, 1 round-trip | Required decision + optional note; closes after one reply | ✓ |
| Yes/No only, 1 round-trip | Decision only, no note | |
| Yes/No + required note | Note always required | |

**User's choice:** Authored **D-63 — Request Reply Model**: reply = Decision (APPROVED/DENIED, required) + Note (optional). Lifecycle PENDING → APPROVED/DENIED, then closed permanently (no reopen, no further replies, no thread). Reply generates a notification (decision + optional note).
**Notes:** Design principle — "A decision is required. An explanation is encouraged, not required." Consistent with D-46.

---

## Eligible coursework for AUTO (D-64)

| Option | Description | Selected |
|--------|-------------|----------|
| After effective close | Quiz at close_at; assignment at deadline + late window (or course soft-delete if no threshold) | |
| After lecturer finalizes grading | Item counts once its score is finalized; independent of window | ✓ |
| Lecturer include/exclude per item | Per-item include-in-grade flag | |

**User's choice:** Authored **D-64 — AUTO Component Eligibility**: an assessment becomes eligible when its score is **finalized**. Quiz → eligible when close_at passed + auto-grading done (≈ close_at). Assignment → eligible when the lecturer finalizes grading, **independent of deadline / late policy / late threshold**. Refines D-58 (not "missing=0" until finalized) and separates eligibility (lecturer-visible, in computation) from publication (student-visible).
**Notes:** User raised the assignment late-threshold (D-02/D-44) themselves — original "past deadline" option was corrected to "effective close," then the user chose the stronger "finalized grading" rule. Design principle — "An assessment contributes to grades when its result is finalized. Eligibility is not the same as publication."

---

## Grade scheme lock (D-65)

| Option | Description | Selected |
|--------|-------------|----------|
| Lock at first publish | Editable until first publish, then locked | |
| Editable anytime, recompute | Weights/structure changeable anytime | |
| Lock weights, add MANUAL ok | Weights locked, continue scoring unpublished components | |

**User's choice:** Authored **D-65 — Grade Scheme Immutability**: the scheme is immutable once created — components, hierarchy, weights, and aggregation structure cannot change. Lecturers may enter scores, publish, and update unpublished scores; they may not add/remove components or change weights/structure.
**Notes:** Stronger than the offered "lock at first publish." Rationale: grade weights are approved academic policy, not operational data. Design principle — "Grade schemes define academic policy. Academic policy is immutable."

---

## AUTO recompute timing after publish (D-66)

| Option | Description | Selected |
|--------|-------------|----------|
| Frozen snapshot at publish; republish to update | Student value frozen; republish replaces snapshot + notifies | ✓ |
| Live auto-recompute | Student value always reflects latest underlying scores | |

**User's choice:** Authored **D-66 — Published Grade Snapshots**: published component values are snapshots — frozen for students. Lecturers always see live recomputed values (AUTO recomputes on any eligible change). Republishing replaces the published snapshot and generates a new notification.
**Notes:** Aligns with D-59/D-65. Design principle — "Publication creates a snapshot. Computation remains live."

---

## MANUAL grade CSV format (D-67)

| Option | Description | Selected |
|--------|-------------|----------|
| Long: one component per file | `student_id,score`, one file per MANUAL component | ✓ |
| Wide: one file many columns | `student_id,midterm,final,project,...` | |
| You decide | Planner picks, following Phase 3 discipline | |

**User's choice:** Authored **D-67 — MANUAL Grade CSV Format**: one component per file, `student_id,score`. Follows Phase 3 CSV discipline — whole-file validation, all-or-nothing commit, row-level error reporting, HTTP 422 on failure, no partial imports.
**Notes:** Design principle — "Grades are imported by component. Gradebook structure should not leak into CSV format."

---

## Claude's Discretion

- **Request creation notifies the targeted lecturer** (bell) — symmetric to the reply→student notification; the app is "no email."
- **Same-transaction notification writes (NOTIF-02)** for grade-publish, republish, and request-reply, via the existing `assignments/service.go:202` pattern.
- **Lecturer actions are NOT audit-logged** — `audit_log` stays admin-only (ADMIN-08); same default as Phase 4.
- **Feature folders** `internal/grades/`, `internal/announcements/`, `internal/requests/` (D-10); migration `000008` (D-06); notification `type` strings, `link` shapes, and student grade-view / lecturer request-inbox UI are the planner's call.
- **Recipient snapshot at send time** for announcement fan-out (per D-53 persist-at-creation).

## Deferred Ideas

- Per-institution configurable grading scale (0–10 / GPA) — out of MVP (D-57).
- Advanced grading policies: best-N quizzes, drop-lowest, weighted quiz groups, alt aggregation (D-56 future).
- Grade export (CSV/PDF transcripts) — tracked as GRADE-V2-01 (v2).
- Request conversation threads / reopen / multi-reply — excluded (D-63).
- Announcement edit/delete / scheduled send / extended read receipts — excluded (D-61).
- Shared lecturer request inbox / request reassignment — excluded (D-62).
