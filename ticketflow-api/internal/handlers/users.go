// internal/handlers/users.go
package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"ticketflow-api/internal/models"
	"ticketflow-api/internal/repository"
)

type UserHandler struct {
	userRepo repository.UserRepository
}

func NewUserHandler(userRepo repository.UserRepository) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

type userResponse struct {
	ID             uuid.UUID   `json:"id"`
	OrganizationID uuid.UUID   `json:"organization_id"`
	Email          string      `json:"email"`
	Role           models.Role `json:"role"`
	FirstName      string      `json:"first_name"`
	LastName       string      `json:"last_name"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

type updateUserRoleRequest struct {
	Role models.Role `json:"role" binding:"required"`
}

func (h *UserHandler) GetAll(c *gin.Context) {
	orgID, ok := mustGetOrgID(c)
	if !ok {
		return
	}

	users, err := h.userRepo.ListByOrganization(orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка отримання списку користувачів",
		})
		return
	}

	response := make([]userResponse, len(users))
	for i, user := range users {
		response[i] = toUserResponse(user)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  response,
		"total": len(response),
	})
}

func (h *UserHandler) UpdateRole(c *gin.Context) {
	orgID, ok := mustGetOrgID(c)
	if !ok {
		return
	}

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "невалідний ідентифікатор користувача",
		})
		return
	}

	var req updateUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !isValidRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "невалідне значення ролі",
		})
		return
	}

	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "користувача не знайдено"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка отримання користувача",
		})
		return
	}

	if user.OrganizationID != orgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "користувача не знайдено"})
		return
	}

	if err := h.userRepo.UpdateRole(userID, req.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка оновлення ролі користувача",
		})
		return
	}

	updated, err := h.userRepo.FindByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка отримання оновленого користувача",
		})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(*updated))
}

func isValidRole(role models.Role) bool {
	switch role {
	case models.RoleClient, models.RoleOperator, models.RoleEngineer, models.RoleAdmin:
		return true
	default:
		return false
	}
}

func toUserResponse(user models.User) userResponse {
	return userResponse{
		ID:             user.ID,
		OrganizationID: user.OrganizationID,
		Email:          user.Email,
		Role:           user.Role,
		FirstName:      user.FirstName,
		LastName:       user.LastName,
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
	}
}
