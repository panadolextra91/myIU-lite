-- Password: 123456
-- Regeneration: go run -e 'package main; import ("fmt";"golang.org/x/crypto/bcrypt"); func main(){h,_:=bcrypt.GenerateFromPassword([]byte("123456"),12);fmt.Println(string(h))}'
INSERT INTO users (username, password_hash, role, must_change_password)
VALUES ('admin', '$2a$12$w2fF0Y7rS0.U.2Kx.9O.N.w.Y0/U3O7E5m0Zq8t1J0R8M8g1E8C6C', 'admin', TRUE);
