package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/GunarsK-portfolio/files-api/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
)

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

func TestDownloadFile_StorageGetObjectError(t *testing.T) {
	testFile := createTestFile()

	mockRepo := &mockRepository{
		getFileByKeyFunc: func(_ context.Context, _, _ string) (*repository.StorageFile, error) {
			return testFile, nil
		},
	}

	mockStore := &mockStorage{
		getObjectFunc: func(_ context.Context, _, _ string) (*minio.Object, error) {
			return nil, errors.New("storage unavailable")
		},
	}

	cfg := createTestConfig()
	handler := New(mockRepo, mockStore, cfg, &mockActionLogRepo{})

	router := setupTestRouter()
	router.GET("/api/v1/files/:fileType/*key", handler.DownloadFile)

	w := performRequest(router, http.MethodGet, "/api/v1/files/portfolio-image/"+testFileKey, nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	if !strings.Contains(w.Body.String(), "file not found in storage") {
		t.Errorf("expected 'file not found in storage' error, got %s", w.Body.String())
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
