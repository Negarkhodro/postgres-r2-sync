
# PostgreSQL Backup and Cloudflare R2 Upload Tool

## Overview

This Go-based application automates PostgreSQL database backups and securely uploads them to Cloudflare R2 storage. It provides a simple, efficient solution for database backup and cloud storage.

## Features

- Automatic PostgreSQL database backup using `pg_dump`
- Secure upload to Cloudflare R2
- Easy configuration via `.env` file
- Automatic local backup file cleanup after upload
- Comprehensive error handling

## Prerequisites

Before you begin, ensure you have:

- Go 1.21 or later
- PostgreSQL installed (local or remote)
- `pg_dump` utility available in your system PATH
- Cloudflare R2 account credentials
- Basic understanding of Go and database management

## Installation

1. Clone the repository:
   ```bash
    sudo sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list'
    wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo apt-key add -
    sudo apt-get update
    sudo apt-get install postgresql-client-16
    pg_dump --version
    git clone https://github.com/raminfp/postgres-r2-sync.git
    cd postgres-backup
```

   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Create a `.env` file in the project root directory

## Configuration

Create a `.env` file with the following configuration:

```env
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_database_password
DB_NAME=your_database_name

# Cloudflare R2 Configuration
R2_ACCOUNT_ID=your_cloudflare_account_id
R2_ACCESS_KEY=your_r2_access_key
R2_SECRET_KEY=your_r2_secret_key
R2_BUCKET_NAME=your_backup_bucket
R2_REGION=auto
```

## Usage

Build the application:
```bash
go build -o postgres-backup
```

Run the backup tool:
```bash
./postgres-backup
```

## Backup Process

The tool performs these steps:
1. Connects to the specified PostgreSQL database
2. Creates a timestamped SQL dump using `pg_dump`
3. Uploads the backup to the specified Cloudflare R2 bucket
4. Removes the local backup file after successful upload

## Logging

- Backup and upload progress is logged to the console
- Errors are logged with detailed information
- Log messages help diagnose any issues during the backup process

## Security Considerations

- Keep your `.env` file private and out of version control
- Use strong, unique credentials for database and R2 access
- Implement appropriate file system permissions
- Rotate credentials periodically

## Troubleshooting

- Ensure `pg_dump` is installed and accessible
- Verify database connection details
- Check R2 bucket permissions and credentials
- Review console logs for specific error messages

## Project Structure

```
.
├── database
      └── database.go
├── entity
      └── r2.go
└── r2
    └── r2_upload.go
├── main.go
├── go.mod
├── go.sum
└── .env
```
