package main

import (
	"errors"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func main() {
	// Create TCP listener on port 8080
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Println("error listening to server ", err)
		return
	}
	defer listener.Close()
	log.Println("Server listening at Port: 8080")
	
	// Accept connections in infinite loop
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		// Handle each connection in its own goroutine
		go runConnection(conn)
	}
}

func runConnection(conn net.Conn) {
	defer conn.Close()
	
	// === READ HTTP REQUEST USING CHUNKED READING ===
	var n int
	var err error
	var headerBuffer []byte
	
	// Read request in small chunks until we find end of headers
	for {
		chunk := make([]byte, 256)
		n, err = conn.Read(chunk)
		if err != nil {
			log.Println("Error reading:", err)
			return
		}
		headerBuffer = append(headerBuffer, chunk[:n]...)
		
		// Check if we have complete headers (marked by \r\n\r\n)
		if strings.Contains(string(headerBuffer), "\r\n\r\n") {
			break
		}
	}
	
	request := string(headerBuffer)
	log.Println(request) // Debug: show full request
	
	// === SPLIT HEADERS AND BODY ===
	requestParts := strings.SplitN(request, "\r\n\r\n", 2)
	headerSection := requestParts[0]
	body := ""
	bodyMap := make(map[string]string)
	
	// Parse body if present
	if len(requestParts) > 1 {
		body = requestParts[1]
		// Parse form data (key=value&key2=value2)
		pairs := strings.Split(body, "&")
		for _, pair := range pairs {
			parts := strings.Split(pair, "=")
			if len(parts) == 2 {
				key := parts[0]
				value := parts[1]
				bodyMap[key] = value
			}
		}
	}
	
	// === PARSE HEADERS ===
	lines := strings.Split(headerSection, "\r\n")
	if len(lines) == 0 {
		log.Println("Invalid request")
		return
	}
	firstLine := lines[0]
	headerLines := lines[1:]
	headerMap := make(map[string]string)
	
	// Parse each header line into key-value pairs
	for _, headerLine := range headerLines {
		parts := strings.SplitN(headerLine, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			headerMap[key] = value
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
	userAgent := headerMap["User-Agent"]
	if strings.Contains(userAgent, "Chrome") {
		browserName = "Chrome"
	} else if strings.Contains(userAgent, "Firefox") {
		browserName = "Firefox"
	} else if strings.Contains(userAgent, "Safari") {
		browserName = "Safari"
	} else {
		browserName = "Unknown Browser"
	}
	
	// === PARSE REQUEST LINE ===
	parts := strings.Split(firstLine, " ")
	if len(parts) < 3 {
		log.Println("Invalid request line")
		return
	}
	method := parts[0]
	path := parts[1]
	
	// === PARSE PATH AND QUERY PARAMETERS ===
	fullPath := strings.Split(path, "?")
	cleanPath := fullPath[0] // Path without query params
	var queryPath string
	var queryMap map[string]string
	queryMap = make(map[string]string)
	
	if len(fullPath) > 1 {
		queryPath = fullPath[1]
		
		if queryPath != "" {
			// Parse query parameters (key=value&key2=value2)
			pairs := strings.Split(queryPath, "&")
			for _, pair := range pairs {
				parts := strings.Split(pair, "=")
				if len(parts) == 2 {
					queryMap[parts[0]] = parts[1]
				}
			}
		}
	}
	
	// === ROUTING AND RESPONSE GENERATION ===
	var response, status string
	var filePath string
	var contentType string
	
	// Determine file path for static files
	
rawPath := cleanPath
cleanedRawPath := filepath.Clean(rawPath)
if strings.Contains(cleanedRawPath, "..") {
    response, status = createResponse("403", "text/plain", "Forbidden", "Access denied")
    goto sendResponse
} else {
    // Safe to add pages prefix
    if cleanPath == "/" {
        filePath = "pages/index.html"
    } else {
        filePath = "pages" + cleanPath
        log.Println("DEBUG: FilePath", filePath)
    }
}	
	
	// Route based on HTTP method
	switch method {
	case "GET":
		// Try to serve static file first
		if fileExists(filePath) {
			content, err := os.ReadFile(filePath)
			if err != nil {
				log.Printf("Error reading file: %v\n", err)
			}
			
			// Determine content type based on file extension
			ext := filepath.Ext(filePath)
			switch ext {
			case ".html":
				contentType = "text/html"
			case ".css":
				contentType = "text/css"
			case ".js":
				contentType = "application/javascript"
			case ".json":
				contentType = "application/json"
			case ".png":
				contentType = "image/png"
			case ".jpg", ".jpeg":
				contentType = "image/jpeg"
			case ".gif":
				contentType = "image/gif"
			case ".ico":
				contentType = "image/x-icon"
			case ".txt":
				contentType = "text/plain"
			default:
				contentType = "application/octet-stream"
			}
			response, status = createResponse("200", contentType, "OK", string(content))
			
		} else {
			// Handle dynamic GET routes
			switch cleanPath {
			case "/hello":
				name := queryMap["name"]
				if name == "" {
					name = "guest"
				}
				response, status = createResponse("200", "text/plain", "OK", "Hello "+browserName+" "+name+" user!")
				
			case "/time":
				currentTime := time.Now()
				formattedTime := currentTime.Format("15:04:05")
				response, status = createResponse("200", "text/plain", "OK", formattedTime)
				
			default:
				response, status = serve404()
			}
		}
		
	case "POST":
		// Handle POST routes
		switch cleanPath {
		case "/test":
			username := getFormValue(bodyMap, "username", "anonymous")
			password := getFormValue(bodyMap, "password", "")
			response, status = createResponse("200", "text/plain", "OK", username+password)
			
		default:
			response, status = createResponse("404", "text/plain", "Not Found", "Route Not found")
		}
		
	case "PUT":
		// Handle PUT routes
		switch cleanPath {
		case "/user":
			username := getFormValue(bodyMap, "username", "anonymous")
			response, status = createResponse("200", "text/plain", "OK", "Updated user: "+username)
			
		default:
			response, status = createResponse("404", "text/plain", "Not Found", "PUT route not found")
		}
		
	case "DELETE":
		// Handle DELETE routes
		switch cleanPath {
		case "/user":
			response, status = createResponse("200", "text/plain", "OK", "User deleted")
			
		default:
			response, status = createResponse("404", "text/plain", "Not Found", "DELETE route not found")
		}
		
	case "PATCH":
		// Handle PATCH routes
		switch cleanPath {
		case "/user":
			username := getFormValue(bodyMap, "username", "anonymous")
			response, status = createResponse("200", "text/plain", "OK", "Patched user: "+username)
			
		default:
			response, status = createResponse("404", "text/plain", "Not Found", "PATCH route not found")
		}
		
	default:
		// Unsupported HTTP method
		response, status = createResponse("405", "text/plain", "Method Not Allowed", "Method not supported")
	}
	
	// === SEND RESPONSE ===
	sendResponse:
	log.Printf("%s %s %s", method, path, status)
	_, err = conn.Write([]byte(response))
	if err != nil {
		log.Println("Error writing response:", err)
		return
	}
}

// createResponse builds a complete HTTP response with proper headers
func createResponse(statusCode, contentType, statusMessage, body string) (string, string) {
	return "HTTP/1.1 " + statusCode + " " + statusMessage +
		"\r\nContent-Type: " + contentType +
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
	content, err := os.ReadFile(cleanedPath)
	if err != nil {
		log.Printf("Error reading 404.html: %v\n", err)
		return createResponse("404", "text/plain", "Not Found", "Route Not Found")
	}
	response, status := createResponse("404", "text/html", "Not Found", string(content))
	return response, status
}