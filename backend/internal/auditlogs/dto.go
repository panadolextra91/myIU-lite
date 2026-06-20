package auditlogs

import (
	"time"
)

func errorEnvelope(code, message string) map[string]interface{} {
	return map[string]interface{}{
		"error": map[string]interface{}{"code": code, "message": message},
	}
}

type AuditLogResponse struct {
	ActorID       *int64    `json:"actor_id"`
	Action        string    `json:"action"`
	TargetType    *string   `json:"target_type"`
	TargetID      *int64    `json:"target_id"`
	OperationID   *string   `json:"operation_id"`
	AffectedCount *int32    `json:"affected_count"`
	Metadata      []byte    `json:"metadata"`
	CreatedAt     time.Time `json:"created_at"`
}

type PaginatedAuditLogs struct {
	Data  []AuditLogResponse `json:"data"`
	Total int64              `json:"total"`
}
