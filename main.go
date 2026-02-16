package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/joho/godotenv"

	"github.com/richeek45/filedrive/cmd"
	"github.com/richeek45/filedrive/handlers"
	"github.com/richeek45/filedrive/middleware"
)

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}

	if err := godotenv.Load(".env"); err != nil {
		log.Println("No .env file found")
	}

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{os.Getenv("FRONTEND_URL")},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	authHandler := handlers.NewAuthHandler()
	userHandler := handlers.NewUserHandler()

	api := router.Group("/api")

	{
		// Auth routes
		auth := api.Group("/auth")
		{
			auth.GET("/google/login", authHandler.GoogleLogin)
			auth.GET("/google/callback", authHandler.GoogleCallback)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/logout", authHandler.Logout)
		}
	}

	protected := api.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/profile", userHandler.GetProfile)
	}

	log.Fatal(router.Run(":8000"))
}
