package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"ticketflow-api/internal/config"
	"ticketflow-api/internal/database"
	"ticketflow-api/internal/handlers"
	"ticketflow-api/internal/repository"
	"ticketflow-api/internal/router"
	"ticketflow-api/internal/security"
	"ticketflow-api/internal/storage"
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

	fileStorage, storagePrefix, err := buildFileStorage(context.Background(), cfg)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	attachmentHandler := handlers.NewAttachmentHandler(database, attachmentScanner, fileStorage, storagePrefix)

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

func buildFileStorage(ctx context.Context, cfg *config.Config) (storage.FileStorage, string, error) {
	switch storage.Provider(cfg.StorageProvider) {
	case storage.ProviderR2:
		endpoint := cfg.R2Endpoint
		if endpoint == "" && cfg.R2AccountID != "" {
			endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.R2AccountID)
		}
		r2Storage, err := storage.NewR2Storage(ctx, storage.R2Config{
			Endpoint:        endpoint,
			AccessKeyID:     cfg.R2AccessKeyID,
			SecretAccessKey: cfg.R2SecretAccessKey,
			Bucket:          cfg.R2Bucket,
		})
		return r2Storage, "attachments", err
	case storage.ProviderLocal, "":
		return storage.NewLocalStorage("."), cfg.LocalStorageDir, nil
	default:
		log.Printf("Невідомий STORAGE_PROVIDER=%q, використовується local", cfg.StorageProvider)
		return storage.NewLocalStorage("."), cfg.LocalStorageDir, nil
	}
}
