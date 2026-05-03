// internal/middleware/auth.go
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"ticketflow-api/internal/models"
	"ticketflow-api/internal/repository"
)

type Claims struct {
	UserID         uuid.UUID   `json:"user_id"`
	OrganizationID uuid.UUID   `json:"organization_id"`
	Role           models.Role `json:"role"`
	TokenVersion   int         `json:"token_version"`
	jwt.RegisteredClaims
}

func AuthMiddleware(jwtSecret string, userRepo repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Відсутній або некоректний токен авторизації",
			})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(
			tokenStr,
			claims,
			func(t *jwt.Token) (interface{}, error) {
				if t.Method != jwt.SigningMethodHS256 {
					return nil, jwt.ErrTokenSignatureInvalid
				}
				return []byte(jwtSecret), nil
			},
		)

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Недійсний або прострочений токен",
			})
			return
		}

		if claims.UserID == uuid.Nil || claims.OrganizationID == uuid.Nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Некоректні дані токена",
			})
			return
		}

		user, err := userRepo.FindByID(claims.UserID)
		if err != nil || user.OrganizationID != claims.OrganizationID {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Користувача не знайдено або токен відкликано",
			})
			return
		}

		if user.TokenVersion != claims.TokenVersion {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Токен відкликано",
			})
			return
		}

		c.Set("user_id", user.ID)
		c.Set("organization_id", user.OrganizationID)
		c.Set("role", user.Role)
		c.Next()
	}
}

func RequireRole(roles ...models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		currentRole, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Доступ заборонено",
			})
			return
		}

		for _, role := range roles {
			if currentRole == role {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "Недостатньо прав для виконання цієї операції",
		})
	}
}
