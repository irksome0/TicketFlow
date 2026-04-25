// internal/handlers/users.go
package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserHandler struct {
	db *gorm.DB
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{db: db}
}

func (h *UserHandler) GetAll(c *gin.Context)     {}
func (h *UserHandler) UpdateRole(c *gin.Context) {}
