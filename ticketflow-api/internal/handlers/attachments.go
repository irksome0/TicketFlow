package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"ticketflow-api/internal/models"
	"ticketflow-api/internal/repository"
)

// ── Константи та допоміжні дані ───────────────────────────────────────────────

const (
	// maxUploadSize визначає максимально допустимий розмір вкладення (5 МБ).
	maxUploadSize = 5 << 20

	// uploadBasePath — кореневий каталог зберігання файлів.
	uploadBasePath = "uploads"
)

// allowedExtensions містить перелік дозволених розширень файлів.
var allowedExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".pdf":  true,
	".txt":  true,
	".log":  true,
}

// extToContentType відповідає розширенню файлу його MIME-тип.
var extToContentType = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".pdf":  "application/pdf",
	".txt":  "text/plain",
	".log":  "text/plain",
}

// ── DTO ───────────────────────────────────────────────────────────────────────

// AttachmentResponse — публічне представлення вкладення без службових полів.
type AttachmentResponse struct {
	ID          uuid.UUID `json:"id"`
	TicketID    uuid.UUID `json:"ticket_id"`
	UploaderID  uuid.UUID `json:"uploader_id"`
	FileName    string    `json:"file_name"`
	FileSize    int       `json:"file_size"`
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`
}

// toAttachmentResponse перетворює модель Attachment на публічне DTO.
func toAttachmentResponse(a models.Attachment) AttachmentResponse {
	return AttachmentResponse{
		ID:          a.ID,
		TicketID:    a.TicketID,
		UploaderID:  a.UploaderID,
		FileName:    a.FileName,
		FileSize:    a.FileSize,
		ContentType: a.ContentType,
		CreatedAt:   a.CreatedAt,
	}
}

// ── Хендлер ───────────────────────────────────────────────────────────────────

// AttachmentHandler обробляє HTTP-запити, пов'язані з вкладеннями.
type AttachmentHandler struct {
	ticketRepo     repository.TicketRepository
	attachmentRepo repository.AttachmentRepository
}

// NewAttachmentHandler створює новий екземпляр AttachmentHandler.
func NewAttachmentHandler(db *gorm.DB) *AttachmentHandler {
	return &AttachmentHandler{
		ticketRepo:     repository.NewTicketRepository(db),
		attachmentRepo: repository.NewAttachmentRepository(db),
	}
}

// ── Обробники маршрутів ───────────────────────────────────────────────────────

// Upload обробляє завантаження файлу-вкладення до заявки.
//
// Доступ: Client (лише до власних заявок), Operator, Engineer.
// Метод: POST /api/v1/tickets/:id/attachments
// Content-Type: multipart/form-data, поле "file".
func (h *AttachmentHandler) Upload(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	orgID := c.MustGet("organization_id").(uuid.UUID)
	role := c.MustGet("role").(models.Role)

	// Адміністратор не має права завантажувати вкладення
	if role == models.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Адміністратор не має права завантажувати вкладення",
		})
		return
	}

	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Некоректний ідентифікатор заявки",
		})
		return
	}

	// Перевірка існування заявки та належності до організації
	ticket, err := h.ticketRepo.FindByID(ticketID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявку не знайдено"})
		return
	}

	if ticket.OrganizationID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Доступ заборонено"})
		return
	}

	// Client може завантажувати вкладення лише до власних заявок
	if role == models.RoleClient && ticket.CreatorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Доступ заборонено"})
		return
	}

	// Отримання файлу з multipart-форми
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Файл не передано. Очікується поле 'file' у multipart/form-data",
		})
		return
	}

	// Перевірка розміру файлу
	if fileHeader.Size > maxUploadSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Розмір файлу перевищує допустимий ліміт у 5 МБ",
		})
		return
	}

	// Перевірка розширення файлу
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !allowedExtensions[ext] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Недопустимий тип файлу. " +
				"Дозволені формати: jpg, jpeg, png, pdf, txt, log",
		})
		return
	}

	contentType := extToContentType[ext]

	// Формування унікального шляху для збереження файлу:
	// uploads/{orgID}/{ticketID}/{uuid}_{originalFileName}
	uploadDir := filepath.Join(
		uploadBasePath,
		orgID.String(),
		ticketID.String(),
	)

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Помилка створення каталогу для збереження файлу",
		})
		return
	}

	originalFileName := filepath.Base(fileHeader.Filename)
	uniqueFileName := fmt.Sprintf(
		"%s_%s",
		uuid.New().String(),
		originalFileName,
	)
	filePath := filepath.Join(uploadDir, uniqueFileName)

	// Збереження файлу на диск
	if err := c.SaveUploadedFile(fileHeader, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Помилка збереження файлу",
		})
		return
	}

	// Збереження метаданих вкладення у базі даних
	attachment := &models.Attachment{
		TicketID:    ticketID,
		UploaderID:  userID,
		FileName:    originalFileName,
		FilePath:    filePath,
		FileSize:    int(fileHeader.Size),
		ContentType: contentType,
	}

	if err := h.attachmentRepo.Create(attachment); err != nil {
		// У разі помилки БД — видаляємо вже збережений файл
		_ = os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Помилка збереження метаданих вкладення",
		})
		return
	}

	c.JSON(http.StatusCreated, toAttachmentResponse(*attachment))
}

// GetByTicket повертає список метаданих усіх вкладень до вказаної заявки.
//
// Доступ: усі ролі в межах організації.
// Client бачить вкладення лише власних заявок.
// Метод: GET /api/v1/tickets/:id/attachments
func (h *AttachmentHandler) GetByTicket(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	orgID := c.MustGet("organization_id").(uuid.UUID)
	role := c.MustGet("role").(models.Role)

	if role == models.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "адміністратор не має доступу до вкладень",
		})
		return
	}

	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Некоректний ідентифікатор заявки",
		})
		return
	}

	// Перевірка існування заявки та належності до організації
	ticket, err := h.ticketRepo.FindByID(ticketID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявку не знайдено"})
		return
	}

	if ticket.OrganizationID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Доступ заборонено"})
		return
	}

	// Client бачить вкладення лише власних заявок
	if role == models.RoleClient && ticket.CreatorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Доступ заборонено"})
		return
	}

	attachments, err := h.attachmentRepo.ListByTicketID(ticketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Помилка отримання списку вкладень",
		})
		return
	}

	response := make([]AttachmentResponse, len(attachments))
	for i, a := range attachments {
		response[i] = toAttachmentResponse(a)
	}

	c.JSON(http.StatusOK, response)
}

// Download обробляє запит на скачування вкладення до заявки.
// Маршрут: GET /api/v1/tickets/:id/attachments/:aid/download
//
// Правила доступу:
//   - Admin  — доступ заборонено
//   - Client — лише власні заявки
//   - Operator, Engineer — будь-яка заявка в межах організації
func (h *AttachmentHandler) Download(c *gin.Context) {
	// Зчитуємо дані автентифікованого користувача з контексту middleware
	userID := c.MustGet("user_id").(uuid.UUID)
	orgID := c.MustGet("organization_id").(uuid.UUID)
	role := c.MustGet("role").(models.Role)

	// Адміністратор не має права на скачування вкладень
	if role == models.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "адміністратор не має доступу до вкладень",
		})
		return
	}

	// Розбираємо ідентифікатор заявки з параметра маршруту :id
	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "невірний формат ідентифікатора заявки",
		})
		return
	}

	// Розбираємо ідентифікатор вкладення з параметра маршруту :aid
	attachmentID, err := uuid.Parse(c.Param("aid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "невірний формат ідентифікатора вкладення",
		})
		return
	}

	// Перевіряємо існування заявки та її належність до поточної організації
	ticket, err := h.ticketRepo.FindByID(ticketID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "заявку не знайдено"})
		return
	}
	if ticket.OrganizationID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "доступ заборонено"})
		return
	}

	// Клієнт має право скачувати вкладення лише власних заявок
	if role == models.RoleClient && ticket.CreatorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "доступ заборонено"})
		return
	}

	// Отримуємо метадані вкладення з бази даних
	attachment, err := h.attachmentRepo.FindByID(attachmentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "вкладення не знайдено"})
		return
	}

	// Захист від IDOR: перевіряємо відповідність вкладення вказаній заявці
	if attachment.TicketID != ticketID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "вкладення не належить до вказаної заявки",
		})
		return
	}

	filePath := attachment.FilePath

	// Перевіряємо фізичну наявність файлу у файловій системі
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "файл не знайдено на сервері",
		})
		return
	}
	if fileInfo.IsDir() {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "шлях вкладення не є файлом",
		})
		return
	}

	// Надсилаємо файл клієнту як вкладення з оригінальним іменем
	c.FileAttachment(filePath, attachment.FileName)
}
