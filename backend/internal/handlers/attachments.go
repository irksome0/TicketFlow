// internal/handlers/attachments.go
package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AttachmentHandler struct {
	db *gorm.DB
}

func NewAttachmentHandler(db *gorm.DB) *AttachmentHandler {
	return &AttachmentHandler{db: db}
}

func (h *AttachmentHandler) Upload(c *gin.Context)      {}
func (h *AttachmentHandler) GetByTicket(c *gin.Context) {}
