package controllers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richeek45/filedrive/dtos"
	"github.com/richeek45/filedrive/models"
	"github.com/richeek45/filedrive/repositories"
)

type FolderController struct {
	Repo     *repositories.FolderRepository
	S3Client *s3.Client
	Bucket   string
}

func formatFolders(folders []models.Folder) []dtos.FolderResponse {
	var response []dtos.FolderResponse

	for _, f := range folders {
		var parentID uuid.UUID
		if f.ParentID != nil {
			parentID = *f.ParentID
		}

		response = append(response, dtos.FolderResponse{
			ID:        f.ID,
			Name:      f.Name,
			ParentID:  parentID,
			CreatedAt: f.CreatedAt,
			IsDeleted: f.IsDeleted,
		})
	}

	return response
}

func (fc *FolderController) CreateFolder(c *gin.Context) {

	var req struct {
		Name     string     `json:"name" binding:"required"`
		ParentID *uuid.UUID `json:"parentId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userRaw, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userRaw.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot parse UUID"})
		return
	}
	folder, err := fc.Repo.CreateFolder(userID, req.Name, req.ParentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, folder)
}

func (fc *FolderController) FindRootFolders(c *gin.Context) {
	userRaw, exists := c.Get("userID")

	isTrash, err := strconv.ParseBool(c.Query("isTrash"))
	if err != nil {
		fmt.Printf("Error converting string \"%s\": %v\n", c.Query("isTrash"), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userRaw.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot parse UUID"})
		return
	}
	parentIDParam := c.Query("parentId")

	if parentIDParam == "" {
		folders, err := fc.Repo.GetRootLevelFolderFromUserID(userID, isTrash)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		response := formatFolders(folders)

		c.JSON(http.StatusOK, response)
		return
	}

	// If parentId present → fetch children
	parentUUID, err := uuid.Parse(parentIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parentId"})
		return
	}

	folders, err := fc.Repo.GetFoldersByParentID(userID, parentUUID, isTrash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response := formatFolders(folders)

	c.JSON(http.StatusOK, response)
}

func (fc *FolderController) RenameFolder(c *gin.Context) {
	var req struct {
		NewName string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	folderID := uuid.MustParse(c.Param("folderId"))
	userID := uuid.MustParse(c.GetString("userID"))

	err := fc.Repo.DB.Model(&models.Folder{}).
		Where("id = ? AND owner_id = ?", folderID, userID).
		Update("name", req.NewName).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Renamed successfully"})
}
