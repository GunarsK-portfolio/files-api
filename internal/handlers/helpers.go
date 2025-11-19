package handlers

import "fmt"

// fileTypeToBucket maps fileType to S3 bucket name using configuration
func (h *Handler) fileTypeToBucket(fileType string) (string, error) {
	switch fileType {
	case "portfolio-image":
		return h.cfg.ImagesBucket, nil
	case "miniature-image":
		return h.cfg.MiniaturesBucket, nil
	case "document":
		return h.cfg.DocumentsBucket, nil
	default:
		return "", fmt.Errorf("invalid fileType: must be portfolio-image, miniature-image, or document")
	}
}
