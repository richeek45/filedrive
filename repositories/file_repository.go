package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/richeek45/filedrive/dtos"
	"github.com/richeek45/filedrive/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FileRepository struct {
	DB *gorm.DB
}

func NewFileRepository(db *gorm.DB) *FileRepository {
	return &FileRepository{DB: db}
}

/*
SELECT f.*
FROM files f
LEFT JOIN resource_permissions rp
    ON f.id = rp.file_id AND rp.user_id = 'user-id-here'
WHERE f.id = 'file-id-here'
    AND f.is_deleted = false
    AND (f.owner_id = 'user-id-here' OR rp.user_id = 'user-id-here')
GROUP BY f.id;

*/

func (r *FileRepository) GetFileByID(fileId uuid.UUID, userId uuid.UUID) (models.File, error) {
	var file models.File

	query := r.DB.Table("file").Select("file.*").
		Joins("LEFT JOIN users ON users.id = ?", userId).
		Joins("LEFT JOIN resource_permission ON resource_permission.user_id = users.id AND resource_permission.file_id = file.id").
		Where("file.id = ? AND file.is_deleted = ?", fileId, false).
		Where("file.owner_id = ? OR resource_permission.user_id = ?", userId, userId).
		Group("file.id")

	err := query.First(&file).Error
	if err != nil {
		return models.File{}, err
	}

	return file, nil
}

func (r *FileRepository) SharedFilesByUserID(userID uuid.UUID) ([]dtos.SharedFileResponse, error) {
	var files []dtos.SharedFileResponse

	err := r.DB.Table("file").
		Select("file.*, resource_permission.permission, users.first_name as shared_by").
		Joins("JOIN resource_permission ON resource_permission.file_id = file.id").
		Joins("JOIN users ON resource_permission.granted_by = users.id").
		Where("resource_permission.user_id = ? ", userID).
		Scan(&files).Error

	if err != nil {
		return nil, err
	}

	return files, nil
}

func (r *FileRepository) DeleteFile(fileID uuid.UUID, userID uuid.UUID, S3Client *s3.Client) error {
	var file models.File

	err := r.DB.Unscoped().Where("id = ? AND owner_id = ?", fileID, userID).First(&file).Error
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	if file.IsDeleted {
		return r.PermanentDeleteFile(&file, S3Client)
	}

	return r.SoftDeleteFile(&file)
}

func (r *FileRepository) SoftDeleteFile(file *models.File) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(file).Updates(map[string]interface{}{
			"is_deleted": true,
			"deleted_at": time.Now(),
		}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *FileRepository) PermanentDeleteFile(file *models.File, S3Client *s3.Client) error {
	_, err := S3Client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(file.BucketName),
		Key:    aws.String(file.ObjectKey),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}

	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Delete(file).Error; err != nil {
			return fmt.Errorf("failed to purge from DB: %w", err)
		}
		return tx.Model(&models.Users{}).Where("id = ?", file.OwnerID).
			UpdateColumn("storage_used", gorm.Expr("storage_used - ?", file.Size)).Error
	})
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
