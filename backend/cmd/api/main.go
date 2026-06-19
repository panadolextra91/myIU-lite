package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/config"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/health"
	"github.com/panadolextra91/myiu-lite/backend/internal/auth"
	"github.com/panadolextra91/myiu-lite/backend/internal/auditlogs"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	router := gin.Default()
	router.Use(func(c *gin.Context) {
		c.SetSameSite(http.SameSiteLaxMode)
		c.Next()
	})
	router.Use(middleware.CORS(cfg.FrontendOrigin))
	health.RegisterRoutes(router, pool)
	auth.RegisterRoutes(router, pool, cfg)
	auditlogs.RegisterRoutes(router, pool, cfg)

	log.Printf("Starting server on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
