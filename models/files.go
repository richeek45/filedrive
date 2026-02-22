package models

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type File struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`

	Name     string     `gorm:"type:varchar(255);not null"`
	OwnerID  uuid.UUID  `gorm:"type:uuid;not null"`
	FolderID *uuid.UUID `gorm:"type:uuid;index"`
	Owner    User       `gorm:"foreignKey:OwnerID"`

	Size     int64   `gorm:"not null;default:0"`
	MimeType *string `gorm:"type:varchar(255)"`

	// S3 metadata
	BucketName string  `gorm:"type:varchar(255);not null"`
	ObjectKey  string  `gorm:"type:text;not null"`
	S3UploadID *string `gorm:"type:text"`
	ETag       *string `gorm:"type:varchar(255)"`

	// Upload tracking
	UploadStatus   string `gorm:"type:varchar(20);default:'pending'"`
	TotalChunks    *int
	UploadedChunks int `gorm:"default:0"`

	// PostgreSQL int[] type
	UploadedPartNumbers pq.Int64Array `gorm:"type:int[]"`

	IsDeleted bool `gorm:"default:false"`

	// Relations
	Folder *Folder `gorm:"foreignKey:FolderID;references:ID;constraint:OnDelete:CASCADE;"`

	gorm.Model
}
