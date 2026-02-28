package repositories

import (
	"github.com/google/uuid"
	"github.com/richeek45/filedrive/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserRepository struct {
	DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) GetAll() ([]models.Users, error) {
	var users []models.Users
	err := r.DB.Find(&users).Error
	return users, err
}

func (r *UserRepository) Create(user *models.Users) error {
	return r.DB.Create(user).Error
}

func (r *UserRepository) GetByID(id uuid.UUID) (*models.Users, error) {
	var user models.Users
	err := r.DB.First(&user, "id = ?", id).Error
	return &user, err
}

func (r *UserRepository) UpsertByGoogleID(user *models.Users) error {
	return r.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "google_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"last_login_at", "picture", "first_name", "last_name"}),
	}).Create(user).Error
}
