# Testing Guide

## Overview

The files-api uses Go's standard `testing` package with httptest for handler
unit tests. This service handles file uploads/downloads to MinIO/S3 storage.

## Quick Commands

```bash
# Run all tests
go test ./internal/handlers/

# Run with coverage
go test -cover ./internal/handlers/

# Generate coverage report
go test -coverprofile=coverage.out ./internal/handlers/
go tool cover -html=coverage.out -o coverage.html

# Run specific test
go test -v -run TestUploadFile_MissingFile ./internal/handlers/

# Run all Delete tests
go test -v -run TestDeleteFile ./internal/handlers/

# Run all Download tests
go test -v -run TestDownloadFile ./internal/handlers/

# Run all Upload tests
go test -v -run TestUploadFile ./internal/handlers/
```

## Test Files

**`handler_test.go`** - 21 tests

| Category | Tests | Coverage |
|----------|-------|----------|
| Health Check | 1 | HealthCheck endpoint |
| Delete File | 4 | Delete + error cases |
| File Type to Bucket | 4 | Bucket mapping |
| Content Type Validation | 1 | Allowed content types (7 subtests) |
| Bucket for File Type | 1 | File type + content type validation (8 subtests) |
| Download File | 3 | Download + error cases |
| Upload File | 4 | Upload validation |
| Constructor | 1 | Handler initialization |
| Context Propagation | 2 | Verifies context passed to repository |

## Key Testing Patterns

**Mock Repository**: Function fields allow per-test behavior customization

```go
mockRepo := &mockRepository{
    getFileByIDFunc: func(ctx context.Context, id int64) (*StorageFile, error) {
        return testFile, nil
    },
}
```

**HTTP Testing**: Uses `httptest.ResponseRecorder` with Gin router

```go
path := "/api/v1/files/portfolio-image/test.png"
w := performRequest(router, http.MethodGet, path, nil)
if w.Code != http.StatusOK { ... }
```

**Multipart Upload Testing**: Creates multipart form data for uploads

```go
body := &bytes.Buffer{}
writer := multipart.NewWriter(body)
part, _ := writer.CreateFormFile("file", "test.png")
part.Write([]byte("fake image data"))
writer.WriteField("fileType", "portfolio-image")
writer.Close()
```

**Test Helpers**: Factory functions for consistent test data

```go
cfg := createTestConfig()
file := createTestFile()
```

## Test Categories

### Success Cases

- Returns expected data
- Sets correct HTTP status (200 OK)
- File type to bucket mapping works

### Error Cases

- Repository errors (500 Internal Server Error)
- Not found errors (404 Not Found)
- Invalid ID format (400 Bad Request)
- Missing required fields (400 Bad Request)
- File too large (400 Bad Request)
- Invalid content type (400 Bad Request)
- Invalid file type (400 Bad Request)

## API Characteristics

Files-api handles file storage operations:

- **Upload**: Requires authentication, validates file type/size
- **Download**: Public access, streams from S3
- **Delete**: Requires authentication, removes from both S3 and database

File types:

- `portfolio-image` -> images bucket
- `miniature-image` -> miniatures bucket
- `document` -> documents bucket

## Storage Layer

The storage layer (`*storage.Storage`) uses the MinIO client directly. Testing
file upload success paths that involve actual S3 operations requires either:

- Integration tests with a real MinIO instance
- A storage interface abstraction for full mock capability

Current tests cover validation, error handling, and repository interactions.

## Contributing Tests

1. Follow naming: `Test<HandlerName>_<Scenario>`
2. Organize by endpoint with section markers
3. Mock only the repository methods needed
4. Check error return values in test setup
5. Verify: `go test -cover ./internal/handlers/`
