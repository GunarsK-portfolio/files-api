package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UploadFile godoc
// @Summary Upload file to S3
// @Description Upload file to MinIO/S3 and create database record
// @Tags files
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "File to upload"
// @Param fileType formData string true "File type: portfolio-image, miniature-image, document"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /files [post]
func (h *Handler) UploadFile(c *gin.Context) {
	// Get file from form
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	// Get required fileType parameter
	fileType := c.PostForm("fileType")
	if fileType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fileType is required (portfolio-image, miniature-image, document)"})
		return
	}

	// Validate file size
	if file.Size > h.cfg.MaxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("file too large (max %d bytes)", h.cfg.MaxFileSize)})
		return
	}

	// Validate file type
	contentType := file.Header.Get("Content-Type")
	if !h.isAllowedContentType(contentType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file type"})
		return
	}

	// Generate unique key
	ext := filepath.Ext(file.Filename)
	key := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	// Open file
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
		return
	}
	defer src.Close()

	// Determine bucket based on file type
	bucket, err := h.getBucketForFileType(fileType, contentType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Upload to S3
	if err := h.storage.PutObject(c.Request.Context(), bucket, key, src, file.Size, contentType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload file"})
		return
	}

	// Create database record
	fileRecord, err := h.repo.CreateFile(c.Request.Context(), bucket, key, file.Filename, fileType, file.Size, contentType)
	if err != nil {
		// Cleanup S3 file if DB insert fails
		_ = h.storage.DeleteObject(c.Request.Context(), bucket, key)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create file record"})
		return
	}

	// Return file info
	c.JSON(http.StatusOK, gin.H{
		"id":       fileRecord.ID,
		"fileName": fileRecord.FileName,
		"fileSize": fileRecord.FileSize,
		"mimeType": fileRecord.MimeType,
		"url":      fmt.Sprintf("/api/v1/files/%s/%s", fileType, key),
		"fileType": fileType,
	})
}

func (h *Handler) isAllowedContentType(contentType string) bool {
	for _, ct := range h.cfg.AllowedFileTypes {
		if strings.HasPrefix(contentType, ct) {
			return true
		}
	}
	return false
}

func (h *Handler) getBucketForFileType(fileType, contentType string) (string, error) {
	// Get bucket for fileType
	bucket, err := fileTypeToBucket(fileType)
	if err != nil {
		return "", err
	}

	// Validate content type matches fileType
	switch fileType {
	case "portfolio-image", "miniature-image":
		if !strings.HasPrefix(contentType, "image/") {
			return "", fmt.Errorf("%s requires image content type", fileType)
		}
	case "document":
		isPDF := strings.HasPrefix(contentType, "application/pdf")
		isWord := strings.HasPrefix(contentType, "application/msword") ||
			strings.HasPrefix(contentType, "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		if !isPDF && !isWord {
			return "", fmt.Errorf("document requires PDF or Word document content type")
		}
	}

	return bucket, nil
}
