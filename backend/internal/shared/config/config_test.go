package config_test

import (
	"os"
	"testing"

	"github.com/panadolextra91/myiu-lite/backend/internal/shared/config"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Run("missing DATABASE_URL fails", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("JWT_SECRET", "test")
		os.Setenv("CLOUDINARY_URL", "test")
		
		_, err := config.Load()
		require.Error(t, err)
	})

	t.Run("missing JWT_SECRET fails", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("DATABASE_URL", "postgres://test")
		os.Setenv("CLOUDINARY_URL", "test")
		
		_, err := config.Load()
		require.Error(t, err)
	})

	t.Run("missing CLOUDINARY_URL fails", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("DATABASE_URL", "postgres://test")
		os.Setenv("JWT_SECRET", "test")
		
		_, err := config.Load()
		require.Error(t, err)
	})

	t.Run("all required vars succeed", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("DATABASE_URL", "postgres://test")
		os.Setenv("JWT_SECRET", "test")
		os.Setenv("CLOUDINARY_URL", "test")
		
		cfg, err := config.Load()
		require.NoError(t, err)
		require.Equal(t, "postgres://test", cfg.DatabaseURL)
		require.Equal(t, "test", cfg.JWTSecret)
		require.Equal(t, "test", cfg.CloudinaryURL)
	})
}
