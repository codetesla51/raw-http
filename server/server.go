

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
	"sync"
	"time"
)
var chunkBufferPool = sync.Pool{
    New: func() interface{} {
        buf := make([]byte, 4096)
        return &buf
    },
}


var requestBufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 8192)
		return &buf
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

type RouteHandler func(req *Request) (response []byte, status string)

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

// readHTTPRequest reads HTTP request headers as bytes
func readHTTPRequest(conn net.Conn) ([]byte, error) {
    bufPtr := requestBufferPool.Get().(*[]byte)
    headerBuffer := (*bufPtr)[:0]

    defer func() {
        if cap(headerBuffer) <= 16384 {
            requestBufferPool.Put(bufPtr)
        }
    }()

    maxHeaderSize := 8192
    endMarker := []byte("\r\n\r\n")

    for {
        conn.SetReadDeadline(time.Now().Add(1 * time.Second))

        if len(headerBuffer) > maxHeaderSize {
            return nil, errors.New("headers too large")
        }

        chunkPtr := chunkBufferPool.Get().(*[]byte)
        chunk := *chunkPtr

        n, err := conn.Read(chunk)
        if err != nil {
            chunkBufferPool.Put(chunkPtr)
            return nil, err
        }

        headerBuffer = append(headerBuffer, chunk[:n]...)
        chunkBufferPool.Put(chunkPtr)

        if bytes.Contains(headerBuffer, endMarker) {
            break
        }
    }

    result := make([]byte, len(headerBuffer))
    copy(result, headerBuffer)
    return result, nil
}

func (r *Router) RunConnection(conn net.Conn) {
	defer conn.Close()

	for {
		requestData, err := readHTTPRequest(conn)
		if err != nil {
			return
		}

		// Split headers and body
		endMarker := []byte("\r\n\r\n")
		parts := bytes.SplitN(requestData, endMarker, 2)
		if len(parts) == 0 {
			return
		}

		headerSection := parts[0]
		var bodyData []byte
		if len(parts) > 1 {
			bodyData = parts[1]
		}

		// Parse header lines
		headerLines := bytes.Split(headerSection, []byte("\r\n"))
		if len(headerLines) == 0 {
			return
		}

		firstLine := headerLines[0]
		remainingHeaders := headerLines[1:]

		// Parse headers
		headerMap := parseHeadersFromBytes(remainingHeaders)

		// Check if we need to read more body data
		contentLengthStr := headerMap["Content-Length"]
		if contentLengthStr != "" {
			contentLength, err := strconv.Atoi(contentLengthStr)
			if err == nil && len(bodyData) < contentLength {
				remainingBytes := contentLength - len(bodyData)
				remainingBuffer := make([]byte, remainingBytes)
				n, err := conn.Read(remainingBuffer)
				if err != nil {
					return
				}
				bodyData = append(bodyData, remainingBuffer[:n]...)
			}
		}

		// Parse request line
		method, pathBytes, err := parseRequestLineFromBytes(firstLine)
		if err != nil {
			return
		}

		// Parse query string
		var queryMap map[string]string
		pathParts := bytes.SplitN(pathBytes, []byte("?"), 2)
		cleanPath := string(pathParts[0])

		if len(pathParts) > 1 {
			queryString := pathParts[1]
			queryMap = parseKeyValuePairsFromBytes(queryString)
		}

		// Parse body data
		var bodyMap map[string]string
		contentType := headerMap["Content-Type"]
		if len(bodyData) > 0 {
			if bytes.Contains([]byte(contentType), []byte("application/json")) {
				bodyMap = parseJSONBodyFromBytes(bodyData)
			} else {
				bodyMap = parseKeyValuePairsFromBytes(bodyData)
			}
		}

		// Detect browser
		browserName := detectBrowser(headerMap["User-Agent"])

		// Security: Check for path traversal
		cleanedPath := filepath.Clean(cleanPath)
		if bytes.Contains([]byte(cleanedPath), []byte("..")) {
			responseBytes, status := CreateResponseBytes("403", "text/plain", "Forbidden", []byte("Access denied"))
			logRequest(method, cleanPath, status)
			conn.Write(responseBytes)

			if headerMap["Connection"] == "close" {
				break
			}
			continue
		}

		var responseBytes []byte
		var status string

		// Determine file path
		var filePath string
		if cleanPath == "/" {
			filePath = "pages/index.html"
		} else {
			filePath = "pages" + cleanPath
		}

		// Try serving static file first
		if fileExists(filePath) {
			content, success := readFileContent(filePath)
			if success {
				contentType := getContentType(filePath)
				responseBytes, status = CreateResponseBytes("200", contentType, "OK", content)
			} else {
				responseBytes, status = serve404Bytes()
			}
		} else {
			// Try routing
			responseBytes, status = r.HandleBytes(method, cleanPath, queryMap, bodyMap, browserName)
		}

		// Log request
		logRequest(method, cleanPath, status)

		// Write response
		_, err = conn.Write(responseBytes)
		if err != nil {
			return
		}

		// Check keep-alive
		if headerMap["Connection"] == "close" {
			break
		}
	}
}

// Parse headers from bytes
func parseHeadersFromBytes(headerLines [][]byte) map[string]string {
	headerMap := make(map[string]string)
	for _, line := range headerLines {
		parts := bytes.SplitN(line, []byte(":"), 2)
		if len(parts) == 2 {
			key := string(bytes.TrimSpace(parts[0]))
			value := string(bytes.TrimSpace(parts[1]))
			headerMap[key] = value
		}
	}
	return headerMap
}

// Parse request line from bytes
func parseRequestLineFromBytes(firstLine []byte) (method string, path []byte, err error) {
	parts := bytes.Split(firstLine, []byte(" "))
	if len(parts) < 3 {
		return "", nil, errors.New("invalid request line")
	}
	return string(parts[0]), parts[1], nil
}

// Parse key-value pairs from bytes (for query strings and form data)
func parseKeyValuePairsFromBytes(data []byte) map[string]string {
	resultMap := make(map[string]string)
	pairs := bytes.Split(data, []byte("&"))

	for _, pair := range pairs {
		parts := bytes.SplitN(pair, []byte("="), 2)
		if len(parts) == 2 {
			key := string(parts[0])
			value := string(parts[1])
			decodedKey := safeURLDecode(key)
			decodedValue := safeURLDecode(value)
			resultMap[decodedKey] = decodedValue
		}
	}
	return resultMap
}

// Parse JSON body from bytes
func parseJSONBodyFromBytes(bodyData []byte) map[string]string {
	var jsonData map[string]any
	result := make(map[string]string)

	if err := json.Unmarshal(bodyData, &jsonData); err != nil {
		return result
	}

	for key, value := range jsonData {
		result[key] = fmt.Sprintf("%v", value)
	}

	return result
}

// Create HTTP response as bytes
func CreateResponseBytes(statusCode, contentType, statusMessage string, body []byte) ([]byte, string) {
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
	buf.Write(body)

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, statusCode
}

// Serve 404 page
func serve404Bytes() ([]byte, string) {
	cleanedPath := filepath.Clean("pages/404.html")
	content, success := readFileContent(cleanedPath)
	if !success {
		return CreateResponseBytes("404", "text/plain", "Not Found", []byte("Route Not Found"))
	}
	return CreateResponseBytes("404", "text/html", "Not Found", content)
}

// Handle routing with bytes
func (r *Router) HandleBytes(method, cleanPath string, queryMap, bodyMap map[string]string, browserName string) ([]byte, string) {
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
	return serve404Bytes()
}

// Log requests with colors
func logRequest(method, path, status string) {
	switch status {
	case "200":
		log.Print(color.GreenString("%s %s %s", method, path, status))
	case "404", "403", "405":
		log.Print(color.RedString("%s %s %s", method, path, status))
	default:
		log.Printf("%s %s %s", method, path, status)
	}
}

// Helper functions
func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

func readFileContent(filePath string) ([]byte, bool) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, false
	}
	return content, true
}

func safeURLDecode(encoded string) string {
	decoded, err := url.QueryUnescape(encoded)
	if err != nil {
		return encoded
	}
	return decoded
}

func detectBrowser(userAgent string) string {
	switch {
	case bytes.Contains([]byte(userAgent), []byte("Chrome")):
		return "Chrome"
	case bytes.Contains([]byte(userAgent), []byte("Firefox")):
		return "Firefox"
	case bytes.Contains([]byte(userAgent), []byte("Safari")):
		return "Safari"
	default:
		return "Unknown Browser"
	}
}

func getContentType(filePath string) string {
	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return contentType
}

func (r *Router) Register(method, path string, handler RouteHandler) {
	if r.routes[method] == nil {
		r.routes[method] = make(map[string]RouteHandler)
	}
	r.routes[method][path] = handler
}