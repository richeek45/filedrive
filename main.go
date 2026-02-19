package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/joho/godotenv"
	"google.golang.org/api/idtoken"
)

func main() {
	ctx := context.Background()

	// Testing to find the authorization type, need to be type = "service-account"
	// creds, _ := google.FindDefaultCredentials(ctx)
	// fmt.Println(string(creds.JSON))

	godotenv.Load()

	audience := os.Getenv("AWS_OIDC_AUDIENCE")
	roleArn := os.Getenv("AWS_ROLE_ARN")
	location := os.Getenv("LOCATION")

	fmt.Print(audience, roleArn, location)

	ts, err := idtoken.NewTokenSource(ctx, audience)
	if err != nil {
		log.Fatalf("failed to create token source: %v", err)
	}

	tok, err := ts.Token()
	if err != nil {
		log.Fatalf("failed to get ID token: %v", err)
	}

	webToken := tok.AccessToken

	fmt.Println(webToken)

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(location),
	)
	if err != nil {
		log.Fatal(err)
	}

	stsClient := sts.NewFromConfig(cfg)

	out, err := stsClient.AssumeRoleWithWebIdentity(ctx, &sts.AssumeRoleWithWebIdentityInput{
		RoleArn:          aws.String(roleArn),
		RoleSessionName:  aws.String("gcp-session"),
		WebIdentityToken: aws.String(webToken),
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Assumed role successfully:", *out.AssumedRoleUser.Arn)
}

// Previously used method
// func main() {
// 	if err := cmd.Execute(); err != nil {
// 		os.Exit(1)
// 	}
// 	ctx := context.Background()

// 	cfg, err := config.LoadDefaultConfig(ctx,
// 		config.WithSharedConfigProfile("rolesanywhere"),
// 	)
// 	if err != nil {
// 		log.Fatalf("failed to load AWS config: %v", err)
// 	}

// 	s3Client := s3.NewFromConfig(cfg)

// 	// Now you can use S3 securely
// 	result, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	fmt.Println(result)

// 	// Initialize database
// 	// db, err := initDB()
// 	// if err != nil {
// 	//     log.Fatal("Failed to connect to database:", err)
// 	// }
// 	// defer db.Close()

// 	// // Initialize S3 client
// 	// s3Client := initS3()

// 	// bucket := "filedrive-bucket"
// 	// key := "test.txt"
// 	// content := "Hello from Roles Anywhere!"

// 	// _, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
// 	// 	Bucket: &bucket,
// 	// 	Key:    &key,
// 	// 	Body:   strings.NewReader(content),
// 	// })
// 	// if err != nil {
// 	// 	log.Fatalf("failed to upload: %v", err)
// 	// }

// 	fmt.Println("File uploaded successfully!")

// 	// Configure CORS based on environment
// 	env := os.Getenv("GO_ENV")
// 	if env == "" {
// 		env = "development"
// 	}

// 	var allowedOrigins []string
// 	if env == "production" {
// 		allowedOrigins = []string{
// 			"https://your-app.pages.dev",
// 			"https://your-custom-domain.com",
// 		}
// 	} else {
// 		// Development - allow localhost
// 		allowedOrigins = []string{os.Getenv("FRONTEND_URL")}
// 	}

// 	router := gin.Default()

// 	router.Use(cors.New(cors.Config{
// 		AllowOrigins:     allowedOrigins,
// 		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
// 		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
// 		ExposeHeaders:    []string{"Content-Length"},
// 		AllowCredentials: true,
// 	}))

// 	authHandler := handlers.NewAuthHandler()
// 	userHandler := handlers.NewUserHandler()

// 	api := router.Group("/api")

// 	{
// 		// Auth routes
// 		auth := api.Group("/auth")
// 		{
// 			auth.GET("/google/login", authHandler.GoogleLogin)
// 			auth.GET("/google/callback", authHandler.GoogleCallback)
// 			auth.POST("/refresh", authHandler.RefreshToken)
// 			auth.POST("/logout", authHandler.Logout)
// 		}
// 	}

// 	protected := api.Group("/")
// 	protected.Use(middleware.AuthMiddleware())
// 	{
// 		protected.GET("/profile", userHandler.GetProfile)
// 		//  protected.GET("/health", healthCheck)
// 		// protected.POST("/upload", uploadFile(s3Client))
// 		// protected.GET("/files", getFiles(db))
// 	}

// 	port := os.Getenv("PORT")
// 	if port == "" {
// 		port = "8080"
// 	}

// 	log.Printf("Server starting in %s mode on port %s", env, port)
// 	router.Run(":" + port)
// }
