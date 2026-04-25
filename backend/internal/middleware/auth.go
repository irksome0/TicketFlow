// internal/middleware/auth.go
package middleware

import (
	"net/http"
	"strings"
	"ticketflow-api/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID         string      `json:"user_id"`
	OrganizationID string      `json:"organization_id"`
	Role           models.Role `json:"role"`
	jwt.RegisteredClaims
}

func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
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
				return []byte(jwtSecret), nil
			},
		)

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Недійсний або прострочений токен",
			})
			return
		}

		// Запис даних користувача у контекст запиту
		c.Set("user_id", claims.UserID)
		c.Set("organization_id", claims.OrganizationID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// RequireRole перевіряє, чи має поточний користувач необхідну роль
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
