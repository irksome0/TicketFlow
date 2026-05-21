package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"ticketflow-api/internal/models"
	"ticketflow-api/internal/repository"
)

const inviteTTL = 7 * 24 * time.Hour

type UserHandler struct {
	userRepo       repository.UserRepository
	inviteRepo     repository.InviteRepository
	frontendOrigin string
}

func NewUserHandler(
	userRepo repository.UserRepository,
	inviteRepo repository.InviteRepository,
	frontendOrigin string,
) *UserHandler {
	return &UserHandler{
		userRepo:       userRepo,
		inviteRepo:     inviteRepo,
		frontendOrigin: frontendOrigin,
	}
}

type userResponse struct {
	ID             uuid.UUID   `json:"id"`
	OrganizationID uuid.UUID   `json:"organization_id"`
	Email          string      `json:"email"`
	Role           models.Role `json:"role"`
	FirstName      string      `json:"first_name"`
	LastName       string      `json:"last_name"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

type updateUserRoleRequest struct {
	Role models.Role `json:"role" binding:"required"`
}

type createInviteRequest struct {
	Email string      `json:"email" binding:"required,email"`
	Role  models.Role `json:"role" binding:"required"`
}

type inviteResponse struct {
	ID        uuid.UUID   `json:"id"`
	Email     string      `json:"email"`
	Role      models.Role `json:"role"`
	InviteURL string      `json:"invite_url,omitempty"`
	ExpiresAt time.Time   `json:"expires_at"`
	UsedAt    *time.Time  `json:"used_at,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
}

type publicInviteResponse struct {
	Email     string      `json:"email"`
	Role      models.Role `json:"role"`
	ExpiresAt time.Time   `json:"expires_at"`
}

type acceptInviteRequest struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Password  string `json:"password" binding:"required,min=8"`
}

func (h *UserHandler) GetAll(c *gin.Context) {
	orgID, ok := mustGetOrgID(c)
	if !ok {
		return
	}

	users, err := h.userRepo.ListByOrganization(orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка отримання списку користувачів",
		})
		return
	}

	response := make([]userResponse, len(users))
	for i, user := range users {
		response[i] = toUserResponse(user)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  response,
		"total": len(response),
	})
}

func (h *UserHandler) UpdateRole(c *gin.Context) {
	orgID, ok := mustGetOrgID(c)
	if !ok {
		return
	}

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "невалідний ідентифікатор користувача",
		})
		return
	}

	var req updateUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !isValidRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "невалідне значення ролі",
		})
		return
	}

	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "користувача не знайдено"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка отримання користувача",
		})
		return
	}

	if user.OrganizationID != orgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "користувача не знайдено"})
		return
	}

	if err := h.userRepo.UpdateRole(userID, req.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка оновлення ролі користувача",
		})
		return
	}

	updated, err := h.userRepo.FindByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка отримання оновленого користувача",
		})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(*updated))
}

func (h *UserHandler) CreateInvite(c *gin.Context) {
	adminID, ok := mustGetUserID(c)
	if !ok {
		return
	}
	orgID, ok := mustGetOrgID(c)
	if !ok {
		return
	}

	var req createInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	email := normalizeEmail(req.Email)
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email є обов'язковим"})
		return
	}
	if !isValidRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "невалідне значення ролі"})
		return
	}

	if _, err := h.userRepo.FindByEmail(email); err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "користувач із таким email вже існує",
		})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка перевірки користувача",
		})
		return
	}

	token, err := generateInviteToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка генерації запрошення",
		})
		return
	}

	invite := &models.OrganizationInvite{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Email:          email,
		Role:           req.Role,
		TokenHash:      hashInviteToken(token),
		ExpiresAt:      time.Now().UTC().Add(inviteTTL),
		CreatedBy:      adminID,
	}

	if err := h.inviteRepo.Create(invite); err != nil {
		if isDuplicateKeyError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "запрошення вже існує"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка створення запрошення",
		})
		return
	}

	c.JSON(http.StatusCreated, toInviteResponse(*invite, h.inviteURL(token)))
}

func (h *UserHandler) GetInvite(c *gin.Context) {
	invite, ok := h.validInviteByToken(c)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, publicInviteResponse{
		Email:     invite.Email,
		Role:      invite.Role,
		ExpiresAt: invite.ExpiresAt,
	})
}

func (h *UserHandler) AcceptInvite(c *gin.Context) {
	invite, ok := h.validInviteByToken(c)
	if !ok {
		return
	}

	var req acceptInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	firstName := normalizeText(req.FirstName)
	lastName := normalizeText(req.LastName)
	if firstName == "" || lastName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ім'я та прізвище є обов'язковими",
		})
		return
	}

	if textLength(firstName) > maxNameLength || textLength(lastName) > maxNameLength {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ім'я та прізвище не повинні перевищувати 100 символів",
		})
		return
	}

	if _, err := h.userRepo.FindByEmail(invite.Email); err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "користувач із таким email вже існує",
		})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка перевірки користувача",
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка хешування пароля",
		})
		return
	}

	user := &models.User{
		ID:             uuid.New(),
		OrganizationID: invite.OrganizationID,
		Email:          invite.Email,
		PasswordHash:   string(hash),
		Role:           invite.Role,
		TokenVersion:   1,
		FirstName:      firstName,
		LastName:       lastName,
	}

	if err := h.inviteRepo.Accept(invite.ID, user, time.Now().UTC()); err != nil {
		if errors.Is(err, repository.ErrInviteAlreadyUsed) {
			c.JSON(http.StatusConflict, gin.H{"error": "запрошення вже використано"})
			return
		}
		if isDuplicateKeyError(err) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "користувач із таким email вже існує",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка прийняття запрошення",
		})
		return
	}

	c.JSON(http.StatusCreated, toUserResponse(*user))
}

func (h *UserHandler) validInviteByToken(c *gin.Context) (*models.OrganizationInvite, bool) {
	token := strings.TrimSpace(c.Param("token"))
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "токен запрошення є обов'язковим"})
		return nil, false
	}

	invite, err := h.inviteRepo.FindByTokenHash(hashInviteToken(token))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "запрошення не знайдено"})
			return nil, false
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка отримання запрошення",
		})
		return nil, false
	}

	if invite.UsedAt != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "запрошення вже використано"})
		return nil, false
	}
	if time.Now().UTC().After(invite.ExpiresAt) {
		c.JSON(http.StatusConflict, gin.H{"error": "строк дії запрошення минув"})
		return nil, false
	}

	return invite, true
}

func isValidRole(role models.Role) bool {
	switch role {
	case models.RoleClient, models.RoleOperator, models.RoleEngineer, models.RoleAdmin:
		return true
	default:
		return false
	}
}

func toUserResponse(user models.User) userResponse {
	return userResponse{
		ID:             user.ID,
		OrganizationID: user.OrganizationID,
		Email:          user.Email,
		Role:           user.Role,
		FirstName:      user.FirstName,
		LastName:       user.LastName,
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
	}
}

func toInviteResponse(invite models.OrganizationInvite, inviteURL string) inviteResponse {
	return inviteResponse{
		ID:        invite.ID,
		Email:     invite.Email,
		Role:      invite.Role,
		InviteURL: inviteURL,
		ExpiresAt: invite.ExpiresAt,
		UsedAt:    invite.UsedAt,
		CreatedAt: invite.CreatedAt,
	}
}

func generateInviteToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func hashInviteToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (h *UserHandler) inviteURL(token string) string {
	origin := firstFrontendOrigin(h.frontendOrigin)
	if origin == "" {
		return "/register/invite/" + token
	}
	return strings.TrimRight(origin, "/") + "/register/invite/" + token
}

func firstFrontendOrigin(value string) string {
	for _, part := range strings.Split(value, ",") {
		origin := strings.TrimSpace(part)
		if origin != "" && origin != "*" {
			return origin
		}
	}
	return ""
}
