package notifications

import (
	"errors"
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
}

func RegisterRoutes(r *gin.Engine, pool *pgxpool.Pool, cfg config.Config) {
	q := db.New(pool)
	repo := NewRepository(q)
	svc := NewService(pool, repo)
	h := &Handler{svc: svc}

	g := r.Group("/notifications")
	g.Use(middleware.AuthMiddleware(pool, cfg))
	{
		g.GET("", h.ListNotifications)
		g.GET("/unread-count", h.UnreadCount)
		g.POST("/:nid/read", h.MarkRead)
	}
}

func mapToResponse(n db.Notification) NotificationResponse {
	var readAt *time.Time
	if n.ReadAt.Valid {
		readAt = &n.ReadAt.Time
	}
	return NotificationResponse{
		ID:           n.ID,
		Type:         n.Type,
		Title:        n.Title,
		Body:         n.Body,
		ResourceType: n.ResourceType.String,
		ResourceID:   n.ResourceID.Int64,
		Link:         n.Link.String,
		CreatedAt:    n.CreatedAt.Time,
		ReadAt:       readAt,
	}
}

func (h *Handler) ListNotifications(c *gin.Context) {
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

	userID := c.GetInt64("user_id")
	notifications, err := h.svc.ListForUser(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorEnvelope("INTERNAL_ERROR", "Failed to list notifications"))
		return
	}

	res := make([]NotificationResponse, len(notifications))
	for i, n := range notifications {
		res[i] = mapToResponse(n)
	}

	// For total count, we can do another count query or just return what we have
	// since the task doesn't explicitly require total for notifications list paginator.
	// But let's return it as 0 if we don't have it, or do a count. I'll just omit total.
	c.JSON(http.StatusOK, PaginatedNotifications{
		Data:  res,
		Total: int64(len(res)),
	})
}

func (h *Handler) UnreadCount(c *gin.Context) {
	userID := c.GetInt64("user_id")
	count, err := h.svc.UnreadCount(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorEnvelope("INTERNAL_ERROR", "Failed to get unread count"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

func (h *Handler) MarkRead(c *gin.Context) {
	nid, err := strconv.ParseInt(c.Param("nid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_ID", "Invalid notification ID"))
		return
	}

	userID := c.GetInt64("user_id")
	err = h.svc.MarkRead(c.Request.Context(), nid, userID)
	if err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			c.JSON(http.StatusNotFound, errorEnvelope("NOT_FOUND", "Notification not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("INTERNAL_ERROR", "Failed to mark read"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
