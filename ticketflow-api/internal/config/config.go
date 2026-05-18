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
	DBRLS      bool

	JWTSecret string
	JWTTTL    time.Duration

	ServerPort     string
	GinMode        string
	FrontendOrigin string

	AttachmentScanEnabled bool

	StorageProvider   string
	LocalStorageDir   string
	R2AccountID       string
	R2Endpoint        string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2Bucket          string
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
		DBRLS:      getEnvAsBool("DB_RLS_ENABLED", true),

		JWTSecret: getEnv("JWT_SECRET", "change-me-in-production"),
		JWTTTL:    time.Duration(getEnvAsInt("JWT_TTL_HOURS", 24)) * time.Hour,

		ServerPort:     getEnv("SERVER_PORT", "8080"),
		GinMode:        getEnv("GIN_MODE", "debug"),
		FrontendOrigin: normalizeOrigin(getEnv("FRONTEND_ORIGIN", "http://localhost:3000")),

		AttachmentScanEnabled: getEnvAsBool("ATTACHMENT_SCAN_ENABLED", true),

		StorageProvider:   strings.ToLower(strings.TrimSpace(getEnv("STORAGE_PROVIDER", "local"))),
		LocalStorageDir:   getEnv("LOCAL_STORAGE_DIR", "uploads"),
		R2AccountID:       strings.TrimSpace(getEnv("R2_ACCOUNT_ID", "")),
		R2Endpoint:        strings.TrimSpace(getEnv("R2_ENDPOINT", "")),
		R2AccessKeyID:     strings.TrimSpace(getEnv("R2_ACCESS_KEY_ID", "")),
		R2SecretAccessKey: strings.TrimSpace(getEnv("R2_SECRET_ACCESS_KEY", "")),
		R2Bucket:          strings.TrimSpace(getEnv("R2_BUCKET", "")),
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

func getEnvAsBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(getEnv(key, "")))
	if raw == "" {
		return fallback
	}

	switch raw {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		log.Printf("РќРµРєРѕСЂРµРєС‚РЅРµ Р·РЅР°С‡РµРЅРЅСЏ %s=%q, РІРёРєРѕСЂРёСЃС‚Р°РЅРѕ %t", key, raw, fallback)
		return fallback
	}
}

func normalizeOrigin(origin string) string {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return "*"
	}
	return origin
}
