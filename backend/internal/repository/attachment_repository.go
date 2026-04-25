package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"ticketflow-api/internal/models"
)

// ── Інтерфейс ─────────────────────────────────────────────────────────────────

// AttachmentRepository визначає контракт для роботи з вкладеннями у сховищі.
type AttachmentRepository interface {
	Create(attachment *models.Attachment) error
	ListByTicketID(ticketID uuid.UUID) ([]models.Attachment, error)
	FindByID(id uuid.UUID) (*models.Attachment, error)
}

// ── Реалізація ────────────────────────────────────────────────────────────────

type attachmentRepository struct {
	db *gorm.DB
}

// NewAttachmentRepository створює новий екземпляр attachmentRepository.
func NewAttachmentRepository(db *gorm.DB) AttachmentRepository {
	return &attachmentRepository{db: db}
}

// Create зберігає запис про вкладення у базі даних.
func (r *attachmentRepository) Create(
	attachment *models.Attachment,
) error {
	return r.db.Create(attachment).Error
}

// ListByTicketID повертає всі вкладення заявки у хронологічному порядку.
func (r *attachmentRepository) ListByTicketID(
	ticketID uuid.UUID,
) ([]models.Attachment, error) {
	var attachments []models.Attachment
	err := r.db.
		Where("ticket_id = ?", ticketID).
		Order("created_at ASC").
		Find(&attachments).Error
	return attachments, err
}

// FindByID повертає вкладення за його ідентифікатором.
func (r *attachmentRepository) FindByID(
	id uuid.UUID,
) (*models.Attachment, error) {
	var attachment models.Attachment
	err := r.db.
		Where("id = ?", id).
		First(&attachment).Error
	if err != nil {
		return nil, err
	}
	return &attachment, nil
}
