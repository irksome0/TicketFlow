// internal/router/router.go
package router

import (
	"ticketflow-api/internal/handlers"
	"ticketflow-api/internal/middleware"
	"ticketflow-api/internal/models"
	"ticketflow-api/internal/repository"

	"github.com/gin-gonic/gin"
)

func Setup(
	jwtSecret string,
	authHandler *handlers.AuthHandler,
	ticketHandler *handlers.TicketHandler,
	attachmentHandler *handlers.AttachmentHandler,
	userHandler *handlers.UserHandler,
	userRepo repository.UserRepository,
) *gin.Engine {
	r := gin.Default()

	r.MaxMultipartMemory = 25 << 20

	public := r.Group("/api/v1")
	{
		public.POST("/auth/register", authHandler.Register)
		public.POST("/auth/login", authHandler.Login)
	}

	protected := r.Group("/api/v1")
	protected.Use(middleware.AuthMiddleware(jwtSecret, userRepo))
	{
		tickets := protected.Group("/tickets")
		{
			tickets.GET("", ticketHandler.ListTickets)
			tickets.POST("", ticketHandler.CreateTicket)
			tickets.GET("/:id", ticketHandler.GetTicket)
			tickets.PATCH("/:id/status", ticketHandler.UpdateTicketStatus)

			tickets.POST("/:id/attachments", attachmentHandler.Upload)
			tickets.GET("/:id/attachments", attachmentHandler.GetByTicket)
			tickets.GET("/:id/attachments/:aid/download", attachmentHandler.Download)
		}

		users := protected.Group("/users")
		users.Use(middleware.RequireRole(models.RoleAdmin))
		{
			users.GET("", userHandler.GetAll)
			users.PATCH("/:id/role", userHandler.UpdateRole)
		}
	}

	return r
}
