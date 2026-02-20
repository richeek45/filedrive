package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/richeek45/filedrive/controllers"
)

func AuthRoutes(api *gin.RouterGroup, authController *controllers.AuthController) {

	auth := api.Group("/auth")
	{
		auth.GET("/google/login", authController.GoogleLogin)
		auth.GET("/google/callback", authController.GoogleCallback)
		auth.POST("/refresh", authController.RefreshToken)
		auth.POST("/logout", authController.Logout)
	}
}
