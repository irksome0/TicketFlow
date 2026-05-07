package repository

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"ticketflow-api/internal/models"
)

var ErrInviteAlreadyUsed = errors.New("invite already used")

type InviteRepository interface {
	Create(invite *models.OrganizationInvite) error
	FindByTokenHash(tokenHash string) (*models.OrganizationInvite, error)
	Accept(inviteID uuid.UUID, user *models.User, usedAt time.Time) error
}

type inviteRepository struct {
	db *gorm.DB
}

func NewInviteRepository(db *gorm.DB) InviteRepository {
	return &inviteRepository{db: db}
}

func (r *inviteRepository) Create(invite *models.OrganizationInvite) error {
	return r.db.Create(invite).Error
}

func (r *inviteRepository) FindByTokenHash(tokenHash string) (*models.OrganizationInvite, error) {
	var invite models.OrganizationInvite
	err := r.db.Where("token_hash = ?", tokenHash).First(&invite).Error
	return &invite, err
}

func (r *inviteRepository) Accept(inviteID uuid.UUID, user *models.User, usedAt time.Time) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.OrganizationInvite{}).
			Where("id = ? AND used_at IS NULL", inviteID).
			Update("used_at", usedAt)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrInviteAlreadyUsed
		}

		return tx.Create(user).Error
	})
}
