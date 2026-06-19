package auditlogs

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
)

const (
	ACCOUNT_CREATE                  = "ACCOUNT_CREATE"
	PASSWORD_RESET                  = "PASSWORD_RESET"
	COURSE_CREATE                   = "COURSE_CREATE"
	COURSE_UPDATE                   = "COURSE_UPDATE"
	COURSE_DELETE                   = "COURSE_DELETE"
	ENROLL_IMPORT                   = "ENROLL_IMPORT"
	LECTURER_IMPORT                 = "LECTURER_IMPORT"
	IMPORT_STUDENTS                 = "IMPORT_STUDENTS"
	IMPORT_LECTURERS                = "IMPORT_LECTURERS"
	STUDENT_REMOVED_FROM_COURSE     = "STUDENT_REMOVED_FROM_COURSE"
	LECTURER_UNASSIGNED_FROM_COURSE = "LECTURER_UNASSIGNED_FROM_COURSE"
	COURSE_SWEEP                    = "COURSE_SWEEP"

	TargetTypeUser   = "user"
	TargetTypeCourse = "course"
)

func WriteAudit(ctx context.Context, q *db.Queries, actorID int64, action, targetType string, targetID, affectedCount *int64, metadata []byte) error {
	params := db.WriteAuditLogParams{
		ActorID: pgtype.Int8{Int64: actorID, Valid: true},
		Action:  action,
	}

	if targetType != "" {
		params.TargetType = pgtype.Text{String: targetType, Valid: true}
	}
	if targetID != nil {
		params.TargetID = pgtype.Int8{Int64: *targetID, Valid: true}
	}
	if affectedCount != nil {
		params.AffectedCount = pgtype.Int4{Int32: int32(*affectedCount), Valid: true}
	}
	if metadata == nil {
		metadata = []byte("{}")
	}
	params.Metadata = metadata

	return q.WriteAuditLog(ctx, params)
}
