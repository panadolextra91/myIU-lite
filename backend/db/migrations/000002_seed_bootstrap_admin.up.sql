-- Password: 123456
-- Regenerate: write a tiny Go program calling
--   bcrypt.GenerateFromPassword([]byte("123456"), 12)
--   (golang.org/x/crypto/bcrypt), then bcrypt.CompareHashAndPassword to verify.
INSERT INTO users (username, password_hash, role, must_change_password)
VALUES ('admin', '$2a$12$Cj8bSBTVEdSMT2nj9kdMbuzN3oxJgn397LzPTJIKy869H2Cw0fcHK', 'admin', TRUE);
