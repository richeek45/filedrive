package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type File struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`

	Name     string     `gorm:"type:varchar(255);not null" json:"name"`
	OwnerID  uuid.UUID  `gorm:"type:uuid;not null;index:idx_files_storage_calc" json:"ownerId"`
	FolderID *uuid.UUID `gorm:"type:uuid;index" json:"folderId"`
	Owner    Users       `gorm:"foreignKey:OwnerID" json:"-"`

	Size     int64   `gorm:"not null;default:0;index:idx_files_storage_calc" json:"size"`
	MimeType *string `gorm:"type:varchar(255)" json:"mimeType"`

	// S3 metadata
	BucketName string `gorm:"type:varchar(255);not null" json:"bucketName"`
	ObjectKey  string  `gorm:"type:text;uniqueIndex;not null" json:"objectKey"`
	S3UploadID *string `gorm:"type:text" json:"s3UploadId"`
	ETag       *string `gorm:"type:varchar(255)" json:"eTag"`

	// Upload tracking
	UploadStatus   string        `gorm:"type:varchar(20);default:'pending';index:idx_files_storage_calc" json:"uploadStatus"`
	TotalChunks    *int          `json:"totalChunks"`
	UploadedChunks int           `gorm:"default:0" json:"uploadedChunks"`
	UploadedPartNumbers int `gorm:"column:uploaded_part_numbers" json:"uploadedPartNumbers"`
	IsDeleted bool `gorm:"default:false;index:idx_files_storage_calc" json:"isDeleted"`

	Folder *Folder `gorm:"foreignKey:FolderID;references:ID;constraint:OnDelete:CASCADE;" json:"-"`

	CreatedAt time.Time      `gorm:"not null;default:now()" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"not null;default:now()" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (f *File) AfterUpdate(tx *gorm.DB) (err error) {
    // Only add to storage if the status just changed to 'completed'
    // and it wasn't already counted.
    if f.UploadStatus == "completed" {
        err = tx.Model(&Users{}).Where("id = ?", f.OwnerID).
            UpdateColumn("storage_used", gorm.Expr("storage_used + ?", f.Size)).Error
    }
    return
}

func (f *File) AfterDelete(tx *gorm.DB) (err error) {
    // Subtract size when a file is hard-deleted
    return tx.Model(&Users{}).Where("id = ?", f.OwnerID).
        UpdateColumn("storage_used", gorm.Expr("storage_used - ?", f.Size)).Error
}