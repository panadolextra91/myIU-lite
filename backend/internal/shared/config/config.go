package config

import (
	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL   string `env:"DATABASE_URL,required"`
	JWTSecret     string `env:"JWT_SECRET,required"`
	CloudinaryURL string `env:"CLOUDINARY_URL,required"`
	Port           string `env:"PORT" envDefault:"8080"`
	FrontendOrigin string `env:"FRONTEND_ORIGIN" envDefault:"http://localhost:5173"`
	CookieSecure   bool   `env:"COOKIE_SECURE" envDefault:"true"`
}

func Load() (Config, error) {
	_ = godotenv.Load() // best-effort load from .env
	return env.ParseAs[Config]()
}
