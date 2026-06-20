package quizzes

import "time"

func errorEnvelope(code, message string) map[string]interface{} {
	return map[string]interface{}{
		"error": map[string]interface{}{"code": code, "message": message},
	}
}

type CreateQuizRequest struct {
	Title        string     `json:"title" binding:"required"`
	PoolSize     int32      `json:"pool_size" binding:"required,min=1"`
	MaxQuestions int32      `json:"max_questions" binding:"required,min=1"`
	MaxGrade     float64    `json:"max_grade" binding:"required,min=0"`
	Shuffle      bool       `json:"shuffle"`
	RetakeCount  int32      `json:"retake_count" binding:"min=1"`
	OpenAt       *time.Time `json:"open_at"`
	CloseAt      *time.Time `json:"close_at"`
}

type QuizResponse struct {
	ID           int64      `json:"id"`
	Title        string     `json:"title"`
	PoolSize     int32      `json:"pool_size"`
	MaxQuestions int32      `json:"max_questions"`
	MaxGrade     float64    `json:"max_grade"`
	Shuffle      bool       `json:"shuffle"`
	RetakeCount  int32      `json:"retake_count"`
	OpenAt       *time.Time `json:"open_at"`
	CloseAt      *time.Time `json:"close_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

type UIOptionRequest struct {
	Text      string `json:"text" binding:"required"`
	IsCorrect bool   `json:"is_correct"`
}

type UIQuestionRequest struct {
	Prompt       string            `json:"prompt" binding:"required"`
	QuestionType string            `json:"question_type" binding:"required,oneof=single multi"`
	Options      []UIOptionRequest `json:"options" binding:"required,min=2"`
}

type StudentOptionView struct {
	ID   int64  `json:"id"`
	Text string `json:"text"`
}

type StudentQuestionView struct {
	ID           int64               `json:"id"`
	Prompt       string              `json:"prompt"`
	QuestionType string              `json:"question_type"`
	Options      []StudentOptionView `json:"options"`
}

type StudentQuizAttemptView struct {
	ID            int64                 `json:"id"`
	QuizID        int64                 `json:"quiz_id"`
	AttemptNumber int32                 `json:"attempt_number"`
	Status        string                `json:"status"`
	Score         *float64              `json:"score"`
	StartedAt     time.Time             `json:"started_at"`
	SubmittedAt   *time.Time            `json:"submitted_at"`
	Questions     []StudentQuestionView `json:"questions"`
	// Selected option IDs keyed by question ID
	SelectedOptions map[int64][]int64 `json:"selected_options"`
	// Review mode fields (only populated if window is closed and attempt is terminal)
	// Correct option IDs keyed by question ID
	CorrectOptions map[int64][]int64 `json:"correct_options,omitempty"`
}

type SubmitAttemptRequest struct {
	Answers map[int64][]int64 `json:"answers"`
}

type SubmitAttemptResponse struct {
	Score         float64 `json:"score"`
	OfficialScore float64 `json:"official_score"`
	Status        string  `json:"status"`
}
