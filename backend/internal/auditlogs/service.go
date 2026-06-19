package auditlogs

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListAuditLogs(ctx context.Context, actorID *int64, action *string, from, to *time.Time, limit, offset int32) ([]AuditLogResponse, int64, error) {
	listArg := db.ListAuditLogsParams{
		Limit:  limit,
		Offset: offset,
	}
	countArg := db.CountAuditLogsParams{}

	if actorID != nil {
		listArg.ActorID = pgtype.Int8{Int64: *actorID, Valid: true}
		countArg.ActorID = pgtype.Int8{Int64: *actorID, Valid: true}
	}
	if action != nil {
		listArg.Action = pgtype.Text{String: *action, Valid: true}
		countArg.Action = pgtype.Text{String: *action, Valid: true}
	}
	if from != nil {
		listArg.From = pgtype.Timestamptz{Time: *from, Valid: true}
		countArg.From = pgtype.Timestamptz{Time: *from, Valid: true}
	}
	if to != nil {
		listArg.To = pgtype.Timestamptz{Time: *to, Valid: true}
		countArg.To = pgtype.Timestamptz{Time: *to, Valid: true}
	}

	rows, err := s.repo.ListAuditLogs(ctx, listArg)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.repo.CountAuditLogs(ctx, countArg)
	if err != nil {
		return nil, 0, err
	}

	res := make([]AuditLogResponse, len(rows))
	for i, r := range rows {
		var aid *int64
		if r.ActorID.Valid {
			v := r.ActorID.Int64
			aid = &v
		}
		var tt *string
		if r.TargetType.Valid {
			v := r.TargetType.String
			tt = &v
		}
		var tid *int64
		if r.TargetID.Valid {
			v := r.TargetID.Int64
			tid = &v
		}
		var oid *string
		if r.OperationID.Valid {
			b := r.OperationID.Bytes
			v := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
			oid = &v
		}
		var ac *int32
		if r.AffectedCount.Valid {
			v := r.AffectedCount.Int32
			ac = &v
		}

		res[i] = AuditLogResponse{
			ActorID:       aid,
			Action:        r.Action,
			TargetType:    tt,
			TargetID:      tid,
			OperationID:   oid,
			AffectedCount: ac,
			Metadata:      r.Metadata,
			CreatedAt:     r.CreatedAt.Time,
		}
	}

	return res, count, nil
}
