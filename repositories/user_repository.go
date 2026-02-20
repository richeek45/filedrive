package repositories

import (
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

func (r *UserRepository) GetAll() ([]models.User, error) {
	var users []models.User
	err := r.DB.Find(&users).Error
	return users, err
}

func (r *UserRepository) Create(user *models.User) error {
	return r.DB.Create(user).Error
}

func (r *UserRepository) UpsertByGoogleID(user *models.User) error {
	// Clauses("OnConflict") handles the "Update if exists" logic in Postgres
	return r.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "google_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"last_login_at", "picture", "first_name", "last_name"}),
	}).Create(user).Error
}
