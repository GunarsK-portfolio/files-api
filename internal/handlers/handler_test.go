package handlers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GunarsK-portfolio/files-api/internal/config"
	"github.com/GunarsK-portfolio/files-api/internal/repository"
	commonConfig "github.com/GunarsK-portfolio/portfolio-common/config"
	commonRepo "github.com/GunarsK-portfolio/portfolio-common/repository"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
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

func performRequest(router *gin.Engine, method, path string, body io.Reader) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, body)
	router.ServeHTTP(w, req)
	return w
}

// =============================================================================
// Delete File Tests
// =============================================================================

func TestDeleteFile_Success(t *testing.T) {
	testFile := createTestFile()
	var deletedBucket, deletedKey string
	var repoDeleteCalled bool

	mockRepo := &mockRepository{
		getFileByIDFunc: func(_ context.Context, _ int64) (*repository.StorageFile, error) {
			return testFile, nil
		},
		deleteFileFunc: func(_ context.Context, _ int64) error {
			repoDeleteCalled = true
			return nil
		},
	}

	mockStore := &mockStorage{
		deleteObjectFunc: func(_ context.Context, bucket, key string) error {
			deletedBucket = bucket
			deletedKey = key
			return nil
		},
	}

	cfg := createTestConfig()
	handler := New(mockRepo, mockStore, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.DELETE("/api/v1/files/:id", handler.DeleteFile)

	w := performRequest(router, http.MethodDelete, "/api/v1/files/1", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify storage was called with correct parameters
	if deletedBucket != testImagesBucket {
		t.Errorf("expected bucket %s, got %s", testImagesBucket, deletedBucket)
	}
	if deletedKey != testFileKey {
		t.Errorf("expected key %s, got %s", testFileKey, deletedKey)
	}

	// Verify repository delete was called
	if !repoDeleteCalled {
		t.Error("expected repository DeleteFile to be called")
	}

	// Verify response message
	if !strings.Contains(w.Body.String(), "file deleted successfully") {
		t.Errorf("expected success message, got %s", w.Body.String())
	}
}

func TestDeleteFile_InvalidID(t *testing.T) {
	mockRepo := &mockRepository{}
	cfg := createTestConfig()
	handler := New(mockRepo, nil, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.DELETE("/api/v1/files/:id", handler.DeleteFile)

	w := performRequest(router, http.MethodDelete, "/api/v1/files/invalid", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "invalid file ID") {
		t.Errorf("expected 'invalid file ID' error, got %s", w.Body.String())
	}
}

func TestDeleteFile_NotFound(t *testing.T) {
	mockRepo := &mockRepository{
		getFileByIDFunc: func(_ context.Context, _ int64) (*repository.StorageFile, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}
	cfg := createTestConfig()
	handler := New(mockRepo, nil, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.DELETE("/api/v1/files/:id", handler.DeleteFile)

	w := performRequest(router, http.MethodDelete, "/api/v1/files/999", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestDeleteFile_RepositoryError(t *testing.T) {
	mockRepo := &mockRepository{
		getFileByIDFunc: func(_ context.Context, _ int64) (*repository.StorageFile, error) {
			return nil, errors.New("database error")
		},
	}
	cfg := createTestConfig()
	handler := New(mockRepo, nil, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.DELETE("/api/v1/files/:id", handler.DeleteFile)

	w := performRequest(router, http.MethodDelete, "/api/v1/files/1", nil)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

// =============================================================================
// File Type to Bucket Tests
// =============================================================================

func TestFileTypeToBucket_PortfolioImage(t *testing.T) {
	cfg := createTestConfig()
	handler := New(&mockRepository{}, nil, cfg, &mockActionLogRepo{})

	bucket, err := handler.fileTypeToBucket("portfolio-image")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if bucket != testImagesBucket {
		t.Errorf("expected bucket %s, got %s", testImagesBucket, bucket)
	}
}

func TestFileTypeToBucket_MiniatureImage(t *testing.T) {
	cfg := createTestConfig()
	handler := New(&mockRepository{}, nil, cfg, &mockActionLogRepo{})

	bucket, err := handler.fileTypeToBucket("miniature-image")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if bucket != testMiniBucket {
		t.Errorf("expected bucket %s, got %s", testMiniBucket, bucket)
	}
}

func TestFileTypeToBucket_Document(t *testing.T) {
	cfg := createTestConfig()
	handler := New(&mockRepository{}, nil, cfg, &mockActionLogRepo{})

	bucket, err := handler.fileTypeToBucket("document")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if bucket != testDocsBucket {
		t.Errorf("expected bucket %s, got %s", testDocsBucket, bucket)
	}
}

func TestFileTypeToBucket_Invalid(t *testing.T) {
	cfg := createTestConfig()
	handler := New(&mockRepository{}, nil, cfg, &mockActionLogRepo{})

	_, err := handler.fileTypeToBucket("invalid-type")
	if err == nil {
		t.Error("expected error for invalid file type")
	}
	if !strings.Contains(err.Error(), "invalid fileType") {
		t.Errorf("expected 'invalid fileType' error, got %v", err)
	}
}

// =============================================================================
// Content Type Validation Tests
// =============================================================================

func TestIsAllowedContentType_ValidImage(t *testing.T) {
	cfg := createTestConfig()
	handler := New(&mockRepository{}, nil, cfg, &mockActionLogRepo{})

	testCases := []struct {
		contentType string
		expected    bool
	}{
		{"image/png", true},
		{"image/jpeg", true},
		{"image/gif", true},
		{"application/pdf", true},
		{"text/plain", false},
		{"application/json", false},
		{"video/mp4", false},
	}

	for _, tc := range testCases {
		t.Run(tc.contentType, func(t *testing.T) {
			result := handler.isAllowedContentType(tc.contentType)
			if result != tc.expected {
				t.Errorf("isAllowedContentType(%s) = %v, want %v", tc.contentType, result, tc.expected)
			}
		})
	}
}

// =============================================================================
// Get Bucket For File Type Tests
// =============================================================================

func TestGetBucketForFileType_ValidCombinations(t *testing.T) {
	cfg := createTestConfig()
	handler := New(&mockRepository{}, nil, cfg, &mockActionLogRepo{})

	testCases := []struct {
		fileType    string
		contentType string
		wantBucket  string
		wantErr     bool
	}{
		{"portfolio-image", "image/png", testImagesBucket, false},
		{"portfolio-image", "image/jpeg", testImagesBucket, false},
		{"miniature-image", "image/png", testMiniBucket, false},
		{"miniature-image", "image/gif", testMiniBucket, false},
		{"document", "application/pdf", testDocsBucket, false},
		// Invalid combinations
		{"portfolio-image", "application/pdf", "", true},
		{"miniature-image", "text/plain", "", true},
		{"document", "image/png", "", true},
	}

	for _, tc := range testCases {
		name := tc.fileType + "_" + tc.contentType
		t.Run(name, func(t *testing.T) {
			bucket, err := handler.getBucketForFileType(tc.fileType, tc.contentType)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if bucket != tc.wantBucket {
					t.Errorf("expected bucket %s, got %s", tc.wantBucket, bucket)
				}
			}
		})
	}
}

// =============================================================================
// Download File Tests
// =============================================================================

func TestDownloadFile_InvalidFileType(t *testing.T) {
	mockRepo := &mockRepository{}
	cfg := createTestConfig()
	handler := New(mockRepo, nil, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.GET("/api/v1/files/:fileType/*key", handler.DownloadFile)

	w := performRequest(router, http.MethodGet, "/api/v1/files/invalid-type/abc123.png", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestDownloadFile_FileNotInDatabase(t *testing.T) {
	mockRepo := &mockRepository{
		getFileByKeyFunc: func(_ context.Context, _, _ string) (*repository.StorageFile, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}
	cfg := createTestConfig()
	handler := New(mockRepo, nil, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.GET("/api/v1/files/:fileType/*key", handler.DownloadFile)

	w := performRequest(router, http.MethodGet, "/api/v1/files/portfolio-image/nonexistent.png", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestDownloadFile_DatabaseError(t *testing.T) {
	mockRepo := &mockRepository{
		getFileByKeyFunc: func(_ context.Context, _, _ string) (*repository.StorageFile, error) {
			return nil, errors.New("database connection error")
		},
	}
	cfg := createTestConfig()
	handler := New(mockRepo, nil, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.GET("/api/v1/files/:fileType/*key", handler.DownloadFile)

	w := performRequest(router, http.MethodGet, "/api/v1/files/portfolio-image/test.png", nil)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

// =============================================================================
// Upload File Validation Tests
// =============================================================================

func TestUploadFile_MissingFile(t *testing.T) {
	mockRepo := &mockRepository{}
	cfg := createTestConfig()
	handler := New(mockRepo, nil, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.POST("/api/v1/files", handler.UploadFile)

	// Request without file
	req := httptest.NewRequest(http.MethodPost, "/api/v1/files", nil)
	req.Header.Set("Content-Type", "multipart/form-data")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "file is required") {
		t.Errorf("expected 'file is required' error, got %s", w.Body.String())
	}
}

func TestUploadFile_MissingFileType(t *testing.T) {
	mockRepo := &mockRepository{}
	cfg := createTestConfig()
	handler := New(mockRepo, nil, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.POST("/api/v1/files", handler.UploadFile)

	// Create multipart request without fileType field
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.png")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := part.Write([]byte("fake image data")); err != nil {
		t.Fatalf("failed to write to part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "fileType is required") {
		t.Errorf("expected 'fileType is required' error, got %s", w.Body.String())
	}
}

func TestUploadFile_FileTooLarge(t *testing.T) {
	mockRepo := &mockRepository{}
	cfg := createTestConfig()
	cfg.MaxFileSize = 100 // Set very small limit for test
	handler := New(mockRepo, nil, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.POST("/api/v1/files", handler.UploadFile)

	// Create multipart request with large file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.png")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	// Write more than 100 bytes
	if _, err := part.Write(make([]byte, 200)); err != nil {
		t.Fatalf("failed to write to part: %v", err)
	}
	if err := writer.WriteField("fileType", "portfolio-image"); err != nil {
		t.Fatalf("failed to write field: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "file too large") {
		t.Errorf("expected 'file too large' error, got %s", w.Body.String())
	}
}

func TestUploadFile_InvalidContentType(t *testing.T) {
	mockRepo := &mockRepository{}
	cfg := createTestConfig()
	handler := New(mockRepo, nil, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.POST("/api/v1/files", handler.UploadFile)

	// Create multipart request with invalid content type
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create a part with custom headers for invalid content type
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="file"; filename="test.exe"`}
	h["Content-Type"] = []string{"application/x-executable"}
	part, err := writer.CreatePart(h)
	if err != nil {
		t.Fatalf("failed to create part: %v", err)
	}
	if _, err := part.Write([]byte("fake executable data")); err != nil {
		t.Fatalf("failed to write to part: %v", err)
	}
	if err := writer.WriteField("fileType", "document"); err != nil {
		t.Fatalf("failed to write field: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "invalid file type") {
		t.Errorf("expected 'invalid file type' error, got %s", w.Body.String())
	}
}

// =============================================================================
// Constructor Tests
// =============================================================================

func TestNew_ReturnsHandler(t *testing.T) {
	mockRepo := &mockRepository{}
	cfg := createTestConfig()
	actionLogRepo := &mockActionLogRepo{}

	handler := New(mockRepo, nil, cfg, actionLogRepo)

	if handler == nil {
		t.Fatal("expected handler to not be nil")
	}
	if handler.repo == nil {
		t.Error("expected repo to be set")
	}
	if handler.cfg == nil {
		t.Error("expected cfg to be set")
	}
	if handler.actionLogRepo == nil {
		t.Error("expected actionLogRepo to be set")
	}
}

// =============================================================================
// Context Propagation Tests
// =============================================================================

type ctxKey struct{}

func TestDeleteFile_ContextPropagation(t *testing.T) {
	var capturedCtx context.Context
	testFile := createTestFile()

	mockRepo := &mockRepository{
		getFileByIDFunc: func(ctx context.Context, _ int64) (*repository.StorageFile, error) {
			capturedCtx = ctx
			return testFile, nil
		},
		deleteFileFunc: func(_ context.Context, _ int64) error {
			return nil
		},
	}

	mockStore := &mockStorage{
		deleteObjectFunc: func(_ context.Context, _, _ string) error {
			return nil
		},
	}

	cfg := createTestConfig()
	handler := New(mockRepo, mockStore, cfg, &mockActionLogRepo{})

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add middleware that injects a sentinel value into the context
	router.Use(func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), ctxKey{}, "test-marker")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	router.DELETE("/api/v1/files/:id", handler.DeleteFile)

	w := performRequest(router, http.MethodDelete, "/api/v1/files/1", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if capturedCtx == nil {
		t.Error("expected context to be propagated to repository")
	}

	// Verify the sentinel value was propagated through
	if capturedCtx.Value(ctxKey{}) != "test-marker" {
		t.Error("context sentinel value was not propagated to repository")
	}
}

func TestDownloadFile_ContextPropagation(t *testing.T) {
	var capturedCtx context.Context

	mockRepo := &mockRepository{
		getFileByKeyFunc: func(ctx context.Context, _, _ string) (*repository.StorageFile, error) {
			capturedCtx = ctx
			return nil, gorm.ErrRecordNotFound
		},
	}

	cfg := createTestConfig()
	handler := New(mockRepo, nil, cfg, &mockActionLogRepo{})

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add middleware that injects a sentinel value into the context
	router.Use(func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), ctxKey{}, "test-marker")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	router.GET("/api/v1/files/:fileType/*key", handler.DownloadFile)

	w := performRequest(router, http.MethodGet, "/api/v1/files/portfolio-image/test.png", nil)

	// Should be 404 due to record not found
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	if capturedCtx == nil {
		t.Error("expected context to be propagated to repository")
	}

	// Verify the sentinel value was propagated through
	if capturedCtx.Value(ctxKey{}) != "test-marker" {
		t.Error("context sentinel value was not propagated to repository")
	}
}
