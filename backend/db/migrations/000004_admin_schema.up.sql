ALTER TABLE users 
    ADD COLUMN full_name TEXT, 
    ADD COLUMN date_of_birth DATE, 
    ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE audit_log 
    ADD COLUMN target_type TEXT, 
    ADD COLUMN target_id BIGINT, 
    ADD COLUMN operation_id UUID NOT NULL DEFAULT gen_random_uuid(), 
    ADD COLUMN affected_count INTEGER;

CREATE INDEX audit_log_action_idx ON audit_log (action);

CREATE TABLE courses (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    term TEXT NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE student_enrollments (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    course_id BIGINT NOT NULL REFERENCES courses(id),
    student_id BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (course_id, student_id)
);

CREATE TABLE course_lecturers (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    course_id BIGINT NOT NULL REFERENCES courses(id),
    lecturer_id BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (course_id, lecturer_id)
);
