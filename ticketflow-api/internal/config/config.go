// internal/config/config.go
package config

import (
	"fmt"
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

	SLATimeZone      string
	SLABusinessStart time.Duration
	SLABusinessEnd   time.Duration
	SLAHolidays      []string
	SLAHighLimit     time.Duration
	SLAMediumLimit   time.Duration
	SLALowLimit      time.Duration

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

		SLATimeZone:      getEnv("SLA_TIMEZONE", "Europe/Kyiv"),
		SLABusinessStart: getEnvAsClock("SLA_BUSINESS_START", 9*time.Hour),
		SLABusinessEnd:   getEnvAsClock("SLA_BUSINESS_END", 18*time.Hour),
		SLAHolidays:      getEnvAsList("SLA_HOLIDAYS"),
		SLAHighLimit:     time.Duration(getEnvAsInt("SLA_HIGH_HOURS", 8)) * time.Hour,
		SLAMediumLimit:   time.Duration(getEnvAsInt("SLA_MEDIUM_HOURS", 24)) * time.Hour,
		SLALowLimit:      time.Duration(getEnvAsInt("SLA_LOW_HOURS", 72)) * time.Hour,

		StorageProvider:   strings.ToLower(strings.TrimSpace(getEnv("STORAGE_PROVIDER", "local"))),
		LocalStorageDir:   getEnv("LOCAL_STORAGE_DIR", "uploads"),
		R2AccountID:       strings.TrimSpace(getEnv("R2_ACCOUNT_ID", "")),
		R2Endpoint:        strings.TrimSpace(getEnv("R2_ENDPOINT", "")),
		R2AccessKeyID:     strings.TrimSpace(getEnv("R2_ACCESS_KEY_ID", "")),
		R2SecretAccessKey: strings.TrimSpace(getEnv("R2_SECRET_ACCESS_KEY", "")),
		R2Bucket:          strings.TrimSpace(getEnv("R2_BUCKET", "")),
	}
}

func getEnvAsClock(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(getEnv(key, ""))
	if raw == "" {
		return fallback
	}

	value, err := time.Parse("15:04", raw)
	if err != nil {
		log.Printf("Некоректне значення %s=%q, використано %s", key, raw, formatClock(fallback))
		return fallback
	}

	return time.Duration(value.Hour())*time.Hour + time.Duration(value.Minute())*time.Minute
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
		log.Printf("Некоректне значення %s=%q, використано %t", key, raw, fallback)
		return fallback
	}
}

func getEnvAsList(key string) []string {
	raw := strings.TrimSpace(getEnv(key, ""))
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func normalizeOrigin(origin string) string {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return "*"
	}
	return origin
}

func formatClock(value time.Duration) string {
	totalMinutes := int(value.Minutes())
	return fmt.Sprintf("%02d:%02d", totalMinutes/60, totalMinutes%60)
}
