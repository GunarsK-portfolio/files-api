package handlers

import "fmt"

// fileTypeToBucket maps fileType to S3 bucket name
func fileTypeToBucket(fileType string) (string, error) {
	switch fileType {
	case "portfolio-image":
		return "images", nil
	case "miniature-image":
		return "miniatures", nil
	case "document":
		return "documents", nil
	default:
		return "", fmt.Errorf("invalid fileType: must be portfolio-image, miniature-image, or document")
	}
}
