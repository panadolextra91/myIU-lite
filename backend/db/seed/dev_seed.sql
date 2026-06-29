-- backend/db/seed/dev_seed.sql
--
-- LOCAL DEV / UAT fixture — NOT a migration (never run against prod/CI).
-- Idempotent: safe to run repeatedly. Re-apply after a `myiu_dev` rebuild.
--
-- Run:
--   docker exec -i myiu-lite-postgres-1 psql -U myiu -d myiu_dev < backend/db/seed/dev_seed.sql
--
-- All accounts log in with password "123456" (bcrypt cost 12, no forced change).
-- The course is ACTIVE with a future end_date so the daily soft-delete sweep
-- (courses past end_date + 1 month) will NOT archive it — that was the cause of
-- the "assignments/quizzes render empty" symptom when seeded with a past date.

-- Accounts (password = 123456) — bcrypt hash reused from the bootstrap admin migration.
INSERT INTO users (username, password_hash, role, must_change_password, full_name)
VALUES
  ('LEC001', '$2a$12$Cj8bSBTVEdSMT2nj9kdMbuzN3oxJgn397LzPTJIKy869H2Cw0fcHK', 'lecturer', false, 'Jane Smith'),
  ('STU001', '$2a$12$Cj8bSBTVEdSMT2nj9kdMbuzN3oxJgn397LzPTJIKy869H2Cw0fcHK', 'student',  false, 'John Doe')
ON CONFLICT (username) WHERE deleted_at IS NULL DO NOTHING;

-- Course CS101 — active (start in the past, end well in the future).
INSERT INTO courses (code, name, term, start_date, end_date)
SELECT 'CS101', 'Intro to Computer Science', 'Spring 2026', CURRENT_DATE - 30, CURRENT_DATE + 180
WHERE NOT EXISTS (SELECT 1 FROM courses WHERE code = 'CS101' AND deleted_at IS NULL);

-- Enrollment + lecturer assignment (unique on (course_id, *_id) → ON CONFLICT no-op).
INSERT INTO student_enrollments (course_id, student_id)
SELECT c.id, u.id FROM courses c, users u
WHERE c.code = 'CS101' AND c.deleted_at IS NULL AND u.username = 'STU001'
ON CONFLICT DO NOTHING;

INSERT INTO course_lecturers (course_id, lecturer_id)
SELECT c.id, u.id FROM courses c, users u
WHERE c.code = 'CS101' AND c.deleted_at IS NULL AND u.username = 'LEC001'
ON CONFLICT DO NOTHING;

-- One assignment.
INSERT INTO assignments (course_id, title, description, deadline, created_by, max_score)
SELECT c.id, 'Homework 1: Variables & Types', 'Implement the chapter 2 exercises.',
       (CURRENT_DATE + 14)::timestamptz, u.id, 100
FROM courses c, users u
WHERE c.code = 'CS101' AND c.deleted_at IS NULL AND u.username = 'LEC001'
  AND NOT EXISTS (SELECT 1 FROM assignments a WHERE a.course_id = c.id AND a.title = 'Homework 1: Variables & Types');

-- One quiz (open now, closes in a week).
INSERT INTO quizzes (course_id, title, pool_size, max_questions, max_grade, shuffle, retake_count, open_at, close_at, created_by)
SELECT c.id, 'Quiz 1: Fundamentals', 10, 5, 100, true, 1, now() - interval '1 day', now() + interval '7 days', u.id
FROM courses c, users u
WHERE c.code = 'CS101' AND c.deleted_at IS NULL AND u.username = 'LEC001'
  AND NOT EXISTS (SELECT 1 FROM quizzes q WHERE q.course_id = c.id AND q.title = 'Quiz 1: Fundamentals');

-- Grade scheme (3 components) so the Gradebook + Grades screens render populated.
INSERT INTO grade_schemes (course_id, created_by)
SELECT c.id, u.id FROM courses c, users u
WHERE c.code = 'CS101' AND c.deleted_at IS NULL AND u.username = 'LEC001'
  AND NOT EXISTS (SELECT 1 FROM grade_schemes gs JOIN courses c2 ON c2.id = gs.course_id WHERE c2.code = 'CS101');

INSERT INTO grade_components (scheme_id, name, weight, source_type, auto_kind)
SELECT s.id, v.name, v.weight, v.source_type, v.auto_kind
FROM grade_schemes s
JOIN courses c ON c.id = s.course_id AND c.code = 'CS101'
CROSS JOIN (VALUES
  ('Midterm',     30, 'MANUAL', NULL::text),
  ('Assignments', 30, 'AUTO',   'ASSIGNMENT_AVERAGE'),
  ('Final',       40, 'MANUAL', NULL::text)
) AS v(name, weight, source_type, auto_kind)
WHERE NOT EXISTS (SELECT 1 FROM grade_components gc WHERE gc.scheme_id = s.id);
