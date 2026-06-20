CREATE UNIQUE INDEX quiz_attempts_inprogress_idx ON quiz_attempts(quiz_id, student_id) WHERE status='IN_PROGRESS';
