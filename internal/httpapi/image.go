package httpapi

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
)

// detectImageContentType sniffs the image MIME type from data bytes, falling
// back to the file extension when DetectContentType returns the generic
// application/octet-stream. Returns an error if the type cannot be resolved
// to a format the vision providers accept.
func detectImageContentType(data []byte, filename string) (string, error) {
	ct := http.DetectContentType(data)
	if isSupportedImageType(ct) {
		return ct, nil
	}

	switch strings.ToLower(filepath.Ext(filename)) {
	case ".jpg", ".jpeg":
		return "image/jpeg", nil
	case ".png":
		return "image/png", nil
	case ".gif":
		return "image/gif", nil
	case ".webp":
		return "image/webp", nil
	}

	return "", fmt.Errorf("unsupported image type (detected %q from bytes, filename %q): expected jpeg, png, gif, or webp", ct, filename)
}

func isSupportedImageType(ct string) bool {
	switch ct {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return true
	}
	return false
}
