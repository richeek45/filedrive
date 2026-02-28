package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/richeek45/filedrive/controllers"
	"github.com/richeek45/filedrive/db"
	"github.com/richeek45/filedrive/repositories"
	"github.com/richeek45/filedrive/routes"
	"github.com/richeek45/filedrive/storage"
	"github.com/richeek45/filedrive/worker"
	"github.com/robfig/cron/v3"
)

func loadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}
	requiredVars := []string{
		"DB_HOST",
		"DB_PORT",
		"DB_NAME",
		"DB_USER",
		"DB_PASSWORD",
		"SSL_MODE",
		"JWT_SECRET",
	}
	for _, v := range requiredVars {
		if os.Getenv(v) == "" {
			log.Fatalf("Required environment variable %s is not set", v)
		}
	}
}

func main() {
	loadEnv()
	db := db.InitDB()

	s3Client := storage.InitS3()
	bucketName := os.Getenv("S3_BUCKET")

	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "development"
	}

	cronJob := cron.New(cron.WithSeconds())
	cronJob.AddFunc("0 0 0 * * *", func() {
		log.Println("--- Starting Storage Sync Job ---")

		err := worker.SyncUserStorage(db)

		if err != nil {
			log.Printf("Storage Sync failed %v", err)
		}
	})

	cronJob.Start()

	controllers.StartCacheCleaner()

	var allowedOrigins []string
	if env == "production" {
		allowedOrigins = []string{
			"https://your-app.pages.dev",
			"https://your-custom-domain.com",
		}
	} else {
		allowedOrigins = []string{os.Getenv("FRONTEND_URL")}
	}

	router := gin.Default()

	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	router.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	api := router.Group("/api")

	userRepo := repositories.NewUserRepository(db)
	userController := &controllers.UserController{Repo: userRepo}
	routes.RegisteredUserRoutes(api, userController)

	folderRepo := repositories.NewFolderRepository(db)
	folderController := &controllers.FolderController{
		Repo:     folderRepo,
		S3Client: s3Client,
		Bucket:   bucketName,
	}
	routes.FolderRoutes(api, folderController)

	fileRepo := repositories.NewFileRepository(db)
	fileController := &controllers.FileController{
		Repo:       fileRepo,
		FolderRepo: folderRepo,
		UserRepo:   userRepo,
		S3Client:   s3Client,
		Bucket:     bucketName,
	}
	routes.FileRoutes(api, fileController)

	authController := controllers.NewAuthController(userRepo)
	routes.AuthRoutes(api, authController)

	for _, route := range router.Routes() {
		fmt.Printf("Method: %s | Path: %s\n", route.Method, route.Path)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting in %s mode on port %s", env, port)
	router.Run(":" + port)
}
