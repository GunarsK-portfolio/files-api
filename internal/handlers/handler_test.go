package handlers

import (
	"strings"
	"testing"
)

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
