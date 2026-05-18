package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"ticketflow-api/internal/config"
	"ticketflow-api/internal/database"
	"ticketflow-api/internal/handlers"
	"ticketflow-api/internal/repository"
	"ticketflow-api/internal/router"
	"ticketflow-api/internal/security"
)

func main() {
	cfg := config.Load()
	gin.SetMode(cfg.GinMode)

	database := database.InitDB(cfg)

	userRepo := repository.NewUserRepository(database)
	inviteRepo := repository.NewInviteRepository(database)
	organizationRepo := repository.NewOrganizationRepository(database)
	ticketRepo := repository.NewTicketRepository(database)

	authHandler := handlers.NewAuthHandler(
		userRepo,
		organizationRepo,
		cfg.JWTSecret,
		cfg.JWTTTL,
	)
	ticketHandler := handlers.NewTicketHandler(ticketRepo)
	userHandler := handlers.NewUserHandler(userRepo, inviteRepo, cfg.FrontendOrigin)
	var attachmentScanner security.AttachmentScanner = security.NoopAttachmentScanner{}
	if cfg.AttachmentScanEnabled {
		attachmentScanner = security.NewSignatureAttachmentScanner()
	}
	attachmentHandler := handlers.NewAttachmentHandler(database, attachmentScanner)

	r := router.Setup(
		cfg.JWTSecret,
		cfg.FrontendOrigin,
		authHandler,
		ticketHandler,
		attachmentHandler,
		userHandler,
		userRepo,
	)

	log.Printf("Сервер запущено на порті %s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("server: %v", err)
	}
}
