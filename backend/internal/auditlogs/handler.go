package auditlogs

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/config"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/middleware"
)

type Handler struct {
	svc *Service
	cfg config.Config
}

func RegisterRoutes(r *gin.Engine, pool *pgxpool.Pool, cfg config.Config) {
	q := db.New(pool)
	repo := NewRepository(q)
	svc := NewService(repo)
	h := &Handler{svc: svc, cfg: cfg}

	g := r.Group("/admin")
	g.Use(middleware.AuthMiddleware(pool, cfg), middleware.RequireRole(db.UserRoleAdmin))
	{
		g.GET("/audit-logs", h.ListAuditLogs)
	}
}

func (h *Handler) ListAuditLogs(c *gin.Context) {
	limit := int32(50)
	offset := int32(0)

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.ParseInt(l, 10, 32); err == nil && parsed > 0 {
			limit = int32(parsed)
			if limit > 200 {
				limit = 200
			}
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.ParseInt(o, 10, 32); err == nil && parsed >= 0 {
			offset = int32(parsed)
		}
	}

	var actorID *int64
	if a := c.Query("actor_id"); a != "" {
		if parsed, err := strconv.ParseInt(a, 10, 64); err == nil {
			actorID = &parsed
		}
	}

	var action *string
	if ac := c.Query("action"); ac != "" {
		action = &ac
	}

	var from, to *time.Time
	if f := c.Query("from"); f != "" {
		if t, err := time.Parse(time.RFC3339, f); err == nil {
			from = &t
		}
	}
	if t := c.Query("to"); t != "" {
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			to = &parsed
		}
	}

	data, total, err := h.svc.ListAuditLogs(c.Request.Context(), actorID, action, from, to, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorEnvelope("INTERNAL_ERROR", "Failed to fetch audit logs"))
		return
	}

	c.JSON(http.StatusOK, PaginatedAuditLogs{
		Data:  data,
		Total: total,
	})
}
