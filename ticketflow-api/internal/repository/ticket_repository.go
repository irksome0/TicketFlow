package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"ticketflow-api/internal/models"
)

type TicketFilter struct {
	Status     *models.TicketStatus
	Priority   *models.TicketPriority
	AssigneeID *uuid.UUID
	CreatorID  *uuid.UUID
}

type TicketRepository interface {
	Create(ticket *models.Ticket) error
	FindByID(id uuid.UUID) (*models.Ticket, error)
	ListByOrganization(orgID uuid.UUID, filter TicketFilter) ([]models.Ticket, error)
	UpdateStatus(
		id uuid.UUID,
		fromStatus models.TicketStatus,
		toStatus models.TicketStatus,
		resolvedAt *time.Time,
		changedBy uuid.UUID,
	) error
	ListStatusHistory(ticketID uuid.UUID) ([]models.TicketStatusHistory, error)
	AssignTo(id uuid.UUID, assigneeID uuid.UUID) error
	CreateComment(comment *models.TicketComment) error
	ListComments(ticketID uuid.UUID) ([]models.TicketComment, error)
}

type ticketRepository struct {
	db *gorm.DB
}

func NewTicketRepository(db *gorm.DB) TicketRepository {
	return &ticketRepository{db: db}
}

func (r *ticketRepository) Create(ticket *models.Ticket) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(ticket).Error; err != nil {
			return err
		}

		return tx.Create(&models.TicketStatusHistory{
			TicketID:   ticket.ID,
			ChangedBy:  ticket.CreatorID,
			FromStatus: nil,
			ToStatus:   ticket.Status,
		}).Error
	})
}

func (r *ticketRepository) FindByID(id uuid.UUID) (*models.Ticket, error) {
	var ticket models.Ticket
	err := r.db.
		Preload("Attachments").
		Preload("Comments").
		Preload("StatusHistory", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Where("id = ?", id).
		First(&ticket).Error
	if err != nil {
		return nil, err
	}
	return &ticket, nil
}

func (r *ticketRepository) ListByOrganization(
	orgID uuid.UUID,
	filter TicketFilter,
) ([]models.Ticket, error) {
	var tickets []models.Ticket

	query := r.db.Where("organization_id = ?", orgID)

	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.Priority != nil {
		query = query.Where("priority = ?", *filter.Priority)
	}
	if filter.AssigneeID != nil {
		query = query.Where("assignee_id = ?", *filter.AssigneeID)
	}
	if filter.CreatorID != nil {
		query = query.Where("creator_id = ?", *filter.CreatorID)
	}

	err := query.
		Preload("StatusHistory", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Order("created_at DESC").
		Find(&tickets).Error

	return tickets, err
}

func (r *ticketRepository) UpdateStatus(
	id uuid.UUID,
	fromStatus models.TicketStatus,
	toStatus models.TicketStatus,
	resolvedAt *time.Time,
	changedBy uuid.UUID,
) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{
			"status":     toStatus,
			"updated_at": time.Now().UTC(),
		}
		if resolvedAt != nil {
			updates["resolved_at"] = resolvedAt
		}
		if toStatus == models.StatusReopened {
			updates["resolved_at"] = nil
		}

		if err := tx.Model(&models.Ticket{}).
			Where("id = ?", id).
			Updates(updates).Error; err != nil {
			return err
		}

		return tx.Create(&models.TicketStatusHistory{
			TicketID:   id,
			ChangedBy:  changedBy,
			FromStatus: &fromStatus,
			ToStatus:   toStatus,
		}).Error
	})
}

func (r *ticketRepository) ListStatusHistory(
	ticketID uuid.UUID,
) ([]models.TicketStatusHistory, error) {
	var history []models.TicketStatusHistory
	err := r.db.
		Where("ticket_id = ?", ticketID).
		Order("created_at ASC").
		Find(&history).Error
	return history, err
}

func (r *ticketRepository) AssignTo(
	id uuid.UUID,
	assigneeID uuid.UUID,
) error {
	return r.db.Model(&models.Ticket{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"assignee_id": assigneeID,
			"updated_at":  time.Now().UTC(),
		}).Error
}

func (r *ticketRepository) CreateComment(
	comment *models.TicketComment,
) error {
	if err := r.db.Create(comment).Error; err != nil {
		return err
	}

	return r.db.Preload("Author").First(comment, "id = ?", comment.ID).Error
}

func (r *ticketRepository) ListComments(
	ticketID uuid.UUID,
) ([]models.TicketComment, error) {
	var comments []models.TicketComment
	err := r.db.
		Preload("Author").
		Where("ticket_id = ?", ticketID).
		Order("created_at ASC").
		Find(&comments).Error
	return comments, err
}
