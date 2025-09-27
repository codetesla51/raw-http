# HTTP Server from Scratch

A lightweight HTTP server built from raw TCP sockets in Go, created to understand HTTP protocol internals and network programming fundamentals.

## Features

- **Raw TCP handling** - Parses HTTP requests directly from socket connections
- **Custom routing** - Simple router with method and path matching
- **Static file serving** - Serves files from a `pages` directory with proper MIME types
- **Template rendering** - Supports Go's `html/template` for dynamic content
- **Connection management** - Keep-alive support with proper connection reuse
- **Security basics** - Path traversal protection and request size/timeout limits
- **Form & JSON parsing** - Handles both URL-encoded forms and JSON request bodies

## Quick Start

```bash
# Clone and run
go mod tidy
go run main.go

# Server starts on http://localhost:8080
```

## Example Usage

```go
router := server.NewRouter()

// Register routes
router.Register("GET", "/welcome", func(req *server.Request) (string, string) {
    return server.CreateResponse("200", "text/html", "OK", "<h1>Hello World</h1>")
})

router.Register("POST", "/login", func(req *server.Request) (string, string) {
    username := req.Body["username"]
    // Handle login logic...
    return server.CreateResponse("200", "text/html", "OK", response)
})
```

## What I Learned

- How HTTP requests are structured and parsed
- TCP connection lifecycle and keep-alive mechanics
- Security considerations (DoS protection, path traversal)
- Go's networking primitives and goroutine-per-connection model
- Template rendering and form data handling
- The critical importance of proper connection reuse for performance

## Project Structure

```
├── server/
│   ├── server.go          # Core HTTP server logic
│   └── server_test.go     # Test suite
├── pages/                 # Static files and templates
│   ├── index.html
│   ├── login.html
│   └── welcome.html
└── main.go               # Example web application
```

## Testing

```bash
go test ./server
```

## Performance Benchmarks

### Connection: close Performance (Bug Fixed)
Tested with `ab -n 5000 -c 1000 -H "Connection: close" http://localhost:8080/ping`:

| Concurrent Connections | Requests/sec | Avg Response Time | Notes |
|------------------------|--------------|-------------------|-------|
| 1000                   | ~250         | 3993ms            | Original buggy version |
| 1000                   | ~282         | 3550ms            | After bug fix, new TCP connection per request |

### Keep-alive Performance  
Tested with `ab -n 5000 -c 1000 -k http://localhost:8080/ping` and `ab -n 1000 -c 100 -k http://localhost:8080/ping`:

| Concurrent Connections | Requests/sec | Avg Response Time | Keep-alive Requests | Command |
|------------------------|--------------|-------------------|---------------------|---------|
| 100                    | ~1710        | 58ms              | 1000/1000          | `ab -n 1000 -c 100 -k` |
| 1000                   | ~1389        | 720ms             | 5000/5000          | `ab -n 5000 -c 1000 -k` |

### Smaller Scale Testing
`ab -n 1000 -c 100 -k http://localhost:8080/ping`:
- **1710 RPS** with 0.585ms mean response time
- Most requests complete in 12-20ms
- 100% connection reuse efficiency

**Performance Summary:**
- **Peak throughput**: ~1710 RPS with keep-alive enabled
- **Connection reuse impact**: 5-6x performance improvement over Connection: close
- **Optimal load**: 100-1000 concurrent connections with keep-alive
- **Reliability**: 0% failure rate across all test scenarios

**Key Optimizations**: 
1. Fixed initial bug affecting request processing (250 → 282 RPS)
2. Removed connection limit loop to enable proper keep-alive support (282 → 1389-1710 RPS)
Combined improvements: 6-7x performance increase from initial buggy version.

## Architecture Insights

The server demonstrates several important HTTP server concepts:

**Connection Management**: Initially limited connection reuse due to a request limit loop. Removing this artificial constraint and implementing proper keep-alive handling resulted in 5-6x performance improvement.

**Concurrency Model**: Uses Go's goroutine-per-connection approach, which scales well for I/O-bound workloads typical of HTTP servers.

**Request Processing**: Direct TCP socket parsing provides insight into HTTP protocol structure while maintaining competitive performance.

## Limitations

This is a learning project, not production software:
- **No HTTPS/TLS support**
- **Basic error handling**
- **Simple routing** (no path parameters or regex)
- **No middleware system**
- **Limited HTTP method support**
- **No load balancing or clustering**

## Why Build This?

Most web development happens at the framework level (Express, Flask, etc.). Building from TCP sockets up helps understand what's actually happening under the hood - HTTP parsing, connection management, and the networking fundamentals that frameworks abstract away.

The performance optimization journey from ~280 RPS to ~1700 RPS by fixing connection reuse demonstrates how low-level implementation details can have dramatic performance impacts.

---

*Built with Go 1.21+*