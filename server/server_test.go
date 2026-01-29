package server

import (
	"net"
	"strings"
	"testing"

	"time"
)

func TestRouter(t *testing.T) {
	router := NewRouter()

	// Test route registration and handling
	router.Register("GET", "/test", func(req *Request) ([]byte, string) {
		return CreateResponseBytes("200", "text/plain", "OK", []byte("test response"))
	})

	response, status := router.Handle("GET", "/test", nil, nil, "Chrome")

	if status != "200" {
		t.Errorf("Expected status 200, got %s", status)
	}

	if !strings.Contains(response, "test response") {
		t.Errorf("Response doesn't contain expected body")
	}
}

func TestRouterNotFound(t *testing.T) {
	router := NewRouter()

	_, status := router.Handle("GET", "/nonexistent", nil, nil, "Chrome")

	if status != "404" {
		t.Errorf("Expected status 404, got %s", status)
	}
}

func TestParseKeyValuePairs(t *testing.T) {
	tests := []struct {
		input    string
		expected map[string]string
	}{
		{
			"key1=value1&key2=value2",
			map[string]string{"key1": "value1", "key2": "value2"},
		},
		{
			"name=John%20Doe&age=30",
			map[string]string{"name": "John Doe", "age": "30"},
		},
		{
			"",
			map[string]string{},
		},
	}

	for _, test := range tests {
		result := parseKeyValuePairs(test.input)

		if len(result) != len(test.expected) {
			t.Errorf("Expected %d pairs, got %d", len(test.expected), len(result))
			continue
		}

		for key, expectedValue := range test.expected {
			if actualValue, exists := result[key]; !exists || actualValue != expectedValue {
				t.Errorf("Expected %s=%s, got %s=%s", key, expectedValue, key, actualValue)
			}
		}
	}
}

func TestParseJSONBody(t *testing.T) {
	jsonBody := `{"name": "John", "age": 30, "active": true}`
	result := parseJSONBody(jsonBody)

	expected := map[string]string{
		"name":   "John",
		"age":    "30",
		"active": "true",
	}

	if len(result) != len(expected) {
		t.Errorf("Expected %d fields, got %d", len(expected), len(result))
	}

	for key, expectedValue := range expected {
		if actualValue, exists := result[key]; !exists || actualValue != expectedValue {
			t.Errorf("Expected %s=%s, got %s=%s", key, expectedValue, key, actualValue)
		}
	}
}

func TestParseRequestLine(t *testing.T) {
	tests := []struct {
		input          string
		expectedMethod string
		expectedPath   string
		shouldError    bool
	}{
		{"GET /test HTTP/1.1", "GET", "/test", false},
		{"POST /api/users HTTP/1.1", "POST", "/api/users", false},
		{"invalid", "", "", true},
	}

	for _, test := range tests {
		method, path, err := parseRequestLine(test.input)

		if test.shouldError {
			if err == nil {
				t.Errorf("Expected error for input: %s", test.input)
			}
			continue
		}

		if err != nil {
			t.Errorf("Unexpected error for input %s: %v", test.input, err)
			continue
		}

		if method != test.expectedMethod {
			t.Errorf("Expected method %s, got %s", test.expectedMethod, method)
		}

		if path != test.expectedPath {
			t.Errorf("Expected path %s, got %s", test.expectedPath, path)
		}
	}
}

func TestParseHeaders(t *testing.T) {
	headers := []string{
		"Content-Type: application/json",
		"Content-Length: 100",
		"User-Agent: Mozilla/5.0",
	}

	result := parseHeaders(headers)

	expected := map[string]string{
		"Content-Type":   "application/json",
		"Content-Length": "100",
		"User-Agent":     "Mozilla/5.0",
	}

	for key, expectedValue := range expected {
		if actualValue, exists := result[key]; !exists || actualValue != expectedValue {
			t.Errorf("Expected %s=%s, got %s=%s", key, expectedValue, key, actualValue)
		}
	}
}

func TestDetectBrowser(t *testing.T) {
	tests := []struct {
		userAgent string
		expected  string
	}{
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36", "Chrome"},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0", "Firefox"},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15", "Safari"},
		{"Unknown User Agent", "Unknown Browser"},
	}

	for _, test := range tests {
		result := detectBrowser(test.userAgent)
		if result != test.expected {
			t.Errorf("Expected %s, got %s for user agent: %s", test.expected, result, test.userAgent)
		}
	}
}

func TestCreateResponse(t *testing.T) {
	response, status := CreateResponse("200", "text/html", "OK", "Hello World")

	if status != "200" {
		t.Errorf("Expected status 200, got %s", status)
	}

	expectedParts := []string{
		"HTTP/1.1 200 OK",
		"Content-Type: text/html",
		"Content-Length: 11",
		"Hello World",
	}

	for _, part := range expectedParts {
		if !strings.Contains(response, part) {
			t.Errorf("Response missing expected part: %s", part)
		}
	}
}

func TestRequestStruct(t *testing.T) {
	// Test that Request struct can be created and accessed
	req := &Request{
		Query:   map[string]string{"q": "test"},
		Body:    map[string]string{"name": "value"},
		Browser: "Chrome",
		Method:  "POST",
		Path:    "/api/test",
	}

	if req.Query["q"] != "test" {
		t.Error("Query parameter not set correctly")
	}

	if req.Body["name"] != "value" {
		t.Error("Body parameter not set correctly")
	}

	if req.Browser != "Chrome" {
		t.Error("Browser not set correctly")
	}

	if req.Method != "POST" {
		t.Error("Method not set correctly")
	}

	if req.Path != "/api/test" {
		t.Error("Path not set correctly")
	}
}

// Integration test
func TestIntegration(t *testing.T) {
	router := NewRouter()

	// Register a test route
	router.Register("GET", "/ping", func(req *Request) ([]byte, string) {
		return CreateResponseBytes("200", "text/plain", "OK", []byte("pong"))
	})

	// Start server in goroutine
	listener, err := net.Listen("tcp", ":0") // Use random available port
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go router.RunConnection(conn)
		}
	}()

	// Give server time to start
	time.Sleep(10 * time.Millisecond)

	// Make a request
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send HTTP request
	request := "GET /ping HTTP/1.1\r\nHost: localhost\r\n\r\n"
	_, err = conn.Write([]byte(request))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read response
	response := make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	responseStr := string(response[:n])

	if !strings.Contains(responseStr, "200 OK") {
		t.Error("Expected 200 OK response")
	}

	if !strings.Contains(responseStr, "pong") {
		t.Error("Expected 'pong' in response body")
	}
}
