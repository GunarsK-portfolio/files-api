# Files API

![CI](https://github.com/GunarsK-portfolio/files-api/workflows/CI/badge.svg)

File upload/download service for portfolio platform.

## Features

- File upload with JWT authentication (validated via auth-service)
- Public file download/streaming
- File deletion (storage + database)
- Semantic file types (portfolio-image, miniature-image, document)
- Database tracking for file metadata
- RESTful API with Swagger documentation
- Health check endpoint

## Tech Stack

- **Language**: Go 1.25
- **Framework**: Gin
- **Database**: PostgreSQL (GORM)
- **Storage**: MinIO (S3-compatible)
- **Documentation**: Swagger/OpenAPI

## Prerequisites

- Go 1.25+
- PostgreSQL (or use Docker Compose)
- MinIO (or use Docker Compose)

## Project Structure

```
files-api/
├── cmd/
│   └── api/              # Application entrypoint
├── internal/
│   ├── config/           # Configuration
│   ├── database/         # Database connection
│   ├── handlers/         # HTTP handlers
│   ├── middleware/       # Authentication (validates with auth-service)
│   ├── repository/       # Data access layer
│   ├── routes/           # Route definitions
│   └── storage/          # MinIO/S3 integration
└── docs/                 # Swagger documentation
```

## Quick Start

### Using Docker Compose

```bash
docker-compose up -d
```

### Local Development

1. Copy environment file:
```bash
cp .env.example .env
```

2. Update `.env` with your configuration:
```env
PORT=8085
DB_HOST=localhost
DB_PORT=5432
DB_USER=portfolio_admin
DB_PASSWORD=portfolio_admin_dev_pass
DB_NAME=portfolio
S3_ENDPOINT=http://localhost:9000
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
S3_USE_SSL=false
AUTH_SERVICE_URL=http://localhost:8084/api/v1
```

3. Start infrastructure (if not running):
```bash
# From infrastructure directory
docker-compose up -d postgres minio flyway auth-service
```

4. Run the service:
```bash
go run cmd/api/main.go
```

## Available Commands

Using Task:
```bash
task run            # Run the service
task build          # Build binary
task fmt            # Format code
task test           # Run tests
task test-coverage  # Run tests with coverage report
task lint           # Run golangci-lint
task vuln           # Check for vulnerabilities
task ci             # Run all CI checks locally
task swagger        # Generate Swagger docs
task clean          # Clean build artifacts
task docker-build   # Build Docker image
task install-tools  # Install dev tools (golangci-lint, govulncheck, etc.)
```

Using Go directly:
```bash
go run cmd/api/main.go       # Run
go build -o bin/files-api cmd/api/main.go  # Build
go test ./...                # Test
```

## API Endpoints

Base URL: `http://localhost:8085/api/v1`

### Health Check
- `GET /health` - Service health status

### Public Endpoints
- `GET /files/{fileType}/{key}` - Download file

### Protected Endpoints (JWT Required)
- `POST /files` - Upload file (multipart: file, fileType)
- `DELETE /files/{id}` - Delete file by ID

**File Types:**
- `portfolio-image` - Professional portfolio project images
- `miniature-image` - Miniature painting photos
- `document` - PDFs, CVs, resumes

## Swagger Documentation

When running, Swagger UI is available at:
- `http://localhost:8085/swagger/index.html`

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8085` |
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_USER` | Database user | `portfolio_admin` |
| `DB_PASSWORD` | Database password | - |
| `DB_NAME` | Database name | `portfolio` |
| `S3_ENDPOINT` | MinIO/S3 endpoint | `http://localhost:9000` |
| `S3_ACCESS_KEY` | MinIO access key | `minioadmin` |
| `S3_SECRET_KEY` | MinIO secret key | `minioadmin` |
| `S3_USE_SSL` | Use SSL for S3 | `false` |
| `AUTH_SERVICE_URL` | Auth service base URL for JWT validation | `http://localhost:8084/api/v1` |
| `MAX_FILE_SIZE` | Max upload size (bytes) | `10485760` (10MB) |
| `ALLOWED_FILE_TYPES` | Allowed MIME types | `image/jpeg,image/jpg,image/png,image/gif,image/webp,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document,application/msword` |

## Integration

This API is used by admin-api for file uploads and by public-api for generating file URLs.

## License

Part of the GunarsK Portfolio platform.
