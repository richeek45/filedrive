package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/richeek45/filedrive/controllers"
	"github.com/richeek45/filedrive/db"
	"github.com/richeek45/filedrive/middleware"
	"github.com/richeek45/filedrive/repositories"
	"github.com/richeek45/filedrive/routes"
	"github.com/richeek45/filedrive/storage"
	"github.com/richeek45/filedrive/worker"
	"github.com/robfig/cron/v3"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
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

func initTracer() (*sdktrace.TracerProvider, error) {
	// Jaeger now supports OTLP natively on port 4317
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "127.0.0.1:4317" // Local fallback
	}
	exporter, err := otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(endpoint),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("filedrive-backend"),
		)),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}

type IPlimiter struct {
	ips map[string]*rate.Limiter
	mu  sync.Mutex
	r   rate.Limit
	b   int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPlimiter {
	return &IPlimiter{
		ips: make(map[string]*rate.Limiter),
		r:   r,
		b:   b,
	}
}

func (i *IPlimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.ips[ip]
	if !exists {
		limiter = rate.NewLimiter(i.r, i.b)
		i.ips[ip] = limiter
	}
	return limiter
}

func RateLimitMiddleware(limiter *IPlimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the client's IP address
		ip := c.ClientIP()
		l := limiter.GetLimiter(ip)

		if !l.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Slow down!",
			})
			return
		}
		c.Next()
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

	cronJob.AddFunc("0 0 */6 * * *", func() {
		log.Println("--- Starting S3 Orphaned Cleanup Job ---")
		worker.CleanupOrphanedS3Objects(db, s3Client, bucketName)
	})

	// 0 * * * * * -> every minute for testing
	cronJob.AddFunc("0 0 2 * * *", func() {
		log.Println("--- Starting Daily Permanent Purge ---")
		worker.PurgeExpiredDeletedFiles(db, s3Client, bucketName)
	})

	cronJob.Start()

	controllers.StartCacheCleaner()

	var allowedOrigins []string
	if env == "production" {
		allowedOrigins = []string{
			"https://filedrive-ctx.pages.dev",
			"https://34.101.234.31.sslip.io",
		}
	} else {
		allowedOrigins = []string{os.Getenv("FRONTEND_URL")}
	}

	router := gin.Default()

	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false
	router.SetTrustedProxies(nil)

	tp, _ := initTracer()
	defer tp.Shutdown(context.Background())

	router.Use(otelgin.Middleware("filedrive-backend"))

	reg := prometheus.NewRegistry()
	promHandler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	router.GET("/metrics", func(c *gin.Context) {
		promHandler.ServeHTTP(c.Writer, c.Request)
	})

	router.Use(middleware.PrometheusMiddleware(reg))
	router.Use(otelgin.Middleware("filedrive-backend"))

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

	limiter := NewIPRateLimiter(5, 10)

	api := router.Group("/api")
	api.Use(RateLimitMiddleware(limiter))

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
