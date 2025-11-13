package handlers

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/GunarsK-portfolio/portfolio-common/audit"
	commonHandlers "github.com/GunarsK-portfolio/portfolio-common/handlers"
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
		commonHandlers.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Get file metadata from database to get original filename
	fileRecord, err := h.repo.GetFileByKey(c.Request.Context(), bucket, key)
	if err != nil {
		commonHandlers.HandleRepositoryError(c, err, "file not found in database", "failed to fetch file record")
		return
	}

	// Get file from S3
	object, err := h.storage.GetObject(c.Request.Context(), bucket, key)
	if err != nil {
		commonHandlers.LogAndRespondError(c, http.StatusNotFound, err, "file not found in storage")
		return
	}
	defer object.Close()

	// Get object info for content type
	stat, err := object.Stat()
	if err != nil {
		commonHandlers.LogAndRespondError(c, http.StatusInternalServerError, err, "failed to get file info")
		return
	}

	// Set headers with original filename
	c.Header("Content-Type", stat.ContentType)
	c.Header("Content-Length", fmt.Sprintf("%d", stat.Size))
	// Use RFC 5987 encoding for filename to prevent header injection and support non-ASCII characters
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(fileRecord.FileName)))

	// Log file download with source tracking
	resourceType := audit.ResourceTypeFile
	source := c.Query("source") // "admin-web", "public-web", or empty
	var sourcePtr *string
	if source != "" {
		sourcePtr = &source
	}
	_ = audit.LogFromContext(c, h.actionLogRepo, audit.ActionFileDownload, &resourceType, &fileRecord.ID, sourcePtr, map[string]interface{}{
		"filename":  fileRecord.FileName,
		"file_type": fileType,
		"size":      fileRecord.FileSize,
		"mime_type": fileRecord.MimeType,
	})

	// Stream file
	if _, err := io.Copy(c.Writer, object); err != nil {
		commonHandlers.LogAndRespondError(c, http.StatusInternalServerError, err, "failed to stream file")
		return
	}
}
