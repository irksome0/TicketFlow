package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"ticketflow-api/internal/models"
	"ticketflow-api/internal/repository"
)

type TicketHandler struct {
	ticketRepo repository.TicketRepository
}

func NewTicketHandler(ticketRepo repository.TicketRepository) *TicketHandler {
	return &TicketHandler{ticketRepo: ticketRepo}
}

type createTicketRequest struct {
	Title       string                `json:"title"       binding:"required,min=3,max=255"`
	Description string                `json:"description" binding:"required,min=10"`
	Priority    models.TicketPriority `json:"priority"    binding:"required"`
}

type updateStatusRequest struct {
	Status models.TicketStatus `json:"status" binding:"required"`
}

type ticketResponse struct {
	ID                    uuid.UUID               `json:"id"`
	OrganizationID        uuid.UUID               `json:"organization_id"`
	CreatorID             uuid.UUID               `json:"creator_id"`
	AssigneeID            *uuid.UUID              `json:"assignee_id"`
	Title                 string                  `json:"title"`
	Description           string                  `json:"description"`
	Status                models.TicketStatus     `json:"status"`
	Priority              models.TicketPriority   `json:"priority"`
	CreatedAt             time.Time               `json:"created_at"`
	ResolvedAt            *time.Time              `json:"resolved_at"`
	UpdatedAt             time.Time               `json:"updated_at"`
	SlaLimitSeconds       int64                   `json:"sla_limit_seconds"`
	ActiveDurationSeconds int64                   `json:"active_duration_seconds"`
	SlaStatus             string                  `json:"sla_status"`
	StatusHistory         []statusHistoryResponse `json:"status_history,omitempty"`
}

type statusHistoryResponse struct {
	ID         uuid.UUID            `json:"id"`
	TicketID   uuid.UUID            `json:"ticket_id"`
	ChangedBy  uuid.UUID            `json:"changed_by"`
	FromStatus *models.TicketStatus `json:"from_status"`
	ToStatus   models.TicketStatus  `json:"to_status"`
	CreatedAt  time.Time            `json:"created_at"`
}

type ticketCommentResponse struct {
	ID        uuid.UUID `json:"id"`
	TicketID  uuid.UUID `json:"ticket_id"`
	AuthorID  uuid.UUID `json:"author_id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type createTicketCommentRequest struct {
	Message string `json:"message" binding:"required"`
}

var validTransitions = map[models.TicketStatus][]models.TicketStatus{
	models.StatusNew:        {models.StatusInProgress, models.StatusPending, models.StatusClosed},
	models.StatusInProgress: {models.StatusPending, models.StatusWaiting, models.StatusOnHold, models.StatusResolved, models.StatusClosed},
	models.StatusPending:    {models.StatusInProgress, models.StatusWaiting, models.StatusOnHold, models.StatusClosed},
	models.StatusWaiting:    {models.StatusInProgress, models.StatusOnHold, models.StatusClosed},
	models.StatusOnHold:     {models.StatusInProgress, models.StatusPending, models.StatusClosed},
	models.StatusResolved:   {models.StatusClosed},
	models.StatusClosed:     {models.StatusReopened},
	models.StatusReopened:   {models.StatusInProgress, models.StatusPending, models.StatusClosed},
}

var roleAllowedTargets = map[models.Role]map[models.TicketStatus]bool{
	models.RoleClient: {
		models.StatusClosed:   true,
		models.StatusReopened: true,
	},
	models.RoleOperator: {
		models.StatusInProgress: true,
		models.StatusPending:    true,
		models.StatusWaiting:    true,
		models.StatusOnHold:     true,
		models.StatusResolved:   true,
		models.StatusClosed:     true,
		models.StatusReopened:   true,
	},
	models.RoleEngineer: {
		models.StatusInProgress: true,
		models.StatusPending:    true,
		models.StatusWaiting:    true,
		models.StatusOnHold:     true,
		models.StatusResolved:   true,
	},
	models.RoleAdmin: {
		models.StatusInProgress: true,
		models.StatusPending:    true,
		models.StatusWaiting:    true,
		models.StatusOnHold:     true,
		models.StatusResolved:   true,
		models.StatusClosed:     true,
		models.StatusReopened:   true,
	},
}

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
		if !isValidStatus(status) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "невалідне значення статусу"})
			return
		}
		filter.Status = &status
	}
	if p := c.Query("priority"); p != "" {
		priority := models.TicketPriority(p)
		if !isValidPriority(priority) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "невалідне значення пріоритету"})
			return
		}
		filter.Priority = &priority
	}

	if role == models.RoleClient {
		userID, ok := mustGetUserID(c)
		if !ok {
			return
		}
		filter.CreatorID = &userID
	}

	tickets, err := h.ticketRepo.ListByOrganization(orgID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "помилка отримання заявок"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "невалідне значення пріоритету"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "помилка створення заявки"})
		return
	}

	created, err := h.ticketRepo.FindByID(ticket.ID)
	if err != nil {
		c.JSON(http.StatusCreated, toTicketResponse(*ticket))
		return
	}

	c.JSON(http.StatusCreated, toTicketResponse(*created))
}

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "помилка отримання заявки"})
		return
	}

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

func (h *TicketHandler) UpdateTicketStatus(c *gin.Context) {
	var req updateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !isValidStatus(req.Status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "невалідне значення статусу"})
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
	userID, ok := mustGetUserID(c)
	if !ok {
		return
	}

	ticket, err := h.ticketRepo.FindByID(ticketID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "заявку не знайдено"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "помилка отримання заявки"})
		return
	}

	if ticket.OrganizationID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "доступ заборонено"})
		return
	}

	if role == models.RoleClient && ticket.CreatorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "доступ заборонено"})
		return
	}

	if !roleAllowedTargets[role][req.Status] {
		c.JSON(http.StatusForbidden, gin.H{"error": "ваша роль не може встановити цей статус"})
		return
	}

	if !isValidTransition(ticket.Status, req.Status) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":            "недопустимий перехід статусу",
			"current_status":   ticket.Status,
			"requested_status": req.Status,
		})
		return
	}

	var resolvedAt *time.Time
	if req.Status == models.StatusResolved {
		now := time.Now().UTC()
		resolvedAt = &now
	}

	if err := h.ticketRepo.UpdateStatus(
		ticketID,
		ticket.Status,
		req.Status,
		resolvedAt,
		userID,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "помилка оновлення статусу"})
		return
	}

	updated, err := h.ticketRepo.FindByID(ticketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "помилка отримання оновленої заявки"})
		return
	}

	c.JSON(http.StatusOK, toTicketResponse(*updated))
}

func (h *TicketHandler) ListComments(c *gin.Context) {
	ticket, ok := h.authorizedTicket(c)
	if !ok {
		return
	}

	comments, err := h.ticketRepo.ListComments(ticket.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "помилка отримання коментарів"})
		return
	}

	response := make([]ticketCommentResponse, len(comments))
	for i, comment := range comments {
		response[i] = toTicketCommentResponse(comment)
	}

	c.JSON(http.StatusOK, response)
}

func (h *TicketHandler) CreateComment(c *gin.Context) {
	ticket, ok := h.authorizedTicket(c)
	if !ok {
		return
	}

	userID, ok := mustGetUserID(c)
	if !ok {
		return
	}

	var req createTicketCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message := strings.TrimSpace(req.Message)
	if message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "коментар не може бути порожнім"})
		return
	}

	comment := &models.TicketComment{
		ID:       uuid.New(),
		TicketID: ticket.ID,
		AuthorID: userID,
		Message:  message,
	}

	if err := h.ticketRepo.CreateComment(comment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "помилка створення коментаря"})
		return
	}

	c.JSON(http.StatusCreated, toTicketCommentResponse(*comment))
}

func (h *TicketHandler) authorizedTicket(c *gin.Context) (*models.Ticket, bool) {
	ticketID, ok := mustParseTicketID(c)
	if !ok {
		return nil, false
	}

	orgID, ok := mustGetOrgID(c)
	if !ok {
		return nil, false
	}
	role, ok := mustGetRole(c)
	if !ok {
		return nil, false
	}

	ticket, err := h.ticketRepo.FindByID(ticketID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "заявку не знайдено"})
			return nil, false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "помилка отримання заявки"})
		return nil, false
	}

	if ticket.OrganizationID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "доступ заборонено"})
		return nil, false
	}

	if role == models.RoleClient {
		userID, ok := mustGetUserID(c)
		if !ok {
			return nil, false
		}
		if ticket.CreatorID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "доступ заборонено"})
			return nil, false
		}
	}

	return ticket, true
}

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
	default:
		return false
	}
}

func isValidStatus(status models.TicketStatus) bool {
	switch status {
	case models.StatusNew,
		models.StatusInProgress,
		models.StatusPending,
		models.StatusWaiting,
		models.StatusOnHold,
		models.StatusResolved,
		models.StatusClosed,
		models.StatusReopened:
		return true
	default:
		return false
	}
}

func toTicketResponse(t models.Ticket) ticketResponse {
	limit := slaLimit(t.Priority)
	active := activeDuration(t, time.Now().UTC())

	history := make([]statusHistoryResponse, len(t.StatusHistory))
	for i, item := range t.StatusHistory {
		history[i] = statusHistoryResponse{
			ID:         item.ID,
			TicketID:   item.TicketID,
			ChangedBy:  item.ChangedBy,
			FromStatus: item.FromStatus,
			ToStatus:   item.ToStatus,
			CreatedAt:  item.CreatedAt,
		}
	}

	return ticketResponse{
		ID:                    t.ID,
		OrganizationID:        t.OrganizationID,
		CreatorID:             t.CreatorID,
		AssigneeID:            t.AssigneeID,
		Title:                 t.Title,
		Description:           t.Description,
		Status:                t.Status,
		Priority:              t.Priority,
		CreatedAt:             t.CreatedAt,
		ResolvedAt:            t.ResolvedAt,
		UpdatedAt:             t.UpdatedAt,
		SlaLimitSeconds:       int64(limit.Seconds()),
		ActiveDurationSeconds: int64(active.Seconds()),
		SlaStatus:             slaStatus(t.Status, active, limit),
		StatusHistory:         history,
	}
}

func toTicketCommentResponse(comment models.TicketComment) ticketCommentResponse {
	return ticketCommentResponse{
		ID:        comment.ID,
		TicketID:  comment.TicketID,
		AuthorID:  comment.AuthorID,
		Message:   comment.Message,
		CreatedAt: comment.CreatedAt,
	}
}

func slaLimit(priority models.TicketPriority) time.Duration {
	switch priority {
	case models.PriorityHigh:
		return 8 * time.Hour
	case models.PriorityLow:
		return 72 * time.Hour
	default:
		return 24 * time.Hour
	}
}

func activeDuration(ticket models.Ticket, now time.Time) time.Duration {
	if len(ticket.StatusHistory) == 0 {
		end := now
		if ticket.ResolvedAt != nil {
			end = *ticket.ResolvedAt
		}
		if end.Before(ticket.CreatedAt) {
			return 0
		}
		return end.Sub(ticket.CreatedAt)
	}

	var total time.Duration
	for i, item := range ticket.StatusHistory {
		start := item.CreatedAt
		end := now
		if i+1 < len(ticket.StatusHistory) {
			end = ticket.StatusHistory[i+1].CreatedAt
		} else if ticket.ResolvedAt != nil && ticket.ResolvedAt.Before(now) {
			end = *ticket.ResolvedAt
		}

		if isActiveSLAStatus(item.ToStatus) && end.After(start) {
			total += end.Sub(start)
		}
	}

	return total
}

func isActiveSLAStatus(status models.TicketStatus) bool {
	switch status {
	case models.StatusNew, models.StatusInProgress, models.StatusReopened:
		return true
	default:
		return false
	}
}

func slaStatus(status models.TicketStatus, active, limit time.Duration) string {
	switch status {
	case models.StatusPending, models.StatusWaiting, models.StatusOnHold:
		return "Paused"
	case models.StatusResolved, models.StatusClosed:
		if active <= limit {
			return "Met"
		}
		return "Breached"
	default:
		if active <= limit {
			return "Within SLA"
		}
		return "Breached"
	}
}

func mustGetUserID(c *gin.Context) (uuid.UUID, bool) {
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "не автентифіковано"})
		return uuid.Nil, false
	}
	id, ok := val.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "некоректний формат user_id"})
		return uuid.Nil, false
	}
	return id, true
}

func mustGetOrgID(c *gin.Context) (uuid.UUID, bool) {
	val, exists := c.Get("organization_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "не автентифіковано"})
		return uuid.Nil, false
	}
	id, ok := val.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "некоректний формат organization_id"})
		return uuid.Nil, false
	}
	return id, true
}

func mustGetRole(c *gin.Context) (models.Role, bool) {
	val, exists := c.Get("role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "не автентифіковано"})
		return "", false
	}
	role, ok := val.(models.Role)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "некоректний формат role"})
		return "", false
	}
	return role, true
}

func mustParseTicketID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "невалідний ідентифікатор заявки"})
		return uuid.Nil, false
	}
	return id, true
}
