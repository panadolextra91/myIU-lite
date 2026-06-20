DROP TABLE IF EXISTS course_lecturers;
DROP TABLE IF EXISTS student_enrollments;
DROP TABLE IF EXISTS courses;

DROP INDEX IF EXISTS audit_log_action_idx;

ALTER TABLE audit_log 
    DROP COLUMN IF EXISTS affected_count,
    DROP COLUMN IF EXISTS operation_id,
    DROP COLUMN IF EXISTS target_id,
    DROP COLUMN IF EXISTS target_type;

ALTER TABLE users 
    DROP COLUMN IF EXISTS is_system,
    DROP COLUMN IF EXISTS date_of_birth,
    DROP COLUMN IF EXISTS full_name;
