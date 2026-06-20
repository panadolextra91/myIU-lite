package notifications

import "time"

type NotificationResponse struct {
	ID           int64      `json:"id"`
	Type         string     `json:"type"`
	Title        string     `json:"title"`
	Body         string     `json:"body"`
	ResourceType string     `json:"resource_type"`
	ResourceID   int64      `json:"resource_id"`
	Link         string     `json:"link"`
	CreatedAt    time.Time  `json:"created_at"`
	ReadAt       *time.Time `json:"read_at"`
}

type PaginatedNotifications struct {
	Data  []NotificationResponse `json:"data"`
	Total int64                  `json:"total"`
}

func errorEnvelope(code, message string) map[string]interface{} {
	return map[string]interface{}{
		"error": map[string]interface{}{"code": code, "message": message},
	}
}
