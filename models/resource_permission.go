package models

import (
	"database/sql/driver"
	"time"

	"github.com/google/uuid"
)

type PermissionType string

const (
	PermissionViewer PermissionType = "viewer"
	PermissionEditor PermissionType = "editor"
	PermissionOwner  PermissionType = "owner"
)

func (self *PermissionType) Scan(value interface{}) error {
	*self = PermissionType(value.([]byte))
	return nil
}

func (self PermissionType) Value() (driver.Value, error) {
	return string(self), nil
}

type ResourcePermission struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`

	FileID   *uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_file_user"`
	FolderID *uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_folder_user"`

	GrantedBy     uuid.UUID `gorm:"type:uuid;not null"`
	GrantedByUser Users     `gorm:"foreignKey:GrantedBy"`

	// user with the access permission
	UserID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_file_user;uniqueIndex:idx_folder_user"`
	User   Users     `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`

	Permission PermissionType `gorm:"type:permission_type;not null;default:'viewer'"`

	CreatedAt time.Time `gorm:"column:created_at;not null;default:now()"`
}
