package users

import "time"

type CreateUserRequest struct {
	ID       string `json:"id" binding:"required"`
	FullName string `json:"full_name" binding:"required"`
	DOB      string `json:"dob" binding:"required"`
	Role     string `json:"role" binding:"required"`
}

type UserResponse struct {
	ID                 int64     `json:"id"`
	Username           string    `json:"username"`
	FullName           string    `json:"full_name"`
	Role               string    `json:"role"`
	DOB                string    `json:"dob"`
	MustChangePassword bool      `json:"must_change_password"`
	CreatedAt          time.Time `json:"created_at"`
}

type PaginatedUsers struct {
	Data  []UserResponse `json:"data"`
	Total int64          `json:"total"`
}

func errorEnvelope(code, message string) map[string]interface{} {
	return map[string]interface{}{
		"error": map[string]interface{}{"code": code, "message": message},
	}
}
