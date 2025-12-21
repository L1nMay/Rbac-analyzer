package config

import (
	"os"
)

type Config struct {
	Addr         string
	DatabaseURL  string
	JWTSecret    string
	BaseURL      string // для генерации ссылок
	ContactEmail string // на сайт
	ContactTG    string
	ContactSite  string
}

func Load() Config {
	return Config{
		Addr:         getenv("APP_ADDR", ":8080"),
		DatabaseURL:  getenv("DATABASE_URL", "postgres://rbac:rbac@db:5432/rbac?sslmode=disable"),
		JWTSecret:    getenv("JWT_SECRET", "dev_secret_change_me"),
		BaseURL:      getenv("BASE_URL", "http://localhost:8080"),
		ContactEmail: getenv("CONTACT_EMAIL", "sales@example.com"),
		ContactTG:    getenv("CONTACT_TG", "@your_tg"),
		ContactSite:  getenv("CONTACT_SITE", "https://example.com"),
	}
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}
