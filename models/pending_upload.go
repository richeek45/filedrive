package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PendingUpload struct {
    ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
    UserID    uuid.UUID `gorm:"type:uuid;index"`
    UploadID  string    `gorm:"not null"`
    S3Key     string     `gorm:"not null;uniqueIndex"`
    FileName  string    `json:"fileName"`
    ParentID  *uuid.UUID `gorm:"type:uuid"`
    TotalParts int      `json:"totalParts"`
    gorm.Model
}