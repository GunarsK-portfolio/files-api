# Files API

![CI](https://github.com/GunarsK-portfolio/files-api/workflows/CI/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/GunarsK-portfolio/files-api)](https://goreportcard.com/report/github.com/GunarsK-portfolio/files-api)
[![codecov](https://codecov.io/gh/GunarsK-portfolio/files-api/graph/badge.svg?token=3O8BMJ1K1B)](https://codecov.io/gh/GunarsK-portfolio/files-api)
[![CodeRabbit](https://img.shields.io/coderabbit/prs/github/GunarsK-portfolio/files-api?label=CodeRabbit&color=2ea44f)](https://coderabbit.ai)

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

- **Language**: Go 1.25.3
- **Framework**: Gin
- **Database**: PostgreSQL (GORM)
- **Storage**: MinIO (S3-compatible)
- **Documentation**: Swagger/OpenAPI

## Prerequisites

- Go 1.25+
- Node.js 22+ and npm 11+
- PostgreSQL (or use Docker Compose)
- MinIO (or use Docker Compose)

## Project Structure

```text
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

1. Update `.env` with your configuration:

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
S3_IMAGES_BUCKET=images
S3_DOCUMENTS_BUCKET=documents
S3_MINIATURES_BUCKET=miniatures
AUTH_SERVICE_URL=http://localhost:8084/api/v1
```

1. Start infrastructure (if not running):

```bash
# From infrastructure directory
docker-compose up -d postgres minio flyway auth-service
```

1. Run the service:

```bash
go run cmd/api/main.go
```

## Available Commands

Using Task:

```bash
# Development
task dev:swagger         # Generate Swagger documentation
task dev:install-tools   # Install dev tools (golangci-lint, govulncheck, etc.)

# Build and run
task build               # Build binary
task test                # Run tests
task test:coverage       # Run tests with coverage report
task clean               # Clean build artifacts

# Code quality
task format              # Format code with gofmt
task tidy                # Tidy and verify go.mod
task lint                # Run golangci-lint
task vet                 # Run go vet

# Security
task security:scan       # Run gosec security scanner
task security:vuln       # Check for vulnerabilities with govulncheck

# Docker
task docker:build        # Build Docker image
task docker:run          # Run service in Docker container
task docker:stop         # Stop running Docker container
task docker:logs         # View Docker container logs

# CI/CD
task ci:all              # Run all CI checks
```

Using Go directly:

```bash
go run cmd/api/main.go                       # Run
go build -o bin/files-api cmd/api/main.go    # Build
go test ./...                                 # Test
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
| `DB_SSLMODE` | PostgreSQL SSL mode | `disable` |
| `S3_ENDPOINT` | MinIO/S3 endpoint URL | `http://localhost:9000` |
| `S3_ACCESS_KEY` | MinIO/S3 access key (optional for AWS IAM) | `minioadmin` |
| `S3_SECRET_KEY` | MinIO/S3 secret key (optional for AWS IAM) | `minioadmin` |
| `S3_USE_SSL` | Use SSL for S3 | `false` |
| `S3_IMAGES_BUCKET` | S3 bucket for portfolio images | `images` |
| `S3_DOCUMENTS_BUCKET` | S3 bucket for documents | `documents` |
| `S3_MINIATURES_BUCKET` | S3 bucket for miniatures | `miniatures` |
| `AUTH_SERVICE_URL` | Auth service URL | `http://localhost:8084` |
| `MAX_FILE_SIZE` | Max upload size (bytes) | `10485760` (10MB) |
| `ALLOWED_FILE_TYPES` | Allowed MIME types | (see docs for full list) |

## Integration

This API is used by admin-api for uploads and by
public-api for file URLs.

## License

[MIT](LICENSE)
