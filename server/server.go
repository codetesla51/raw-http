/*
HTTP/HTTPS Server from Raw TCP Sockets

Author: Uthman Dev
GitHub: https://github.com/codetesla51
Repository: https://github.com/codetesla51/raw-http

This package implements a basic HTTP/HTTPS server built directly on top of TCP sockets
without using Go's net/http package. It demonstrates fundamental HTTP protocol
handling including request parsing, routing, and response generation.

Features:
- Raw TCP connection handling with keep-alive support
- Custom HTTP request/response parsing
- Simple routing system with method and path matching
- Static file serving with MIME type detection
- Form data and JSON body parsing
- Basic security protections (path traversal, request limits)
- HTTPS/TLS support with certificate-based encryption
- Concurrent connection handling with goroutines
- Memory-optimized buffer pooling for high performance
- Graceful error handling and recovery

Limitations:
- Basic error handling and recovery
- Simple routing (no path parameters or wildcards)
- Limited HTTP method and header support
- Not suitable for production use without enhancements

This is primarily an educational project to understand HTTP internals,
TLS encryption, and network programming fundamentals in Go.
*/

package server

import (
	"bytes"
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
	"sync"
	"time"
)

var requestBufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 8192)
		return buf
	},
}

var chunkBufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 256)
		return buf
	},
}

var responseBufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

type Request struct {
	Query   map[string]string
	Body    map[string]string
	Browser string
	Method  string
	Path    string
}

type RouteHandler func(req *Request) (response, status string)

type Router struct {
	routes map[string]map[string]RouteHandler
}

func NewRouter() *Router {
	return &Router{
		routes: make(map[string]map[string]RouteHandler),
	}
}

func init() {
	mime.AddExtensionType(".html", "text/html")
	mime.AddExtensionType(".htm", "text/html")
	mime.AddExtensionType(".css", "text/css")
	mime.AddExtensionType(".js", "application/javascript")
	mime.AddExtensionType(".json", "application/json")
	mime.AddExtensionType(".txt", "text/plain")
	mime.AddExtensionType(".xml", "application/xml")
	mime.AddExtensionType(".csv", "text/csv")

	mime.AddExtensionType(".jpg", "image/jpeg")
	mime.AddExtensionType(".jpeg", "image/jpeg")
	mime.AddExtensionType(".png", "image/png")
	mime.AddExtensionType(".gif", "image/gif")
	mime.AddExtensionType(".svg", "image/svg+xml")
	mime.AddExtensionType(".webp", "image/webp")
	mime.AddExtensionType(".ico", "image/x-icon")
	mime.AddExtensionType(".bmp", "image/bmp")

	mime.AddExtensionType(".mp4", "video/mp4")
	mime.AddExtensionType(".webm", "video/webm")
	mime.AddExtensionType(".avi", "video/x-msvideo")
	mime.AddExtensionType(".mov", "video/quicktime")
	mime.AddExtensionType(".wmv", "video/x-ms-wmv")

	mime.AddExtensionType(".mp3", "audio/mpeg")
	mime.AddExtensionType(".wav", "audio/wav")
	mime.AddExtensionType(".ogg", "audio/ogg")
	mime.AddExtensionType(".m4a", "audio/mp4")
	mime.AddExtensionType(".flac", "audio/flac")

	mime.AddExtensionType(".woff", "font/woff")
	mime.AddExtensionType(".woff2", "font/woff2")
	mime.AddExtensionType(".ttf", "font/ttf")
	mime.AddExtensionType(".otf", "font/otf")
	mime.AddExtensionType(".eot", "application/vnd.ms-fontobject")

	mime.AddExtensionType(".pdf", "application/pdf")
	mime.AddExtensionType(".doc", "application/msword")
	mime.AddExtensionType(".docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	mime.AddExtensionType(".xls", "application/vnd.ms-excel")
	mime.AddExtensionType(".xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

	mime.AddExtensionType(".zip", "application/zip")
	mime.AddExtensionType(".tar", "application/x-tar")
	mime.AddExtensionType(".gz", "application/gzip")
	mime.AddExtensionType(".rar", "application/vnd.rar")
	mime.AddExtensionType(".7z", "application/x-7z-compressed")
}

func readHTTPRequest(conn net.Conn) (string, error) {
	bufPtr := requestBufferPool.Get().(*[]byte)
	headerBuffer := (*bufPtr)[:0]
	
	defer func() {
		if cap(headerBuffer) <= 16384 {
			requestBufferPool.Put(bufPtr)
		}
	}()

	maxHeaderSize := 8192
	timeout := time.Now().Add(10 * time.Second)

	for {
		conn.SetReadDeadline(timeout)
		if len(headerBuffer) > maxHeaderSize {
			return "", errors.New("headers too large")
		}
		
		chunkPtr := chunkBufferPool.Get().(*[]byte)
		chunk := *chunkPtr
		
		n, err := conn.Read(chunk)
		if err != nil {
			chunkBufferPool.Put(chunkPtr)
			return "", err
		}
		
		headerBuffer = append(headerBuffer, chunk[:n]...)
		chunkBufferPool.Put(chunkPtr)

		if strings.Contains(string(headerBuffer), "\r\n\r\n") {
			break
		}
	}

	return string(headerBuffer), nil
}

func (r *Router) RunConnection(conn net.Conn) {
	defer conn.Close()

	for {
		request, err := readHTTPRequest(conn)
		if err != nil {
			return
		}

		requestParts := strings.SplitN(request, "\r\n\r\n", 2)
		headerSection := requestParts[0]
		body := ""
		var bodyMap map[string]string
		var headerMap map[string]string

		lines := strings.Split(headerSection, "\r\n")
		if len(lines) == 0 {
			log.Println("Invalid request")
			return
		}
		firstLine := lines[0]
		headerLines := lines[1:]
		headerMap = parseHeaders(headerLines)

		if len(requestParts) > 1 {
			body = requestParts[1]
			contentType := headerMap["Content-Type"]
			if strings.Contains(contentType, "application/json") {
				bodyMap = parseJSONBody(body)
			} else {
				bodyMap = parseKeyValuePairs(body)
			}
		}

		contentLengthStr := headerMap["Content-Length"]
		if contentLengthStr != "" {
			contentLength, err := strconv.Atoi(contentLengthStr)
			if err == nil && len(body) < contentLength {
				remainingBytes := contentLength - len(body)
				remainingBuffer := make([]byte, remainingBytes)
				_, err := conn.Read(remainingBuffer)
				if err != nil {
					log.Println("Error reading remaining body:", err)
					return
				}
				body += string(remainingBuffer)
			}
		}

		var browserName string
		browserName = detectBrowser(headerMap["User-Agent"])

		method, path, err := parseRequestLine(firstLine)
		if err != nil {
			return
		}

		fullPath := strings.Split(path, "?")
		cleanPath := fullPath[0]
		var queryPath string
		var queryMap map[string]string

		if len(fullPath) > 1 {
			queryPath = fullPath[1]
			if queryPath != "" {
				queryMap = parseKeyValuePairs(queryPath)
			}
		}

		var response, status string
		var filePath string

		rawPath := cleanPath
		cleanedRawPath := filepath.Clean(rawPath)
		if strings.Contains(cleanedRawPath, "..") {
			response, status = CreateResponse("403", "text/plain", "Forbidden", "Access denied")
			goto sendResponse
		} else {
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

func CreateResponse(statusCode, contentType, statusMessage, body string) (string, string) {
	buf := responseBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	
	defer func() {
		if buf.Cap() <= 16384 {
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
	buf.WriteString(body)

	return buf.String(), statusCode
}

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

func getFormValue(bodyMap map[string]string, key string, defaultValue string) string {
	if value, exists := bodyMap[key]; exists {
		return value
	}
	return defaultValue
}

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