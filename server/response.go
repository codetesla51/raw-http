package server

import (
	"bytes"
	"path/filepath"
	"strconv"
)

// CreateResponseBytes builds an HTTP response as bytes
func CreateResponseBytes(statusCode, contentType, statusMessage string, body []byte) ([]byte, string) {
	buf := responseBufferPool.Get().(*bytes.Buffer)
	buf.Reset()

	defer func() {
		if buf.Cap() <= maxPoolBufferSize {
			responseBufferPool.Put(buf)
		}
	}()

	buf.WriteString("HTTP/1.1 ")
	buf.WriteString(statusCode)
	buf.WriteString(" ")
	buf.WriteString(statusMessage)
	buf.WriteString("\r\nContent-Type: ")
	buf.WriteString(contentType)
	buf.WriteString("\r\nConnection: keep-alive")
	buf.WriteString("\r\nContent-Length: ")
	buf.WriteString(strconv.Itoa(len(body)))
	buf.WriteString("\r\n\r\n")
	buf.Write(body)

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, statusCode
}

// CreateResponse builds an HTTP response as string (for compatibility)
func CreateResponse(statusCode, contentType, statusMessage, body string) (string, string) {
	responseBytes, status := CreateResponseBytes(statusCode, contentType, statusMessage, []byte(body))
	return string(responseBytes), status
}

// serve404Bytes returns a 404 response, using custom page if available
func serve404Bytes() ([]byte, string) {
	cleanedPath := filepath.Clean("pages/404.html")
	content, success := readFileContent(cleanedPath)
	if !success {
		return CreateResponseBytes("404", "text/plain", "Not Found", []byte("Route Not Found"))
	}
	return CreateResponseBytes("404", "text/html", "Not Found", content)
}
