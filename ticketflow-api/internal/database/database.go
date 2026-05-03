// internal/database/database.go
package database

import (
	"fmt"
	"log"
	"strings"

	"ticketflow-api/internal/config"
	"ticketflow-api/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDB(cfg *config.Config) *gorm.DB {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBSSLMode,
		cfg.DBTimeZone,
	)

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Помилка підключення до бази даних: %v", err)
	}

	bootstrapPostgres(database)

	if err := database.AutoMigrate(
		&models.Organization{},
		&models.User{},
		&models.Ticket{},
		&models.TicketStatusHistory{},
		&models.Attachment{},
		&models.TicketComment{},
	); err != nil {
		log.Fatalf("Помилка міграції бази даних: %v", err)
	}

	log.Println("Підключення до бази даних встановлено успішно")
	return database
}

func bootstrapPostgres(database *gorm.DB) {
	statements := []string{
		`CREATE EXTENSION IF NOT EXISTS pgcrypto`,
		createEnumSQL("user_role", []string{"client", "operator", "engineer", "admin"}),
		createEnumSQL("ticket_priority", []string{"High", "Medium", "Low"}),
		createEnumSQL("ticket_status", []string{
			"New",
			"In Progress",
			"Pending",
			"Waiting for Customer",
			"On Hold",
			"Resolved",
			"Closed",
			"Reopened",
		}),
	}

	for _, statement := range statements {
		if err := database.Exec(statement).Error; err != nil {
			log.Fatalf("Помилка підготовки PostgreSQL схеми: %v", err)
		}
	}

	ensureEnumValues(database, "ticket_status", []string{
		"Pending",
		"Waiting for Customer",
		"On Hold",
	})
}

func createEnumSQL(name string, values []string) string {
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = "'" + strings.ReplaceAll(value, "'", "''") + "'"
	}

	return fmt.Sprintf(`
DO $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = '%s') THEN
		CREATE TYPE %s AS ENUM (%s);
	END IF;
END
$$;`, name, name, strings.Join(quoted, ", "))
}

func ensureEnumValues(database *gorm.DB, name string, values []string) {
	for _, value := range values {
		statement := fmt.Sprintf(
			"ALTER TYPE %s ADD VALUE IF NOT EXISTS '%s'",
			name,
			strings.ReplaceAll(value, "'", "''"),
		)
		if err := database.Exec(statement).Error; err != nil {
			log.Printf("Не вдалося оновити enum %s значенням %q: %v", name, value, err)
		}
	}
}
