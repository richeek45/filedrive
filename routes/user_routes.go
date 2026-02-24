package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/richeek45/filedrive/controllers"
	"github.com/richeek45/filedrive/middleware"
)

func RegisteredUserRoutes(api *gin.RouterGroup, userController *controllers.UserController) {

	protected := api.Group("/users")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/profile", userController.GetProfile)
		protected.POST("/", userController.CreateUser)
		//  protected.GET("/health", healthCheck)
		// protected.PUT("/me", userController.UpdateProfile) // /api/users/me
		// protected.DELETE("/me", userController.DeleteAccount)

		// // Admin-only sub-grouping
		// admin := protected.Group("/admin")
		// admin.Use(middleware.AdminOnly())
		// {
		//     admin.GET("/stats", userController.GetSystemStats)
		// }
	}
}
