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
	var s3Bucket, s3Key, s3ContentType string
	var s3Size int64
	var dbBucket, dbKey string
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
			dbBucket = bucket
			dbKey = key
			createdFile.S3Key = key
			return createdFile, nil
		},
	}

	mockStore := &mockStorage{
		putObjectFunc: func(_ context.Context, bucket, key string, _ io.Reader, size int64, contentType string) error {
			s3Bucket = bucket
			s3Key = key
			s3Size = size
			s3ContentType = contentType
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

	// Verify storage was called with correct parameters
	if s3Bucket != testImagesBucket {
		t.Errorf("expected S3 bucket %s, got %s", testImagesBucket, s3Bucket)
	}
	if s3Key == "" {
		t.Error("expected S3 key to be generated")
	}
	// Key should be server-generated UUID, not the client filename
	if s3Key == "test-upload.png" {
		t.Error("S3 key should be server-generated UUID, not client filename")
	}
	if s3Size != 13 { // "fake png data" is 13 bytes
		t.Errorf("expected S3 size 13, got %d", s3Size)
	}
	if s3ContentType != "image/png" {
		t.Errorf("expected S3 content type image/png, got %s", s3ContentType)
	}

	// Verify database was called
	if !dbCreateCalled {
		t.Error("expected repository CreateFile to be called")
	}

	// Verify S3 and DB received the same bucket/key (consistency check)
	if dbBucket != s3Bucket {
		t.Errorf("S3 bucket (%s) and DB bucket (%s) should match", s3Bucket, dbBucket)
	}
	if dbKey != s3Key {
		t.Errorf("S3 key (%s) and DB key (%s) should match", s3Key, dbKey)
	}

	// Verify response contains file info
	if !strings.Contains(w.Body.String(), "fileName") {
		t.Errorf("expected response to contain fileName, got %s", w.Body.String())
	}
}

// =============================================================================
// Upload File Error Tests
// =============================================================================

func TestUploadFile_S3Error_DBNotTouched(t *testing.T) {
	var dbCreateCalled bool

	mockRepo := &mockRepository{
		createFileFunc: func(_ context.Context, _, _, _, _ string, _ int64, _ string) (*repository.StorageFile, error) {
			dbCreateCalled = true
			return nil, nil
		},
	}
	mockStore := &mockStorage{
		putObjectFunc: func(_ context.Context, _, _ string, _ io.Reader, _ int64, _ string) error {
			return errors.New("S3 connection error")
		},
	}

	cfg := createTestConfig()
	handler := New(mockRepo, mockStore, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.POST("/api/v1/files", handler.UploadFile)

	req, w, err := createMultipartRequest("test.png", "image/png", "portfolio-image", []byte("data"))
	if err != nil {
		t.Fatalf("failed to create multipart request: %v", err)
	}
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	if !strings.Contains(w.Body.String(), "failed to upload file") {
		t.Errorf("expected 'failed to upload file' error, got %s", w.Body.String())
	}

	// Verify DB was not touched when S3 fails
	if dbCreateCalled {
		t.Error("DB CreateFile should not be called when S3 upload fails")
	}
}

func TestUploadFile_DBError_CleansUpS3Object(t *testing.T) {
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

	req, w, err := createMultipartRequest("test.png", "image/png", "portfolio-image", []byte("data"))
	if err != nil {
		t.Fatalf("failed to create multipart request: %v", err)
	}
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
	var s3Bucket, dbBucket string
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
		createFileFunc: func(_ context.Context, bucket, key, _, _ string, _ int64, _ string) (*repository.StorageFile, error) {
			dbCreateCalled = true
			dbBucket = bucket
			createdFile.S3Key = key
			return createdFile, nil
		},
	}

	mockStore := &mockStorage{
		putObjectFunc: func(_ context.Context, bucket, _ string, _ io.Reader, _ int64, _ string) error {
			s3Bucket = bucket
			return nil
		},
	}

	cfg := createTestConfig()
	handler := New(mockRepo, mockStore, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.POST("/api/v1/files", handler.UploadFile)

	req, w, err := createMultipartRequest("test-document.pdf", "application/pdf", "document", []byte("PDF data"))
	if err != nil {
		t.Fatalf("failed to create multipart request: %v", err)
	}
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	if !dbCreateCalled {
		t.Error("expected repository CreateFile to be called")
	}

	// Verify correct bucket selection for documents
	if s3Bucket != testDocsBucket {
		t.Errorf("expected S3 bucket %s, got %s", testDocsBucket, s3Bucket)
	}
	if dbBucket != testDocsBucket {
		t.Errorf("expected DB bucket %s, got %s", testDocsBucket, dbBucket)
	}
}

func TestUploadFile_WordDocument(t *testing.T) {
	var s3Bucket string
	var dbCreateCalled bool

	createdFile := &repository.StorageFile{
		ID:       1,
		S3Key:    "generated-uuid.docx",
		S3Bucket: testDocsBucket,
		FileName: "test-document.docx",
		FileSize: 9,
		MimeType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		FileType: "document",
	}

	mockRepo := &mockRepository{
		createFileFunc: func(_ context.Context, bucket, key, _, _ string, _ int64, _ string) (*repository.StorageFile, error) {
			dbCreateCalled = true
			createdFile.S3Key = key
			return createdFile, nil
		},
	}

	mockStore := &mockStorage{
		putObjectFunc: func(_ context.Context, bucket, _ string, _ io.Reader, _ int64, _ string) error {
			s3Bucket = bucket
			return nil
		},
	}

	cfg := createTestConfig()
	// Add Word document to allowed types
	cfg.AllowedFileTypes = append(cfg.AllowedFileTypes,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	handler := New(mockRepo, mockStore, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.POST("/api/v1/files", handler.UploadFile)

	// Create multipart request with Word document
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="file"; filename="test-document.docx"`}
	h["Content-Type"] = []string{"application/vnd.openxmlformats-officedocument.wordprocessingml.document"}
	part, _ := writer.CreatePart(h)
	_, _ = part.Write([]byte("Word data"))
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

	// Verify correct bucket selection for Word documents
	if s3Bucket != testDocsBucket {
		t.Errorf("expected S3 bucket %s, got %s", testDocsBucket, s3Bucket)
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

func TestUploadFile_DBError_S3CleanupFailure_ReturnsOriginalError(t *testing.T) {
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

	req, w, err := createMultipartRequest("test.png", "image/png", "portfolio-image", []byte("data"))
	if err != nil {
		t.Fatalf("failed to create multipart request: %v", err)
	}
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

	// S3 cleanup error should NOT be leaked in response (security)
	if strings.Contains(w.Body.String(), "S3 cleanup failed") {
		t.Error("S3 cleanup error should not be leaked in response")
	}
}

// =============================================================================
// Upload File Hostile Filename Tests
// =============================================================================

func TestUploadFile_HostileFilenames(t *testing.T) {
	// Test that hostile filenames are handled safely.
	// Defense-in-depth: Go's mime/multipart sanitizes filenames to base name only,
	// and the handler generates UUID keys for S3, so hostile filenames cannot
	// affect storage paths. The sanitized filename is stored in DB for display.

	testCases := []struct {
		name             string
		filename         string
		expectedFilename string // What Go's multipart sanitizes it to
		expectError      bool
	}{
		{
			name:             "path traversal dots",
			filename:         "../../../etc/passwd",
			expectedFilename: "passwd", // Go sanitizes to base name
			expectError:      false,
		},
		{
			name:             "path traversal encoded",
			filename:         "..%2F..%2F..%2Fetc%2Fpasswd",
			expectedFilename: "..%2F..%2F..%2Fetc%2Fpasswd", // URL encoding preserved
			expectError:      false,
		},
		{
			name:             "directory separator",
			filename:         "foo/bar/../../secret.png",
			expectedFilename: "secret.png", // Go sanitizes to base name
			expectError:      false,
		},
		{
			// Windows backslash paths: Go only sanitizes on Windows, not Linux.
			// On Linux, backslash is a valid filename character, so the full string is kept.
			// We test that even unsanitized, the S3 key is still a safe UUID.
			name:             "windows path backslashes",
			filename:         "C:\\Windows\\config\\SAM",
			expectedFilename: "", // Platform-dependent, checked separately
			expectError:      false,
		},
		{
			name:             "very long filename",
			filename:         strings.Repeat("a", 500) + ".png",
			expectedFilename: strings.Repeat("a", 500) + ".png",
			expectError:      false,
		},
		{
			name:             "simple valid filename",
			filename:         "normal-file.png",
			expectedFilename: "normal-file.png",
			expectError:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var storedFilename string
			var s3Key string

			createdFile := &repository.StorageFile{
				ID:       1,
				S3Key:    "uuid-key.png",
				S3Bucket: testImagesBucket,
				FileName: tc.expectedFilename,
				FileSize: 8,
				MimeType: "image/png",
				FileType: "portfolio-image",
			}

			mockRepo := &mockRepository{
				createFileFunc: func(_ context.Context, _, key, fileName, _ string, _ int64, _ string) (*repository.StorageFile, error) {
					storedFilename = fileName
					s3Key = key
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

			req, w, err := createMultipartRequest(tc.filename, "image/png", "portfolio-image", []byte("png data"))
			if err != nil {
				t.Fatalf("failed to create multipart request: %v", err)
			}
			router.ServeHTTP(w, req)

			if tc.expectError {
				if w.Code == http.StatusOK {
					t.Errorf("expected error for hostile filename %q, got success", tc.filename)
				}
			} else {
				if w.Code != http.StatusOK {
					t.Errorf("expected success, got status %d: %s", w.Code, w.Body.String())
					return
				}

				// Verify filename was sanitized by Go's multipart and stored
				// Skip check if expectedFilename is empty (platform-dependent behavior)
				if tc.expectedFilename != "" && storedFilename != tc.expectedFilename {
					t.Errorf("expected stored filename %q, got %q", tc.expectedFilename, storedFilename)
				}

				// S3 key should be a generated UUID, not containing path traversal
				if strings.Contains(s3Key, "..") || strings.Contains(s3Key, "/") {
					t.Errorf("S3 key should not contain path traversal components: %s", s3Key)
				}
			}
		})
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

	req, w, err := createMultipartRequest("miniature.png", "image/png", "miniature-image", []byte("png data"))
	if err != nil {
		t.Fatalf("failed to create multipart request: %v", err)
	}
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
