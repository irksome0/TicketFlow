package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"ticketflow-api/internal/models"
)

// ── Фільтр ───────────────────────────────────────────────────────────────────

// TicketFilter містить опціональні параметри фільтрації списку заявок.
type TicketFilter struct {
	Status     *models.TicketStatus
	Priority   *models.TicketPriority
	AssigneeID *uuid.UUID
	CreatorID  *uuid.UUID
}

// ── Інтерфейс ─────────────────────────────────────────────────────────────────

type TicketRepository interface {
	Create(ticket *models.Ticket) error
	FindByID(id uuid.UUID) (*models.Ticket, error)
	ListByOrganization(
		orgID uuid.UUID,
		filter TicketFilter,
	) ([]models.Ticket, error)
	UpdateStatus(
		id uuid.UUID,
		status models.TicketStatus,
		resolvedAt *time.Time,
	) error
	AssignTo(id uuid.UUID, assigneeID uuid.UUID) error
	CreateComment(comment *models.TicketComment) error
	ListComments(ticketID uuid.UUID) ([]models.TicketComment, error)
}

// ── Реалізація ────────────────────────────────────────────────────────────────

type ticketRepository struct {
	db *gorm.DB
}

// NewTicketRepository створює новий екземпляр ticketRepository.
func NewTicketRepository(db *gorm.DB) TicketRepository {
	return &ticketRepository{db: db}
}

// Create зберігає нову заявку у базі даних.
func (r *ticketRepository) Create(ticket *models.Ticket) error {
	return r.db.Create(ticket).Error
}

// FindByID повертає заявку за її ідентифікатором разом із вкладеннями
// та коментарями.
func (r *ticketRepository) FindByID(
	id uuid.UUID,
) (*models.Ticket, error) {
	var ticket models.Ticket
	err := r.db.
		Preload("Attachments").
		Preload("Comments").
		Where("id = ?", id).
		First(&ticket).Error
	if err != nil {
		return nil, err
	}
	return &ticket, nil
}

// ListByOrganization повертає список заявок організації з урахуванням
// опціональних фільтрів. Записи впорядковано від найновіших до найстаріших.
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
		Order("created_at DESC").
		Find(&tickets).Error

	return tickets, err
}

// UpdateStatus оновлює статус заявки. Якщо новий статус — Resolved,
// встановлюється час вирішення (resolved_at). При переході до будь-якого
// іншого статусу поле resolved_at скидається.
func (r *ticketRepository) UpdateStatus(
	id uuid.UUID,
	status models.TicketStatus,
	resolvedAt *time.Time,
) error {
	updates := map[string]any{
		"status":      status,
		"resolved_at": resolvedAt,
		"updated_at":  time.Now(),
	}
	return r.db.Model(&models.Ticket{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// AssignTo призначає заявку конкретному виконавцю.
func (r *ticketRepository) AssignTo(
	id uuid.UUID,
	assigneeID uuid.UUID,
) error {
	return r.db.Model(&models.Ticket{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"assignee_id": assigneeID,
			"updated_at":  time.Now(),
		}).Error
}

// CreateComment зберігає новий коментар до заявки.
func (r *ticketRepository) CreateComment(
	comment *models.TicketComment,
) error {
	return r.db.Create(comment).Error
}

// ListComments повертає всі коментарі заявки у хронологічному порядку.
func (r *ticketRepository) ListComments(
	ticketID uuid.UUID,
) ([]models.TicketComment, error) {
	var comments []models.TicketComment
	err := r.db.
		Where("ticket_id = ?", ticketID).
		Order("created_at ASC").
		Find(&comments).Error
	return comments, err
}
