-- Reverse Phase-4 touch
ALTER TABLE assignments DROP COLUMN IF EXISTS grading_finalized_at;
ALTER TABLE assignments DROP COLUMN IF EXISTS max_score;

-- Drop Phase-5 tables in reverse-dependency order
DROP TABLE IF EXISTS requests;
DROP TABLE IF EXISTS announcement_recipients;
DROP TABLE IF EXISTS announcements;
DROP TABLE IF EXISTS grade_publications;
DROP TABLE IF EXISTS grade_scores;
DROP TABLE IF EXISTS grade_components;
DROP TABLE IF EXISTS grade_schemes;
