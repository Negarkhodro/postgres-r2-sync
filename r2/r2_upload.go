package r2

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"postgres-r2-sync/entity"

	"time"
)

type ServiceBackup struct {
	entity.BackupConfig
}

func (cfg *ServiceBackup) CreateR2Client() (*s3.Client, error) {
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

func (cfg *ServiceBackup) UploadToCloudflareR2(client *s3.Client, backupPath string) error {
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
