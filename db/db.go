package db

import (
	"fmt"
	"os"

	"github.com/richeek45/filedrive/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func InitDB() *gorm.DB {
	dbHost := os.Getenv("DB_HOST")
	dbName := os.Getenv("DB_NAME")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbPort := os.Getenv("DB_PORT")
	sslMode := os.Getenv("SSL_MODE")
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s", dbHost, dbUser, dbPassword, dbName, dbPort, sslMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})

	if err != nil {
		panic("failed to connect to database: " + err.Error())
	}

	db.Exec(`
		DO $$ BEGIN
			CREATE TYPE permission_type AS ENUM ('viewer', 'editor', 'owner');
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$;
	`)

	err = db.AutoMigrate(&models.User{}, &models.File{}, &models.Folder{}, &models.ResourcePermission{})
	if err != nil {
		panic("failed to migrate database: " + err.Error())
	}

	fmt.Println("Connected to database!", dsn)
	return db
}
