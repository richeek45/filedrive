package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richeek45/filedrive/repositories"
)

type FolderController struct {
	Repo *repositories.FolderRepostory
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

	userID := userRaw.(uuid.UUID)
	folder, err := fc.Repo.CreateFolder(userID, req.Name, req.ParentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, folder)
}

// GET /api/folder?parentId=xxx
func (fc *FolderController) FindRootFolders(c *gin.Context) {
	userRaw, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID := userRaw.(uuid.UUID)
	parentIDParam := c.Query("parentId")

	// If no parentId → root folders
	if parentIDParam == "" {
		folders, err := fc.Repo.GetRootLevelFolderFromUserID(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, folders)
	}

	// If parentId present → fetch children
	parentUUID, err := uuid.Parse(parentIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parentId"})
		return
	}

	folders, err := fc.Repo.GetFoldersByParentID(userID, parentUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, folders)
}
