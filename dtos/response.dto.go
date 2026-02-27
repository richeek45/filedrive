package dtos

import (
	"time"

	"github.com/google/uuid"
)

type FileResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	MimeType  *string   `json:"mimeType"`
	CreatedAt time.Time `json:"createdAt"`
	IsDeleted bool      `json:"isDeleted"`
}
