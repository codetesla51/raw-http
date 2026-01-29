# raw-http

A lightweight HTTP/1.1 server built from raw TCP sockets in Go.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Tests](https://img.shields.io/badge/tests-21%20passing-brightgreen.svg)]()

## Table of Contents

- [About](#about)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Routing](#routing)
- [Request Object](#request-object)
- [Response Helpers](#response-helpers)
- [Configuration](#configuration)
- [Static Files](#static-files)
- [Custom 404 Page](#custom-404-page)
- [TLS/HTTPS](#tlshttps)
- [Testing](#testing)
- [Performance](#performance)
- [Limitations](#limitations)
- [License](#license)

## About

raw-http is an HTTP/1.1 server implementation that handles:

- Request parsing (method, path, headers, body)
- Route matching with path parameters (`/users/:id`)
- Query string and body parsing (JSON + form-encoded)
- Static file serving with MIME detection
- Keep-alive connections
- TLS/HTTPS support
- Panic recovery
- Graceful shutdown

**This is a learning project.** It works for small applications but is not battle-tested. For production, use Go's `net/http` package.

## Installation

### As a dependency

```bash
go get github.com/codetesla51/raw-http@v1.0.0
```

Then import in your code:

```go
import "github.com/codetesla51/raw-http/server"
```

### Build from source

```bash
git clone https://github.com/codetesla51/raw-http.git
cd raw-http
go build -o server main.go
./server
```

Server starts on `http://localhost:8080` (auto-increments if port is busy).

## Quick Start

Here's a complete working server:

```go
package main

import (
    "log"

    "github.com/codetesla51/raw-http/server"
)

func main() {
    // Create server
    srv := server.NewServer(":8080")

    // Register routes
    srv.Register("GET", "/ping", func(req *server.Request) ([]byte, string) {
        return server.CreateResponseBytes("200", "text/plain", "OK", []byte("pong"))
    })

    srv.Register("GET", "/users/:id", func(req *server.Request) ([]byte, string) {
        userID := req.PathParams["id"]
        return server.CreateResponseBytes("200", "text/plain", "OK", []byte("User: "+userID))
    })

    srv.Register("POST", "/api/data", func(req *server.Request) ([]byte, string) {
        name := req.Body["name"]
        if name == "" {
            return server.Serve400("name is required")
        }
        return server.Serve201("created: " + name)
    })

    // Start server (blocks until Ctrl+C)
    if err := srv.ListenAndServe(); err != nil {
        log.Fatal(err)
    }
}
```

Test it:

```bash
curl http://localhost:8080/ping           # pong
curl http://localhost:8080/users/42       # User: 42
curl -X POST -d "name=john" http://localhost:8080/api/data
```

## Routing

### Register Routes

```go
router.Register(method, path, handler)
```

### Path Parameters

Use `:param` syntax to capture URL segments:

```go
router.Register("GET", "/users/:id", func(req *server.Request) ([]byte, string) {
    userID := req.PathParams["id"]  // "123" from /users/123
    return server.CreateResponseBytes("200", "text/plain", "OK", []byte(userID))
})

router.Register("GET", "/posts/:postId/comments/:commentId", func(req *server.Request) ([]byte, string) {
    postID := req.PathParams["postId"]
    commentID := req.PathParams["commentId"]
    // ...
})
```

### Query Parameters

```go
router.Register("GET", "/search", func(req *server.Request) ([]byte, string) {
    q := req.Query["q"]           // /search?q=golang
    page := req.Query["page"]     // /search?q=golang&page=2
    // ...
})
```

### POST Body

Form-encoded and JSON bodies are automatically parsed:

```go
router.Register("POST", "/users", func(req *server.Request) ([]byte, string) {
    name := req.Body["name"]
    email := req.Body["email"]
    // ...
})
```

## Request Object

Handlers receive `*server.Request`:

| Field | Type | Description |
|-------|------|-------------|
| `Method` | `string` | HTTP method (GET, POST, PUT, DELETE) |
| `Path` | `string` | Request path without query string |
| `PathParams` | `map[string]string` | URL parameters from route (`:id`) |
| `Query` | `map[string]string` | Query string parameters |
| `Body` | `map[string]string` | Parsed request body |
| `Headers` | `map[string]string` | HTTP headers |
| `Browser` | `string` | Detected browser name |

## Response Helpers

### Build Custom Response

```go
server.CreateResponseBytes(statusCode, contentType, statusMessage, body)

// Example
return server.CreateResponseBytes("200", "application/json", "OK", []byte(`{"ok":true}`))
```

### Status Code Helpers

| Function | Code | Use Case |
|----------|------|----------|
| `Serve201(msg)` | 201 | Resource created |
| `Serve204()` | 204 | Success, no content |
| `Serve400(msg)` | 400 | Bad request / validation error |
| `Serve401(msg)` | 401 | Authentication required |
| `Serve403(msg)` | 403 | Access denied |
| `Serve405(method, path)` | 405 | Method not allowed |
| `Serve429(msg)` | 429 | Rate limit exceeded |
| `Serve500(msg)` | 500 | Internal server error |
| `Serve502(msg)` | 502 | Bad gateway |
| `Serve503(msg)` | 503 | Service unavailable |

Example:

```go
router.Register("POST", "/login", func(req *server.Request) ([]byte, string) {
    if req.Body["password"] == "" {
        return server.Serve400("password required")
    }
    if !authenticate(req.Body["user"], req.Body["password"]) {
        return server.Serve401("invalid credentials")
    }
    return server.Serve201("logged in")
})
```

## Configuration

### Using Server with Config

```go
cfg := &server.Config{
    ReadTimeout:     60 * time.Second,
    WriteTimeout:    30 * time.Second,
    MaxBodySize:     50 * 1024 * 1024,  // 50MB
    EnableKeepAlive: true,
}

srv := server.NewServerWithConfig(":8080", cfg)
srv.Register("GET", "/ping", handler)
srv.ListenAndServe()
```

### Using Router Directly

```go
cfg := server.DefaultConfig()
cfg.ReadTimeout = 60 * time.Second

router := server.NewRouterWithConfig(cfg)
router.ListenAndServe(":8080")
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `ReadTimeout` | `time.Duration` | 30s | Max time to read entire request |
| `WriteTimeout` | `time.Duration` | 30s | Max time to write response |
| `IdleTimeout` | `time.Duration` | 120s | Keep-alive timeout |
| `MaxHeaderSize` | `int` | 8192 | Max header size (bytes) |
| `MaxBodySize` | `int64` | 10MB | Max request body size |
| `EnableKeepAlive` | `bool` | true | HTTP/1.1 keep-alive |
| `EnableLogging` | `bool` | false | Log requests to stdout |

## Static Files

Files in `pages/` directory are served automatically:

```
pages/
├── index.html      → GET /index.html
├── styles.css      → GET /styles.css
├── js/
│   └── app.js      → GET /js/app.js
└── 404.html        → Custom 404 page
```

MIME types are detected automatically (.html, .css, .js, .png, .jpg, etc).

Path traversal attacks (`/../etc/passwd`) are blocked.

## Custom 404 Page

Create `pages/404.html`:

```html
<!DOCTYPE html>
<html>
<head>
    <title>404 - Not Found</title>
</head>
<body>
    <h1>404</h1>
    <p>Page not found.</p>
</body>
</html>
```

This page is returned for any unmatched route. If the file doesn't exist, the server returns plain text "Route Not Found".

## TLS/HTTPS

Enable HTTPS with a single line:

```go
srv := server.NewServer(":8080")
srv.EnableTLS(":8443", "server.crt", "server.key")
srv.Register("GET", "/ping", handler)
srv.ListenAndServe()  // Serves HTTP on 8080 and HTTPS on 8443
```

### Generate Certificates

```bash
# Generate self-signed certificate (development only)
openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt -days 365 -nodes
```

Place `server.crt` and `server.key` in the project root.

For production, use Let's Encrypt:

```bash
certbot certonly --standalone -d yourdomain.com
cp /etc/letsencrypt/live/yourdomain.com/fullchain.pem server.crt
cp /etc/letsencrypt/live/yourdomain.com/privkey.pem server.key
```

## Testing

Run tests:

```bash
go test ./server/... -v
```

21 tests cover:
- HTTP parsing (request line, headers, body)
- Route matching (exact, pattern, params)
- Response formatting
- Error handling

## Technical Internals

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Server Struct                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │   Router    │  │  TLS Config │  │  Graceful Shutdown  │  │
│  └──────┬──────┘  └─────────────┘  └─────────────────────┘  │
└─────────┼───────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────┐
│                    Connection Handler                        │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ Buffer Pool │  │  Request    │  │  Keep-Alive Loop    │  │
│  │  (sync.Pool)│  │  Parser     │  │                     │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Buffer Pooling

Three `sync.Pool` instances reduce garbage collection pressure:

| Pool | Buffer Size | Purpose |
|------|-------------|---------|
| `chunkBufferPool` | 4KB | Reading from TCP connection |
| `requestBufferPool` | 8KB | Accumulating request headers |
| `responseBufferPool` | Dynamic | Building HTTP responses |

Buffers larger than 16KB are discarded to prevent memory bloat.

```go
// How it works internally
buf := chunkBufferPool.Get().(*[]byte)
defer chunkBufferPool.Put(buf)
n, _ := conn.Read(*buf)
```

### Graceful Shutdown

The server handles `SIGINT` and `SIGTERM` signals:

1. Stop accepting new connections
2. Wait for active connections to finish (2 second grace period)
3. Close all listeners
4. Exit cleanly

```go
// Automatic signal handling
srv := server.NewServer(":8080")
srv.ListenAndServe()  // Blocks until Ctrl+C

// Or use custom context for programmatic shutdown
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
srv.ListenAndServeContext(ctx)
```

### Keep-Alive Connections

HTTP/1.1 keep-alive is enabled by default:

- Connections are reused for multiple requests
- Idle timeout: 120 seconds (configurable)
- Reduces TCP handshake overhead
- Significantly improves throughput (5k → 11k req/sec)

### Request Parsing

Zero-allocation parsing where possible:

1. Read raw bytes from connection into pooled buffer
2. Split headers from body at `\r\n\r\n` marker
3. Parse request line: `METHOD /path HTTP/1.1`
4. Parse headers into map (single allocation)
5. Parse body based on Content-Type (JSON or form-encoded)

### Panic Recovery

Every connection handler is wrapped with recovery:

```go
defer func() {
    if err := recover(); err != nil {
        log.Printf("PANIC recovered: %v\n%s", err, debug.Stack())
        conn.Write(errorResponse500)
    }
}()
```

A panic in one handler won't crash the server.

### Path Traversal Protection

Static file serving blocks directory traversal attempts:

```go
// These are blocked:
// /../etc/passwd
// /pages/../../../etc/passwd
// /%2e%2e/etc/passwd

if strings.Contains(cleanPath, "..") {
    return Serve403()
}
```

## Performance

Benchmarks on 8-core system:

| Scenario | Concurrency | Requests/sec | Latency |
|----------|-------------|--------------|---------|
| GET /ping | 100 | 5,601 | 17.9ms |
| GET /ping | 500 | 11,042 | 45.3ms |
| POST with body | 100 | 5,773 | 17.3ms |

Run your own:

```bash
# Install Apache Bench
sudo apt install apache2-utils

# Benchmark
ab -n 10000 -c 100 -k http://localhost:8080/ping
```

## Limitations

| Limitation | Impact |
|------------|--------|
| Not production-tested | Use for learning/small projects only |
| Single process | No clustering support |
| No middleware system | Implement yourself if needed |
| No observability | No built-in metrics/tracing |
| ~5k connection ceiling | Performance degrades at high concurrency |

**For production applications, use Go's `net/http` package.**

## License

MIT



