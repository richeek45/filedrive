package worker

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/richeek45/filedrive/models"
	"gorm.io/gorm"
)

func SyncUserStorage(db *gorm.DB) error {
	log.Println("Starting storage synchronization...")

	const batchSize = 100
	var totalUsers int64

	db.Model(&models.Users{}).Count(&totalUsers)

	for i := 0; i < int(totalUsers); i += batchSize {
		var userIDs []uuid.UUID

		err := db.Model(&models.Users{}).
			Limit(batchSize).
			Offset(i).
			Pluck("id", &userIDs).
			Error

		if err != nil {
			log.Printf("Error fetching user batch: %v", err)
			continue
		}

		err = db.Exec(`
			UPDATE users u
			SET storage_used = (
				SELECT COALESCE(SUM(f.size), 0)
				FROM file f
				WHERE f.owner_id = u.id
				AND f.upload_status = 'completed'
			)
			WHERE u.id IN ?`, userIDs).Error

		if err != nil {
			log.Printf("Transaction failed for batch at offset %d: %v", i, err)
		} else {
			log.Printf("Processed %d/%d users...", i+len(userIDs), totalUsers)
		}

		time.Sleep(50 * time.Millisecond)
	}

	return nil
}
