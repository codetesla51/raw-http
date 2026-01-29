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
func Serve400(msg string) ([]byte, string) {
	if msg == "" {
		msg = "Bad Request"
	}
	return CreateResponseBytes("400", "text/plain", "Bad Request", []byte(msg))
}

// 401 Unauthorized - client authentication required
func Serve401(msg string) ([]byte, string) {
	if msg == "" {
		msg = "Authentication required"
	}
	return CreateResponseBytes("401", "text/plain", "Unauthorized", []byte(msg))
}

// 403 Forbidden - authenticated but access denied
func Serve403(msg string) ([]byte, string) {
	if msg == "" {
		msg = "Access denied"
	}
	return CreateResponseBytes("403", "text/plain", "Forbidden", []byte(msg))
}

// 405 Method Not Allowed - wrong HTTP method
func Serve405(method, path string) ([]byte, string) {
	msg := "Method " + method + " not allowed for " + path
	return CreateResponseBytes("405", "text/plain", "Method Not Allowed", []byte(msg))
}

// 429 Too Many Requests - rate limit exceeded
func Serve429(msg string) ([]byte, string) {
	if msg == "" {
		msg = "Rate limit exceeded"
	}
	return CreateResponseBytes("429", "text/plain", "Too Many Requests", []byte(msg))
}

// 500 Internal Server Error
func Serve500(msg string) ([]byte, string) {
	if msg == "" {
		msg = "Internal server error"
	}
	return CreateResponseBytes("500", "text/plain", "Internal Server Error", []byte(msg))
}

// 502 Bad Gateway
func Serve502(msg string) ([]byte, string) {
	if msg == "" {
		msg = "Bad gateway"
	}
	return CreateResponseBytes("502", "text/plain", "Bad Gateway", []byte(msg))
}

// 503 Service Unavailable
func Serve503(msg string) ([]byte, string) {
	if msg == "" {
		msg = "Service unavailable"
	}
	return CreateResponseBytes("503", "text/plain", "Service Unavailable", []byte(msg))
}

// 201 Created - resource successfully created
func Serve201(body string) ([]byte, string) {
	if body == "" {
		body = "Resource created"
	}
	return CreateResponseBytes("201", "text/plain", "Created", []byte(body))
}

// 204 No Content - successful but no content to return
func Serve204() ([]byte, string) {
	return CreateResponseBytes("204", "text/plain", "No Content", []byte(""))
}

// 301 Moved Permanently - use for redirects (note: requires Location header in real use)
func Serve301(url string) ([]byte, string) {
	msg := "Moved to " + url
	return CreateResponseBytes("301", "text/plain", "Moved Permanently", []byte(msg))
}

// 302 Found - temporary redirect
func Serve302(url string) ([]byte, string) {
	msg := "Found at " + url
	return CreateResponseBytes("302", "text/plain", "Found", []byte(msg))
}
