package grades

type ComponentInput struct {
	Name        string   `json:"name" binding:"required"`
	Weight      float64  `json:"weight" binding:"required,gt=0,lte=100"`
	ParentIndex *int     `json:"parent_index"` // used during creation to link children to parents
	SourceType  *string  `json:"source_type"`  // AUTO or MANUAL, nil if composite
	AutoKind    *string  `json:"auto_kind"`    // QUIZ_AVERAGE or ASSIGNMENT_AVERAGE
}

type SchemeRequest struct {
	Components []ComponentInput `json:"components" binding:"required,min=1"`
}

type ComponentResponse struct {
	ID         int64               `json:"id"`
	ParentID   *int64              `json:"parent_id"`
	Name       string              `json:"name"`
	Weight     float64             `json:"weight"`
	SourceType *string             `json:"source_type"`
	AutoKind   *string             `json:"auto_kind"`
}

type SchemeResponse struct {
	ID         int64               `json:"id"`
	CourseID   int64               `json:"course_id"`
	Components []ComponentResponse `json:"components"`
}

type ScoreEntryRequest struct {
	StudentID int64   `json:"student_id" binding:"required"`
	Score     float64 `json:"score" binding:"required"`
}

type ComputedComponent struct {
	ComponentID int64   `json:"component_id"`
	Score       float64 `json:"score"`
}

type OverallResponse struct {
	StudentID  int64               `json:"student_id"`
	Overall    float64             `json:"overall"`
	Components []ComputedComponent `json:"components"`
}

type StudentGradesResponse struct {
	StudentID  int64               `json:"student_id"`
	Overall    *float64            `json:"overall"`
	Components []ComputedComponent `json:"components"`
}

func errorEnvelope(code, message string) map[string]interface{} {
	return map[string]interface{}{
		"error": map[string]interface{}{"code": code, "message": message},
	}
}
