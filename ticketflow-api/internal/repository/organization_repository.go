package repository

import (
	"ticketflow-api/internal/models"

	"gorm.io/gorm"
)

type OrganizationRepository interface {
	CreateWithAdmin(organization *models.Organization, admin *models.User) error
}

type organizationRepository struct {
	db *gorm.DB
}

func NewOrganizationRepository(db *gorm.DB) OrganizationRepository {
	return &organizationRepository{db: db}
}

func (r *organizationRepository) CreateWithAdmin(
	organization *models.Organization,
	admin *models.User,
) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(organization).Error; err != nil {
			return err
		}

		admin.OrganizationID = organization.ID
		if err := tx.Create(admin).Error; err != nil {
			return err
		}

		return nil
	})
}
