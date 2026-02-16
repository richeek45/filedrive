package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserHandler struct{}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Fetch user from database using userID
	// user, err := db.GetUserByID(userID.(string))

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"email":   c.GetString("userEmail"),
		// Add more user data
	})
}
