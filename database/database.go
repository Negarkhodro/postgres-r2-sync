package database

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"postgres-r2-sync/entity"
	"time"
)

type ServiceDB struct {
	entity.BackupConfig
}

func (cfg *ServiceDB) generateBackupFilePath() string {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	return filepath.Join(cfg.BackupDir, fmt.Sprintf("%s_%s.sql", cfg.DBName, timestamp))
}

func (cfg *ServiceDB) CreatePostgresBackup() (string, error) {
	if err := cfg.ensureBackupDir(); err != nil {
		return "", err
	}
	backupPath := cfg.generateBackupFilePath()

	if err := cfg.ExecuteBackupCommand(backupPath); err != nil {
		return "", err
	}

	if err := cfg.validateBackupFile(backupPath); err != nil {
		return "", err
	}

	log.Printf("Backup created successfully: %s", backupPath)
	return backupPath, nil
}

func (cfg *ServiceDB) ExecuteBackupCommand(backupPath string) error {
	cmd := exec.Command("bash", "-c", fmt.Sprintf(
		"PGPASSWORD=%s pg_dump -h %s -p %s -U %s -d %s -f %s --no-privileges --no-tablespaces --format=plain --encoding=UTF8 --verbose",
		cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBName, backupPath,
	))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("backup failed: %w, output: %s", err, string(output))
	}
	return nil
}

func (cfg *ServiceDB) validateBackupFile(backupPath string) error {
	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}
	if fileInfo.Size() == 0 {
		return fmt.Errorf("backup file is empty")
	}
	return nil
}

func (cfg *ServiceDB) ensureBackupDir() error {
	if err := os.MkdirAll(cfg.BackupDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}
	return nil
}

func (cfg *ServiceDB) CleanupBackup(backupPath string) {
	if err := os.Remove(backupPath); err != nil {
		log.Printf("Warning: Failed to remove local backup file: %v", err)
	}
}
