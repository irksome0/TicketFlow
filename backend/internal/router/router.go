// internal/router/router.go
package router

import (
	"ticketflow-api/internal/handlers"
	"ticketflow-api/internal/middleware"
	"ticketflow-api/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Setup(db *gorm.DB, jwtSecret string) *gin.Engine {
	r := gin.Default()

	// Ліміт розміру тіла запиту (для вкладень — 5 МБ)
	r.MaxMultipartMemory = 5 << 20

	authHandler := handlers.NewAuthHandler(db, jwtSecret)
	ticketHandler := handlers.NewTicketHandler(db)
	attachmentHandler := handlers.NewAttachmentHandler(db)
	userHandler := handlers.NewUserHandler(db)

	// Публічні маршрути (без автентифікації)
	public := r.Group("/api/v1")
	{
		public.POST("/auth/register", authHandler.Register)
		public.POST("/auth/login", authHandler.Login)
	}

	// Захищені маршрути
	protected := r.Group("/api/v1")
	protected.Use(middleware.AuthMiddleware(jwtSecret))
	{
		// Маршрути заявок
		tickets := protected.Group("/tickets")
		{
			tickets.GET("", ticketHandler.GetAll)
			tickets.POST("", ticketHandler.Create)
			tickets.GET("/:id", ticketHandler.GetByID)
			tickets.PATCH("/:id/status", ticketHandler.UpdateStatus)

			// Вкладення
			tickets.POST("/:id/attachments", attachmentHandler.Upload)
			tickets.GET("/:id/attachments", attachmentHandler.GetByTicket)
		}

		// Маршрути управління користувачами (лише Admin)
		users := protected.Group("/users")
		users.Use(middleware.RequireRole(models.RoleAdmin))
		{
			users.GET("", userHandler.GetAll)
			users.PATCH("/:id/role", userHandler.UpdateRole)
		}
	}

	return r
}
