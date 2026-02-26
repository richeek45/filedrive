package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richeek45/filedrive/models"
	"github.com/richeek45/filedrive/repositories"
)

type UserController struct {
	Repo *repositories.UserRepository
}

func (r *UserController) FindUsers(c *gin.Context) {
	users, err := r.Repo.GetAll()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch users"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

func (r *UserController) CreateUser(c *gin.Context) {
	var user models.Users
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.Repo.Create(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create user"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": user})
}

func (r *UserController) GetByID(id uuid.UUID) (*models.Users, error) {
	var user models.Users
	err := r.Repo.DB.First(&user, "id = ?", id).Error
	return &user, err
}

func (r *UserController) GetProfile(c *gin.Context) {
	userRaw, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	userStr := userRaw.(string)
	parsedID, err := uuid.Parse(userStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format in token. Please re-login."})
		return
	}

	user, err := r.GetByID(parsedID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found in database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"email":   user.Email,
		"name":    user.FirstName,
		"picture": user.Picture,
		"storageUsed": user.StorageUsed,
		"storageLimit": user.StorageLimit,
	})
}
