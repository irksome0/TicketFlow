package repository

import (
	"ticketflow-api/internal/models"

	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *models.User) error
	FindByEmail(email string) (*models.User, error)
	FindByID(id string) (*models.User, error)
	ListByOrganization(orgID string) ([]models.User, error)
	UpdateRole(id string, role models.Role) error
}

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository створює новий екземпляр userRepository.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *userRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error
	return &user, err
}

func (r *userRepository) FindByID(id string) (*models.User, error) {
	var user models.User
	err := r.db.Where("id = ?", id).First(&user).Error
	return &user, err
}

func (r *userRepository) ListByOrganization(
	orgID string,
) ([]models.User, error) {
	var users []models.User
	err := r.db.Where("organization_id = ?", orgID).Find(&users).Error
	return users, err
}

func (r *userRepository) UpdateRole(id string, role models.Role) error {
	return r.db.Model(&models.User{}).
		Where("id = ?", id).
		Update("role", role).Error
}
