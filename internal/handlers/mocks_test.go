package handlers

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/GunarsK-portfolio/files-api/internal/config"
	"github.com/GunarsK-portfolio/files-api/internal/repository"
	commonConfig "github.com/GunarsK-portfolio/portfolio-common/config"
	commonRepo "github.com/GunarsK-portfolio/portfolio-common/repository"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

// =============================================================================
// Test Constants
// =============================================================================

const (
	testFileName     = "test-image.png"
	testFileKey      = "abc123-def456.png"
	testBucket       = "images"
	testMimeType     = "image/png"
	testFileType     = "portfolio-image"
	testFileSize     = int64(1024)
	testMaxFileSize  = int64(10485760) // 10MB
	testImagesBucket = "images"
	testDocsBucket   = "documents"
	testMiniBucket   = "miniatures"
)

// =============================================================================
// Mock Repository
// =============================================================================

type mockRepository struct {
	createFileFunc   func(ctx context.Context, bucket, key, fileName, fileType string, fileSize int64, mimeType string) (*repository.StorageFile, error)
	getFileByIDFunc  func(ctx context.Context, id int64) (*repository.StorageFile, error)
	getFileByKeyFunc func(ctx context.Context, bucket, key string) (*repository.StorageFile, error)
	deleteFileFunc   func(ctx context.Context, id int64) error
}

func (m *mockRepository) CreateFile(ctx context.Context, bucket, key, fileName, fileType string, fileSize int64, mimeType string) (*repository.StorageFile, error) {
	if m.createFileFunc != nil {
		return m.createFileFunc(ctx, bucket, key, fileName, fileType, fileSize, mimeType)
	}
	return nil, nil
}

func (m *mockRepository) GetFileByID(ctx context.Context, id int64) (*repository.StorageFile, error) {
	if m.getFileByIDFunc != nil {
		return m.getFileByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockRepository) GetFileByKey(ctx context.Context, bucket, key string) (*repository.StorageFile, error) {
	if m.getFileByKeyFunc != nil {
		return m.getFileByKeyFunc(ctx, bucket, key)
	}
	return nil, nil
}

func (m *mockRepository) DeleteFile(ctx context.Context, id int64) error {
	if m.deleteFileFunc != nil {
		return m.deleteFileFunc(ctx, id)
	}
	return nil
}

// =============================================================================
// Mock Storage
// =============================================================================

type mockStorage struct {
	getObjectFunc    func(ctx context.Context, bucket, key string) (*minio.Object, error)
	putObjectFunc    func(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string) error
	deleteObjectFunc func(ctx context.Context, bucket, key string) error
	statObjectFunc   func(ctx context.Context, bucket, key string) (minio.ObjectInfo, error)
}

func (m *mockStorage) GetObject(ctx context.Context, bucket, key string) (*minio.Object, error) {
	if m.getObjectFunc != nil {
		return m.getObjectFunc(ctx, bucket, key)
	}
	return nil, nil
}

func (m *mockStorage) PutObject(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string) error {
	if m.putObjectFunc != nil {
		return m.putObjectFunc(ctx, bucket, key, reader, size, contentType)
	}
	return nil
}

func (m *mockStorage) DeleteObject(ctx context.Context, bucket, key string) error {
	if m.deleteObjectFunc != nil {
		return m.deleteObjectFunc(ctx, bucket, key)
	}
	return nil
}

func (m *mockStorage) StatObject(ctx context.Context, bucket, key string) (minio.ObjectInfo, error) {
	if m.statObjectFunc != nil {
		return m.statObjectFunc(ctx, bucket, key)
	}
	return minio.ObjectInfo{}, nil
}

// =============================================================================
// Mock Action Log Repository
// =============================================================================

type mockActionLogRepo struct{}

func (m *mockActionLogRepo) LogAction(_ *commonRepo.ActionLog) error {
	return nil
}

func (m *mockActionLogRepo) GetActionsByType(_ string, _ int) ([]commonRepo.ActionLog, error) {
	return nil, nil
}

func (m *mockActionLogRepo) GetActionsByResource(_ string, _ int64) ([]commonRepo.ActionLog, error) {
	return nil, nil
}

func (m *mockActionLogRepo) GetActionsByUser(_ int64, _ int) ([]commonRepo.ActionLog, error) {
	return nil, nil
}

func (m *mockActionLogRepo) CountActionsByResource(_ string, _ int64) (int64, error) {
	return 0, nil
}

// =============================================================================
// Test Helpers
// =============================================================================

type ctxKey struct{}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func createTestConfig() *config.Config {
	return &config.Config{
		S3Config: commonConfig.S3Config{
			ImagesBucket:     testImagesBucket,
			DocumentsBucket:  testDocsBucket,
			MiniaturesBucket: testMiniBucket,
		},
		MaxFileSize:      testMaxFileSize,
		AllowedFileTypes: []string{"image/png", "image/jpeg", "image/gif", "application/pdf"},
	}
}

func createTestFile() *repository.StorageFile {
	return &repository.StorageFile{
		ID:        1,
		S3Key:     testFileKey,
		S3Bucket:  testBucket,
		FileName:  testFileName,
		FileSize:  testFileSize,
		MimeType:  testMimeType,
		FileType:  testFileType,
		CreatedAt: time.Now(),
	}
}

func performRequest(router *gin.Engine, method, path string, body io.Reader, headers ...map[string]string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, body)
	if len(headers) > 0 {
		for key, value := range headers[0] {
			req.Header.Set(key, value)
		}
	}
	router.ServeHTTP(w, req)
	return w
}

// createMultipartRequest creates a multipart form request for file upload testing.
// Returns the request and recorder, or an error if request creation fails.
func createMultipartRequest(filename, contentType, fileType string, fileContent []byte) (*http.Request, *httptest.ResponseRecorder, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create file part with custom headers
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="file"; filename="` + filename + `"`}
	h["Content-Type"] = []string{contentType}
	part, err := writer.CreatePart(h)
	if err != nil {
		return nil, nil, err
	}
	if _, err := part.Write(fileContent); err != nil {
		return nil, nil, err
	}

	// Add fileType field
	if err := writer.WriteField("fileType", fileType); err != nil {
		return nil, nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, nil, err
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	return req, w, nil
}
