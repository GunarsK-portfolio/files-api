# Testing Guide

## Overview

The files-api uses Go's standard `testing` package with httptest for handler
and route-level unit tests. **48 tests total** across handlers and routes.
This service handles file uploads/downloads to MinIO/S3 storage.

## Quick Commands

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run handler tests only
go test -v ./internal/handlers/

# Run route RBAC tests only
go test -v ./internal/routes/

# Run all Delete tests
go test -v -run DeleteFile ./internal/handlers/

# Run all Download tests
go test -v -run DownloadFile ./internal/handlers/

# Run all Upload tests
go test -v -run UploadFile ./internal/handlers/

# Run permission tests
go test -v -run Permission ./internal/routes/
```

## Test Files

### `internal/handlers/` - 35 tests

| File | Tests | Coverage |
| ---- | ----- | -------- |
| `delete_test.go` | 7 | Success, invalid ID, not found, errors, context |
| `download_test.go` | 6 | Invalid type, not found, errors, traversal |
| `upload_test.go` | 15 | Success, validation, S3/DB errors, cleanup, hostiles |
| `handler_test.go` | 7 | Bucket mapping, content types, constructor |

### `internal/routes/` - 13 tests

| Category | Tests | Coverage |
| -------- | ----- | -------- |
| Files Routes Forbidden | 2 | POST/DELETE return 403 without permission |
| Files Routes Allowed | 2 | POST/DELETE accessible with correct permission |
| Permission Hierarchy | 6 | delete > edit > read > none hierarchy |
| Middleware Error Handling | 3 | No scopes (401), invalid format (500), repo errors |

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

// With custom headers
headers := map[string]string{"Authorization": "Bearer token"}
w := performRequest(router, http.MethodGet, path, nil, headers)
```

**Multipart Upload Testing**: Use the `createMultipartRequest` helper

```go
req, w, err := createMultipartRequest(
    "test.png", "image/png", "portfolio-image", []byte("data"))
if err != nil {
    t.Fatalf("failed to create multipart request: %v", err)
}
router.ServeHTTP(w, req)
```

For edge cases requiring custom multipart structure (missing fields, oversized files):

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
- S3 storage errors (500 Internal Server Error)
- Database errors with S3 cleanup verification
- Path traversal attempts (400/404 rejection)

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

The storage layer uses the `storage.ObjectStore` interface, which enables
mocking for unit tests. The concrete implementation (`*storage.Storage`) uses
the MinIO client.

**Mock Storage**: Function fields allow per-test S3 behavior customization

```go
mockStore := &mockStorage{
    deleteObjectFunc: func(ctx context.Context, bucket, key string) error {
        return nil
    },
}
handler := New(mockRepo, mockStore, cfg, &mockActionLogRepo{})
```

Tests cover S3 operations (upload, download, delete), validation, error handling,
and repository interactions. The upload success path is fully tested with mocks.

## Contributing Tests

1. Follow naming: `Test<HandlerName>_<Scenario>` or `Test<HandlerName>_<Condition>_<ExpectedBehavior>`
2. Organize by endpoint with section markers
3. Mock only the repository methods needed
4. Use `createMultipartRequest` helper for upload tests
5. Check error return values in test setup
6. Verify: `go test -cover ./internal/handlers/`

## Test Helper Functions

Located in `mocks_test.go`:

| Helper | Purpose |
| ------ | ------- |
| `setupTestRouter()` | Creates Gin router in test mode |
| `createTestConfig()` | Creates config with test bucket names |
| `createTestFile()` | Creates sample StorageFile struct |
| `performRequest(...)` | Executes HTTP request with optional headers |
| `createMultipartRequest(...)` | Creates multipart upload request |
