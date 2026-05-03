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

// ── Handler ───────────────────────────────────────────────────────────────────

type TicketHandler struct {
	ticketRepo repository.TicketRepository
}

func NewTicketHandler(ticketRepo repository.TicketRepository) *TicketHandler {
	return &TicketHandler{ticketRepo: ticketRepo}
}

// ── DTO ───────────────────────────────────────────────────────────────────────

type createTicketRequest struct {
	Title       string                `json:"title"       binding:"required,min=3,max=255"`
	Description string                `json:"description" binding:"required,min=10"`
	Priority    models.TicketPriority `json:"priority"    binding:"required"`
}

type updateStatusRequest struct {
	Status models.TicketStatus `json:"status" binding:"required"`
}

type ticketResponse struct {
	ID             uuid.UUID             `json:"id"`
	OrganizationID uuid.UUID             `json:"organization_id"`
	CreatorID      uuid.UUID             `json:"creator_id"`
	AssigneeID     *uuid.UUID            `json:"assignee_id"`
	Title          string                `json:"title"`
	Description    string                `json:"description"`
	Status         models.TicketStatus   `json:"status"`
	Priority       models.TicketPriority `json:"priority"`
	CreatedAt      time.Time             `json:"created_at"`
	ResolvedAt     *time.Time            `json:"resolved_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

// ── Матриця переходів статусів ────────────────────────────────────────────────

// validTransitions визначає дозволені переходи між статусами заявки.
var validTransitions = map[models.TicketStatus][]models.TicketStatus{
	models.StatusNew:        {models.StatusInProgress, models.StatusClosed},
	models.StatusInProgress: {models.StatusResolved, models.StatusClosed},
	models.StatusResolved:   {models.StatusClosed},
	models.StatusClosed:     {models.StatusReopened},
	models.StatusReopened:   {models.StatusInProgress, models.StatusClosed},
}

// roleAllowedTargets визначає статуси, до яких роль може переводити заявку.
var roleAllowedTargets = map[models.Role]map[models.TicketStatus]bool{
	models.RoleClient: {
		models.StatusClosed:   true,
		models.StatusReopened: true,
	},
	models.RoleOperator: {
		models.StatusInProgress: true,
		models.StatusResolved:   true,
		models.StatusClosed:     true,
		models.StatusReopened:   true,
	},
	models.RoleEngineer: {
		models.StatusInProgress: true,
		models.StatusResolved:   true,
	},
	models.RoleAdmin: {
		models.StatusInProgress: true,
		models.StatusResolved:   true,
		models.StatusClosed:     true,
		models.StatusReopened:   true,
	},
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// ListTickets — GET /api/v1/tickets
// Повертає список заявок організації поточного користувача.
// Підтримує фільтрацію через query-параметри: status, priority.
func (h *TicketHandler) ListTickets(c *gin.Context) {
	orgID, ok := mustGetOrgID(c)
	if !ok {
		return
	}

	role, ok := mustGetRole(c)
	if !ok {
		return
	}

	filter := repository.TicketFilter{}

	if s := c.Query("status"); s != "" {
		status := models.TicketStatus(s)
		filter.Status = &status
	}
	if p := c.Query("priority"); p != "" {
		priority := models.TicketPriority(p)
		filter.Priority = &priority
	}

	// Client бачить лише власні заявки.
	if role == models.RoleClient {
		userID, ok := mustGetUserID(c)
		if !ok {
			return
		}
		filter.CreatorID = &userID
	}

	tickets, err := h.ticketRepo.ListByOrganization(orgID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка отримання заявок",
		})
		return
	}

	response := make([]ticketResponse, len(tickets))
	for i, t := range tickets {
		response[i] = toTicketResponse(t)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  response,
		"total": len(response),
	})
}

// CreateTicket — POST /api/v1/tickets
// Створює нову заявку. Автором є поточний автентифікований користувач.
func (h *TicketHandler) CreateTicket(c *gin.Context) {
	var req createTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := mustGetUserID(c)
	if !ok {
		return
	}
	orgID, ok := mustGetOrgID(c)
	if !ok {
		return
	}

	if !isValidPriority(req.Priority) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "невалідне значення пріоритету",
		})
		return
	}

	ticket := &models.Ticket{
		ID:             uuid.New(),
		OrganizationID: orgID,
		CreatorID:      userID,
		Title:          req.Title,
		Description:    req.Description,
		Status:         models.StatusNew,
		Priority:       req.Priority,
	}

	if err := h.ticketRepo.Create(ticket); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка створення заявки",
		})
		return
	}

	c.JSON(http.StatusCreated, toTicketResponse(*ticket))
}

// GetTicket — GET /api/v1/tickets/:id
// Повертає детальну інформацію про заявку разом із вкладеннями та коментарями.
func (h *TicketHandler) GetTicket(c *gin.Context) {
	ticketID, ok := mustParseTicketID(c)
	if !ok {
		return
	}

	orgID, ok := mustGetOrgID(c)
	if !ok {
		return
	}
	role, ok := mustGetRole(c)
	if !ok {
		return
	}

	ticket, err := h.ticketRepo.FindByID(ticketID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "заявку не знайдено"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка отримання заявки",
		})
		return
	}

	// Захист multi-tenant: перевірка належності заявки до організації.
	if ticket.OrganizationID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "доступ заборонено"})
		return
	}

	if role == models.RoleClient {
		userID, ok := mustGetUserID(c)
		if !ok {
			return
		}
		if ticket.CreatorID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "доступ заборонено"})
			return
		}
	}

	c.JSON(http.StatusOK, toTicketResponse(*ticket))
}

// UpdateTicketStatus — PATCH /api/v1/tickets/:id/status
// Змінює статус заявки відповідно до матриці переходів та ролі користувача.
func (h *TicketHandler) UpdateTicketStatus(c *gin.Context) {
	var req updateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ticketID, ok := mustParseTicketID(c)
	if !ok {
		return
	}
	orgID, ok := mustGetOrgID(c)
	if !ok {
		return
	}
	role, ok := mustGetRole(c)
	if !ok {
		return
	}

	ticket, err := h.ticketRepo.FindByID(ticketID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "заявку не знайдено"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка отримання заявки",
		})
		return
	}

	// Захист multi-tenant.
	if ticket.OrganizationID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "доступ заборонено"})
		return
	}

	if role == models.RoleClient {
		userID, ok := mustGetUserID(c)
		if !ok {
			return
		}
		if ticket.CreatorID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "доступ заборонено"})
			return
		}
	}

	// Перевірка дозволу ролі на цільовий статус.
	if !roleAllowedTargets[role][req.Status] {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "ваша роль не може встановити цей статус",
		})
		return
	}

	// Перевірка валідності переходу статусу.
	if !isValidTransition(ticket.Status, req.Status) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":            "недопустимий перехід статусу",
			"current_status":   ticket.Status,
			"requested_status": req.Status,
		})
		return
	}

	// Встановлення resolved_at лише при переході до Resolved.
	var resolvedAt *time.Time
	if req.Status == models.StatusResolved {
		now := time.Now()
		resolvedAt = &now
	}

	if err := h.ticketRepo.UpdateStatus(ticketID, req.Status, resolvedAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка оновлення статусу",
		})
		return
	}

	// Повернути оновлений стан заявки.
	updated, err := h.ticketRepo.FindByID(ticketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка отримання оновленої заявки",
		})
		return
	}

	c.JSON(http.StatusOK, toTicketResponse(*updated))
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func isValidTransition(from, to models.TicketStatus) bool {
	allowed, exists := validTransitions[from]
	if !exists {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

func isValidPriority(p models.TicketPriority) bool {
	switch p {
	case models.PriorityHigh, models.PriorityMedium, models.PriorityLow:
		return true
	}
	return false
}

func toTicketResponse(t models.Ticket) ticketResponse {
	return ticketResponse{
		ID:             t.ID,
		OrganizationID: t.OrganizationID,
		CreatorID:      t.CreatorID,
		AssigneeID:     t.AssigneeID,
		Title:          t.Title,
		Description:    t.Description,
		Status:         t.Status,
		Priority:       t.Priority,
		CreatedAt:      t.CreatedAt,
		ResolvedAt:     t.ResolvedAt,
		UpdatedAt:      t.UpdatedAt,
	}
}

// mustGetUserID витягує UserID з контексту Gin.
func mustGetUserID(c *gin.Context) (uuid.UUID, bool) {
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "не автентифіковано"})
		return uuid.Nil, false
	}
	id, ok := val.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "некоректний формат user_id",
		})
		return uuid.Nil, false
	}
	return id, true
}

// mustGetOrgID витягує OrganizationID з контексту Gin.
func mustGetOrgID(c *gin.Context) (uuid.UUID, bool) {
	val, exists := c.Get("organization_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "не автентифіковано"})
		return uuid.Nil, false
	}
	id, ok := val.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "некоректний формат organization_id",
		})
		return uuid.Nil, false
	}
	return id, true
}

// mustGetRole витягує Role з контексту Gin.
func mustGetRole(c *gin.Context) (models.Role, bool) {
	val, exists := c.Get("role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "не автентифіковано"})
		return "", false
	}
	role, ok := val.(models.Role)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "некоректний формат role",
		})
		return "", false
	}
	return role, true
}

// mustParseTicketID парсить UUID заявки з параметрів маршруту.
func mustParseTicketID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "невалідний ідентифікатор заявки",
		})
		return uuid.Nil, false
	}
	return id, true
}
