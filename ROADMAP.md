# raw-http Roadmap

## Tier 2: Essential Features (Build Next)

These three features make the server complete and usable. Each is 1-2 hours to implement.

### Feature 1: Cookies Support

**What it does:** Parse incoming cookies and set response cookies for authentication/sessions.

**How to implement:**

1. **Parse incoming cookies** (in `request.go`):
```go
// After parsing headers, extract Cookie header
func parseCookies(headerMap map[string]string) map[string]string {
    cookies := make(map[string]string)
    cookieHeader := headerMap["Cookie"]
    if cookieHeader == "" {
        return cookies
    }
    
    // Format: "name1=value1; name2=value2; name3=value3"
    parts := strings.Split(cookieHeader, ";")
    for _, part := range parts {
        part = strings.TrimSpace(part)
        kv := strings.SplitN(part, "=", 2)
        if len(kv) == 2 {
            cookies[kv[0]] = kv[1]
        }
    }
    return cookies
}
```

2. **Add to Request struct** (in `request.go`):
```go
type Request struct {
    // ... existing fields ...
    Cookies map[string]string  // NEW
}
```

3. **Call in processRequest** (in `router.go`):
```go
cookies := parseCookies(headerMap)
// Pass to Request struct
req := &Request{
    // ... existing ...
    Cookies: cookies,
}
```

4. **Generate Set-Cookie headers** (new function in `response.go`):
```go
type Cookie struct {
    Name     string
    Value    string
    Path     string        // "/" is default
    Domain   string        // Optional
    Expires  time.Time     // Optional
    MaxAge   int           // Seconds, -1 = delete
    HttpOnly bool          // true = prevent JS access
    Secure   bool          // true = HTTPS only
    SameSite string        // "Strict", "Lax", "None"
}

func formatSetCookie(c Cookie) string {
    // Build: "Set-Cookie: name=value; Path=/; HttpOnly; Secure"
    parts := []string{c.Name + "=" + c.Value}
    
    if c.Path != "" {
        parts = append(parts, "Path="+c.Path)
    }
    if c.HttpOnly {
        parts = append(parts, "HttpOnly")
    }
    if c.Secure {
        parts = append(parts, "Secure")
    }
    if c.SameSite != "" {
        parts = append(parts, "SameSite="+c.SameSite)
    }
    if !c.Expires.IsZero() {
        parts = append(parts, "Expires="+c.Expires.UTC().Format(http.TimeFormat))
    }
    if c.MaxAge > 0 {
        parts = append(parts, fmt.Sprintf("Max-Age=%d", c.MaxAge))
    }
    
    return strings.Join(parts, "; ")
}
```

5. **Update CreateResponseBytes** to accept cookies:
```go
// Modify signature
func CreateResponseBytes(status, contentType string, cookies []Cookie, body []byte) []byte {
    response := fmt.Sprintf("HTTP/1.1 %s\r\n"+
        "Content-Type: %s\r\n"+
        "Content-Length: %d\r\n",
        status, contentType, len(body))
    
    // Add Set-Cookie headers
    for _, c := range cookies {
        response += "Set-Cookie: " + formatSetCookie(c) + "\r\n"
    }
    
    response += "Connection: keep-alive\r\n\r\n"
    return []byte(response + string(body))
}
```

**Usage in handlers:**
```go
router.Register("GET", "/login", func(req *server.Request) ([]byte, string) {
    // Check if already has session cookie
    sessionID := req.Cookies["session_id"]
    if sessionID != "" {
        return []byte("Already logged in"), "200 OK"
    }
    
    // Set new session cookie
    cookie := server.Cookie{
        Name:     "session_id",
        Value:    generateSessionID(),
        Path:     "/",
        HttpOnly: true,
        Secure:   true,
        MaxAge:   3600,  // 1 hour
    }
    
    return server.CreateResponseBytes("200", "text/html", []server.Cookie{cookie}, body)
})
```

**Test:**
```bash
# See cookies sent
curl -i http://localhost:8080/login

# Send cookies back
curl -b "session_id=abc123" http://localhost:8080/dashboard
```

---

### Feature 2: Structured Logging

**What it does:** Log requests in JSON format with timestamp, method, path, status, response time, and IP address.

**How to implement:**

1. **Create structured logger** (modify `logging.go`):
```go
package server

import (
    "encoding/json"
    "fmt"
    "log"
    "time"
)

type RequestLog struct {
    Timestamp   time.Time `json:"timestamp"`
    Method      string    `json:"method"`
    Path        string    `json:"path"`
    Status      string    `json:"status"`
    DurationMs  int64     `json:"duration_ms"`
    ClientIP    string    `json:"client_ip"`
    UserAgent   string    `json:"user_agent"`
    BytesSent   int       `json:"bytes_sent"`
}

func logRequestStructured(rl RequestLog, useJSON bool) {
    if useJSON {
        data, _ := json.Marshal(rl)
        log.Println(string(data))
    } else {
        // Human-readable fallback
        log.Printf("%s %s %s %dms from %s",
            rl.Method, rl.Path, rl.Status, rl.DurationMs, rl.ClientIP)
    }
}
```

2. **Add timing to processRequest** (in `router.go`):
```go
func (r *Router) processRequest(conn net.Conn, requestData []byte) ([]byte, string, bool) {
    startTime := time.Now()  // START
    
    // ... parsing and routing logic ...
    
    responseBytes, status := r.routeRequest(...)
    
    // TIMING
    duration := time.Since(startTime)
    
    if r.config.EnableLogging {
        rl := RequestLog{
            Timestamp:  startTime,
            Method:     method,
            Path:       cleanPath,
            Status:     status,
            DurationMs: duration.Milliseconds(),
            ClientIP:   conn.RemoteAddr().String(),
            UserAgent:  headerMap["User-Agent"],
            BytesSent:  len(responseBytes),
        }
        logRequestStructured(rl, true)  // JSON output
    }
    
    return responseBytes, status, shouldClose
}
```

3. **Add to Config** (in `config.go`):
```go
type Config struct {
    // ... existing ...
    LogFormat string  // "json" or "text"
}

func DefaultConfig() *Config {
    return &Config{
        // ... existing ...
        LogFormat: "json",
    }
}
```

**Usage:**
```bash
# Server logs to stdout in JSON
go run main.go

# Output looks like:
# {"timestamp":"2026-01-28T15:30:15Z","method":"GET","path":"/ping","status":"200 OK","duration_ms":2,"client_ip":"127.0.0.1:54321","user_agent":"curl/7.68.0","bytes_sent":4}
```

**Parse logs with `jq`:**
```bash
./raw-http 2>&1 | jq '.status, .duration_ms'
```

---

### Feature 3: Error Handler Middleware

**What it does:** Catch panics and handler errors, return proper error responses.

**How to implement:**

1. **Create error handling wrapper** (in `router.go`):
```go
func (r *Router) handleWithRecovery(handler RouteHandler, req *Request) (respBytes []byte, status string) {
    defer func() {
        if err := recover(); err != nil {
            log.Printf("Handler panic: %v\n%s", err, debug.Stack())
            respBytes, status = server.CreateResponseBytes(
                "500",
                "text/plain",
                "Internal Server Error",
                []byte(fmt.Sprintf("Error: %v", err)),
            )
        }
    }()
    
    respBytes, status = handler(req)
    return
}
```

2. **Use in routeRequest** (in `router.go`):
```go
func (r *Router) routeRequest(method, cleanPath string, queryMap, bodyMap map[string]string, browserName string) ([]byte, string) {
    // ... existing routing logic ...
    
    if exists {
        req := &Request{
            Query:   queryMap,
            Body:    bodyMap,
            Browser: browserName,
            Method:  method,
            Path:    cleanPath,
        }
        // Wrap handler with recovery
        return r.handleWithRecovery(handler, req)  // WRAPPED
    }
    
    return serve404Bytes()
}
```

3. **Add error response helpers** (in `response.go`):
```go
func Serve400(msg string) ([]byte, string) {
    return CreateResponseBytes("400", "text/plain", "Bad Request", []byte(msg))
}

func Serve404() ([]byte, string) {
    return CreateResponseBytes("404", "text/html", "Not Found", loadPage("404.html"))
}

func Serve500(msg string) ([]byte, string) {
    return CreateResponseBytes("500", "text/plain", "Internal Server Error", []byte(msg))
}
```

**Usage in handlers:**
```go
router.Register("GET", "/data", func(req *server.Request) ([]byte, string) {
    id := req.Query["id"]
    if id == "" {
        return server.Serve400("id parameter required")
    }
    
    // If handler panics, middleware catches it and returns 500
    data := processData(id)  // Could panic
    return server.CreateResponseBytes("200", "application/json", data)
})
```

---

### Feature 4: File Uploads (for later)

**What it does:** Accept multipart/form-data with file fields.

**How to implement (summary):**

1. Parse `multipart/form-data` boundary from `Content-Type` header
2. Read body in chunks, find boundary markers
3. Extract headers and body for each part
4. For file parts: write to disk or buffer in memory
5. For form parts: add to bodyMap like regular form data

**Why defer:** Complex parsing logic, rarely needed unless your app specifically requires uploads. Add when you actually need it.

---

## Tier 3: Optional Features (Add Later)

These are useful but not essential. Build only if your app needs them.

### Feature 5: Sessions (Build on Cookies)

**What it does:** Server-side session storage using cookies for client identification.

**How to implement:**

1. **Create session store** (new file `server/session.go`):
```go
package server

import (
    "crypto/rand"
    "encoding/hex"
    "sync"
    "time"
)

type SessionData struct {
    Values    map[string]interface{}
    CreatedAt time.Time
    ExpiresAt time.Time
}

type SessionStore struct {
    mu       sync.RWMutex
    sessions map[string]*SessionData  // sessionID -> data
}

func NewSessionStore() *SessionStore {
    return &SessionStore{
        sessions: make(map[string]*SessionData),
    }
}

// Generate random session ID
func generateSessionID() string {
    b := make([]byte, 16)
    rand.Read(b)
    return hex.EncodeToString(b)
}

// Create new session
func (s *SessionStore) Create(expiresIn time.Duration) string {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    sessionID := generateSessionID()
    s.sessions[sessionID] = &SessionData{
        Values:    make(map[string]interface{}),
        CreatedAt: time.Now(),
        ExpiresAt: time.Now().Add(expiresIn),
    }
    return sessionID
}

// Get session
func (s *SessionStore) Get(sessionID string) map[string]interface{} {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    session, exists := s.sessions[sessionID]
    if !exists || time.Now().After(session.ExpiresAt) {
        return nil
    }
    return session.Values
}

// Set value in session
func (s *SessionStore) Set(sessionID string, key string, value interface{}) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if session, exists := s.sessions[sessionID]; exists {
        session.Values[key] = value
    }
}

// Delete session
func (s *SessionStore) Delete(sessionID string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    delete(s.sessions, sessionID)
}

// Cleanup expired sessions (call periodically)
func (s *SessionStore) CleanupExpired() {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    now := time.Now()
    for id, session := range s.sessions {
        if now.After(session.ExpiresAt) {
            delete(s.sessions, id)
        }
    }
}
```

2. **Add to Router** (in `router.go`):
```go
type Router struct {
    mu       sync.RWMutex
    routes   map[string]map[string]RouteHandler
    config   *Config
    sessions *SessionStore  // NEW
}

func NewRouter() *Router {
    return &Router{
        routes:   make(map[string]map[string]RouteHandler),
        config:   DefaultConfig(),
        sessions: NewSessionStore(),  // NEW
    }
}
```

3. **Add helper to Request** (in `request.go`):
```go
type Request struct {
    // ... existing fields ...
    Session map[string]interface{}  // Current session data
    SessionID string                 // Session ID from cookie
}
```

4. **Extract session in processRequest** (in `router.go`):
```go
// After parsing cookies
sessionID := cookies["session_id"]
sessionData := r.sessions.Get(sessionID)
if sessionData == nil {
    sessionData = make(map[string]interface{})
}

req := &Request{
    // ... existing ...
    Session:   sessionData,
    SessionID: sessionID,
}
```

**Usage in handlers:**
```go
router.Register("POST", "/login", func(req *server.Request) ([]byte, string) {
    username := req.Body["username"]
    password := req.Body["password"]
    
    // Verify credentials
    if validatePassword(username, password) {
        // Create new session
        sessionID := router.sessions.Create(24 * time.Hour)
        router.sessions.Set(sessionID, "user_id", username)
        
        // Set session cookie
        cookie := server.Cookie{
            Name:     "session_id",
            Value:    sessionID,
            HttpOnly: true,
            MaxAge:   86400,  // 24 hours
        }
        
        return server.CreateResponseBytes("200", "text/html", []server.Cookie{cookie}, []byte("Logged in"))
    }
    
    return server.Serve401("Invalid credentials")
})

// Protected endpoint
router.Register("GET", "/dashboard", func(req *server.Request) ([]byte, string) {
    userID, exists := req.Session["user_id"].(string)
    if !exists || userID == "" {
        return server.Serve401("Not logged in")
    }
    
    return server.CreateResponseBytes("200", "text/html", nil, []byte("Welcome "+userID))
})
```

5. **Background cleanup** (in `main.go`):
```go
// Cleanup expired sessions every 5 minutes
go func() {
    ticker := time.Tick(5 * time.Minute)
    for range ticker {
        router.sessions.CleanupExpired()
    }
}()
```

---

### Feature 6: Rate Limiting

**What it does:** Limit requests per IP address to prevent abuse.

**How to implement:**

1. **Create rate limiter** (new file `server/ratelimit.go`):
```go
package server

import (
    "sync"
    "time"
)

type RateLimitEntry struct {
    Requests  int
    ResetTime time.Time
}

type RateLimiter struct {
    mu        sync.RWMutex
    limits    map[string]*RateLimitEntry  // IP -> rate info
    maxReqs   int        // Requests per window
    window    time.Duration  // Time window
}

func NewRateLimiter(maxReqs int, window time.Duration) *RateLimiter {
    return &RateLimiter{
        limits:  make(map[string]*RateLimitEntry),
        maxReqs: maxReqs,
        window:  window,
    }
}

// Check if IP is rate limited
func (rl *RateLimiter) Allow(ip string) bool {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    now := time.Now()
    entry, exists := rl.limits[ip]
    
    // New entry or window expired
    if !exists || now.After(entry.ResetTime) {
        rl.limits[ip] = &RateLimitEntry{
            Requests:  1,
            ResetTime: now.Add(rl.window),
        }
        return true
    }
    
    // Check if over limit
    if entry.Requests >= rl.maxReqs {
        return false
    }
    
    entry.Requests++
    return true
}

// Get remaining requests for IP
func (rl *RateLimiter) Remaining(ip string) int {
    rl.mu.RLock()
    defer rl.mu.RUnlock()
    
    entry, exists := rl.limits[ip]
    if !exists {
        return rl.maxReqs
    }
    
    remaining := rl.maxReqs - entry.Requests
    if remaining < 0 {
        return 0
    }
    return remaining
}
```

2. **Add to Router** (in `router.go`):
```go
type Router struct {
    // ... existing ...
    rateLimiter *RateLimiter  // NEW
}

func NewRouter() *Router {
    return &Router{
        routes:      make(map[string]map[string]RouteHandler),
        config:      DefaultConfig(),
        sessions:    NewSessionStore(),
        rateLimiter: NewRateLimiter(100, time.Minute),  // 100 req/min
    }
}
```

3. **Check in RunConnection** (in `router.go`):
```go
for {
    requestData, err := readHTTPRequest(conn, r.config)
    if err != nil {
        return
    }
    
    // Check rate limit
    clientIP := strings.Split(conn.RemoteAddr().String(), ":")[0]
    if !r.rateLimiter.Allow(clientIP) {
        resp, _ := CreateResponseBytes("429", "text/plain", 
            "Too Many Requests", []byte("Rate limit exceeded"))
        conn.Write(resp)
        continue
    }
    
    responseBytes, _, shouldClose := r.processRequest(conn, requestData)
    conn.Write(responseBytes)
    
    if shouldClose {
        break
    }
}
```

**Usage:**
```go
// Create custom limiter
limiter := server.NewRateLimiter(1000, time.Minute)  // 1000 req/min per IP

// Use different limits for different endpoints
if strings.HasPrefix(path, "/api/expensive") {
    limiter := server.NewRateLimiter(10, time.Minute)
    if !limiter.Allow(clientIP) {
        return server.Serve429()
    }
}
```

---

### Feature 7: File Uploads

**What it does:** Accept `multipart/form-data` with files and form fields.

**How to implement:**

1. **Parse multipart** (new file `server/upload.go`):
```go
package server

import (
    "bytes"
    "fmt"
    "io"
    "strings"
)

type FormFile struct {
    Filename string
    Content  []byte
    Size     int64
}

type MultipartForm struct {
    Fields map[string]string
    Files  map[string]*FormFile
}

// Parse multipart/form-data
func ParseMultipart(body []byte, boundary string) (*MultipartForm, error) {
    form := &MultipartForm{
        Fields: make(map[string]string),
        Files:  make(map[string]*FormFile),
    }
    
    // Split by boundary
    parts := bytes.Split(body, []byte("--"+boundary))
    
    for _, part := range parts {
        if len(part) == 0 {
            continue
        }
        
        // Split headers and content
        parts := bytes.SplitN(part, []byte("\r\n\r\n"), 2)
        if len(parts) != 2 {
            continue
        }
        
        headers := string(parts[0])
        content := parts[1]
        
        // Extract Content-Disposition
        disposition := extractHeader(headers, "Content-Disposition")
        
        if isFile(disposition) {
            // File field
            filename := extractFilename(disposition)
            name := extractFieldName(disposition)
            form.Files[name] = &FormFile{
                Filename: filename,
                Content:  bytes.TrimSuffix(content, []byte("\r\n")),
                Size:     int64(len(content)),
            }
        } else {
            // Form field
            name := extractFieldName(disposition)
            form.Fields[name] = string(bytes.TrimSuffix(content, []byte("\r\n")))
        }
    }
    
    return form, nil
}

// Helper functions
func extractHeader(headers, name string) string {
    start := strings.Index(headers, name+":")
    if start == -1 {
        return ""
    }
    start += len(name) + 1
    end := strings.Index(headers[start:], "\r\n")
    if end == -1 {
        end = len(headers[start:])
    }
    return strings.TrimSpace(headers[start : start+end])
}

func extractFieldName(disposition string) string {
    start := strings.Index(disposition, `name="`)
    if start == -1 {
        return ""
    }
    start += 6
    end := strings.Index(disposition[start:], `"`)
    return disposition[start : start+end]
}

func extractFilename(disposition string) string {
    start := strings.Index(disposition, `filename="`)
    if start == -1 {
        return ""
    }
    start += 10
    end := strings.Index(disposition[start:], `"`)
    return disposition[start : start+end]
}

func isFile(disposition string) bool {
    return strings.Contains(disposition, "filename=")
}
```

2. **Use in handlers** (in `main.go`):
```go
router.Register("POST", "/upload", func(req *server.Request) ([]byte, string) {
    // Get boundary from Content-Type header
    contentType := req.Headers["Content-Type"]
    // Format: "multipart/form-data; boundary=----WebKitFormBoundary..."
    boundary := extractBoundary(contentType)
    
    form, err := server.ParseMultipart([]byte(req.Body), boundary)
    if err != nil {
        return server.Serve400("Invalid multipart form")
    }
    
    // Access form fields
    title := form.Fields["title"]
    
    // Access uploaded file
    if file, exists := form.Files["document"]; exists {
        // Save to disk or process
        saveFile(file.Filename, file.Content)
        return server.CreateResponseBytes("200", "text/html", 
            nil, []byte("File uploaded: "+file.Filename))
    }
    
    return server.Serve400("No file provided")
})
```

**Limitations:**
- Simple implementation - no streaming (loads entire request in memory)
- For large files, use request streaming (Phase 3 feature)
- Files stored in memory temporarily before processing

---

## Implementation Priority

**Build immediately:**
1. Cookies (enables auth)
2. Structured Logging (enables debugging)
3. Error Handling (stability)

**Build when needed:**
4. Sessions (multi-request user tracking)
5. Rate Limiting (prevent abuse)
6. File Uploads (handle user files)

---

## Phase 1: Core Features (Extended)


### P1.1 Middleware System
**Effort:** Medium | **Impact:** High
- Request/response interceptor chain
- Logging middleware (structured output)
- Recovery middleware (already exists, formalize)
- CORS middleware
- Request ID generation and propagation

**Implementation:**
```go
type Middleware func(next Handler) Handler
type Handler func(req *Request) ([]byte, string)

router.Use(loggingMiddleware)
router.Use(recoveryMiddleware)
```

### P1.2 Cookies Support
**Effort:** Low | **Impact:** High
- Parse `Cookie:` header into map
- Generate `Set-Cookie:` headers
- HttpOnly, Secure, SameSite flags
- Expiry and Max-Age

**Files to modify:** request.go (parse), response.go (generate)

### P1.3 File Upload Support (Multipart)
**Effort:** High | **Impact:** Medium
- Parse `multipart/form-data` content type
- Handle file streams
- Temporary file handling
- Size limits per file
- Memory efficiency (don't load entire file into memory)

**Files:** New file upload.go

### P1.4 Response Compression (Gzip)
**Effort:** Low | **Impact:** Medium
- Detect `Accept-Encoding: gzip`
- Compress response body
- Add `Content-Encoding: gzip` header
- Configurable compression level

**Files:** New compress.go

### P1.5 Structured Access Logging
**Effort:** Low | **Impact:** Medium
- JSON-formatted request logs (optional)
- Include: method, path, status, latency, IP, user-agent
- Log to file or stdout
- Separate debug vs access logs

**Files:** Modify logging.go

---

## Phase 2: Advanced Features

### P2.1 Session Management
**Effort:** High | **Impact:** Medium
- In-memory session store
- Session middleware
- Cookie-based session ID
- Expiry and cleanup

### P2.2 Rate Limiting
**Effort:** Medium | **Impact:** Medium
- Token bucket algorithm
- Per-IP rate limiting middleware
- Configurable rates per endpoint
- Return 429 Too Many Requests

### P2.3 Request Context/Values
**Effort:** Medium | **Impact:** High
- Add context.Context support
- Store request-scoped values (user, session, etc.)
- Pass through handlers without globals

### P2.4 Error Handling Middleware
**Effort:** Low | **Impact:** High
- Catch errors from handlers
- Return appropriate error responses
- Custom error types

### P2.5 Custom Response Headers
**Effort:** Low | **Impact:** Low
- Add X-* headers standardly
- Server identification header
- Cache control headers
- CORS headers

---

## Phase 3: Performance & Scalability

### P3.1 Connection Pooling
**Effort:** High | **Impact:** Medium
- Limit concurrent connections (semaphore)
- Connection queue when at limit
- Return 503 Service Unavailable when full
- Metrics on connection pool utilization

### P3.2 Request Body Streaming
**Effort:** High | **Impact:** Medium
- Don't buffer entire body in memory
- Stream body for large uploads
- Configurable buffer chunk size
- Progress callbacks

### P3.3 Response Buffering Strategy
**Effort:** Medium | **Impact:** Medium
- Small responses: buffer fully
- Large responses: stream to client
- Configurable threshold

### P3.4 Metrics & Observability
**Effort:** Medium | **Impact:** High
- Request count, latency histogram
- Connection count and duration
- Memory usage tracking
- Goroutine count
- Export to Prometheus format (optional)

---

## Optimization Roadmap to 20k RPS

Current bottleneck analysis:
- Lock contention on `router.mu` during lookups (RWMutex helps but still a factor)
- String allocations in header/body parsing
- Memory allocations in request/response formatting
- Context creation overhead
- HTTP date generation on every response

### Optimization Phase 1: Memory (Est +15-20% RPS)

**OPT-1.1: Reduce Allocations in Parsing**
- Pre-allocate buffers for common header counts
- Use byte comparison instead of string
- Reuse header maps instead of creating new ones
- Pool request/response objects

```go
// Instead of:
headers := make(map[string]string)  // allocation

// Use pool:
headers := headerMapPool.Get().(map[string]string)
defer func() {
    for k := range headers { delete(headers, k) }  // clear
    headerMapPool.Put(headers)
}()
```

**OPT-1.2: Optimize Buffer Pools**
- Separate pools for different sizes (4KB, 8KB, 16KB)
- Tune pool sizes based on typical request distribution
- Reduce GC pressure with larger buffers

**OPT-1.3: Cache HTTP Date Header**
- HTTP spec requires date string in every response
- Generate once per second, reuse
- Saves allocations and formatting

```go
// Atomic update every second
var cachedDate atomic.Value  // string

func httpDate() string {
    return cachedDate.Load().(string)
}

// Update once per second in background goroutine
ticker := time.Tick(time.Second)
go func() {
    for range ticker {
        cachedDate.Store(time.Now().UTC().Format(http.TimeFormat))
    }
}()
```

**Expected gain: 1,000-2,000 RPS**

---

### Optimization Phase 2: Lock Contention (Est +10-15% RPS)

**OPT-2.1: Route Lookup Optimization**
- Use read-only route trie after startup
- Lock only during Register() calls
- Lock-free reads with atomic.Value for entire routes map

```go
// After startup, wrap entire routes map in atomic.Value
var routesMap atomic.Value  // map[string]map[string]RouteHandler

// Reads don't need lock
routes := routesMap.Load().(map[string]map[string]RouteHandler)

// Only writes need lock
r.mu.Lock()
newRoutes := copyAndModify(routes)
routesMap.Store(newRoutes)
r.mu.Unlock()
```

**OPT-2.2: Config Access**
- Make config immutable after startup
- No locks needed for config reads

**OPT-2.3: Connection Pool Lock Optimization**
- Use sync/atomic.Int64 for counters instead of mutex
- Only lock when actually needed for queue

**Expected gain: 500-1,500 RPS**

---

### Optimization Phase 3: I/O Optimization (Est +10-15% RPS)

**OPT-3.1: Syscall Reduction**
- Use TCP_NODELAY to reduce buffering delays
- Batch writes where possible
- Use writev syscall for multiple buffers

```go
conn := net.Conn.(net.TCPConn)
conn.SetNoDelay(true)
```

**OPT-3.2: Read Deadline Reduction**
- Only set deadline once per connection, not per request
- Reuse timeout value

**OPT-3.3: Buffer Reuse in Response Building**
- Use bytes.Buffer from pool instead of fmt.Sprintf
- Avoid string concatenation

```go
// Instead of:
response := fmt.Sprintf("HTTP/1.1 %s...", status)
HTTP Server from Scratch
This lightweight HTTP server was built from raw TCP sockets in Go to understand HTTP protocol internals and network programming fundamentals. No frameworks, no shortcuts - just pure socket programming.


// Use:
buf := bytesHTTP Server from Scratch
This lightweight HTTP server was built from raw TCP sockets in Go to understand HTTP protocol internals and network programming fundamentals. No frameworks, no shortcuts - just pure socket programming.

BufferPool.Get().(*bytes.Buffer)
buf.Reset()
buf.WriteString("HTTP/1.1 ")
buf.WriteString(status)
// ...
response := buf.Bytes()
bytesBufferPool.Put(buf)
```

**Expected gain: 1,000-1,500 RPS**
HTTP Server from Scratch
This lightweight HTTP server was built from raw TCP sockets in Go to understand HTTP protocol internals and network programming fundamentals. No frameworks, no shortcuts - just pure socket programming.


---

### Optimization Phase 4: CPU Optimization (Est +5-10% RPS)

**OPT-4.1: String Operations**
- Replace strings.Contains with bytes.Index
- Replace strings.Split with manual loop
- Pre-compile common patterns

**OPT-4.2: HHTTP Server from Scratch
This lightweight HTTP server was built from raw TCP sockets in Go to understand HTTP protocol internals and network programming fundamentals. No frameworks, no shortcuts - just pure socket programming.

eader Parsing**
- Use bytes instead of strings throughout parsing
- Avoid lowercase conversion unless necessary
- Cache field indices

**OPT-4.3: Response Formatting**
- Pre-build common response patterns
- Use strconv instead of fmt for numbers

**Expected gain: 500-1,000 RPS**

---

### Optimization Phase 5: Advanced (Est +5-10% RPS)

**OPT-5.1: Goroutine Pool**
- Instead of spawning goroutine per connection
- Use worker pool for initial connection accept
- Profile to find if it helps or hurts

**OPT-5.2: epoll/kqueue Integration**
- Use native I/O multiplexing for BSD/Linux
- More connections per goroutine
- Significant complexity increase

**OPT-5.3: SIMD/Vectorization**
- Use SIMD for header parsing if available
- Profile first - may not help much

**Expected gain: 500-1,000 RPS**

---

## Projected 20k RPS Roadmap

| Phase | Feature/Optimization | Est. RPS Gain | Cumulative | Effort |
|-------|---------------------|---------------|-----------|--------|
| Current | Baseline | - | 11,635 | - |
| 1 | Memory optimizations | +2,000 | 13,635 | 2 days |
| 2 | Lock reduction | +1,000 | 14,635 | 1 day |
| 3 | I/O optimization | +1,500 | 16,135 | 2 days |
| 4 | CPU tuning | +750 | 16,885 | 1 day |
| 5 | Advanced (pool/epoll) | +1,000 | 17,885 | 3-5 days |
| Advanced | Micro-optimizations | +2,000+ | 20,000+ | Ongoing |

---

## Implementation Priority

**Immediate (Week 1):**
1. Middleware system (unlocks many features)
2. Cookies support
3. Phase 1 optimizations (memory, buffers)

**Short-term (Week 2-3):**
4. Structured logging
5. Phase 2 optimizations (locks)
6. Error handling middleware
7. Request context support

**Medium-term (Week 4-6):**
8. File uploads
9. Response compression
10. Connection pooling
11. Phase 3-4 optimizations

**Long-term (Week 7+):**
12. Session management
13. Rate limiting
14. Metrics/observability
15. Phase 5 advanced optimizations

---

## Testing Strategy

For each feature:
- Unit tests for parsing/formatting
- Integration tests with curl/ab
- Benchmark before and after
- Memory profile (pprof)
- Profile with flame graphs

Before each optimization:
```bash
go test -bench . -benchmem ./server/
go tool pprof http://localhost:8081/debug/pprof/profile?seconds=30
```

---

## Notes on 20k RPS

Current state (11,635 RPS) is very good for a from-scratch implementation.

Getting to 20k requires:
1. Reducing allocations (biggest win)
2. Reducing lock contention
3. Better I/O syscall patterns
4. CPU optimization in hot paths
5. Potentially connection-level pooling

Beyond 20k typically requires:
- epoll/kqueue event loops (architectural change)
- Partial HTTP/2 support
- Advanced connection reuse strategies
- Platform-specific optimizations

For comparison:
- net/http std lib: 15-20k RPS simple handlers
- nginx: 50k+ RPS (C, optimized, event-driven)
- fasthttp (Go): 30k+ RPS (aggressive optimizations, HTTP/1.1 only)

Target of 20k is realistic and achieves 2x improvement from current baseline.
