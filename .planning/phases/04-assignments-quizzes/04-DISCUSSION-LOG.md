# Phase 4: Assignments & Quizzes - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-20
**Phase:** 4-Assignments & Quizzes
**Areas discussed:** Nộp bài & trễ hạn, Soạn quiz (CSV/UI), Lượt làm & kết quả quiz, Notification

---

## Nộp bài & trễ hạn (Assignment submission & late handling)

### Resubmission model
| Option | Description | Selected |
|--------|-------------|----------|
| Ghi đè tới hạn | Overwrite — latest replaces previous, within window | |
| One-shot 1 lần | Single submission, no edits | |
| Giữ lịch sử các lần nộp | Each submission a separate version; grade latest, history viewable | ✓ |

**User's choice:** **D-44** — Versioned submissions; never overwrite. Multiple submissions while window open, each a new version, latest = active version graded, history preserved.

### Late handling
| Option | Description | Selected |
|--------|-------------|----------|
| Chỉ gắn cờ late | Flag only — record is_late/submitted_at/late_duration; lecturer decides penalty | ✓ |
| Tự trừ điểm | Auto-penalty configured per assignment | |

**User's choice:** **D-45** — Late flagged only (`is_late`, `submitted_at`, `late_duration`); no automatic penalty; lecturer makes the academic judgment.

### Grading inputs
| Option | Description | Selected |
|--------|-------------|----------|
| Điểm + feedback text | Score required + optional feedback; student views both | ✓ |
| Chỉ điểm số | Score only | |

**User's choice:** **D-46** — Score required, feedback optional; student views both.

**Notes:** All three returned as full decision records (decision + rationale + relationships + trade-off + design principle). Principle: the system records facts, the lecturer makes academic judgments.

---

## Soạn quiz (CSV/UI) — Quiz authoring & question model

### "Max number of questions" semantics
| Option | Description | Selected |
|--------|-------------|----------|
| Ngân hàng + rút ngẫu nhiên | Pool of N, draw M per attempt | ✓ |
| Danh sách cố định | Fixed list, max is just an authoring cap | |

**User's choice:** **D-47** — Question-bank model; pool N, random M per attempt; retakes get new sets; shuffle randomizes selection + question order + answer order.

### Options per question / correctness
| Option | Description | Selected |
|--------|-------------|----------|
| 4 đáp án, 1 đúng | Fixed A–D, exactly 1 correct | (CSV) |
| 2–N đáp án, 1 đúng | Variable options, 1 correct | |
| Cho phép nhiều đúng | Multi-correct allowed | (UI) |

**User's choice:** **D-48** — CSV = fixed 4 choices / 1 correct (`question,A,B,C,D,correct`); UI = single-choice (radio) **and** multi-choice (checkbox); auto-grade exact-match all-or-nothing (no partial credit in MVP).

### Quiz availability
| Option | Description | Selected |
|--------|-------------|----------|
| Mở khi course active | Always available while course active; no window/timer | |
| Có cửa sổ open/close | Lecturer sets open/close timestamps | ✓ |
| Có timer mỗi lượt | Per-attempt countdown | |

**User's choice:** **D-49** — Quiz availability window (Open At / Close At); no late submission for quizzes; review allowed after submission; no per-attempt timer (deferred). User chose the window explicitly over the lean default.

---

## Lượt làm & kết quả quiz — Attempts, scoring & answer reveal

### Official retake score
| Option | Description | Selected |
|--------|-------------|----------|
| Cao nhất | MAX across attempts | ✓ |
| Mới nhất | Latest attempt | |
| Trung bình | Average | |

**User's choice:** **D-50** — Official score = MAX across completed attempts; gradebook stores official score only; attempt history kept.

### Answer reveal vs remaining retakes (QUIZ-03 non-leak)
| Option | Description | Selected |
|--------|-------------|----------|
| Hoãn đáp án tới khi hết lượt | Withhold correct answers while retakes remain | (refined) |
| Review đầy đủ ngay mỗi lần | Reveal correct answers immediately | |

**User's choice:** **D-51** — Reveal policy is **window-bound, not attempt-bound**: while window open → score + submitted answers + per-question correct/incorrect status only (no correct answers); after window closes → correct answers visible, regardless of retakes. Reframed the option around the window for fairness across all students.

### Attempt counting
| Option | Description | Selected |
|--------|-------------|----------|
| Khi NỘP | Consumed on submit | |
| Khi BẮT ĐẦU | Consumed on start | ✓ |

**User's choice:** **D-52** — Attempt consumed on START; states IN_PROGRESS/SUBMITTED/AUTO_SUBMITTED; resumable while IN_PROGRESS (no extra retake); AUTO_SUBMITTED when window closes.

**Notes:** Principle: assessment integrity is governed by the quiz window, not individual attempt status; opening an assessment is participation.

---

## Notification primitive

### Content persistence
| Option | Description | Selected |
|--------|-------------|----------|
| Denormalize text sẵn | Render title/body at creation; store directly | ✓ |
| Reference, render khi đọc | Store type + target_id; render at read | |

**User's choice:** **D-53** — Persist fully-rendered title/body at creation; row = recipient_id/type/title/body/resource_type/resource_id/link/created_at/read_at; stable even if resource changes (D-29/D-40).

### Where students see notifications
| Option | Description | Selected |
|--------|-------------|----------|
| Trung tâm thông báo (chuông) | Bell + badge + list page + deep-link | ✓ |
| Inline trong từng trang | In-page only | |
| Cả hai | Both | |

**User's choice:** **D-54** — Centralized notification center: bell in header + unread badge + list page + mark-read-on-click + deep-link. Out of scope: real-time push, dropdown previews, categories, preferences.

### Phase 4 notification events
| Option | Description | Selected |
|--------|-------------|----------|
| Chỉ khi chấm assignment | Assignment grade only (quiz shows score inline) | ✓ |
| Assignment + quiz score | Both notify | |

**User's choice:** **D-55** — Phase 4 notifies only on assignment grading (same transaction, NOTIF-02); quiz grading is synchronous (score shown immediately, D-49) → no quiz notification.

---

## Claude's Discretion

- Feature-folder layout (D-10): `internal/assignments/`, `internal/quizzes/`, notification feature, new `internal/shared/cloudinary/`; migrations `000006+`.
- Ownership/authz (AUTH-05): lecturer-of-course (`course_lecturers`) authors/grades; enrolled student (`student_enrollments`) submits/takes; user_id from JWT; reads filter soft-deleted courses.
- Audit-logging of lecturer actions — default OFF (audit is admin-only, ADMIN-08), pending researcher/planner confirmation.
- Notification read-marker UX details, type-enum strings, link URL shapes, badge-count query — planner's call.

## Deferred Ideas

- Per-attempt quiz timers / duration limits / lockdown browser (D-49 future).
- Partial-credit / weighted / negative-marking quiz grading (D-48 future).
- Notification real-time push / dropdown previews / categories / preferences (D-54).
- Notification templates / localization (D-53 future).
- Announcement / assignment-creation / enrollment / gradebook notifications — Phase 5 (D-55 future).
- Late submission for quizzes — explicitly excluded (D-49).
