package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/richeek45/filedrive/controllers"
	"github.com/richeek45/filedrive/db"
	"github.com/richeek45/filedrive/repositories"
	"github.com/richeek45/filedrive/routes"
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

	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "development"
	}

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

	authController := controllers.NewAuthController(userRepo)
	routes.AuthRoutes(api, authController)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting in %s mode on port %s", env, port)
	router.Run(":" + port)
}
