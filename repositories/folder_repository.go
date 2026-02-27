package repositories

import (
	"github.com/google/uuid"
	"github.com/richeek45/filedrive/models"
	"gorm.io/gorm"
)

type FolderRepository struct {
	DB *gorm.DB
}

func NewFolderRepository(db *gorm.DB) *FolderRepository {
	return &FolderRepository{DB: db}
}

func (r *FolderRepository) EnsureFolderExists(userID uuid.UUID, parentID *uuid.UUID, name string) (uuid.UUID, error) {
    var folder models.Folder
    
    err := r.DB.Where(models.Folder{
        OwnerID:  userID,
        ParentID: parentID,
        Name:     name,
    }).FirstOrCreate(&folder).Error

    return folder.ID, err
}

func (r *FolderRepository) CreateFolder(userID uuid.UUID, folderName string, parentID *uuid.UUID) (*models.Folder, error) {

	folder := models.Folder{
		Name:     folderName,
		ParentID: parentID,
		OwnerID:  userID,
	}

	if err := r.DB.Create(&folder).Error; err != nil {
		return nil, err
	}

	return &folder, nil
}

func (r *FolderRepository) GetRootLevelFolderFromUserID(userID uuid.UUID) ([]models.Folder, error) {

	var folders []models.Folder

	err := r.DB.
		Where("owner_id = ? AND parent_id IS NULL AND is_deleted = false", userID).
		Find(&folders).Error

	return folders, err
}

func (r *FolderRepository) GetFoldersByParentID(userID uuid.UUID, parentID uuid.UUID) ([]models.Folder, error) {

	var folders []models.Folder

	err := r.DB.
		Where("owner_id = ? AND parent_id = ? AND is_deleted = false", userID, parentID).
		Find(&folders).Error

	return folders, err
}
