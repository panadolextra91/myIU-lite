-- Phase 5: Gradebook, Announcements & Requests

-- Grade Scheme: one immutable scheme per course (D-65)
CREATE TABLE grade_schemes (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    course_id BIGINT NOT NULL REFERENCES courses(id),
    created_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(course_id)
);

-- Grade Components: hierarchical weighted tree (composite or leaf)
-- Leaf = source_type IS NOT NULL; Composite = source_type IS NULL
-- Depth (max 2 levels) enforced in Go, NOT in SQL
CREATE TABLE grade_components (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    scheme_id BIGINT NOT NULL REFERENCES grade_schemes(id),
    parent_id BIGINT NULL REFERENCES grade_components(id),
    name TEXT NOT NULL,
    weight NUMERIC NOT NULL CHECK (weight > 0 AND weight <= 100),
    source_type TEXT NULL CHECK (source_type IN ('AUTO','MANUAL')),
    auto_kind TEXT NULL CHECK (auto_kind IN ('QUIZ_AVERAGE','ASSIGNMENT_AVERAGE')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Grade Scores: MANUAL leaf values only (AUTO computed live)
CREATE TABLE grade_scores (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    component_id BIGINT NOT NULL REFERENCES grade_components(id),
    student_id BIGINT NOT NULL REFERENCES users(id),
    score NUMERIC NOT NULL CHECK (score >= 0 AND score <= 100),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(component_id, student_id)
);

-- Grade Publications: frozen snapshot per top-level component per student (D-66)
CREATE TABLE grade_publications (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    component_id BIGINT NOT NULL REFERENCES grade_components(id),
    student_id BIGINT NOT NULL REFERENCES users(id),
    value NUMERIC NOT NULL,
    published_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(component_id, student_id)
);

-- Announcements: immutable, NO updated_at (D-61)
CREATE TABLE announcements (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    course_id BIGINT NOT NULL REFERENCES courses(id),
    author_id BIGINT NOT NULL REFERENCES users(id),
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    audience_type TEXT NOT NULL CHECK (audience_type IN ('ALL_STUDENTS','SPECIFIC_STUDENTS')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Announcement Recipients: for SPECIFIC_STUDENTS audience
CREATE TABLE announcement_recipients (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    announcement_id BIGINT NOT NULL REFERENCES announcements(id),
    student_id BIGINT NOT NULL REFERENCES users(id),
    UNIQUE(announcement_id, student_id)
);

-- Requests: single round-trip, NO updated_at (D-63)
CREATE TABLE requests (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    course_id BIGINT NOT NULL REFERENCES courses(id),
    student_id BIGINT NOT NULL REFERENCES users(id),
    targeted_lecturer_id BIGINT NOT NULL REFERENCES users(id),
    type TEXT NOT NULL CHECK (type IN ('LEAVE_EARLY','ABSENCE','CUSTOM')),
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','APPROVED','DENIED')),
    reply_note TEXT NULL,
    replied_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes (mirroring 000006 style)
CREATE INDEX grade_components_scheme_id_idx ON grade_components(scheme_id);
CREATE INDEX grade_components_parent_id_idx ON grade_components(parent_id);
CREATE INDEX grade_publications_student_id_idx ON grade_publications(student_id);
CREATE INDEX announcement_recipients_announcement_id_idx ON announcement_recipients(announcement_id);
CREATE INDEX requests_lecturer_status_idx ON requests(targeted_lecturer_id, status);
CREATE INDEX requests_student_id_idx ON requests(student_id);

-- Phase-4 touch: add max_score + grading_finalized_at to assignments (D-57, D-64)
ALTER TABLE assignments ADD COLUMN max_score NUMERIC NOT NULL DEFAULT 100;
ALTER TABLE assignments ADD COLUMN grading_finalized_at TIMESTAMPTZ NULL;
