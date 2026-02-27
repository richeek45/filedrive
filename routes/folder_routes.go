package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/richeek45/filedrive/controllers"
	"github.com/richeek45/filedrive/middleware"
)

func FolderRoutes(api *gin.RouterGroup, folderController *controllers.FolderController) {
	folderApi := api.Group("/folders")
	folderApi.Use(middleware.AuthMiddleware())

	{
		folderApi.GET("/", folderController.FindRootFolders)
		folderApi.POST("/", folderController.CreateFolder)
		folderApi.PATCH("/:folderId/rename", folderController.RenameFolder)
	}
}
