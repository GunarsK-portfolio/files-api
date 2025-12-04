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

	"github.com/GunarsK-portfolio/files-api/internal/repository"
	"github.com/gin-gonic/gin"
)

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
// Upload File Success Tests
// =============================================================================

func TestUploadFile_Success(t *testing.T) {
	var uploadedBucket, uploadedKey, uploadedContentType string
	var uploadedSize int64
	var dbCreateCalled bool

	createdFile := &repository.StorageFile{
		ID:       1,
		S3Key:    "generated-uuid.png",
		S3Bucket: testImagesBucket,
		FileName: "test-upload.png",
		FileSize: 15,
		MimeType: "image/png",
		FileType: "portfolio-image",
	}

	mockRepo := &mockRepository{
		createFileFunc: func(_ context.Context, bucket, key, fileName, fileType string, fileSize int64, mimeType string) (*repository.StorageFile, error) {
			dbCreateCalled = true
			createdFile.S3Key = key
			return createdFile, nil
		},
	}

	mockStore := &mockStorage{
		putObjectFunc: func(_ context.Context, bucket, key string, _ io.Reader, size int64, contentType string) error {
			uploadedBucket = bucket
			uploadedKey = key
			uploadedSize = size
			uploadedContentType = contentType
			return nil
		},
	}

	cfg := createTestConfig()
	handler := New(mockRepo, mockStore, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.POST("/api/v1/files", handler.UploadFile)

	// Create multipart request with valid file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create PNG file part with proper headers
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="file"; filename="test-upload.png"`}
	h["Content-Type"] = []string{"image/png"}
	part, err := writer.CreatePart(h)
	if err != nil {
		t.Fatalf("failed to create part: %v", err)
	}
	if _, err := part.Write([]byte("fake png data")); err != nil {
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

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify storage was called
	if uploadedBucket != testImagesBucket {
		t.Errorf("expected bucket %s, got %s", testImagesBucket, uploadedBucket)
	}
	if uploadedKey == "" {
		t.Error("expected key to be generated")
	}
	if uploadedSize != 13 { // "fake png data" is 13 bytes
		t.Errorf("expected size 13, got %d", uploadedSize)
	}
	if uploadedContentType != "image/png" {
		t.Errorf("expected content type image/png, got %s", uploadedContentType)
	}

	// Verify database was called
	if !dbCreateCalled {
		t.Error("expected repository CreateFile to be called")
	}

	// Verify response contains file info
	if !strings.Contains(w.Body.String(), "fileName") {
		t.Errorf("expected response to contain fileName, got %s", w.Body.String())
	}
}

// =============================================================================
// Upload File Error Tests
// =============================================================================

func TestUploadFile_S3Error(t *testing.T) {
	mockRepo := &mockRepository{}
	mockStore := &mockStorage{
		putObjectFunc: func(_ context.Context, _, _ string, _ io.Reader, _ int64, _ string) error {
			return errors.New("S3 connection error")
		},
	}

	cfg := createTestConfig()
	handler := New(mockRepo, mockStore, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.POST("/api/v1/files", handler.UploadFile)

	// Create multipart request
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="file"; filename="test.png"`}
	h["Content-Type"] = []string{"image/png"}
	part, _ := writer.CreatePart(h)
	_, _ = part.Write([]byte("data"))
	_ = writer.WriteField("fileType", "portfolio-image")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	if !strings.Contains(w.Body.String(), "failed to upload file") {
		t.Errorf("expected 'failed to upload file' error, got %s", w.Body.String())
	}
}

func TestUploadFile_DBErrorWithS3Cleanup(t *testing.T) {
	var s3CleanupCalled bool
	var cleanupBucket, cleanupKey string

	mockRepo := &mockRepository{
		createFileFunc: func(_ context.Context, _, _, _, _ string, _ int64, _ string) (*repository.StorageFile, error) {
			return nil, errors.New("database error")
		},
	}

	mockStore := &mockStorage{
		putObjectFunc: func(_ context.Context, _, _ string, _ io.Reader, _ int64, _ string) error {
			return nil // S3 upload succeeds
		},
		deleteObjectFunc: func(_ context.Context, bucket, key string) error {
			s3CleanupCalled = true
			cleanupBucket = bucket
			cleanupKey = key
			return nil
		},
	}

	cfg := createTestConfig()
	handler := New(mockRepo, mockStore, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.POST("/api/v1/files", handler.UploadFile)

	// Create multipart request
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="file"; filename="test.png"`}
	h["Content-Type"] = []string{"image/png"}
	part, _ := writer.CreatePart(h)
	_, _ = part.Write([]byte("data"))
	_ = writer.WriteField("fileType", "portfolio-image")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	// Verify S3 cleanup was attempted
	if !s3CleanupCalled {
		t.Error("expected S3 cleanup to be called after database error")
	}
	if cleanupBucket != testImagesBucket {
		t.Errorf("expected cleanup bucket %s, got %s", testImagesBucket, cleanupBucket)
	}
	if cleanupKey == "" {
		t.Error("expected cleanup key to be set")
	}
}

// =============================================================================
// Upload File Document Tests
// =============================================================================

func TestUploadFile_PDFDocument(t *testing.T) {
	var uploadedBucket string
	var dbCreateCalled bool

	createdFile := &repository.StorageFile{
		ID:       1,
		S3Key:    "generated-uuid.pdf",
		S3Bucket: testDocsBucket,
		FileName: "test-document.pdf",
		FileSize: 8,
		MimeType: "application/pdf",
		FileType: "document",
	}

	mockRepo := &mockRepository{
		createFileFunc: func(_ context.Context, bucket, key, fileName, fileType string, fileSize int64, mimeType string) (*repository.StorageFile, error) {
			dbCreateCalled = true
			uploadedBucket = bucket
			createdFile.S3Key = key
			return createdFile, nil
		},
	}

	mockStore := &mockStorage{
		putObjectFunc: func(_ context.Context, _, _ string, _ io.Reader, _ int64, _ string) error {
			return nil
		},
	}

	cfg := createTestConfig()
	handler := New(mockRepo, mockStore, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.POST("/api/v1/files", handler.UploadFile)

	// Create multipart request with PDF file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="file"; filename="test-document.pdf"`}
	h["Content-Type"] = []string{"application/pdf"}
	part, _ := writer.CreatePart(h)
	_, _ = part.Write([]byte("PDF data"))
	_ = writer.WriteField("fileType", "document")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	if !dbCreateCalled {
		t.Error("expected repository CreateFile to be called")
	}

	if uploadedBucket != testDocsBucket {
		t.Errorf("expected bucket %s, got %s", testDocsBucket, uploadedBucket)
	}
}

func TestUploadFile_InvalidFileTypeForImage(t *testing.T) {
	mockRepo := &mockRepository{}
	mockStore := &mockStorage{}
	cfg := createTestConfig()
	handler := New(mockRepo, mockStore, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.POST("/api/v1/files", handler.UploadFile)

	// Try to upload PDF as portfolio-image
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="file"; filename="test.pdf"`}
	h["Content-Type"] = []string{"application/pdf"}
	part, _ := writer.CreatePart(h)
	_, _ = part.Write([]byte("PDF data"))
	_ = writer.WriteField("fileType", "portfolio-image")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "requires image content type") {
		t.Errorf("expected 'requires image content type' error, got %s", w.Body.String())
	}
}

func TestUploadFile_InvalidFileTypeForDocument(t *testing.T) {
	mockRepo := &mockRepository{}
	mockStore := &mockStorage{}
	cfg := createTestConfig()
	handler := New(mockRepo, mockStore, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.POST("/api/v1/files", handler.UploadFile)

	// Try to upload image as document
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="file"; filename="test.png"`}
	h["Content-Type"] = []string{"image/png"}
	part, _ := writer.CreatePart(h)
	_, _ = part.Write([]byte("image data"))
	_ = writer.WriteField("fileType", "document")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "requires PDF or Word document") {
		t.Errorf("expected 'requires PDF or Word document' error, got %s", w.Body.String())
	}
}

func TestUploadFile_DBErrorWithS3CleanupFailure(t *testing.T) {
	// Test that cleanup failure is logged but doesn't change the response
	var s3CleanupCalled bool

	mockRepo := &mockRepository{
		createFileFunc: func(_ context.Context, _, _, _, _ string, _ int64, _ string) (*repository.StorageFile, error) {
			return nil, errors.New("database error")
		},
	}

	mockStore := &mockStorage{
		putObjectFunc: func(_ context.Context, _, _ string, _ io.Reader, _ int64, _ string) error {
			return nil // S3 upload succeeds
		},
		deleteObjectFunc: func(_ context.Context, _, _ string) error {
			s3CleanupCalled = true
			return errors.New("S3 cleanup failed") // Cleanup also fails
		},
	}

	cfg := createTestConfig()
	handler := New(mockRepo, mockStore, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.POST("/api/v1/files", handler.UploadFile)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="file"; filename="test.png"`}
	h["Content-Type"] = []string{"image/png"}
	part, _ := writer.CreatePart(h)
	_, _ = part.Write([]byte("data"))
	_ = writer.WriteField("fileType", "portfolio-image")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should still return 500 for the original DB error
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	// Verify cleanup was attempted
	if !s3CleanupCalled {
		t.Error("expected S3 cleanup to be attempted")
	}

	// Original error message should still be returned
	if !strings.Contains(w.Body.String(), "failed to create file record") {
		t.Errorf("expected 'failed to create file record' error, got %s", w.Body.String())
	}
}

func TestUploadFile_MiniatureImage(t *testing.T) {
	var uploadedBucket string

	createdFile := &repository.StorageFile{
		ID:       1,
		S3Key:    "generated-uuid.png",
		S3Bucket: testMiniBucket,
		FileName: "miniature.png",
		FileSize: 8,
		MimeType: "image/png",
		FileType: "miniature-image",
	}

	mockRepo := &mockRepository{
		createFileFunc: func(_ context.Context, bucket, key, _, _ string, _ int64, _ string) (*repository.StorageFile, error) {
			uploadedBucket = bucket
			createdFile.S3Key = key
			return createdFile, nil
		},
	}

	mockStore := &mockStorage{
		putObjectFunc: func(_ context.Context, _, _ string, _ io.Reader, _ int64, _ string) error {
			return nil
		},
	}

	cfg := createTestConfig()
	handler := New(mockRepo, mockStore, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.POST("/api/v1/files", handler.UploadFile)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="file"; filename="miniature.png"`}
	h["Content-Type"] = []string{"image/png"}
	part, _ := writer.CreatePart(h)
	_, _ = part.Write([]byte("png data"))
	_ = writer.WriteField("fileType", "miniature-image")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	if uploadedBucket != testMiniBucket {
		t.Errorf("expected bucket %s, got %s", testMiniBucket, uploadedBucket)
	}
}

// =============================================================================
// Upload File Context Propagation Test
// =============================================================================

func TestUploadFile_ContextPropagation(t *testing.T) {
	var capturedCtx context.Context

	mockRepo := &mockRepository{
		createFileFunc: func(ctx context.Context, _, _, _, _ string, _ int64, _ string) (*repository.StorageFile, error) {
			capturedCtx = ctx
			return &repository.StorageFile{ID: 1, FileName: "test.png"}, nil
		},
	}

	mockStore := &mockStorage{
		putObjectFunc: func(_ context.Context, _, _ string, _ io.Reader, _ int64, _ string) error {
			return nil
		},
	}

	cfg := createTestConfig()
	handler := New(mockRepo, mockStore, cfg, &mockActionLogRepo{})

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add middleware that injects a sentinel value into the context
	router.Use(func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), ctxKey{}, "upload-test-marker")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	router.POST("/api/v1/files", handler.UploadFile)

	// Create multipart request
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="file"; filename="test.png"`}
	h["Content-Type"] = []string{"image/png"}
	part, _ := writer.CreatePart(h)
	_, _ = part.Write([]byte("data"))
	_ = writer.WriteField("fileType", "portfolio-image")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	if capturedCtx == nil {
		t.Error("expected context to be propagated to repository")
	}

	if capturedCtx.Value(ctxKey{}) != "upload-test-marker" {
		t.Error("context sentinel value was not propagated to repository")
	}
}
