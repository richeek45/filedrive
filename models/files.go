package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type File struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`

	Name     string     `gorm:"type:varchar(255);not null" json:"name"`
	OwnerID  uuid.UUID  `gorm:"type:uuid;not null" json:"ownerId"`
	FolderID *uuid.UUID `gorm:"type:uuid;index" json:"folderId"`
	Owner    User       `gorm:"foreignKey:OwnerID" json:"-"`

	Size     int64   `gorm:"not null;default:0" json:"size"`
	MimeType *string `gorm:"type:varchar(255)" json:"mimeType"`

	// S3 metadata
	BucketName string `gorm:"type:varchar(255);not null" json:"bucketName"`
	ObjectKey  string  `gorm:"type:text;uniqueIndex;not null" json:"objectKey"`
	S3UploadID *string `gorm:"type:text" json:"s3UploadId"`
	ETag       *string `gorm:"type:varchar(255)" json:"eTag"`

	// Upload tracking
	UploadStatus   string        `gorm:"type:varchar(20);default:'pending'" json:"uploadStatus"`
	TotalChunks    *int          `json:"totalChunks"`
	UploadedChunks int           `gorm:"default:0" json:"uploadedChunks"`
	UploadedPartNumbers int `gorm:"column:uploaded_part_numbers" json:"uploadedPartNumbers"`
	IsDeleted bool `gorm:"default:false" json:"isDeleted"`

	Folder *Folder `gorm:"foreignKey:FolderID;references:ID;constraint:OnDelete:CASCADE;" json:"-"`

	CreatedAt time.Time      `gorm:"not null;default:now()" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"not null;default:now()" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}