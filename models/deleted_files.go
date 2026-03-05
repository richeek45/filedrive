package models

import (
	"time"

	"github.com/google/uuid"
)

type DeletedFile struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	OriginalFileID uuid.UUID `gorm:"type:uuid;not null"`
	OwnerID        uuid.UUID `gorm:"type:uuid;not null;index"`
	Name           string    `gorm:"type:varchar(255)"`
	Size           int64     `gorm:"not null"`
	BucketName     string    `gorm:"type:varchar(255)"`
	// This will be the new S3 key (e.g., delete/old-key)
	ObjectKey string `gorm:"type:text;not null"`
	// Store the old key so we can restore it exactly where it was
	OriginalKey string     `gorm:"type:text"`
	MimeType    *string    `gorm:"type:varchar(255)"`
	FolderID    *uuid.UUID `gorm:"type:uuid"`

	DeletedAt time.Time `gorm:"not null;default:now()"`
}

type FailedS3Deletion struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	BucketName string    `gorm:"type:varchar(255)"`
	ObjectKey  string    `gorm:"type:text"`
	CreatedAt  time.Time `gorm:"default:now()"`
}
