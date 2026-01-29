package server

import (
	"bytes"
	"log"
	"net"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RouteHandler is a function that handles an HTTP request
type RouteHandler func(req *Request) (response []byte, status string)

// Router manages HTTP routes and dispatches requests
type Router struct {
	mu     sync.RWMutex
	routes map[string]map[string]RouteHandler
	config *Config
}

// NewRouter creates a new Router instance
func NewRouter() *Router {
	return &Router{
		routes: make(map[string]map[string]RouteHandler),
		config: DefaultConfig(),
	}

}

// router instance with config
func NewRouterWithConfig(config *Config) *Router {
	return &Router{
		routes: make(map[string]map[string]RouteHandler),
		config: config,
	}

}

// Register adds a route handler for a method and path
func (r *Router) Register(method, path string, handler RouteHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.routes[method] == nil {
		r.routes[method] = make(map[string]RouteHandler)
	}
	r.routes[method][path] = handler
}

// HandleBytes routes a request and returns response bytes
func (r *Router) HandleBytes(method, cleanPath string, queryMap, bodyMap map[string]string, browserName string) ([]byte, string) {
	r.mu.RLock()
	methodRoutes, exists := r.routes[method]
	r.mu.RUnlock()

	if !exists {
		return serve404Bytes()
	}

	// Try to find a matching route
	r.mu.RLock()
	defer r.mu.RUnlock()

	var handler RouteHandler
	var pathParams map[string]string
	found := false

	// First try exact match (faster)
	if exactHandler, ok := methodRoutes[cleanPath]; ok {
		handler = exactHandler
		pathParams = make(map[string]string)
		found = true
	} else {
		// Try pattern matching
		for pattern, h := range methodRoutes {
			params, matched := matchRoute(cleanPath, pattern)
			if matched {
				handler = h
				pathParams = params
				found = true
				break
			}
		}
	}

	if !found {
		return serve404Bytes()
	}
	req := &Request{
		Method:     method,
		Path:       cleanPath,
		PathParams: pathParams, // ← The extracted params like {"id": "123"}
		Query:      queryMap,
		Body:       bodyMap,
		Browser:    browserName,
	}

	return handler(req)
}

// Handle routes a request and returns response string (for compatibility)
func (r *Router) Handle(method, cleanPath string, queryMap, bodyMap map[string]string, browserName string) (string, string) {
	responseBytes, status := r.HandleBytes(method, cleanPath, queryMap, bodyMap, browserName)
	return string(responseBytes), status
}

// RunConnection handles an HTTP connection (supports keep-alive)
func (r *Router) RunConnection(conn net.Conn) {
	defer conn.Close()

	defer func() {
		if err := recover(); err != nil {
			log.Printf("PANIC recovered: %v\n%s", err, debug.Stack())
			errorResponse, _ := CreateResponseBytes(
				"500",
				"text/plain",
				"Internal Server Error",
				[]byte("Internal server error occurred"),
			)
			conn.Write(errorResponse)
		}
	}()

	for {
		// Read request
		requestData, err := readHTTPRequest(conn, r.config)
		if err != nil {
			return
		}

		// Parse and handle request
		responseBytes, _, shouldClose := r.processRequest(conn, requestData)

		// Send response
		conn.Write(responseBytes)

		if shouldClose {
			break
		}
	}
}

// processRequest parses and handles a single HTTP request
func (r *Router) processRequest(conn net.Conn, requestData []byte) ([]byte, string, bool) {
	// Split headers and body
	endMarker := []byte("\r\n\r\n")
	parts := bytes.SplitN(requestData, endMarker, 2)
	if len(parts) == 0 {
		resp, status := CreateResponseBytes("400", "text/plain", "Bad Request", []byte("Invalid request"))
		return resp, status, true
	}

	headerSection := parts[0]
	var bodyData []byte
	if len(parts) > 1 {
		bodyData = parts[1]
	}

	// Parse header lines
	headerLines := bytes.Split(headerSection, []byte("\r\n"))
	if len(headerLines) == 0 {
		resp, status := CreateResponseBytes("400", "text/plain", "Bad Request", []byte("No headers"))
		return resp, status, true
	}

	firstLine := headerLines[0]
	remainingHeaders := headerLines[1:]

	// Parse request line
	method, pathBytes, err := parseRequestLineFromBytes(firstLine)
	if err != nil {
		resp, status := CreateResponseBytes("400", "text/plain", "Bad Request", []byte("Invalid request line"))
		return resp, status, true
	}

	// Parse headers
	headerMap := parseHeadersFromBytes(remainingHeaders)

	// Read remaining body if needed
	bodyData = r.readRemainingBody(conn, headerMap, bodyData)

	// Parse query string
	var queryMap map[string]string
	pathParts := bytes.SplitN(pathBytes, []byte("?"), 2)
	cleanPath := string(pathParts[0])

	if len(pathParts) > 1 {
		queryMap = parseKeyValuePairsFromBytes(pathParts[1])
	}

	// Parse body
	var bodyMap map[string]string
	contentType := headerMap["Content-Type"]
	if len(bodyData) > 0 {
		if strings.Contains(contentType, "application/json") {
			bodyMap = parseJSONBodyFromBytes(bodyData)
		} else {
			bodyMap = parseKeyValuePairsFromBytes(bodyData)
		}
	}

	// Detect browser
	browserName := detectBrowser(headerMap["User-Agent"])

	// Route request
	responseBytes, status := r.routeRequest(method, cleanPath, queryMap, bodyMap, browserName)

	if r.config.EnableLogging {
		logRequest(method, cleanPath, status)
	}

	// Check if connection should close
	shouldClose := headerMap["Connection"] == "close"

	return responseBytes, status, shouldClose
}

// readRemainingBody reads body data if Content-Length indicates more data
func (r *Router) readRemainingBody(conn net.Conn, headerMap map[string]string, bodyData []byte) []byte {
	contentLengthStr := headerMap["Content-Length"]
	if contentLengthStr == "" {
		return bodyData
	}

	contentLength, err := strconv.Atoi(contentLengthStr)
	if err != nil || len(bodyData) >= contentLength {
		return bodyData
	}

	remainingBytes := contentLength - len(bodyData)
	remainingBuffer := make([]byte, remainingBytes)
	totalRead := 0

	conn.SetReadDeadline(time.Now().Add(r.config.ReadTimeout))

	for totalRead < remainingBytes {
		n, err := conn.Read(remainingBuffer[totalRead:])
		if err != nil {
			return bodyData
		}
		totalRead += n
	}

	return append(bodyData, remainingBuffer[:totalRead]...)
}

// routeRequest determines how to handle a request (static file or route)
func (r *Router) routeRequest(method, cleanPath string, queryMap, bodyMap map[string]string, browserName string) ([]byte, string) {
	// Determine file path
	var filePath string
	if cleanPath == "/" {
		filePath = "pages/index.html"
	} else {
		filePath = "pages" + cleanPath
	}

	// Security: Check for path traversal
	baseDir := "pages"
	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return CreateResponseBytes("500", "text/plain", "Internal Server Error", []byte("Server configuration error"))
	}
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return CreateResponseBytes("500", "text/plain", "Internal Server Error", []byte("Path resolution error"))
	}

	isPathTraversal := !strings.HasPrefix(absFilePath, absBaseDir)

	// Serve static file if exists (with path traversal protection)
	if !isPathTraversal && FileExists(filePath) {
		content, success := readFileContent(filePath)
		if success {
			contentType := getContentType(filePath)
			return CreateResponseBytes("200", contentType, "OK", content)
		}
		return serve404Bytes()
	}

	// Path traversal attempt
	if isPathTraversal {
		return CreateResponseBytes("403", "text/plain", "Forbidden", []byte("Access denied"))
	}

	// Try routing
	return r.HandleBytes(method, cleanPath, queryMap, bodyMap, browserName)
}
