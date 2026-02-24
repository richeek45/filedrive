package repositories

import (
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
    return r.DB.Model(&models.File{}).
        Where("id = ? AND owner_id = ?", fileID, userID).
        Updates(map[string]interface{}{
            "is_deleted": true,
            "deleted_at": time.Now(),
        }).Error
}

func (r *FileRepository) DeleteFile(fileId uuid.UUID, userId uuid.UUID) error {
    var file models.File
    err := r.DB.Where("id = ? AND owner_id = ? AND is_deleted = true", fileId, userId).Delete(file).Error
    return err
}

func NewFileRepository(db *gorm.DB) *FileRepository {
	return &FileRepository{DB: db}
}

func (r *FileRepository) GetFiles(userId uuid.UUID, folderID *uuid.UUID) ([]models.File, error) {
	var files []models.File
	query := r.DB.Where("owner_id = ? AND is_deleted = false", userId)

	if folderID != nil {
		query = query.Where("folder_id = ?", *folderID)
	} else {
		query = query.Where("folder_id IS NULL") // Assuming root files have NULL parent
	}

	err := query.Find(&files).Error
	return files, err
}

func (r *FileRepository) UpsertFilePending(file *models.File) error {
	// We use Upsert (On Conflict) so if the user resumes an upload,
	// we just update the existing record based on ObjectKey or ID
	return r.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "object_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"updated_at", "s3_upload_id"}),
	}).Create(file).Error
}

func (r *FileRepository) FinalizeFile(uploadID string, partsCount int, finalETag string, status string) error {
	return r.DB.Model(&models.File{}).
		Where("s3_upload_id = ?", uploadID).
		Updates(map[string]interface{}{
			"upload_status":         status,
			"s3_upload_id":          nil, // Clear the upload ID once done
			"uploaded_chunks":       partsCount,
			"uploaded_part_numbers": partsCount,
			"e_tag":                 finalETag,
		}).Error
}
