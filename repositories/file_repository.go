package repositories

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/richeek45/filedrive/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FileRepository struct {
	DB *gorm.DB
}

func (r *FileRepository) GetFileByID(fileId uuid.UUID, userId uuid.UUID) (models.File, error) {
	var file models.File
	query := r.DB.Where("owner_id = ? AND id = ? AND is_deleted = false", userId, fileId)
	err := query.First(&file).Error
	return file, err
}

func (r *FileRepository) SoftDeleteFile(fileID uuid.UUID, userID uuid.UUID) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		var file models.File
		if err := tx.Where("id = ? AND owner_id = ?", fileID, userID).Error; err != nil {
			return err
		}

		if err := tx.Model(&file).Updates(map[string]interface{}{
			"is_deleted": true,
			"deleted_at": time.Now(),
		}).Error; err != nil {
			return err
		}
		return tx.Model(&models.Users{}).Where("id = ?", file.OwnerID).
			UpdateColumn("storage_used", gorm.Expr("storage_used - ?", file.Size)).Error
	})
}

func (r *FileRepository) DeleteFile(fileId uuid.UUID, userId uuid.UUID) error {
	var file models.File
	err := r.DB.Where("id = ? AND owner_id = ? AND is_deleted = true", fileId, userId).Delete(file).Error
	return err
}

func NewFileRepository(db *gorm.DB) *FileRepository {
	return &FileRepository{DB: db}
}

func (r *FileRepository) GetFiles(userId uuid.UUID, folderID *uuid.UUID, isTrash bool) ([]models.File, error) {
	var files []models.File
	fmt.Println(isTrash, folderID, userId)
	query := r.DB.Unscoped().Where("owner_id = ? AND is_deleted = ?", userId, isTrash)
	// Need to remove is_deleted bool from migration

	if !isTrash {
		if folderID != nil {
			query = query.Where("folder_id = ?", *folderID)
		} else {
			query = query.Where("folder_id IS NULL")
		}
	} else {
		if folderID != nil {
			query = query.Where("folder_id = ?", *folderID)
		}
	}

	err := query.Find(&files).Error
	return files, err
}

func (r *FileRepository) UpsertFilePending(file *models.File, pendingEntry *models.PendingUpload) error {
	// We use Upsert (On Conflict) so if the user resumes an upload,
	// we just update the existing record based on ObjectKey or ID
	return r.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "object_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"updated_at", "s3_upload_id", "upload_status"}),
		}).Create(file).Error
		if err != nil {
			return err
		}

		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "s3_key"}}, // Or "upload_id" depending on your constraint
			DoUpdates: clause.AssignmentColumns([]string{"upload_id", "updated_at"}),
		}).Create(pendingEntry).Error
	})
}

func (r *FileRepository) FinalizeFile(uploadID string, partsCount int, finalETag string, status string) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		var file models.File
		if err := tx.Where("s3_upload_id = ?", uploadID).First(&file).Error; err != nil {
			return err
		}

		if err := tx.Model(&file).Updates(map[string]interface{}{
			"upload_status":         status,
			"e_tag":                 finalETag,
			"s3_upload_id":          nil,
			"uploaded_chunks":       partsCount,
			"uploaded_part_numbers": partsCount,
		}).Error; err != nil {
			return err
		}

		// Update the user storage directly here instead of a hook
		return tx.Model(&models.Users{}).Where("id = ?", file.OwnerID).
			UpdateColumn("storage_used", gorm.Expr("storage_used + ?", file.Size)).Error
	})
}
