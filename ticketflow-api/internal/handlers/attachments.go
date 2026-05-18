package handlers

import (
	"errors"
	"fmt"
	"io"
	"mime"
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
	"ticketflow-api/internal/security"
	"ticketflow-api/internal/storage"
)

// ── Константи та допоміжні дані ───────────────────────────────────────────────

const (
	// maxUploadSize визначає максимально допустимий розмір вкладення (25 МБ).
	maxUploadSize = 25 << 20
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
	scanner        security.AttachmentScanner
	storage        storage.FileStorage
	storagePrefix  string
}

// NewAttachmentHandler створює новий екземпляр AttachmentHandler.
func NewAttachmentHandler(db *gorm.DB, scanner security.AttachmentScanner, fileStorage storage.FileStorage, storagePrefix string) *AttachmentHandler {
	if scanner == nil {
		scanner = security.NoopAttachmentScanner{}
	}
	return &AttachmentHandler{
		ticketRepo:     repository.NewTicketRepository(db),
		attachmentRepo: repository.NewAttachmentRepository(db),
		scanner:        scanner,
		storage:        fileStorage,
		storagePrefix:  storagePrefix,
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
			"error": "Розмір файлу перевищує допустимий ліміт у 25 МБ",
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

	originalFileName := filepath.Base(fileHeader.Filename)
	uniqueFileName := fmt.Sprintf(
		"%s_%s",
		uuid.New().String(),
		originalFileName,
	)
	storageKey := h.buildStorageKey(orgID, ticketID, uniqueFileName)

	tempFile, err := os.CreateTemp("", "ticketflow-attachment-*"+ext)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка підготовки тимчасового файлу",
		})
		return
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	src, err := fileHeader.Open()
	if err != nil {
		_ = tempFile.Close()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка відкриття вкладення",
		})
		return
	}
	if _, err := io.Copy(tempFile, src); err != nil {
		_ = src.Close()
		_ = tempFile.Close()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка збереження тимчасового файлу",
		})
		return
	}
	_ = src.Close()
	if err := tempFile.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка завершення запису тимчасового файлу",
		})
		return
	}

	if err := h.scanner.ScanFile(tempPath); err != nil {
		if errors.Is(err, security.ErrMaliciousAttachment) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "файл не пройшов антивірусну перевірку",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка антивірусної перевірки",
		})
		return
	}

	fileForStorage, err := os.Open(tempPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка відкриття перевіреного файлу",
		})
		return
	}
	defer fileForStorage.Close()

	if err := h.storage.Save(c.Request.Context(), storageKey, fileForStorage, contentType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка збереження файлу у сховищі",
		})
		return
	}

	// Збереження метаданих вкладення у базі даних
	attachment := &models.Attachment{
		TicketID:    ticketID,
		UploaderID:  userID,
		FileName:    originalFileName,
		FilePath:    storageKey,
		FileSize:    int(fileHeader.Size),
		ContentType: contentType,
	}

	if err := h.attachmentRepo.Create(attachment); err != nil {
		_ = h.storage.Delete(c.Request.Context(), storageKey)
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

	file, err := h.storage.Open(c.Request.Context(), attachment.FilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "файл не знайдено у сховищі",
		})
		return
	}
	defer file.Close()

	c.Header("Content-Type", attachment.ContentType)
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{
		"filename": attachment.FileName,
	}))
	c.Header("Content-Length", fmt.Sprintf("%d", attachment.FileSize))
	if _, err := io.Copy(c.Writer, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "помилка передачі файлу"})
		return
	}
}

func (h *AttachmentHandler) buildStorageKey(orgID, ticketID uuid.UUID, fileName string) string {
	parts := []string{
		orgID.String(),
		ticketID.String(),
		fileName,
	}
	if h.storagePrefix != "" {
		parts = append([]string{h.storagePrefix}, parts...)
	}
	return strings.Join(parts, "/")
}
