# HTTP/HTTPS Server from Scratch

A lightweight HTTP/HTTPS server built from raw TCP sockets in Go to understand HTTP protocol internals and network programming fundamentals. No frameworks - just socket programming, HTTP parsing, and TLS encryption.

**Peak performance: 10,000+ RPS** | **Built for learning and understanding fundamentals**

## Quick Start

```bash
git clone https://github.com/codetesla51/raw-http
cd raw-http
go mod tidy
go run main.go

# Server starts on http://localhost:8080
# HTTPS available on https://localhost:8443 (if certificates present)
```

Try it:
```bash
curl http://localhost:8080/ping
# Returns: pong

curl -X POST http://localhost:8080/login \
  -d "username=admin&password=secret"
# Returns: Login successful HTML page
```

## Features

- **Raw TCP handling** - Parses HTTP requests directly from socket connections
- **Custom routing** - Simple router with method and path matching  
- **Static file serving** - Serves files from `pages/` with proper MIME types
- **Template rendering** - Supports Go's `html/template` for dynamic content
- **Connection management** - Keep-alive support with proper connection reuse
- **Form & JSON parsing** - Handles both URL-encoded forms and JSON request bodies
- **Panic recovery** - Graceful handler panic recovery with stack trace logging
- **Security basics** - Path traversal protection and request limits
- **HTTPS/TLS support** - Optional encrypted connections with certificate support
- **Graceful shutdown** - Clean server termination with signal handling
- **Bytes-optimized processing** - Zero-copy parsing with strategic buffer pooling
- **High-performance networking** - Sub-millisecond response times under optimal load

## Panic Recovery

The server includes robust panic recovery middleware that prevents handler panics from crashing the entire server.

### How It Works

Every incoming connection is wrapped with a `defer/recover` mechanism that:
1. **Catches panics** - Recovers from any panic in handler code
2. **Logs stack traces** - Captures full stack trace for debugging
3. **Returns 500 error** - Sends proper HTTP error response to client
4. **Keeps server alive** - Connection pool remains healthy

```go
defer func() {
    if r := recover(); r != nil {
        log.Printf("PANIC recovered: %v\n%s", r, debug.Stack())
        // Server continues running
    }
}()
```

### Benefits

- **Server stability** - A single handler panic won't crash your server
- **Request isolation** - Panic in one request doesn't affect others
- **Debug visibility** - Full stack traces logged for troubleshooting
- **Client experience** - Returns proper 500 error instead of connection drop
- **Production ready** - Handles unexpected errors gracefully

### Testing Panic Recovery

```bash
# Trigger a test panic
curl http://localhost:8080/panic

# Server logs the panic but continues running
# Returns: HTTP 500 Internal Server Error
```

The server will log something like:
```
2026/01/08 09:00:12 PANIC recovered: test panic
goroutine 35 [running]:
runtime/debug.Stack()
...
```

But the server stays alive and continues handling requests.

## Performance Optimization Journey

### Strategic Buffer Pooling

The server uses **strategic buffer pooling** - a technique for reusing large memory buffers instead of constantly allocating and deallocating them. The key insight: **pool large buffers (4KB+), allocate small ones directly**.

#### Implementation

The server uses two optimized buffer pools:

1. **Request Buffer Pool** (8KB) - Reuses buffers for reading incoming HTTP requests
2. **Response Buffer Pool** - Reuses `bytes.Buffer` for building HTTP responses

```go
var requestBufferPool = sync.Pool{
    New: func() interface{} {
        buf := make([]byte, 8192)
        return &buf
    },
}
```

Small read chunks (4KB) are allocated directly - we learned that pooling tiny buffers actually *hurts* performance due to lock contention overhead.

### Bytes Throughout

The entire request/response pipeline works with `[]byte` instead of strings:
- Zero-copy HTTP parsing
- Direct byte operations (`bytes.Split`, `bytes.Contains`)
- Minimal string conversions (only at API boundaries)

This eliminates unnecessary allocations in the hot path.

### Performance Results

| Concurrency | RPS | Avg Response Time | Status |
|-------------|-----|-------------------|--------|
| **c=1** | **~12,000** | **<0.1ms** | Peak |
| **c=10** | **~10,000** | **1.0ms** | Optimal |
| **c=100** | **~10,000** | **10ms** | Good |
| c=1000 | 307 | 3,256ms | **Failure** |

**Key findings:**
- **Sweet spot: 10-100 concurrent connections** - Consistent 10k RPS
- **Single connection: 12k RPS** - Zero lock contention
- **Breaking point: ~1000 concurrent** - System limits reached (file descriptors, goroutine overhead)
- **Zero failures** up to c=100 in sustained testing

### Optimization Impact Timeline

| Stage | RPS | Improvement |
|-------|-----|-------------|
| Initial string-based | ~7,000 | Baseline |
| Added small buffer pools (256B) | ~4,000 | **-43%** (pools hurt) |
| Removed small pools, kept large (8KB) | ~9,400 | **+34%** |
| Full bytes conversion | **~10,000** | **+43%** |

**Total improvement: +43% from strategic optimization**

**Lesson learned:** Premature optimization is real - we initially made performance *worse* by pooling everything. The winning strategy: profile first, optimize strategically.

## Graceful Shutdown

The server supports graceful shutdown, allowing in-flight requests to complete before stopping. Listens for `SIGINT` (Ctrl+C) and `SIGTERM` signals.

### How It Works

1. **Signal Detection** - Captures interrupt signals
2. **Stop Accepting Connections** - Listeners stop immediately
3. **Grace Period** - 2-second wait for active connections
4. **Clean Shutdown** - Proper resource cleanup

```bash
# Start server
go run main.go

# Graceful stop
^C
# Shutting down server...
# Server stopped.
```

## HTTPS Configuration

### Included Self-Signed Certificates

Includes self-signed certificates for local development:
- `server.crt` - Certificate
- `server.key` - Private key

Server auto-detects these files and enables HTTPS on port 8443.

### Production Certificates

For production, use certificates from a CA (like Let's Encrypt):

```bash
# Get certificates
certbot certonly --standalone -d yourdomain.com

# Copy to project
cp /etc/letsencrypt/live/yourdomain.com/fullchain.pem server.crt
cp /etc/letsencrypt/live/yourdomain.com/privkey.pem server.key
chmod 600 server.key
```

### Generate Self-Signed (Testing)

```bash
openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt -days 365 -nodes
```

## Example Usage

```go
router := server.NewRouter()

// Handlers now return []byte instead of string
router.Register("POST", "/login", func(req *server.Request) ([]byte, string) {
    username := req.Body["username"]
    browser := req.Browser
    
    if username == "admin" {
        html := "<h1>Welcome " + username + "!</h1>"
        return server.CreateResponseBytes("200", "text/html", "OK", []byte(html))
    }
    return server.CreateResponseBytes("401", "text/html", "Unauthorized", 
        []byte("<h1>Login Failed</h1>"))
})

// Simple API endpoint
router.Register("GET", "/ping", func(req *server.Request) ([]byte, string) {
    return server.CreateResponseBytes("200", "text/plain", "OK", []byte("pong"))
})
```

## Under the Hood

- **Bytes-first processing:** Zero-copy HTTP parsing with `[]byte` operations
- **Strategic buffer pooling:** `sync.Pool` for large buffers (8KB+), direct allocation for small ones
- **TCP connection pooling:** HTTP/1.1 keep-alive implementation
- **Panic recovery middleware:** Defer/recover pattern preventing handler crashes
- **Goroutine-per-connection:** Leverages Go's concurrency model
- **Custom HTTP parser:** Zero-dependency request parsing
- **MIME detection:** Comprehensive content-type handling
- **TLS/SSL layer:** Optional HTTPS encryption
- **Signal handling:** Context-based graceful shutdown

## Project Structure

```
├── server/
│   ├── server.go          # Bytes-optimized HTTP server with strategic pooling
│   └── server_test.go     # Test suite
├── pages/                 # Static files and templates
│   ├── index.html
│   ├── login.html
│   └── styles.css
├── server.crt             # Self-signed certificate
├── server.key             # Private key
├── main.go                # Example web application
└── README.md
```

## What I Learned

Building from TCP sockets up provided deep insights into:

- **HTTP protocol internals** - Request structure, parsing, and protocol mechanics
- **TCP connection lifecycle** - Keep-alive, connection reuse, and state management
- **Performance optimization** - When to pool, when to allocate, profiling-driven development
- **Memory management** - Buffer reuse strategies and GC pressure reduction
- **Bytes vs strings** - The performance cost of string conversions
- **Error handling** - Panic recovery patterns and production reliability
- **Go's networking primitives** - `net` package, goroutines, and concurrency patterns
- **TLS/SSL encryption** - Certificate management and secure connections
- **Security fundamentals** - Path traversal, DoS protection, input validation
- **Graceful shutdown** - Signal handling and clean resource cleanup
- **Real-world constraints** - File descriptor limits, goroutine overhead, system boundaries

**Key lesson:** Optimization without measurement is guesswork. We initially made performance *worse* by blindly adding buffer pools everywhere. Strategic, measured optimization (profile → change one thing → measure) is the only reliable approach.

## Testing

```bash
go test ./server
```

### Load Testing

Test with ApacheBench:

```bash
# Optimal performance test
ab -n 100000 -c 10 -k http://localhost:8080/ping

# High load test  
ab -n 100000 -c 100 -k http://localhost:8080/ping

# Breaking point test (expect failures)
ab -n 10000 -c 1000 -k http://localhost:8080/ping
```

**Note:** Performance degrades significantly above c=100 due to system limits. For production workloads requiring >100 concurrent connections, use Go's `net/http` package.

### Testing Panic Recovery

```bash
# Test that panics don't crash the server
curl http://localhost:8080/panic

# Server should return 500 but stay alive
# Try another request to verify
curl http://localhost:8080/ping
# Should return: pong
```

## Limitations

This is a learning project demonstrating HTTP fundamentals:

- **Concurrency limit:** ~100 connections before degradation
- **Simple routing:** No path parameters or regex
- **No middleware system**
- **Limited HTTP method support**
- **Basic error handling**
- **No rate limiting or DDoS protection**
- **Synthetic benchmarks:** Real apps with DB/logic will be much slower
- **Not production-ready:** Use `net/http` for real applications

## Why Build This?

Understanding what happens beneath web frameworks - HTTP parsing, connection management, TLS encryption, memory optimization, error recovery, and networking fundamentals. 

The optimization journey (7k → 4k → 10k RPS) demonstrates how low-level implementation choices dramatically impact performance, and how measurement-driven optimization is essential.

## Routes Available

- `/` - Home page
- `/welcome` - Dynamic template demo
- `/login` - Form handling example
- `/hello` - About page
- `/ping` - API endpoint (used for benchmarking)
- `/panic` - Test panic recovery (returns 500 but server stays alive)

---

*Built with Go 1.21+ • Created by [Uthman](https://github.com/codetesla51) • Learning project focused on HTTP/HTTPS internals and performance optimization fundamentals*