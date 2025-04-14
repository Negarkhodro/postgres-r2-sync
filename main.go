package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"postgres-r2-sync/database"
	"postgres-r2-sync/entity"
	"postgres-r2-sync/r2"
)

func loadConfig() (*entity.BackupConfig, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	return &entity.BackupConfig{
		DBHost:       os.Getenv("DB_HOST"),
		DBPort:       os.Getenv("DB_PORT"),
		DBUser:       os.Getenv("DB_USER"),
		DBPassword:   os.Getenv("DB_PASSWORD"),
		DBName:       os.Getenv("DB_NAME"),
		R2AccountID:  os.Getenv("R2_ACCOUNT_ID"),
		R2AccessKey:  os.Getenv("R2_ACCESS_KEY"),
		R2SecretKey:  os.Getenv("R2_SECRET_KEY"),
		R2BucketName: os.Getenv("R2_BUCKET_NAME"),
		R2Region:     os.Getenv("R2_REGION"),
		BackupDir:    "/tmp/postgres_backups",
	}, nil
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Create a ServiceDB instance using the configuration
	serviceDB := &database.ServiceDB{BackupConfig: *cfg}

	// Create PostgreSQL backup
	backupPath, err := serviceDB.CreatePostgresBackup()
	if err != nil {
		log.Fatalf("Backup creation failed: %v", err)
	}

	defer serviceDB.CleanupBackup(backupPath)

	// Create an R2 client using the configuration
	serviceBackup := &r2.ServiceBackup{BackupConfig: *cfg}
	r2Client, err := serviceBackup.CreateR2Client()
	if err != nil {
		log.Fatalf("R2 client creation failed: %v", err)
	}

	// Upload the backup to R2
	if err := serviceBackup.UploadToCloudflareR2(r2Client, backupPath); err != nil {
		log.Fatalf("R2 upload failed: %v", err)
	}

	log.Println("Database backup and R2 upload completed successfully")
}
