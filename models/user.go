package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type User struct {
	// gorm.Model includes ID, CreatedAt, UpdatedAt, DeletedAt
	gorm.Model

	// uniqueIndex:name creates a composite unique index on both fields
	FirstName string `gorm:"uniqueIndex:idx_full_name;not null" json:"first_name" binding:"required"`
	LastName  string `gorm:"uniqueIndex:idx_full_name;not null" json:"last_name" binding:"required"`
	Picture   string `gorm:"type:text" json:"picture" binding:"omitempty,url"`
	Email     string `gorm:"uniqueIndex;not null" json:"email" binding:"required,email"`
	Country   string `gorm:"not null;default:'Unknown'" json:"country"`

	// Use a check constraint to ensure age is realistic
	Age int `gorm:"not null;check:age >= 0 AND age < 150" json:"age"`

	Files   []File   `gorm:"foreignKey:OwnerID"`
	Folders []Folder `gorm:"foreignKey:OwnerID"`

	Permissions        []ResourcePermission `gorm:"foreignKey:UserID"`
	GrantedPermissions []ResourcePermission `gorm:"foreignKey:GrantedBy"`

	GoogleID    string    `gorm:"uniqueIndex" json:"google_id"`
	LastLoginAt time.Time `json:"last_login_at"`
}

type TokenDetails struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}
