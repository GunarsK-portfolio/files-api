package routes

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GunarsK-portfolio/files-api/internal/config"
	"github.com/GunarsK-portfolio/files-api/internal/handlers"
	"github.com/GunarsK-portfolio/files-api/internal/repository"
	common "github.com/GunarsK-portfolio/portfolio-common/middleware"
	commonrepo "github.com/GunarsK-portfolio/portfolio-common/repository"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

func init() {
	gin.SetMode(gin.TestMode)
}

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
	return &repository.StorageFile{ID: 1}, nil
}

func (m *mockRepository) GetFileByID(ctx context.Context, id int64) (*repository.StorageFile, error) {
	if m.getFileByIDFunc != nil {
		return m.getFileByIDFunc(ctx, id)
	}
	return &repository.StorageFile{ID: id, S3Bucket: "test", S3Key: "test.jpg"}, nil
}

func (m *mockRepository) GetFileByKey(ctx context.Context, bucket, key string) (*repository.StorageFile, error) {
	if m.getFileByKeyFunc != nil {
		return m.getFileByKeyFunc(ctx, bucket, key)
	}
	return &repository.StorageFile{ID: 1, S3Bucket: bucket, S3Key: key}, nil
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

func (m *mockActionLogRepo) LogAction(log *commonrepo.ActionLog) error {
	return nil
}

func (m *mockActionLogRepo) GetActionsByType(actionType string, limit int) ([]commonrepo.ActionLog, error) {
	return nil, nil
}

func (m *mockActionLogRepo) GetActionsByResource(resourceType string, resourceID int64) ([]commonrepo.ActionLog, error) {
	return nil, nil
}

func (m *mockActionLogRepo) GetActionsByUser(userID int64, limit int) ([]commonrepo.ActionLog, error) {
	return nil, nil
}

func (m *mockActionLogRepo) CountActionsByResource(resourceType string, resourceID int64) (int64, error) {
	return 0, nil
}

// =============================================================================
// Test Helpers
// =============================================================================

func injectScopes(scopes map[string]string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("scopes", scopes)
		c.Next()
	}
}

func setupRouterWithScopes(t *testing.T, scopes map[string]string) *gin.Engine {
	t.Helper()

	router := gin.New()
	cfg := &config.Config{}
	handler := handlers.New(&mockRepository{}, &mockStorage{}, cfg, &mockActionLogRepo{})

	v1 := router.Group("/api/v1")
	v1.Use(injectScopes(scopes))
	{
		v1.POST("/files", common.RequirePermission(common.ResourceFiles, common.LevelEdit), handler.UploadFile)
		v1.DELETE("/files/:id", common.RequirePermission(common.ResourceFiles, common.LevelDelete), handler.DeleteFile)
	}

	return router
}

func performRequest(t *testing.T, router *gin.Engine, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// =============================================================================
// Route Permission Definitions
// =============================================================================

type routePermission struct {
	method   string
	path     string
	resource string
	level    string
}

var protectedRoutes = []routePermission{
	{"POST", "/api/v1/files", common.ResourceFiles, common.LevelEdit},
	{"DELETE", "/api/v1/files/1", common.ResourceFiles, common.LevelDelete},
}

// =============================================================================
// Route Permission Tests
// =============================================================================

func TestProtectedRoutes_Forbidden_WithoutPermission(t *testing.T) {
	for _, route := range protectedRoutes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			router := setupRouterWithScopes(t, map[string]string{})
			w := performRequest(t, router, route.method, route.path)

			if w.Code != http.StatusForbidden {
				t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if response["error"] != "insufficient permissions" {
				t.Errorf("error = %v, want 'insufficient permissions'", response["error"])
			}
			if response["resource"] != route.resource {
				t.Errorf("resource = %v, want %q", response["resource"], route.resource)
			}
			if response["required"] != route.level {
				t.Errorf("required = %v, want %q", response["required"], route.level)
			}
		})
	}
}

func TestProtectedRoutes_Allowed_WithPermission(t *testing.T) {
	for _, route := range protectedRoutes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			scopes := map[string]string{route.resource: route.level}
			router := setupRouterWithScopes(t, scopes)
			w := performRequest(t, router, route.method, route.path)

			// We only verify authorization passes (not 403/401).
			// Handler may return 400/404/500 due to missing body or mock defaults.
			if w.Code == http.StatusForbidden {
				t.Errorf("got 403 Forbidden with permission %s:%s", route.resource, route.level)
			}
			if w.Code == http.StatusUnauthorized {
				t.Errorf("got 401 Unauthorized - scopes not injected")
			}
		})
	}
}

// =============================================================================
// Permission Hierarchy Tests
// =============================================================================

func TestPermissionHierarchy(t *testing.T) {
	tests := []struct {
		name       string
		granted    string
		required   string
		method     string
		path       string
		wantAccess bool
	}{
		{"delete grants delete", common.LevelDelete, common.LevelDelete, "DELETE", "/api/v1/files/1", true},
		{"delete grants edit", common.LevelDelete, common.LevelEdit, "POST", "/api/v1/files", true},
		{"edit grants edit", common.LevelEdit, common.LevelEdit, "POST", "/api/v1/files", true},
		{"edit denies delete", common.LevelEdit, common.LevelDelete, "DELETE", "/api/v1/files/1", false},
		{"read denies edit", common.LevelRead, common.LevelEdit, "POST", "/api/v1/files", false},
		{"read denies delete", common.LevelRead, common.LevelDelete, "DELETE", "/api/v1/files/1", false},
		{"none denies edit", common.LevelNone, common.LevelEdit, "POST", "/api/v1/files", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scopes := map[string]string{common.ResourceFiles: tt.granted}
			router := setupRouterWithScopes(t, scopes)
			w := performRequest(t, router, tt.method, tt.path)

			gotAccess := w.Code != http.StatusForbidden
			if gotAccess != tt.wantAccess {
				t.Errorf("granted=%s required=%s: gotAccess=%v wantAccess=%v (status=%d)",
					tt.granted, tt.required, gotAccess, tt.wantAccess, w.Code)
			}
		})
	}
}

// =============================================================================
// Middleware Error Handling Tests
// =============================================================================

func TestRoutes_NoScopes_Unauthorized(t *testing.T) {
	router := gin.New()
	cfg := &config.Config{}
	handler := handlers.New(&mockRepository{}, &mockStorage{}, cfg, &mockActionLogRepo{})

	// Route without scope injection middleware
	router.DELETE("/api/v1/files/:id",
		common.RequirePermission(common.ResourceFiles, common.LevelDelete),
		handler.DeleteFile,
	)

	req, _ := http.NewRequest("DELETE", "/api/v1/files/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d (no scopes = unauthorized)", w.Code, http.StatusUnauthorized)
	}
}

func TestRoutes_InvalidScopesFormat_InternalError(t *testing.T) {
	router := gin.New()
	cfg := &config.Config{}
	handler := handlers.New(&mockRepository{}, &mockStorage{}, cfg, &mockActionLogRepo{})

	// Inject invalid scopes format
	router.Use(func(c *gin.Context) {
		c.Set("scopes", "invalid-format")
		c.Next()
	})

	router.DELETE("/api/v1/files/:id",
		common.RequirePermission(common.ResourceFiles, common.LevelDelete),
		handler.DeleteFile,
	)

	req, _ := http.NewRequest("DELETE", "/api/v1/files/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d (invalid scopes = internal error)", w.Code, http.StatusInternalServerError)
	}
}
