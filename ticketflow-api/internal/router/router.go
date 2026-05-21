// internal/router/router.go
package router

import (
	"net/http"
	"strings"
	"time"

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
	r.Use(middleware.SecurityHeaders())
	r.Use(corsMiddleware(frontendOrigin))
	r.Use(middleware.RateLimiter(middleware.RateLimitConfig{
		Name:   "global",
		Limit:  300,
		Window: time.Minute,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	authLimiter := middleware.RateLimiter(middleware.RateLimitConfig{
		Name:    "auth",
		Limit:   10,
		Window:  time.Minute,
		KeyFunc: middleware.ClientIPRouteKey,
	})
	uploadLimiter := middleware.RateLimiter(middleware.RateLimitConfig{
		Name:   "upload",
		Limit:  20,
		Window: time.Minute,
	})

	api := r.Group("/api/v1")

	public := api.Group("")
	{
		public.POST("/auth/register", authLimiter, authHandler.Register)
		public.POST("/auth/register-organization", authLimiter, authHandler.RegisterOrganization)
		public.POST("/auth/login", authLimiter, authHandler.Login)
		public.GET("/users/invites/:token", userHandler.GetInvite)
		public.POST("/users/invites/:token/accept", authLimiter, userHandler.AcceptInvite)
	}

	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(jwtSecret, userRepo))
	{
		tickets := protected.Group("/tickets")
		{
			tickets.GET("", ticketHandler.ListTickets)
			tickets.POST("", ticketHandler.CreateTicket)
			tickets.GET("/:id", ticketHandler.GetTicket)
			tickets.PATCH("/:id/status", ticketHandler.UpdateTicketStatus)
			tickets.GET("/:id/comments", ticketHandler.ListComments)
			tickets.POST("/:id/comments", ticketHandler.CreateComment)

			tickets.POST("/:id/attachments", uploadLimiter, attachmentHandler.Upload)
			tickets.GET("/:id/attachments", attachmentHandler.GetByTicket)
			tickets.GET("/:id/attachments/:aid/download", attachmentHandler.Download)
		}

		users := protected.Group("/users")
		users.Use(middleware.RequireRole(models.RoleAdmin))
		{
			users.GET("", userHandler.GetAll)
			users.POST("/invites", userHandler.CreateInvite)
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
