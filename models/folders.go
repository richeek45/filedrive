package models

import (
	"time"

	"github.com/google/uuid"
)

type Folder struct {
	ID   uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name string    `gorm:"type:varchar(255);not null"`

	OwnerID uuid.UUID `gorm:"type:uuid;not null"`
	Owner   User      `gorm:"foreignKey:OwnerID;constraint:OnDelete:CASCADE;"`

	// Self reference
	ParentID *uuid.UUID `gorm:"type:uuid;index"`
	Parent   *Folder    `gorm:"foreignKey:ParentID;references:ID;constraint:OnDelete:CASCADE;"`

	Folders []Folder `gorm:"foreignKey:ParentID;references:ID"`
	Files   []File   `gorm:"foreignKey:FolderID;references:ID"`

	IsDeleted bool `gorm:"default:false"`

	CreatedAt time.Time `gorm:"not null;default:now()"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
}
