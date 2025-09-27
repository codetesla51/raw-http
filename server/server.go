package server

import (
	"errors"
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

func init() {
	// Extra mime types
	mime.AddExtensionType(".svg", "image/svg+xml")
	mime.AddExtensionType(".webp", "image/webp")
	mime.AddExtensionType(".mp4", "video/mp4")
	mime.AddExtensionType(".webm", "video/webm")
	mime.AddExtensionType(".mp3", "audio/mpeg")
	mime.AddExtensionType(".wav", "audio/wav")
	mime.AddExtensionType(".ogg", "audio/ogg")
	mime.AddExtensionType(".pdf", "application/pdf")
	mime.AddExtensionType(".zip", "application/zip")
	mime.AddExtensionType(".tar", "application/x-tar")
	mime.AddExtensionType(".gz", "application/gzip")
	mime.AddExtensionType(".xml", "application/xml")
	mime.AddExtensionType(".csv", "text/csv")
	mime.AddExtensionType(".woff", "font/woff")
	mime.AddExtensionType(".woff2", "font/woff2")
	mime.AddExtensionType(".ttf", "font/ttf")
	mime.AddExtensionType(".otf", "font/otf")
	mime.AddExtensionType(".eot", "application/vnd.ms-fontobject")
}
func readHTTPRequest(conn net.Conn) (string, error) {
	var headerBuffer []byte

	// Read request in small chunks until we find end of headers
	for {
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
func runConnection(conn net.Conn) {
	defer conn.Close()
	requestCount := 0
	for requestCount < 3 {
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

		// Parse body if present
		if len(requestParts) > 1 {

			body = requestParts[1]
			bodyMap = parseKeyValuePairs(body)
		}

		// === PARSE HEADERS ===
		lines := strings.Split(headerSection, "\r\n")
		if len(lines) == 0 {
			log.Println("Invalid request")
			return
		}
		firstLine := lines[0]
		headerLines := lines[1:]
		var headerMap map[string]string
		headerMap = parseHeaders(headerLines)

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
			}
		}

		// Route based on HTTP method
		response, status = handleRoute(method, cleanPath, contentType, queryMap, bodyMap, browserName, filePath)
		// === SEND RESPONSE ===
	sendResponse:
		log.Printf("%s %s %s", method, path, status)
		_, err = conn.Write([]byte(response))
		if err != nil {
			log.Println("Error writing response:", err)
			return
		}
		if headerMap["Connection"] == "close" {
			break
		}

		//   log.Printf("Request %d completed, keeping connection alive", requestCount+1)

	}
	//   log.Println("Connection closed after", requestCount, "requests")

}

// createResponse builds a complete HTTP response with proper headers
func createResponse(statusCode, contentType, statusMessage, body string) (string, string) {
	return "HTTP/1.1 " + statusCode + " " + statusMessage +
		"\r\nContent-Type: " + contentType +
		"\r\nConnection: keep-alive" + // Add this line
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
		return createResponse("404", "text/plain", "Not Found", "Route Not Found")
	}
	response, status := createResponse("404", "text/html", "Not Found", string(content))
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
		return "", "", errors.New("Invalid request line")
		
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

func handleRoute(method, cleanPath, contentType string, queryMap, bodyMap map[string]string, browserName, filePath string) (response, status string) {

	switch method {
	case "GET":
		// Try to serve static file first
		if fileExists(filePath) {
			content, success := readFileContent(filePath)
			if success {
				contentType = getContentType(filePath)
				response, status = createResponse("200", contentType, "OK", string(content))
			} else {
				response, status = serve404()
			}
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
