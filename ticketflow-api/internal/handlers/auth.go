package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"ticketflow-api/internal/models"
	"ticketflow-api/internal/repository"
)

// AuthHandler містить залежності обробника автентифікації.
type AuthHandler struct {
	userRepo         repository.UserRepository
	organizationRepo repository.OrganizationRepository
	jwtSecret        string
	jwtTTL           time.Duration
}

// NewAuthHandler створює новий екземпляр AuthHandler.
func NewAuthHandler(
	userRepo repository.UserRepository,
	organizationRepo repository.OrganizationRepository,
	jwtSecret string,
	jwtTTL time.Duration,
) *AuthHandler {
	return &AuthHandler{
		userRepo:         userRepo,
		organizationRepo: organizationRepo,
		jwtSecret:        jwtSecret,
		jwtTTL:           jwtTTL,
	}
}

// ── DTO ──────────────────────────────────────────────────────────────────────

type registerRequest struct {
	Email          string `json:"email"           binding:"required,email"`
	Password       string `json:"password"        binding:"required,min=8"`
	FirstName      string `json:"first_name"      binding:"required"`
	LastName       string `json:"last_name"       binding:"required"`
	OrganizationID string `json:"organization_id" binding:"required,uuid"`
}

type registerOrganizationRequest struct {
	OrganizationName string `json:"organization_name" binding:"required"`
	Email            string `json:"email"             binding:"required,email"`
	Password         string `json:"password"          binding:"required,min=8"`
	FirstName        string `json:"first_name"        binding:"required"`
	LastName         string `json:"last_name"         binding:"required"`
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type userPayload struct {
	ID             uuid.UUID   `json:"id"`
	Email          string      `json:"email"`
	FirstName      string      `json:"first_name"`
	LastName       string      `json:"last_name"`
	Role           models.Role `json:"role"`
	OrganizationID uuid.UUID   `json:"organization_id"`
}

type authResponse struct {
	Token string      `json:"token"`
	User  userPayload `json:"user"`
}

// ── JWT Claims ────────────────────────────────────────────────────────────────

// Claims визначає структуру корисного навантаження JWT-токена.
type Claims struct {
	UserID         uuid.UUID   `json:"user_id"`
	Role           models.Role `json:"role"`
	OrganizationID uuid.UUID   `json:"organization_id"`
	TokenVersion   int         `json:"token_version"`
	jwt.RegisteredClaims
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// Register — POST /api/v1/auth/register
// Реєструє нового користувача з роллю Client та повертає JWT-токен.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	email := normalizeEmail(req.Email)
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

	orgID, err := uuid.Parse(req.OrganizationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "невалідний ідентифікатор організації",
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка хешування пароля",
		})
		return
	}

	user := &models.User{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Email:          email,
		PasswordHash:   string(hash),
		Role:           models.RoleClient,
		TokenVersion:   1,
		FirstName:      firstName,
		LastName:       lastName,
	}

	if err := h.userRepo.Create(user); err != nil {
		if isDuplicateKeyError(err) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "користувач із таким email вже існує",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка створення користувача",
		})
		return
	}

	token, err := h.generateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка генерації токена",
		})
		return
	}

	c.JSON(http.StatusCreated, authResponse{
		Token: token,
		User:  toUserPayload(user),
	})
}

// RegisterOrganization — POST /api/v1/auth/register-organization
// Створює нову клієнтську організацію та першого адміністратора цієї організації.
func (h *AuthHandler) RegisterOrganization(c *gin.Context) {
	var req registerOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	organizationName := normalizeText(req.OrganizationName)
	firstName := normalizeText(req.FirstName)
	lastName := normalizeText(req.LastName)
	email := normalizeEmail(req.Email)

	if organizationName == "" || firstName == "" || lastName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "назва організації, ім'я та прізвище є обов'язковими",
		})
		return
	}
	if textLength(firstName) > maxNameLength || textLength(lastName) > maxNameLength {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ім'я та прізвище не повинні перевищувати 100 символів",
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка хешування пароля",
		})
		return
	}

	organization := &models.Organization{
		ID:   uuid.New(),
		Name: organizationName,
	}
	admin := &models.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
		Role:         models.RoleAdmin,
		TokenVersion: 1,
		FirstName:    firstName,
		LastName:     lastName,
	}

	if err := h.organizationRepo.CreateWithAdmin(organization, admin); err != nil {
		if isDuplicateKeyError(err) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "організація або користувач з такими даними вже існує",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка створення організації",
		})
		return
	}

	token, err := h.generateToken(admin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка генерації токена",
		})
		return
	}

	c.JSON(http.StatusCreated, authResponse{
		Token: token,
		User:  toUserPayload(admin),
	})
}

// Login — POST /api/v1/auth/login
// Автентифікує користувача та повертає JWT-токен.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	email := normalizeEmail(req.Email)
	user, err := h.userRepo.FindByEmail(email)
	if err != nil {
		// Однакова відповідь для обох випадків — захист від user enumeration.
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "невірний email або пароль",
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(req.Password),
	); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "невірний email або пароль",
		})
		return
	}

	token, err := h.generateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "помилка генерації токена",
		})
		return
	}

	c.JSON(http.StatusOK, authResponse{
		Token: token,
		User:  toUserPayload(user),
	})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (h *AuthHandler) generateToken(user *models.User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:         user.ID,
		Role:           user.Role,
		OrganizationID: user.OrganizationID,
		TokenVersion:   user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(h.jwtTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}

func toUserPayload(u *models.User) userPayload {
	return userPayload{
		ID:             u.ID,
		Email:          u.Email,
		FirstName:      u.FirstName,
		LastName:       u.LastName,
		Role:           u.Role,
		OrganizationID: u.OrganizationID,
	}
}

func isDuplicateKeyError(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
