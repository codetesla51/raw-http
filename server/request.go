package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// Request represents an incoming HTTP request
type Request struct {
	Method     string
	Path       string
	Query      map[string]string
	PathParams map[string]string
	Body       map[string]string
	Headers    map[string]string
	Browser    string
}

// readHTTPRequest reads HTTP request headers from a connection
func readHTTPRequest(conn net.Conn, config *Config) ([]byte, error) {
	bufPtr := requestBufferPool.Get().(*[]byte)
	headerBuffer := (*bufPtr)[:0]

	defer func() {
		if cap(headerBuffer) <= maxPoolBufferSize {
			requestBufferPool.Put(bufPtr)
		}
	}()

	endMarker := []byte("\r\n\r\n")

	for {
		conn.SetReadDeadline(time.Now().Add(config.ReadTimeout))

		if len(headerBuffer) > config.MaxHeaderSize {
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

// parseRequestLineFromBytes extracts method and path from request line
func parseRequestLineFromBytes(firstLine []byte) (method string, path []byte, err error) {
	parts := bytes.Split(firstLine, []byte(" "))
	if len(parts) < 3 {
		return "", nil, errors.New("invalid request line")
	}
	return string(parts[0]), parts[1], nil
}

// parseHeadersFromBytes parses HTTP headers from byte slices
func parseHeadersFromBytes(headerLines [][]byte) map[string]string {
	headerMap := make(map[string]string, len(headerLines))
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

// parseKeyValuePairsFromBytes parses URL-encoded key-value pairs
func parseKeyValuePairsFromBytes(data []byte) map[string]string {
	resultMap := make(map[string]string, 8)
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

// parseJSONBodyFromBytes parses a JSON body into a string map
func parseJSONBodyFromBytes(bodyData []byte) map[string]string {
	var jsonData map[string]any
	result := make(map[string]string, 8)

	if err := json.Unmarshal(bodyData, &jsonData); err != nil {
		return result
	}

	for key, value := range jsonData {
		result[key] = fmt.Sprintf("%v", value)
	}

	return result
}

// safeURLDecode decodes a URL-encoded string, returning original on error
func safeURLDecode(encoded string) string {
	decoded, err := url.QueryUnescape(encoded)
	if err != nil {
		return encoded
	}
	return decoded
}

// detectBrowser determines browser from User-Agent header
func detectBrowser(userAgent string) string {
	switch {
	case strings.Contains(userAgent, "Chrome"):
		return "Chrome"
	case strings.Contains(userAgent, "Firefox"):
		return "Firefox"
	case strings.Contains(userAgent, "Safari"):
		return "Safari"
	default:
		return "Unknown Browser"
	}
}
func matchRoute(requestPath string, routePattern string) (map[string]string, bool) {
	// Split both into parts
	requestParts := strings.Split(strings.Trim(requestPath, "/"), "/")
	patternParts := strings.Split(strings.Trim(routePattern, "/"), "/")

	// Must have same number of segments
	if len(requestParts) != len(patternParts) {
		return nil, false
	}

	// Extract parameters
	params := make(map[string]string)

	for i := 0; i < len(requestParts); i++ {
		if strings.HasPrefix(patternParts[i], ":") {

			paramName := patternParts[i][1:]
			params[paramName] = requestParts[i]
		} else if requestParts[i] != patternParts[i] {

			return nil, false
		}
	}

	return params, true
}

// --- Compatibility functions for tests ---
// These functions wrap the bytes-based parsers to accept strings.
// They exist ONLY to simplify unit tests (see server_test.go).
// Production code should use the bytes versions directly.

// parseRequestLine parses request line from string (TEST ONLY)
// Wrapper around parseRequestLineFromBytes for test convenience
func parseRequestLine(line string) (method string, path string, err error) {
	parts := strings.Split(line, " ")
	if len(parts) < 3 {
		return "", "", errors.New("invalid request line")
	}
	return parts[0], parts[1], nil
}

// parseHeaders parses headers from string slice (TEST ONLY)
// Wrapper around parseHeadersFromBytes for test convenience
func parseHeaders(headerLines []string) map[string]string {
	headerMap := make(map[string]string, len(headerLines))
	for _, line := range headerLines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			headerMap[key] = value
		}
	}
	return headerMap
}

// parseKeyValuePairs parses URL-encoded string (TEST ONLY)
// Wrapper around parseKeyValuePairsFromBytes for test convenience
func parseKeyValuePairs(data string) map[string]string {
	return parseKeyValuePairsFromBytes([]byte(data))
}

// parseJSONBody parses JSON string (TEST ONLY)
// Wrapper around parseJSONBodyFromBytes for test convenience
func parseJSONBody(body string) map[string]string {
	return parseJSONBodyFromBytes([]byte(body))
}
