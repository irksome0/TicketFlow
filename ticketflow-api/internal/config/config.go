// internal/config/config.go
package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	DBTimeZone string

	JWTSecret string
	JWTTTL    time.Duration

	ServerPort     string
	GinMode        string
	FrontendOrigin string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не знайдено, використовуються змінні середовища")
	}

	return &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "ticketflow"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
		DBTimeZone: getEnv("DB_TIMEZONE", "UTC"),

		JWTSecret: getEnv("JWT_SECRET", "change-me-in-production"),
		JWTTTL:    time.Duration(getEnvAsInt("JWT_TTL_HOURS", 24)) * time.Hour,

		ServerPort:     getEnv("SERVER_PORT", "8080"),
		GinMode:        getEnv("GIN_MODE", "debug"),
		FrontendOrigin: normalizeOrigin(getEnv("FRONTEND_ORIGIN", "http://localhost:3000")),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	raw := strings.TrimSpace(getEnv(key, ""))
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		log.Printf("Некоректне значення %s=%q, використано %d", key, raw, fallback)
		return fallback
	}

	return value
}

func normalizeOrigin(origin string) string {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return "*"
	}
	return origin
}
