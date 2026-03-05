package worker

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/richeek45/filedrive/models"
	"gorm.io/gorm"
)

func PurgeExpiredDeletedFiles(db *gorm.DB, s3Client *s3.Client, bucketName string) {
	go func() {
		startTime := time.Now()

		tx := db.Begin()
		defer tx.Rollback()

		var locked bool
		tx.Raw("SELECT pg_try_advisory_xact_lock(654321)").Scan(&locked)
		if !locked {
			log.Println("Purge Worker: Already running. Skipping.")
			return
		}

		const totalLimit = 50000
		const batchSize = 1000
		processedCount := 0

		// expiryDate := time.Now().AddDate(0, 0, -30) // 30days for prod
		expiryDate := time.Now().Add(-30 * time.Second)

		log.Printf("Purge Worker: Starting cleanup for files deleted before %s", expiryDate.Format("2006-01-02"))

		for processedCount < totalLimit {
			var filesToPurge []models.File

			err := db.Unscoped().
				Limit(batchSize).
				Where("is_deleted = ? AND deleted_at < ?", true, expiryDate).
				Find(&filesToPurge).Error
			if err != nil {
				log.Fatalf("Failed to get deleted_at files: %w", err)
				break
			}

			if len(filesToPurge) == 0 {
				break
			}

			var objectKeys []types.ObjectIdentifier
			for _, f := range filesToPurge {
				key := f.ObjectKey
				objectKeys = append(objectKeys, types.ObjectIdentifier{Key: &key})
			}

			output, err := s3Client.DeleteObjects(context.TODO(), &s3.DeleteObjectsInput{
				Bucket: aws.String(bucketName),
				Delete: &types.Delete{Objects: objectKeys, Quiet: aws.Bool(false)},
			})

			if err != nil {
				log.Fatalf("S3 API error: %w", err)
				break
			}

			var successfullyDeletedKeys []string
			for _, o := range output.Deleted {
				successfullyDeletedKeys = append(successfullyDeletedKeys, *o.Key)
			}

			if len(successfullyDeletedKeys) > 0 {
				db.Unscoped().
					Where("object_key IN ?", successfullyDeletedKeys).
					Delete(&models.File{})

				processedCount += len(successfullyDeletedKeys)
			}
			log.Printf("Cleanup Progress: %d/%d", processedCount, totalLimit)
		}
		elapsed := time.Since(startTime)
		log.Printf("Cleanup complete in %d. Goroutine exiting and lock released.", elapsed)
	}()
}

func CleanupOrphanedS3Objects(db *gorm.DB, s3Client *s3.Client, bucketName string) {
	go func() {
		// We use a transaction-level advisory lock so it clears if the app crashes
		tx := db.Begin()
		defer tx.Rollback()

		var locked bool
		// '123456' is an arbitrary unique ID for this specific lock type
		err := tx.Raw("SELECT pg_try_advisory_xact_lock(123456)").Scan(&locked).Error

		if err != nil || !locked {
			log.Println("Cleanup job already running or failed to lock. Skipping.")
			return
		}

		const totalLimit = 50000
		const batchSize = 1000
		processedCount := 0

		log.Printf("Lock acquired. Starting background S3 cleanup...")

		for processedCount < totalLimit {

			var failedDeletions []models.FailedS3Deletion

			if err := db.Limit(1000).Find(&failedDeletions).Error; err != nil {
				return
			}

			if len(failedDeletions) == 0 {
				return
			}

			var objectKeys []types.ObjectIdentifier
			for _, file := range failedDeletions {
				key := file.ObjectKey
				objectKeys = append(objectKeys, types.ObjectIdentifier{Key: &key})
			}

			output, err := s3Client.DeleteObjects(context.TODO(), &s3.DeleteObjectsInput{
				Bucket: aws.String(bucketName),
				Delete: &types.Delete{Objects: objectKeys, Quiet: aws.Bool(false)},
			})

			if err != nil {
				log.Printf("S3 API Error: %v", err)
				return
			}

			var successfullyDeleted []string
			for _, o := range output.Deleted {
				successfullyDeleted = append(successfullyDeleted, *o.Key)
			}

			if len(successfullyDeleted) > 0 {
				// Use the main DB instance for the actual deletion to keep it separate from the lock tx
				db.Where("bucket_name = ? AND object_key IN ?", bucketName, successfullyDeleted).
					Delete(&models.FailedS3Deletion{})
			}

			processedCount += len(failedDeletions)
			log.Printf("Cleanup Progress: %d/%d", processedCount, totalLimit)
		}

		log.Println("Cleanup complete. Goroutine exiting and lock released.")
	}()
}
