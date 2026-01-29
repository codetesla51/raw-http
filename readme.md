# raw-http

High-performance HTTP/1.1 server built from raw TCP sockets in Go. Suitable for small to medium applications where you need control, performance, and simplicity over framework abstractions.

**Note:** This is a learning project demonstrating HTTP server implementation. While suitable for small applications, it is not battle-tested like established servers (net/http, nginx, etc.).

## What Is This?

A complete, focused HTTP server implementation. No dependencies, no framework overhead. Built from TCP sockets with direct HTTP/1.1 protocol handling.

**Use cases:**
- Small to medium web applications
- Embedded HTTP servers in tools
- APIs and microservices
- Applications where you control all request handling
- Systems where you want minimal dependencies

## Limitations

**This is a learning project.** It works well for small applications but has limitations compared to battle-tested servers:

- **Not battle-tested in production** - Limited real-world deployment history
- **Single-process only** - No built-in clustering or multi-process support
- **No built-in observability** - You implement logging, metrics, tracing
- **Smaller community** - Not as proven or audited as net/http or nginx
- **Concurrency ceiling** - Performance degrades beyond 5k concurrent connections
- **Limited tooling** - No profiling tools, debuggers, or diagnostic utilities like mature frameworks have

**When NOT to use this:**
- Applications requiring proven, battle-tested infrastructure
- Large teams needing comprehensive framework features
- Applications requiring extensive middleware ecosystem
- Systems that need built-in ORM, session management, clustering
- High-traffic production systems without significant testing

## How to Run

```bash
git clone https://github.com/codetesla51/raw-http
cd raw-http
go build -o server main.go
./server
```

Server listens on `http://localhost:8080` (auto-increments port if busy).  
HTTPS on `https://localhost:8443` if `server.crt` and `server.key` exist.

## What It Does

✓ Accepts HTTP/1.1 connections over TCP  
✓ Parses requests (method, path, headers, body)  
✓ Routes to handlers using exact + pattern matching (`/users/:id`)  
✓ Serves static files with MIME type detection  
✓ Parses JSON and form-encoded request bodies  
✓ Recovers from handler panics (won't crash)  
✓ Protects against path traversal attacks  
✓ Supports keep-alive connections  
✓ Configurable timeouts and size limits  
✓ TLS/HTTPS support  
✓ Graceful shutdown  
✓ Memory efficient (buffer pooling)  

## What It Doesn't Do

✗ Middleware system (you can build this on top)  
✗ Authentication/authorization (you add this)  
✗ Request validation (you validate input)  
✗ Rate limiting (you implement if needed)  
✗ Compression (compression is optional app logic)  
✗ Sessions/cookies abstractions (parse headers yourself)  
✗ Logging (implement in handlers)  
✗ Database access (you add this)  

It's a **server**, not a **framework**. You handle business logic.

## Example Usage

### Basic Handler

```go
router.Register("GET", "/ping", func(req *server.Request) ([]byte, string) {
	return server.CreateResponseBytes("200", "text/plain", "OK", []byte("pong"))
})
```

### Path Parameters

```go
router.Register("GET", "/users/:id", func(req *server.Request) ([]byte, string) {
	userID := req.PathParams["id"]
	body := []byte("User: " + userID)
	return server.CreateResponseBytes("200", "text/plain", "OK", body)
})

// curl http://localhost:8080/users/123
// Response: User: 123
```

### Query Parameters

```go
router.Register("GET", "/search", func(req *server.Request) ([]byte, string) {
	query := req.Query["q"]
	if query == "" {
		return server.Serve400("missing 'q' parameter")
	}
	response := []byte("Search results for: " + query)
	return server.CreateResponseBytes("200", "text/plain", "OK", response)
})

// curl http://localhost:8080/search?q=golang
// Response: Search results for: golang
```

### POST with Body Parsing

```go
router.Register("POST", "/api/users", func(req *server.Request) ([]byte, string) {
	name := req.Body["name"]
	email := req.Body["email"]
	
	if name == "" || email == "" {
		return server.Serve400("name and email required")
	}
	
	// Your logic here (save to DB, etc.)
	response := []byte(`{"id":1,"name":"` + name + `","email":"` + email + `"}`)
	return server.CreateResponseBytes("201", "application/json", "Created", response)
})

// curl -X POST http://localhost:8080/api/users \
//   -d "name=John&email=john@example.com"
// Response: {"id":1,"name":"John","email":"john@example.com"}
```

### JSON Request Parsing

```go
router.Register("POST", "/api/data", func(req *server.Request) ([]byte, string) {
	// JSON body is automatically parsed into req.Body
	data := req.Body["key"]
	
	if data == "" {
		return server.Serve400("key field required")
	}
	
	return server.CreateResponseBytes("200", "application/json", "OK", 
		[]byte(`{"received":"` + data + `"}`))
})

// curl -X POST http://localhost:8080/api/data \
//   -H "Content-Type: application/json" \
//   -d '{"key":"value"}'
```

### Error Responses

```go
router.Register("GET", "/api/protected", func(req *server.Request) ([]byte, string) {
	token := req.Headers["Authorization"]
	
	if token == "" {
		return server.Serve401("missing authorization header")
	}
	
	if token != "Bearer valid-token" {
		return server.Serve403("invalid token")
	}
	
	return server.CreateResponseBytes("200", "text/plain", "OK", []byte("Access granted"))
})
```

### Static Files

```go
// Automatically served from pages/ directory
// pages/index.html    → GET /index.html
// pages/styles.css    → GET /styles.css
// pages/image.png     → GET /image.png

router.Register("GET", "/", func(req *server.Request) ([]byte, string) {
	return server.CreateResponseBytes("200", "text/html", "OK", 
		[]byte("<h1>Home Page</h1>"))
})
```

### Custom Configuration

```go
cfg := server.DefaultConfig()
cfg.MaxBodySize = 50 * 1024 * 1024      // 50MB uploads
cfg.ReadTimeout = 60 * time.Second       // Longer timeout for slow clients
cfg.EnableLogging = true                 // Log all requests

router := server.NewRouterWithConfig(cfg)
```

## Core Abstractions

- **Router** - Registers and dispatches routes
- **Request** - Parsed HTTP request (method, path, headers, body, query params, path params)
- **Config** - Configurable timeouts, size limits, logging toggle
- **Response helpers** - `Serve400()`, `Serve401()`, `Serve500()`, etc.

All in `server/` package. Main application code in `main.go`.

## Performance

Benchmarks on 8-core system (keep-alive enabled):

| Profile | Concurrency | RPS | Latency | Notes |
|---------|-------------|-----|---------|-------|
| Standard load | 100 | 5,601 | 17.9ms | Realistic production scenario |
| High throughput | 500 | 11,042 | 45.3ms | Peak capacity |
| POST requests | 100 | 5,773 | 17.3ms | Body parsing included |

**Real-world baseline:** 5,600+ RPS for typical application handlers  
**Peak capability:** 11,000+ RPS for simple endpoints  
**Scaling:** Horizontal scaling recommended beyond 5k concurrent connections

## Testing

```bash
go test ./server/... -v
```

21 tests covering HTTP parsing, routing, path parameters, and error handling.

## Implementation Details

**Strategic buffer pooling:** `sync.Pool` for 8KB buffers, direct allocation for small reads  
**Zero-copy parsing:** Entire pipeline works with `[]byte`  
**Keep-alive support:** HTTP/1.1 connection reuse  
**Panic recovery:** Defer/recover prevents handler crashes  
**Path traversal protection:** Validated file access  
**TLS/HTTPS:** Optional certificate-based encryption  
**Graceful shutdown:** Signal handling with connection draining  
**MIME detection:** Automatic content-type for static files  

## When to Use This vs net/http

| Aspect | raw-http | net/http |
|--------|----------|----------|
| **Simplicity** | Minimal, full control | Feature-rich |
| **Startup time** | Very fast | Slightly slower |
| **Dependencies** | Zero | Standard library |
| **Learning curve** | Low (straightforward) | Higher (many abstractions) |
| **Concurrency** | 5-10k req/s typical | 10k+ req/s typical |
| **Middleware** | You implement | Built-in patterns |
| **Use case** | Small-medium apps | Any scale |

**Choose raw-http if:** You want simplicity, full control, and prefer explicit over implicit.  
**Choose net/http if:** You need middleware ecosystem, larger team, or standard library comfort.

## Configuration

```go
cfg := server.DefaultConfig()
cfg.ReadTimeout = 30 * time.Second    // Request timeout
cfg.WriteTimeout = 30 * time.Second   // Response timeout
cfg.IdleTimeout = 120 * time.Second   // Keep-alive timeout
cfg.MaxHeaderSize = 8192              // Max header bytes
cfg.MaxBodySize = 10 * 1024 * 1024    // Max POST/PUT size
cfg.EnableKeepAlive = true            // HTTP/1.1 keep-alive
cfg.EnableLogging = false             // Request logging

router := server.NewRouterWithConfig(cfg)
```

## TLS/HTTPS

Server automatically enables HTTPS if `server.crt` and `server.key` exist:

```bash
# Generate self-signed certificate (testing)
openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt -days 365 -nodes

# For production, use Let's Encrypt
certbot certonly --standalone -d yourdomain.com
cp /etc/letsencrypt/live/yourdomain.com/fullchain.pem server.crt
cp /etc/letsencrypt/live/yourdomain.com/privkey.pem server.key
chmod 600 server.key
```

HTTPS listens on `https://localhost:8443`.

## Static File Serving

Drop HTML/CSS/images in `pages/` directory. Server auto-detects MIME types:

```
pages/
  index.html
  styles.css
  image.png
```

Access as `/index.html`, `/styles.css`, etc. Path traversal attempts are blocked.

## Code Structure

```
main.go                  Entry point, route registration, listeners
server/
  router.go              Connection handling, routing, request dispatch
  request.go              HTTP parsing, path parameter extraction
  response.go             Response formatting, status code helpers
  config.go               Configuration and defaults
  static.go               File serving utilities
  pool.go                 Memory buffer pooling
  mime.go                 MIME type detection
  logging.go              Request logging (optional)
  server_test.go          21 unit tests
```

## Response Helpers

Convenient status code builders:

```go
return server.Serve400("validation failed")    // 400 Bad Request
return server.Serve401("auth required")        // 401 Unauthorized
return server.Serve403("access denied")        // 403 Forbidden
return server.Serve405("GET", "/path")         // 405 Method Not Allowed
return server.Serve429()                       // 429 Too Many Requests
return server.Serve500("database error")       // 500 Internal Error
return server.Serve502("upstream timeout")     // 502 Bad Gateway
return server.Serve503()                       // 503 Service Unavailable
return server.Serve201("user created")         // 201 Created
return server.Serve204()                       // 204 No Content
```

## License

MIT



