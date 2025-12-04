package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/GunarsK-portfolio/files-api/internal/repository"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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

func TestDeleteFile_S3DeleteError(t *testing.T) {
	testFile := createTestFile()

	mockRepo := &mockRepository{
		getFileByIDFunc: func(_ context.Context, _ int64) (*repository.StorageFile, error) {
			return testFile, nil
		},
	}

	mockStore := &mockStorage{
		deleteObjectFunc: func(_ context.Context, _, _ string) error {
			return errors.New("S3 delete failed")
		},
	}

	cfg := createTestConfig()
	handler := New(mockRepo, mockStore, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.DELETE("/api/v1/files/:id", handler.DeleteFile)

	w := performRequest(router, http.MethodDelete, "/api/v1/files/1", nil)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	if !strings.Contains(w.Body.String(), "failed to delete file from storage") {
		t.Errorf("expected 'failed to delete file from storage' error, got %s", w.Body.String())
	}
}

func TestDeleteFile_DBDeleteError(t *testing.T) {
	testFile := createTestFile()
	var s3DeleteCalled bool

	mockRepo := &mockRepository{
		getFileByIDFunc: func(_ context.Context, _ int64) (*repository.StorageFile, error) {
			return testFile, nil
		},
		deleteFileFunc: func(_ context.Context, _ int64) error {
			return errors.New("database delete failed")
		},
	}

	mockStore := &mockStorage{
		deleteObjectFunc: func(_ context.Context, _, _ string) error {
			s3DeleteCalled = true
			return nil
		},
	}

	cfg := createTestConfig()
	handler := New(mockRepo, mockStore, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.DELETE("/api/v1/files/:id", handler.DeleteFile)

	w := performRequest(router, http.MethodDelete, "/api/v1/files/1", nil)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	// Verify S3 was called first
	if !s3DeleteCalled {
		t.Error("expected S3 delete to be called")
	}

	if !strings.Contains(w.Body.String(), "failed to delete file record") {
		t.Errorf("expected 'failed to delete file record' error, got %s", w.Body.String())
	}
}

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
