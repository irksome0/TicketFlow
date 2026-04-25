// internal/handlers/tickets.go
package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TicketHandler struct {
	db *gorm.DB
}

func NewTicketHandler(db *gorm.DB) *TicketHandler {
	return &TicketHandler{db: db}
}

func (h *TicketHandler) GetAll(c *gin.Context)       {}
func (h *TicketHandler) Create(c *gin.Context)       {}
func (h *TicketHandler) GetByID(c *gin.Context)      {}
func (h *TicketHandler) UpdateStatus(c *gin.Context) {}
