// internal/router/router.go
package router

import (
	"net/http"
	"strings"

	"ticketflow-api/internal/handlers"
	"ticketflow-api/internal/middleware"
	"ticketflow-api/internal/models"
	"ticketflow-api/internal/repository"

	"github.com/gin-gonic/gin"
)

func Setup(
	jwtSecret string,
	frontendOrigin string,
	authHandler *handlers.AuthHandler,
	ticketHandler *handlers.TicketHandler,
	attachmentHandler *handlers.AttachmentHandler,
	userHandler *handlers.UserHandler,
	userRepo repository.UserRepository,
) *gin.Engine {
	r := gin.Default()

	r.MaxMultipartMemory = 25 << 20
	r.Use(corsMiddleware(frontendOrigin))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	public := r.Group("/api/v1")
	{
		public.POST("/auth/register", authHandler.Register)
		public.POST("/auth/register-organization", authHandler.RegisterOrganization)
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

func corsMiddleware(frontendOrigin string) gin.HandlerFunc {
	allowedOrigins := splitOrigins(frontendOrigin)

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowOrigin := matchOrigin(origin, allowedOrigins)
		if allowOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowOrigin)
			c.Header("Vary", "Origin")
		}

		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		c.Header("Access-Control-Allow-Credentials", "false")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func splitOrigins(value string) []string {
	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		origin := strings.TrimSpace(part)
		if origin != "" {
			origins = append(origins, origin)
		}
	}
	if len(origins) == 0 {
		return []string{"*"}
	}
	return origins
}

func matchOrigin(origin string, allowed []string) string {
	for _, item := range allowed {
		if item == "*" {
			return "*"
		}
		if origin != "" && origin == item {
			return origin
		}
	}
	return ""
}
