package repositories

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
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
	encodedKey := url.PathEscape(file.ObjectKey)
	copySource := file.BucketName + "/" + encodedKey
	trashKey := "delete/" + file.ObjectKey
	_, err := S3Client.CopyObject(context.TODO(), &s3.CopyObjectInput{
		Bucket:     aws.String(file.BucketName),
		CopySource: aws.String(copySource),
		Key:        aws.String(trashKey),
	})
	if err != nil {
		return fmt.Errorf("failed to copy to delete folder: %w", err)
	}

	_, err = S3Client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(file.BucketName),
		Key:    aws.String(file.ObjectKey),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}

	return r.DB.Transaction(func(tx *gorm.DB) error {
		deletedEntry := models.DeletedFile{
			ID:             uuid.New(),
			OriginalFileID: file.ID,
			OwnerID:        file.OwnerID,
			Name:           file.Name,
			Size:           file.Size,
			BucketName:     file.BucketName,
			ObjectKey:      trashKey,
			OriginalKey:    file.ObjectKey,
			MimeType:       file.MimeType,
			FolderID:       file.FolderID,
		}

		if err := tx.Create(&deletedEntry).Error; err != nil {
			return err
		}

		if err := tx.Unscoped().Delete(file).Error; err != nil {
			return fmt.Errorf("failed to purge from DB: %w", err)
		}
		return tx.Model(&models.Users{}).Where("id = ?", file.OwnerID).
			UpdateColumn("storage_used", gorm.Expr("storage_used - ?", file.Size)).Error
	})
}

func (r *FileRepository) RestoreFileById(fileId uuid.UUID, userId uuid.UUID, S3Client *s3.Client) error {
	var file models.File

	err := r.DB.Unscoped().Where("id = ? AND owner_id = ?", fileId, userId).First(&file).Error
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	if file.IsDeleted {
		if err := r.DB.Unscoped().Model(&file).Updates(map[string]interface{}{
			"is_deleted": false,
			"deleted_at": nil,
		}).Error; err != nil {
			return err
		}
		return nil
	}

	return nil
}

func (r *FileRepository) RestoreDeletedFiles(userId uuid.UUID, S3Client *s3.Client) error {
	var deletedFiles []models.DeletedFile
	if err := r.DB.Where("owner_id = ?", userId).Find(&deletedFiles).Error; err != nil {
		return err
	}
	if len(deletedFiles) == 0 {
		return nil
	}

	var (
		mu                sync.Mutex
		filesToRestore    []models.File
		s3KeysToDelete    []types.ObjectIdentifier
		totalRestoredSize int64
	)

	const maxWorkers = 10
	jobs := make(chan models.DeletedFile, len(deletedFiles))
	results := make(chan error, len(deletedFiles))

	for w := 1; w < maxWorkers; w++ {
		go func() {
			for f := range jobs {
				encodedKey := url.PathEscape(f.ObjectKey)
				copySource := f.BucketName + "/" + encodedKey

				_, err := S3Client.CopyObject(context.TODO(), &s3.CopyObjectInput{
					Bucket:     aws.String(f.BucketName),
					CopySource: aws.String(copySource),
					Key:        aws.String(f.OriginalKey),
				})

				if err != nil {
					results <- fmt.Errorf("S3 Copy failed for %s: %w", f.Name, err)
					continue
				}

				mu.Lock()

				s3KeysToDelete = append(s3KeysToDelete, types.ObjectIdentifier{Key: aws.String(f.ObjectKey)})
				filesToRestore = append(filesToRestore, models.File{
					ID: f.OriginalFileID, Name: f.Name, OwnerID: f.OwnerID,
					FolderID: f.FolderID, Size: f.Size, MimeType: f.MimeType,
					BucketName: f.BucketName, ObjectKey: f.OriginalKey, UploadStatus: "completed",
				})
				totalRestoredSize += f.Size
				mu.Unlock()
				results <- nil
			}
		}()
	}

	for _, df := range deletedFiles {
		jobs <- df
	}
	close(jobs)

	for i := 0; i < len(deletedFiles); i++ {
		if err := <-results; err != nil {
			return err // Stop early if any copy fails
		}
	}

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&filesToRestore).Error; err != nil {
			return err
		}
		if err := tx.Where("owner_id = ?", userId).Delete(&models.DeletedFile{}).Error; err != nil {
			return err
		}
		return tx.Model(&models.Users{}).Where("id = ?", userId).
			UpdateColumn("storage_used", gorm.Expr("storage_used + ?", totalRestoredSize)).Error
	})
	if err != nil {
		return err
	}

	_, err = S3Client.DeleteObjects(context.TODO(), &s3.DeleteObjectsInput{
		Bucket: aws.String(deletedFiles[0].BucketName),
		Delete: &types.Delete{Objects: s3KeysToDelete},
	})

	if err != nil {
		var failures []models.FailedS3Deletion
		for _, obj := range s3KeysToDelete {
			failures = append(failures, models.FailedS3Deletion{
				BucketName: deletedFiles[0].BucketName,
				ObjectKey:  *obj.Key,
			})
		}
		_ = r.DB.Create(&failures).Error
		fmt.Println("S3 batch delete failed; logged keys for later cleanup")
	}

	return nil
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
