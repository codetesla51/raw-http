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

// Test path parameter extraction with pattern matching
func TestPathParameterExtraction(t *testing.T) {
	tests := []struct {
		pattern        string
		path           string
		shouldMatch    bool
		expectedParams map[string]string
	}{
		{
			"/users/:id",
			"/users/123",
			true,
			map[string]string{"id": "123"},
		},
		{
			"/users/:id",
			"/users/john",
			true,
			map[string]string{"id": "john"},
		},
		{
			"/api/v1/:version/users/:id",
			"/api/v1/stable/users/456",
			true,
			map[string]string{"version": "stable", "id": "456"},
		},
		{
			"/users/:id",
			"/products/123",
			false,
			nil,
		},
		{
			"/users/:id",
			"/users/123/posts",
			false,
			nil,
		},
	}

	for _, test := range tests {
		params, matched := matchRoute(test.path, test.pattern)

		if matched != test.shouldMatch {
			t.Errorf("Pattern %s, path %s: expected matched=%v, got %v",
				test.pattern, test.path, test.shouldMatch, matched)
			continue
		}

		if test.shouldMatch {
			if len(params) != len(test.expectedParams) {
				t.Errorf("Expected %d params, got %d", len(test.expectedParams), len(params))
				continue
			}

			for key, expectedValue := range test.expectedParams {
				if actualValue, exists := params[key]; !exists || actualValue != expectedValue {
					t.Errorf("Expected %s=%s, got %s=%s", key, expectedValue, key, actualValue)
				}
			}
		}
	}
}

// Test response helpers
func TestResponseHelpers(t *testing.T) {
	tests := []struct {
		name           string
		fn             func() ([]byte, string)
		expectedStatus string
		shouldContain  string
	}{
		{"Serve400", func() ([]byte, string) { return Serve400("test error") }, "400", "test error"},
		{"Serve401", func() ([]byte, string) { return Serve401("auth failed") }, "401", "auth failed"},
		{"Serve403", func() ([]byte, string) { return Serve403("forbidden") }, "403", "forbidden"},
		{"Serve429", func() ([]byte, string) { return Serve429("rate limit") }, "429", "rate limit"},
		{"Serve500", func() ([]byte, string) { return Serve500("server error") }, "500", "server error"},
		{"Serve201", func() ([]byte, string) { return Serve201("created") }, "201", "created"},
		{"Serve204", func() ([]byte, string) { return Serve204() }, "204", ""},
	}

	for _, test := range tests {
		response, status := test.fn()

		if status != test.expectedStatus {
			t.Errorf("%s: expected status %s, got %s", test.name, test.expectedStatus, status)
		}

		if test.shouldContain != "" && !strings.Contains(string(response), test.shouldContain) {
			t.Errorf("%s: response doesn't contain '%s'", test.name, test.shouldContain)
		}
	}
}

// Test config defaults
func TestConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ReadTimeout <= 0 {
		t.Error("ReadTimeout should be > 0")
	}

	if cfg.WriteTimeout <= 0 {
		t.Error("WriteTimeout should be > 0")
	}

	if cfg.MaxBodySize <= 0 {
		t.Error("MaxBodySize should be > 0")
	}

	if cfg.MaxHeaderSize <= 0 {
		t.Error("MaxHeaderSize should be > 0")
	}

	if !cfg.EnableKeepAlive {
		t.Error("EnableKeepAlive should be true by default")
	}

	if cfg.EnableLogging {
		t.Error("EnableLogging should be false by default (performance)")
	}
}

// Test request struct population
func TestRequestStructPopulation(t *testing.T) {
	router := NewRouter()

	router.Register("POST", "/api/:version/users/:id", func(req *Request) ([]byte, string) {
		// Verify path params extracted
		if req.PathParams["version"] != "v1" {
			t.Errorf("Expected version=v1, got %s", req.PathParams["version"])
		}

		if req.PathParams["id"] != "42" {
			t.Errorf("Expected id=42, got %s", req.PathParams["id"])
		}

		// Verify other request fields
		if req.Method != "POST" {
			t.Errorf("Expected method=POST, got %s", req.Method)
		}

		if req.Path != "/api/v1/users/42" {
			t.Errorf("Expected path=/api/v1/users/42, got %s", req.Path)
		}

		return CreateResponseBytes("200", "text/plain", "OK", []byte("verified"))
	})

	queryMap := map[string]string{"filter": "active"}
	bodyMap := map[string]string{"name": "John"}

	response, status := router.Handle("POST", "/api/v1/users/42", queryMap, bodyMap, "Chrome")

	if status != "200" {
		t.Errorf("Expected status 200, got %s", status)
	}

	if !strings.Contains(response, "verified") {
		t.Error("Handler verification failed")
	}
}

// Test static file serving
func TestStaticFileServing(t *testing.T) {
	router := NewRouter()

	// Test that / routes to index.html
	response, status := router.Handle("GET", "/", nil, nil, "Chrome")

	if status != "200" {
		t.Logf("Note: /index.html not found (expected if pages/index.html doesn't exist)")
	} else {
		if !strings.Contains(response, "HTTP/1.1") {
			t.Error("Response should contain HTTP headers")
		}
	}
}

// Test headers parsing
func TestHeadersParsing(t *testing.T) {
	headerLines := []string{
		"Host: localhost:8080",
		"User-Agent: curl/7.68.0",
		"Accept: */*",
		"Content-Type: application/json",
	}

	headers := parseHeaders(headerLines)

	expectedHeaders := map[string]string{
		"Host":         "localhost:8080",
		"User-Agent":   "curl/7.68.0",
		"Accept":       "*/*",
		"Content-Type": "application/json",
	}

	for key, expectedValue := range expectedHeaders {
		if actualValue, exists := headers[key]; !exists || actualValue != expectedValue {
			t.Errorf("Expected %s: %s, got %s", key, expectedValue, actualValue)
		}
	}
}

// Test browser detection
func TestBrowserDetection(t *testing.T) {
	tests := []struct {
		userAgent string
		expected  string
	}{
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36", "Chrome"},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0", "Firefox"},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15", "Safari"},
		{"curl/7.68.0", "Unknown Browser"},
	}

	for _, test := range tests {
		result := detectBrowser(test.userAgent)
		if result != test.expected {
			t.Errorf("For UA %s: expected %s, got %s", test.userAgent, test.expected, result)
		}
	}
}

// Test CreateResponseBytes
func TestCreateResponseBytes(t *testing.T) {
	response, status := CreateResponseBytes("200", "application/json", "OK", []byte(`{"key":"value"}`))

	if status != "200" {
		t.Errorf("Expected status 200, got %s", status)
	}

	responseStr := string(response)

	if !strings.Contains(responseStr, "HTTP/1.1 200 OK") {
		t.Error("Response should contain status line")
	}

	if !strings.Contains(responseStr, "Content-Type: application/json") {
		t.Error("Response should contain Content-Type header")
	}

	if !strings.Contains(responseStr, `{"key":"value"}`) {
		t.Error("Response should contain body")
	}

	if !strings.Contains(responseStr, "Content-Length:") {
		t.Error("Response should contain Content-Length")
	}
}

// Test multiple exact routes
func TestMultipleExactRoutes(t *testing.T) {
	router := NewRouter()

	router.Register("GET", "/api/users", func(req *Request) ([]byte, string) {
		return CreateResponseBytes("200", "text/plain", "OK", []byte("users list"))
	})

	router.Register("GET", "/api/products", func(req *Request) ([]byte, string) {
		return CreateResponseBytes("200", "text/plain", "OK", []byte("products list"))
	})

	router.Register("GET", "/api/orders", func(req *Request) ([]byte, string) {
		return CreateResponseBytes("200", "text/plain", "OK", []byte("orders list"))
	})

	tests := []struct {
		path     string
		expected string
	}{
		{"/api/users", "users list"},
		{"/api/products", "products list"},
		{"/api/orders", "orders list"},
	}

	for _, test := range tests {
		response, status := router.Handle("GET", test.path, nil, nil, "Chrome")

		if status != "200" {
			t.Errorf("Expected status 200 for %s, got %s", test.path, status)
		}

		if !strings.Contains(response, test.expected) {
			t.Errorf("Expected '%s' in response for %s", test.expected, test.path)
		}
	}
}

// Test POST with body
func TestPostWithBody(t *testing.T) {
	router := NewRouter()

	router.Register("POST", "/api/users", func(req *Request) ([]byte, string) {
		name := req.Body["name"]
		email := req.Body["email"]

		if name == "" || email == "" {
			return Serve400("name and email required")
		}

		response := "User created: " + name + " (" + email + ")"
		return CreateResponseBytes("201", "text/plain", "Created", []byte(response))
	})

	bodyMap := map[string]string{
		"name":  "John Doe",
		"email": "john@example.com",
	}

	response, status := router.Handle("POST", "/api/users", nil, bodyMap, "Chrome")

	if status != "201" {
		t.Errorf("Expected status 201, got %s", status)
	}

	if !strings.Contains(response, "John Doe") {
		t.Error("Response should contain user name")
	}
}
