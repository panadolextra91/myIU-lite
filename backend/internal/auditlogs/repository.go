package auditlogs

import (
	"context"

	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
)

type Repository struct {
	q *db.Queries
}

func NewRepository(q *db.Queries) *Repository {
	return &Repository{q: q}
}

func (r *Repository) ListAuditLogs(ctx context.Context, arg db.ListAuditLogsParams) ([]db.AuditLog, error) {
	return r.q.ListAuditLogs(ctx, arg)
}

func (r *Repository) CountAuditLogs(ctx context.Context, arg db.CountAuditLogsParams) (int64, error) {
	return r.q.CountAuditLogs(ctx, arg)
}
