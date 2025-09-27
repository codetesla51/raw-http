// Server package comment block:
/*
HTTP Server from Raw TCP Sockets

Author: Uthman Dev
GitHub: https://github.com/codetesla51
Repository: https://github.com/codetesla51/raw-http

This package implements a basic HTTP server built directly on top of TCP sockets
without using Go's net/http package. It demonstrates fundamental HTTP protocol
handling including request parsing, routing, and response generation.

Features:
- Raw TCP connection handling with keep-alive support
- Custom HTTP request/response parsing
- Simple routing system with method and path matching
- Static file serving with MIME type detection
- Form data and JSON body parsing
- Basic security protections (path traversal, request limits)

Limitations:
- No HTTPS/TLS support
- Basic error handling and recovery
- Simple routing (no path parameters or wildcards)
- Limited HTTP method and header support
- Not suitable for production use

This is primarily an educational project to understand HTTP internals
and network programming fundamentals in Go.
*/

// ========================================================================

package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fatih/color"
	"log"
	"mime"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Request struct {
	Query   map[string]string
	Body    map[string]string
	Browser string
	Method  string
	Path    string
}
type RouteHandler func(req *Request) (response, status string)

// Router manages route registration and handling
type Router struct {
	routes map[string]map[string]RouteHandler
	// method -> path -> handler
}

// NewRouter creates a new router instance
func NewRouter() *Router {
	return &Router{
		routes: make(map[string]map[string]RouteHandler),
	}
}

func init() {
	// Basic web MIME types (most important)
	mime.AddExtensionType(".html", "text/html")
	mime.AddExtensionType(".htm", "text/html")
	mime.AddExtensionType(".css", "text/css")
	mime.AddExtensionType(".js", "application/javascript")
	mime.AddExtensionType(".json", "application/json")
	mime.AddExtensionType(".txt", "text/plain")
	mime.AddExtensionType(".xml", "application/xml")
	mime.AddExtensionType(".csv", "text/csv")

	// Images
	mime.AddExtensionType(".jpg", "image/jpeg")
	mime.AddExtensionType(".jpeg", "image/jpeg")
	mime.AddExtensionType(".png", "image/png")
	mime.AddExtensionType(".gif", "image/gif")
	mime.AddExtensionType(".svg", "image/svg+xml")
	mime.AddExtensionType(".webp", "image/webp")
	mime.AddExtensionType(".ico", "image/x-icon")
	mime.AddExtensionType(".bmp", "image/bmp")

	// Video
	mime.AddExtensionType(".mp4", "video/mp4")
	mime.AddExtensionType(".webm", "video/webm")
	mime.AddExtensionType(".avi", "video/x-msvideo")
	mime.AddExtensionType(".mov", "video/quicktime")
	mime.AddExtensionType(".wmv", "video/x-ms-wmv")

	// Audio
	mime.AddExtensionType(".mp3", "audio/mpeg")
	mime.AddExtensionType(".wav", "audio/wav")
	mime.AddExtensionType(".ogg", "audio/ogg")
	mime.AddExtensionType(".m4a", "audio/mp4")
	mime.AddExtensionType(".flac", "audio/flac")

	// Fonts
	mime.AddExtensionType(".woff", "font/woff")
	mime.AddExtensionType(".woff2", "font/woff2")
	mime.AddExtensionType(".ttf", "font/ttf")
	mime.AddExtensionType(".otf", "font/otf")
	mime.AddExtensionType(".eot", "application/vnd.ms-fontobject")

	// Documents
	mime.AddExtensionType(".pdf", "application/pdf")
	mime.AddExtensionType(".doc", "application/msword")
	mime.AddExtensionType(".docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	mime.AddExtensionType(".xls", "application/vnd.ms-excel")
	mime.AddExtensionType(".xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

	// Archives
	mime.AddExtensionType(".zip", "application/zip")
	mime.AddExtensionType(".tar", "application/x-tar")
	mime.AddExtensionType(".gz", "application/gzip")
	mime.AddExtensionType(".rar", "application/vnd.rar")
	mime.AddExtensionType(".7z", "application/x-7z-compressed")
}

func readHTTPRequest(conn net.Conn) (string, error) {
	var headerBuffer []byte
	maxHeaderSize := 8192
	timeout := time.Now().Add(10 * time.Second)
	// Read request in small chunks until we find end of headers
	for {
		conn.SetReadDeadline(timeout)
		if len(headerBuffer) > maxHeaderSize {
			return "", errors.New("headers too large")
		}
		chunk := make([]byte, 256)
		n, err := conn.Read(chunk)
		if err != nil {
			return "", err
		}
		headerBuffer = append(headerBuffer, chunk[:n]...)

		// Check if we have complete headers (marked by \r\n\r\n)
		if strings.Contains(string(headerBuffer), "\r\n\r\n") {
			break
		}
	}

	return string(headerBuffer), nil

}

func (r *Router) RunConnection(conn net.Conn) {
	defer conn.Close()

	for {
		// === READ HTTP REQUEST USING CHUNKED READING ===

		request, err := readHTTPRequest(conn)
		if err != nil {
			log.Println("Error reading request:", err)
			return
		}

		// === SPLIT HEADERS AND BODY ===
		requestParts := strings.SplitN(request, "\r\n\r\n", 2)
		headerSection := requestParts[0]
		body := ""
		var bodyMap map[string]string
		var headerMap map[string]string
		// === PARSE HEADERS ===
		lines := strings.Split(headerSection, "\r\n")
		if len(lines) == 0 {
			log.Println("Invalid request")
			return
		}
		firstLine := lines[0]
		headerLines := lines[1:]
		headerMap = parseHeaders(headerLines)
		// Parse body if present
		if len(requestParts) > 1 {
			body = requestParts[1]
			if len(requestParts) > 1 {
				body = requestParts[1]
				contentType := headerMap["Content-Type"]
				if strings.Contains(contentType, "application/json") {

					bodyMap = parseJSONBody(body)

				} else {

					bodyMap = parseKeyValuePairs(body)
				}
			}
		}
		// === HANDLE CONTENT-LENGTH (READ REMAINING BODY IF NEEDED) ===
		contentLengthStr := headerMap["Content-Length"]
		if contentLengthStr != "" {
			contentLength, err := strconv.Atoi(contentLengthStr)
			if err == nil && len(body) < contentLength {
				remainingBytes := contentLength - len(body)

				// Create buffer for remaining data
				remainingBuffer := make([]byte, remainingBytes)

				// Read the exact amount needed
				_, err := conn.Read(remainingBuffer)
				if err != nil {
					log.Println("Error reading remaining body:", err)
					return
				}

				// Append to existing body
				body += string(remainingBuffer)
			}
		}

		// === DETECT BROWSER FROM USER-AGENT ===
		var browserName string
		browserName = detectBrowser(headerMap["User-Agent"])

		// === PARSE REQUEST LINE ===
		method, path, err := parseRequestLine(firstLine)
		if err != nil {
			return
		}

		// === PARSE PATH AND QUERY PARAMETERS ===
		fullPath := strings.Split(path, "?")
		cleanPath := fullPath[0] // Path without query params
		var queryPath string
		var queryMap map[string]string

		if len(fullPath) > 1 {
			queryPath = fullPath[1]
			if queryPath != "" {
				// Parse query parameters (key=value&key2=value2)
				queryMap = parseKeyValuePairs(queryPath)
			}
		}

		// === ROUTING AND RESPONSE GENERATION ===
		var response, status string
		var filePath string

		// Determine file path for static files
		rawPath := cleanPath
		cleanedRawPath := filepath.Clean(rawPath)
		if strings.Contains(cleanedRawPath, "..") {
			response, status = CreateResponse("403", "text/plain", "Forbidden", "Access denied")
			goto sendResponse
		} else {
			// Safe to add pages prefix
			if cleanPath == "/" {
				filePath = "pages/index.html"
			} else {
				filePath = "pages" + cleanPath
			}
		}

		if fileExists(filePath) {
			content, success := readFileContent(filePath)
			if success {
				contentType := getContentType(filePath)
				response, status = CreateResponse("200", contentType, "OK", string(content))
			} else {
				response, status = serve404()
			}
		} else {
			response, status = r.Handle(method, cleanPath, queryMap, bodyMap, browserName)
		}

		// === SEND RESPONSE ===

	sendResponse:
		switch status {
		case "200":
			log.Print(color.GreenString("%s %s %s", method, path, status))
		case "404", "403", "405":
			log.Print(color.RedString("%s %s %s", method, path, status))
		default:
			log.Printf("%s %s %s", method, path, status)
		}
		_, err = conn.Write([]byte(response))
		if err != nil {
			log.Println("Error writing response:", err)
			return
		}
		if headerMap["Connection"] == "close" {
			break
		}

	}
}

// CreateResponse builds a complete HTTP response with proper headers
func CreateResponse(statusCode, contentType, statusMessage, body string) (string, string) {
	return "HTTP/1.1 " + statusCode + " " + statusMessage +
		"\r\nContent-Type: " + contentType +
		"\r\nConnection: keep-alive" +
		"\r\nContent-Length: " + strconv.Itoa(len(body)) +
		"\r\n\r\n" + body, statusCode
}

// fileExists checks if a file exists at the given path
func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	log.Printf("Error checking file %s: %v\n", filePath, err)
	return false
}

// getFormValue retrieves a value from form data with a default fallback
func getFormValue(bodyMap map[string]string, key string, defaultValue string) string {

	if value, exists := bodyMap[key]; exists {
		return value
	}
	return defaultValue
}

// serve404 returns a 404 Not Found response, serving custom 404 page if available
func serve404() (string, string) {
	cleanedPath := filepath.Clean("pages/404.html")
	content, success := readFileContent(cleanedPath)
	if !success {
		log.Printf("Error reading 404.html")
		return CreateResponse("404", "text/plain", "Not Found", "Route Not Found")
	}
	response, status := CreateResponse("404", "text/html", "Not Found", string(content))
	return response, status
}

func detectBrowser(userAgent string) string {
	if strings.Contains(userAgent, "Chrome") {
		return "Chrome"
	} else if strings.Contains(userAgent, "Firefox") {
		return "Firefox"
	} else if strings.Contains(userAgent, "Safari") {
		return "Safari"
	} else {
		return "Unknown Browser"
	}
}

func readFileContent(filePath string) ([]byte, bool) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Error reading file %s: %v", filePath, err)
		return nil, false
	}
	return content, true
}

func safeURLDecode(encoded string) string {
	decoded, err := url.QueryUnescape(encoded)
	if err != nil {
		log.Printf("URL decode failed for: %s", encoded)
		return encoded
	}
	return decoded
}

func parseKeyValuePairs(data string) map[string]string {
	resultMap := make(map[string]string)
	pairs := strings.Split(data, "&")
	for _, pair := range pairs {
		parts := strings.Split(pair, "=")

		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]
			decodedKey := safeURLDecode(key)
			decodedValue := safeURLDecode(value)

			resultMap[decodedKey] = decodedValue
		}
	}
	return resultMap
}

func parseRequestLine(firstLine string) (method, path string, err error) {
	parts := strings.Split(firstLine, " ")
	if len(parts) < 3 {
		return "", "", errors.New("invalid request line")
	}
	return parts[0], parts[1], nil
}

func parseHeaders(headerLines []string) map[string]string {
	headerMap := make(map[string]string)
	for _, headerLine := range headerLines {
		parts := strings.SplitN(headerLine, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			headerMap[key] = value
		}
	}
	return headerMap
}

func (r *Router) Register(method, path string, handler RouteHandler) {
	if r.routes[method] == nil {
		r.routes[method] = make(map[string]RouteHandler)
	}
	r.routes[method][path] = handler
}

// Handle processes a request through the registered routes
func (r *Router) Handle(method, cleanPath string, queryMap, bodyMap map[string]string, browserName string) (response, status string) {
	if methodRoutes, exists := r.routes[method]; exists {
		if handler, exists := methodRoutes[cleanPath]; exists {
			req := &Request{
				Query:   queryMap,
				Body:    bodyMap,
				Browser: browserName,
				Method:  method,
				Path:    cleanPath,
			}
			return handler(req)
		}
	}
	response, status = serve404()
	return response, status
}

func getContentType(filePath string) string {
	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return contentType
}
func parseJSONBody(body string) map[string]string {
	var jsonData map[string]any
	result := make(map[string]string)

	if err := json.Unmarshal([]byte(body), &jsonData); err != nil {
		log.Printf("JSON parse error: %v", err)
		return result
	}

	for key, value := range jsonData {
		result[key] = fmt.Sprintf("%v", value)
	}

	return result
}
