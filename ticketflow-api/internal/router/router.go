// internal/router/router.go
package router

import (
	"ticketflow-api/internal/handlers"
	"ticketflow-api/internal/middleware"
	"ticketflow-api/internal/models"

	"github.com/gin-gonic/gin"
)

func Setup(
	jwtSecret string,
	authHandler *handlers.AuthHandler,
	ticketHandler *handlers.TicketHandler,
	attachmentHandler *handlers.AttachmentHandler,
	userHandler *handlers.UserHandler,
) *gin.Engine {
	r := gin.Default()

	// Ліміт розміру тіла запиту (для вкладень — 5 МБ)
	r.MaxMultipartMemory = 5 << 20

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
			tickets.GET("", ticketHandler.ListTickets)
			tickets.POST("", ticketHandler.CreateTicket)
			tickets.GET("/:id", ticketHandler.GetTicket)
			tickets.PATCH("/:id/status", ticketHandler.UpdateTicketStatus)

			// Вкладення
			tickets.POST("/:id/attachments", attachmentHandler.Upload)
			tickets.GET("/:id/attachments", attachmentHandler.GetByTicket)
			tickets.GET("/:id/attachments/:aid/download", attachmentHandler.Download)
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
