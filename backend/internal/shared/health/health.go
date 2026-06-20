package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
)

func RegisterRoutes(r *gin.Engine, pool *pgxpool.Pool) {
	queries := db.New(pool)
	
	r.GET("/healthz", func(c *gin.Context) {
		count, err := queries.HealthCountUsers(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db down"})
			return
		}
		
		_ = count 
		
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}
