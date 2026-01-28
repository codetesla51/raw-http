# raw-http

High-performance HTTP/HTTPS server built from raw TCP sockets in Go. No frameworks, no abstractions - just socket I/O, protocol parsing, and efficient request handling.

**Status:** Active Development
**Performance:** 9,800-11,600 RPS (benchmark results below)
**Production Ready:** No - use for learning, experimentation, or embedded scenarios only

## Installation

```bash
git clone https://github.com/codetesla51/raw-http
cd raw-http
go mod tidy
go build
./raw-http
```

Server listens on `http://localhost:8080` by default (auto-increments if busy).
TLS on `https://localhost:8443` (if `server.crt` and `server.key` present).

## Usage

### Basic Server Setup

```go
package main

import "github.com/codetesla51/raw-http/server"

func main() {
    // Default configuration (30s timeout, 10MB max body)
    router := server.NewRouter()
    
    // Or use custom configuration
    cfg := server.DefaultConfig()
    cfg.ReadTimeout = 60 * time.Second
    cfg.MaxBodySize = 50 * 1024 * 1024  // 50MB
    cfg.EnableLogging = true
    router := server.NewRouterWithConfig(cfg)
    
    // Register routes
    router.Register("GET", "/ping", func(req *server.Request) ([]byte, string) {
        return server.CreateResponseBytes("200", "text/plain", "OK", []byte("pong"))
    })
}
```

### Configuration

Configuration is set via `Config` struct with `DefaultConfig()` providing sensible defaults:

```go
type Config struct {
    ReadTimeout     time.Duration  // Max time to wait for request (default: 30s)
    WriteTimeout    time.Duration  // Max time to send response (default: 30s)
    IdleTimeout     time.Duration  // Max time between requests (default: 120s)
    MaxHeaderSize   int            // Max header bytes, rejects larger (default: 8192)
    MaxBodySize     int64          // Max request body, rejects larger (default: 10MB)
    EnableKeepAlive bool           // Allow connection reuse (default: true)
    EnableLogging   bool           // Log requests to stdout (default: false)
}
```

Create custom configs and pass to `NewRouterWithConfig(cfg)`. The config object applies to all connections and routes - it controls defaults only, not runtime behavior.

## Features

- Raw TCP socket handling with HTTP/1.1 protocol compliance
- Custom routing engine (method + path matching)
- Keep-alive connection support (configurable)
- Static file serving with MIME type detection
- Form data and JSON request parsing
- Request context (headers, query parameters, body)
- Panic recovery (handlers can't crash the server)
- Path traversal protection
- HTTPS/TLS support (optional, certificate-based)
- Graceful shutdown with signal handling
- Buffer pooling for memory efficiency
- Configurable timeouts and size limits

## Performance

Benchmark results on 8-core system with keep-alive enabled.

| Scenario | Concurrency | Requests | RPS | Response Time | Status |
|----------|-------------|----------|-----|---------------|--------|
| Baseline | 100 | 10k | 9,818 | 10.2ms | Stable |
| Sustained | 200 | 50k | 9,713 | 20.6ms | Stable |
| High Load | 500 | 100k | 11,635 | 43ms | Stable |
| Extreme | 1,000 | 100k | 11,303 | 88.5ms | Stable |
| Stress | 5,000 | 100k | 8,930 | 559ms | Degraded |
| POST (body) | 100 | 10k | 6,617 | 15.1ms | 35% slower |
| Static Files | 100 | 10k | 6,349 | 15.8ms | Disk I/O |

Peak throughput: **11,635 RPS** (500 concurrent connections)
Optimal latency: **9,818 RPS** at 100 concurrent connections
Breaking point: ~5,000 concurrent connections (OS file descriptor limits)
Zero failures: Up to 5,000 concurrent connections

GET requests on simple handlers (like /ping) are fastest. POST requests and file I/O are 35% slower due to body/disk overhead.

## Architecture

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed design, Go concepts, and implementation details.

Structure:
- `main.go` - Entry point, route registration
- `server/router.go` - Connection handling, routing logic
- `server/request.go` - HTTP request parsing
- `server/response.go` - HTTP response formatting
- `server/config.go` - Configuration and defaults
- `server/static.go` - File serving utilities
- `server/pool.go` - Memory buffer pooling
- `server/mime.go` - Content-type mapping
- `server/logging.go` - Request logging (disabled by default)

## Testing

Run tests:
```bash
go test ./server/...
```

Run benchmarks:
```bash
# Moderate load
ab -n 10000 -c 100 -k http://localhost:8080/ping

# Heavy load
ab -n 100000 -c 500 -k http://localhost:8080/ping
```

## Not Production Ready

This project is suitable for:
- Learning HTTP protocol mechanics
- Embedded servers in tools and utilities
- Experimentation with network programming
- Educational purposes

Do not use for:
- Public-facing web services
- Mission-critical applications
- Handling untrusted input at scale
- Replacing established web servers (nginx, Apache, Go's net/http)

Production use requires:
- Request logging and metrics
- Security hardening and audit
- Connection pooling and limits
- Advanced caching strategies
- Comprehensive error handling
- Load testing under your specific workload

## License

MIT


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