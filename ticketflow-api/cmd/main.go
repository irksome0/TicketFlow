package main

import (
	"log"
	"time"

	"ticketflow-api/internal/config"
	"ticketflow-api/internal/database"
	"ticketflow-api/internal/handlers"
	"ticketflow-api/internal/repository"
	"ticketflow-api/internal/router"
)

func main() {
	cfg := config.Load()

	database := database.InitDB(cfg)

	// ── Репозиторії ───────────────────────────────────────────────────────────
	userRepo := repository.NewUserRepository(database)
	ticketRepo := repository.NewTicketRepository(database)

	// ── Обробники ─────────────────────────────────────────────────────────────
	authHandler := handlers.NewAuthHandler(
		userRepo,
		cfg.JWTSecret,
		24*time.Hour,
	)
	ticketHandler := handlers.NewTicketHandler(ticketRepo)
	userHandler := handlers.NewUserHandler(userRepo)
	attachmentHandler := handlers.NewAttachmentHandler(database)

	// ── Маршрутизатор ─────────────────────────────────────────────────────────
	r := router.Setup(
		cfg.JWTSecret,
		authHandler,
		ticketHandler,
		attachmentHandler,
		userHandler,
		userRepo,
	)

	log.Printf("Сервер запущено на порту %s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("server: %v", err)
	}
}
