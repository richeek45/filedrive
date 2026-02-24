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
	}

	uploadApi := api.Group("/upload")
	{
		// Multipart Upload Routes
		uploadApi.POST("/initiate", fileController.InitiateMultiPartUpload)
		uploadApi.POST("/presign-part", fileController.PresignPart)
		uploadApi.POST("/complete", fileController.CompleteMultipartUpload)
	}
}