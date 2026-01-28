package server

import (
	"mime"
	"os"
	"path/filepath"
)

// FileExists checks if a file exists at the given path
func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

// readFileContent reads entire file content
func readFileContent(filePath string) ([]byte, bool) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, false
	}
	return content, true
}

// getContentType determines MIME type from file extension
func getContentType(filePath string) string {
	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return contentType
}
