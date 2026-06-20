package lifecycle

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/auditlogs"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
)

func StartSweeper(ctx context.Context, pool *pgxpool.Pool, systemID int64) {
	q := db.New(pool)
	
	// Catch-up run on startup
	if _, err := runSweep(ctx, pool, q, systemID); err != nil {
		log.Printf("Startup sweep failed: %v", err)
	}

	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				func() {
					defer func() {
						if r := recover(); r != nil {
							log.Printf("sweep panic: %v", r)
						}
					}()
					if _, err := runSweep(ctx, pool, q, systemID); err != nil {
						log.Printf("Daily sweep failed: %v", err)
					}
				}()
			}
		}
	}()
}

func runSweep(ctx context.Context, pool *pgxpool.Pool, q *db.Queries, systemID int64) (int64, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	cmdTag, err := tx.Exec(ctx, `
		UPDATE courses 
		SET deleted_at = now() 
		WHERE deleted_at IS NULL AND end_date < now() - interval '1 month'
	`)
	if err != nil {
		return 0, err
	}

	n := cmdTag.RowsAffected()
	if n > 0 {
		qtx := q.WithTx(tx)
		err = auditlogs.WriteAudit(ctx, qtx, systemID, auditlogs.COURSE_SWEEP, auditlogs.TargetTypeCourse, nil, &n, nil)
		if err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	return n, nil
}
