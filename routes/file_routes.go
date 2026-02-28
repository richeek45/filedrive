package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/richeek45/filedrive/controllers"
	"github.com/richeek45/filedrive/middleware"
)

func FileRoutes(api *gin.RouterGroup, fileController *controllers.FileController) {
	fileApi := api.Group("/files")
	fileApi.Use(middleware.AuthMiddleware())
	{
		fileApi.GET("/", fileController.GetFilesFromParentFolder)
		fileApi.GET("/:fileId/download", fileController.GetDownloadURL)
		fileApi.PATCH("/:fileId/rename", fileController.RenameFile)
		fileApi.PATCH("/:fileId/trash", fileController.MoveToTrash) // Add this for soft delete
		fileApi.GET("/sync-active-uploads", fileController.SyncUserUploads)
		fileApi.POST("/share", fileController.ShareFilesToUsersByEmails)
	}

	uploadApi := fileApi.Group("/uploads")
	{
		// Multipart Upload Routes
		uploadApi.POST("/initiate", fileController.InitiateMultiPartUpload)
		uploadApi.POST("/presign-part", fileController.PresignPart)
		uploadApi.POST("/complete", fileController.CompleteMultipartUpload)
	}
}
