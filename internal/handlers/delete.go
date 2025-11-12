package handlers

import (
	"net/http"
	"strconv"

	commonHandlers "github.com/GunarsK-portfolio/portfolio-common/handlers"
	"github.com/gin-gonic/gin"
)

// DeleteFile godoc
// @Summary Delete file from S3 and database
// @Description Delete file by ID from both S3 storage and database
// @Tags files
// @Produce json
// @Param id path int true "File ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /files/{id} [delete]
func (h *Handler) DeleteFile(c *gin.Context) {
	// Get file ID from path
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		commonHandlers.RespondError(c, http.StatusBadRequest, "invalid file ID")
		return
	}

	// Get file from database to get S3 details
	file, err := h.repo.GetFileByID(c.Request.Context(), id)
	if err != nil {
		commonHandlers.HandleRepositoryError(c, err, "file not found", "failed to fetch file")
		return
	}

	// Map fileType to bucket
	bucket, err := fileTypeToBucket(file.FileType)
	if err != nil {
		commonHandlers.LogAndRespondError(c, http.StatusInternalServerError, err, "invalid file type in database")
		return
	}

	// Delete from S3
	if err := h.storage.DeleteObject(c.Request.Context(), bucket, file.S3Key); err != nil {
		commonHandlers.LogAndRespondError(c, http.StatusInternalServerError, err, "failed to delete file from storage")
		return
	}

	// Delete from database
	if err := h.repo.DeleteFile(c.Request.Context(), id); err != nil {
		commonHandlers.LogAndRespondError(c, http.StatusInternalServerError, err, "failed to delete file record")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "file deleted successfully"})
}
