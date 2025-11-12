package handlers

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// DownloadFile godoc
// @Summary Download file from S3
// @Description Stream file from MinIO/S3 storage
// @Tags files
// @Produce octet-stream
// @Param fileType path string true "File type: portfolio-image, miniature-image, document"
// @Param key path string true "File key/path"
// @Success 200 {file} binary
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /files/{fileType}/{key} [get]
func (h *Handler) DownloadFile(c *gin.Context) {
	fileType := c.Param("fileType")
	key := c.Param("key")
	// Remove leading slash from key (gin's *key param includes the /)
	if len(key) > 0 && key[0] == '/' {
		key = key[1:]
	}

	// Map fileType to bucket
	bucket, err := fileTypeToBucket(fileType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get file metadata from database to get original filename
	fileRecord, err := h.repo.GetFileByKey(c.Request.Context(), bucket, key)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found in database"})
		return
	}

	// Get file from S3
	object, err := h.storage.GetObject(c.Request.Context(), bucket, key)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found in storage"})
		return
	}
	defer object.Close()

	// Get object info for content type
	stat, err := object.Stat()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get file info"})
		return
	}

	// Set headers with original filename
	c.Header("Content-Type", stat.ContentType)
	c.Header("Content-Length", fmt.Sprintf("%d", stat.Size))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileRecord.FileName))

	// Stream file
	if _, err := io.Copy(c.Writer, object); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to stream file"})
		return
	}
}
