# Phase 3: Admin Provisioning & Course Lifecycle - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-20
**Phase:** 3-Admin Provisioning & Course Lifecycle
**Areas discussed:** Tài khoản & CSV, Khóa học & ghi danh, Audit log, Sweep tự xóa, Admin UI surface & removal scope

---

## Tài khoản & CSV (Accounts & CSV)

| Option | Description | Selected |
|--------|-------------|----------|
| 1 file có cột role | SV + GV chung 1 file, phân biệt bằng cột `role`, 1 endpoint | |
| Tách theo role | Upload riêng cho SV và GV, mỗi loại schema riêng | ✓ (D-24) |

| Option | Description | Selected |
|--------|-------------|----------|
| Thêm full_name | Lưu họ tên để UI hiển thị | ✓ (D-25) |
| Tối thiểu | Chỉ id + role, không tên/email | |
| full_name + email | Lưu cả tên và email | |

| Option | Description | Selected |
|--------|-------------|----------|
| Lưu date_of_birth | Lưu DOB → ADMIN-04 reset tự sinh lại DDMMYYYY | ✓ (D-26) |
| Không lưu | DOB chỉ dùng tạm lúc tạo, admin nhập lại khi reset | |

| Option | Description | Selected |
|--------|-------------|----------|
| JSON list + 422 | All-or-nothing, lỗi → 422 + {row, field, message}, FE render bảng | ✓ (D-27) |
| Tải file lỗi về | Trả file CSV lỗi để admin tải | |

**User's choice:** D-24 separate CSV files/actions per role; D-25 store full_name (no email); D-26 store date_of_birth for auto-reset; D-27 all-or-nothing 422 with structured errors.
**Notes:** Student CSV = `student_id, full_name, dob (DD/MM/YYYY)`; Lecturer CSV = `lecturer_id, full_name, dob`; extra columns ignored. username=ID, password=DDMMYYYY. Design principle: explicit business workflows over a generic import mechanism.

---

## Khóa học & ghi danh (Courses & Enrollment)

| Option | Description | Selected |
|--------|-------------|----------|
| code+name+start/end | Minimal course fields | |
| + term + description | Add term + description | partial → D-28 (term yes, description no) |

| Option | Description | Selected |
|--------|-------------|----------|
| Soft-delete | Set deleted_at, filter on reads | ✓ (D-29) |
| Hard delete | Physical delete | |

| Option | Description | Selected |
|--------|-------------|----------|
| Theo khóa, cộng dồn | Per-course CSV, additive + idempotent | ✓ (D-30) |
| Theo khóa, thay toàn bộ | Per-course CSV replaces roster | |
| File phẳng | One flat course_id,user_id file | |

| Option | Description | Selected |
|--------|-------------|----------|
| Nhiều GV + nhiều SV | enrollments(course_id,user_id), role from users | partial → D-31 (multi yes, but two separate tables) |
| Đúng 1 GV/khóa | Exactly one lecturer per course | |

| Option | Description | Selected |
|--------|-------------|----------|
| CSV riêng cho GV | Lecturer assignment via per-course CSV (mirrors D-30) | ✓ (D-32) |
| Gán qua UI | Assign lecturer via UI only | |
| Cả hai | Both CSV and UI | |

**User's choice:** D-28 fields code/name/term/start/end (no description); D-29 soft-delete only; D-30 per-course additive idempotent student CSV; D-31 separate `student_enrollments` + `course_lecturers` tables (1+ lecturers, 0+ students); D-32 lecturer assignment via CSV mirroring D-30.
**Notes:** `term` is first-class, not derived from dates. myIU is not the course-registration system — it manages membership after registration. CSV is add-only; removing membership is a separate action.

---

## Audit log

| Option | Description | Selected |
|--------|-------------|----------|
| 1 dòng / thao tác | One audit row per bulk op + operation_id + affected_count | ✓ (D-33) |
| 1 dòng / entity | One audit row per affected entity | |
| Lai | Batch row + detail rows | |

| Option | Description | Selected |
|--------|-------------|----------|
| Ghi diff before/after | Store old→new values on update | |
| Chỉ ghi giá trị mới | Store event + new values only | partial → D-34 (no diffs; actor/action/target only) |

| Option | Description | Selected |
|--------|-------------|----------|
| Trigger chặn ở DB | BEFORE UPDATE/DELETE triggers raise | ✓ (D-35) |
| REVOKE quyền | Revoke UPDATE/DELETE from app DB role | |
| Kỷ luật ở app | App-level discipline only | |

| Option | Description | Selected |
|--------|-------------|----------|
| Có, trang xem read-only | Admin read-only audit viewer with filters | ✓ (D-36) |
| Chỉ ghi, xem sau | Write-only this phase | |

**User's choice:** D-33 one audit row per bulk op (operation_id + affected_count); D-34 payload actor/action/target_type/target_id/timestamp/metadata, no before/after diffs; D-35 append-only via DB triggers; D-36 ship admin read-only audit viewer.
**Notes:** Operational detail goes in dedicated detail tables (if needed), not the audit log. REVOKE/role separation deferred to a future hardening phase as defense-in-depth.

---

## Sweep tự xóa (Course Lifecycle Sweep)

| Option | Description | Selected |
|--------|-------------|----------|
| In-process, hàng ngày | In-process daily job + startup catch-up | ✓ (D-37) |
| Job/process riêng | External cron / separate process | |

| Option | Description | Selected |
|--------|-------------|----------|
| actor_id = NULL | NULL actor + COURSE_SWEEP action | |
| Tạo user 'system' | Dedicated SYSTEM account as actor | ✓ (D-38) |

| Option | Description | Selected |
|--------|-------------|----------|
| 1 dòng / lần sweep | One row per sweep when ≥1 course affected | ✓ (D-39) |
| 1 dòng / khóa | One row per swept course | |
| Luôn ghi, kể cả 0 | Always log, even affected_count=0 | |

| Option | Description | Selected |
|--------|-------------|----------|
| Không cascade | Only set courses.deleted_at; related records hidden via active-course queries | ✓ (D-40) |
| Cascade soft-delete | Cascade deleted_at to related records | |

**User's choice:** D-37 in-process daily sweep + startup catch-up; D-38 dedicated non-loginable SYSTEM actor; D-39 one audit row only when ≥1 course affected (none on 0); D-40 no cascade.
**Notes:** Sweep is idempotent, not time-critical. Process-execution visibility for no-op days belongs to logs/metrics, not audit. Multi-instance locking deferred.

---

## Admin UI surface & removal scope

| Option | Description | Selected |
|--------|-------------|----------|
| Có, dựng sidebar | First full admin sidebar (Dashboard/Users/Academic/System) | ✓ (D-41) |
| Chưa, nav tạm | Temporary header nav | |

| Option | Description | Selected |
|--------|-------------|----------|
| Có, trang chi tiết khóa | Read-only roster page (Overview/Students/Lecturers) | ✓ (D-42) |
| Chỉ upload, không xem | Upload only, no roster view | |

| Option | Description | Selected |
|--------|-------------|----------|
| Defer sang sau | Removal out of scope for Phase 3 | |
| Làm luôn phase 3 | Manual student remove / lecturer unassign from roster page | ✓ (D-43) |

**User's choice:** Discuss both → D-41 build the full admin sidebar now (fulfils D-21); D-42 read-only course roster page; D-43 manual membership removal from the roster page (UI, audited).
**Notes:** Removal stays UI-only; CSV remains additive. Removal actions audited as STUDENT_REMOVED_FROM_COURSE / LECTURER_UNASSIGNED_FROM_COURSE.

## Claude's Discretion

- Backend feature-folder grouping for the phase (per D-10) and explicit audit-INSERT vs admin middleware.
- Admin Dashboard widget content (default: simple counts).
- Concrete action-code strings / target_type values, accounts-list search/pagination, manual-create form layout.
- Reset is single-user (ADMIN-04) from the Accounts list, reusing stored DOB.

## Deferred Ideas

- Course retention/archival policy (~5yr; archive without cascade) — not MVP (D-29, D-40).
- Audit-history before/after reconstruction — not MVP (D-34).
- Audit-log security hardening (dedicated DB roles + REVOKE + role separation) — future hardening phase (D-35).
- Multi-instance sweep hardening (dedicated scheduler / cron / distributed lock) — future (D-37).
- Lecturer/student-facing roster visibility — later phases (D-42).
