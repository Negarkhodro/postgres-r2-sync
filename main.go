package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/joho/godotenv"
)

type BackupConfig struct {
	DBHost       string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string
	R2AccountID  string
	R2AccessKey  string
	R2SecretKey  string
	R2BucketName string
	R2Region     string
	BackupDir    string
}

func loadConfig() (*BackupConfig, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	return &BackupConfig{
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

func (cfg *BackupConfig) ensureBackupDir() error {
	if err := os.MkdirAll(cfg.BackupDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}
	return nil
}

func (cfg *BackupConfig) generateBackupFilePath() string {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	return filepath.Join(cfg.BackupDir, fmt.Sprintf("%s_%s.sql", cfg.DBName, timestamp))
}

func (cfg *BackupConfig) executeBackupCommand(backupPath string) error {
	cmd := exec.Command("bash", "-c", fmt.Sprintf(
		"PGPASSWORD=%s pg_dump -h %s -p %s -U %s -d %s -f %s --no-privileges --no-tablespaces --format=plain --encoding=UTF8 --verbose",
		cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBName, backupPath,
	))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("backup failed: %w, output: %s", err, string(output))
	}
	return nil
}

func (cfg *BackupConfig) validateBackupFile(backupPath string) error {
	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}
	if fileInfo.Size() == 0 {
		return fmt.Errorf("backup file is empty")
	}
	return nil
}

func (cfg *BackupConfig) createPostgresBackup() (string, error) {
	if err := cfg.ensureBackupDir(); err != nil {
		return "", err
	}

	backupPath := cfg.generateBackupFilePath()

	if err := cfg.executeBackupCommand(backupPath); err != nil {
		return "", err
	}

	if err := cfg.validateBackupFile(backupPath); err != nil {
		return "", err
	}

	log.Printf("Backup created successfully: %s", backupPath)
	return backupPath, nil
}

func (cfg *BackupConfig) createR2Client() (*s3.Client, error) {
	r2Endpoint := fmt.Sprintf("%s.r2.cloudflarestorage.com", cfg.R2AccountID)
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			PartitionID:   "r2",
			URL:           fmt.Sprintf("https://%s", r2Endpoint),
			SigningRegion: cfg.R2Region,
		}, nil
	})

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				MaxVersion: tls.VersionTLS13,
			},
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: true,
		},
		Timeout: 10 * time.Minute,
	}

	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.R2Region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.R2AccessKey, cfg.R2SecretKey, ""),
		),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load R2 config: %w", err)
	}

	return s3.NewFromConfig(awsCfg), nil
}

func (cfg *BackupConfig) uploadToCloudflareR2(client *s3.Client, backupPath string) error {
	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	backupFilename := filepath.Base(backupPath)
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(cfg.R2BucketName),
		Key:    aws.String(backupFilename),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("R2 upload failed: %w", err)
	}

	log.Printf("Successfully uploaded backup to R2: %s", backupFilename)
	return nil
}

func cleanupBackup(backupPath string) {
	if err := os.Remove(backupPath); err != nil {
		log.Printf("Warning: Failed to remove local backup file: %v", err)
	}
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	backupPath, err := cfg.createPostgresBackup()
	if err != nil {
		log.Fatalf("Backup creation failed: %v", err)
	}
	defer cleanupBackup(backupPath)

	r2Client, err := cfg.createR2Client()
	if err != nil {
		log.Fatalf("R2 client creation failed: %v", err)
	}

	if err := cfg.uploadToCloudflareR2(r2Client, backupPath); err != nil {
		log.Fatalf("R2 upload failed: %v", err)
	}

	log.Println("Database backup and R2 upload completed successfully")
}
