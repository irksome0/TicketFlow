// internal/database/database.go
package database

import (
	"fmt"
	"log"
	"ticketflow-api/internal/config"
	"ticketflow-api/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDB(cfg *config.Config) *gorm.DB {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Помилка підключення до бази даних: %v", err)
	}

	// AutoMigrate створює таблиці, якщо вони ще не існують
	if err := database.AutoMigrate(
		&models.Organization{},
		&models.User{},
		&models.Ticket{},
		&models.Attachment{},
		&models.TicketComment{},
	); err != nil {
		log.Fatalf("Помилка міграції бази даних: %v", err)
	}

	log.Println("Підключення до бази даних встановлено успішно")
	return database
}
