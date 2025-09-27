# HTTP Server from Scratch

A lightweight HTTP server built from raw TCP sockets in Go to understand HTTP protocol internals and network programming fundamentals. No frameworks - just socket programming and HTTP parsing.

**Peak performance: 4,000  RPS** | **Built for learning, not production**

## Quick Start

```bash
git clone https://github.com/codetesla51/raw-http
cd raw-http
go mod tidy
go run main.go

# Server starts on http://localhost:8080
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
- **Security basics** - Path traversal protection and request limits

## Example Usage

```go
router := server.NewRouter()

// Handle form submissions with browser detection
router.Register("POST", "/login", func(req *server.Request) (string, string) {
    username := req.Body["username"]
    browser := req.Browser // "Chrome", "Firefox", etc.
    
    if username == "admin" {
        return server.CreateResponse("200", "text/html", "OK", 
            "<h1>Welcome "+username+"!</h1><p>Browser: "+browser+"</p>")
    }
    return server.CreateResponse("401", "text/html", "Unauthorized", 
        "<h1>Login Failed</h1>")
})

// Simple API endpoint
router.Register("GET", "/ping", func(req *server.Request) (string, string) {
    return server.CreateResponse("200", "text/plain", "OK", "pong")
})
```

## Performance

### Peak Performance: 4,000 RPS

| Concurrency Level | RPS | Response Time | Performance Tier |
|-------------------|-----|---------------|------------------|
| **10** | **4,000** | **0.252ms** | **Peak Performance** |
| 50 | 2,926 | 0.342ms | Excellent |
| 100 | 2,067 | 0.484ms | Very Good |
| 200 | 2,082 | 0.480ms | Very Good |
| 500 | 2,286 | 0.437ms | Good |
| 1000 | 1,463 | 0.683ms | Moderate Load |
| 1500 | 1,981 | 0.505ms | High Load |
| 2000 | 1,507 | 0.664ms | Stress Test |

**Key insights:** 
- **Optimal range:** 10-200 concurrent connections (2,000-4,000 RPS)
- **Sweet spot:** Low concurrency scenarios achieve sub-millisecond response times
- **Scaling:** Maintains good performance up to 500 concurrent connections
- **Reliability:** 0% failure rate across all load levels tested

### Historical Performance Journey
- **Initial buggy version:** ~250-282 RPS (Connection: close with processing bug)
- **Bug fix:** 1,389 RPS (Connection: close, but fixed request handling) 
- **Keep-alive optimization:** 1,710 RPS (enabled proper connection reuse)
- **Peak performance discovery:** 4,000 RPS (optimal concurrency level)

**Total improvement:** 6-7x from initial version

## Under the Hood

- **TCP Connection Pooling:** HTTP/1.1 keep-alive implementation
- **Custom HTTP Parser:** Zero-dependency request parsing with chunked reading
- **Goroutine-per-connection:** Leverages Go's concurrency model
- **MIME Detection:** Comprehensive content-type handling for static files
- **Memory Efficient:** Streams request bodies instead of full buffering

## Project Structure

```
├── server/
│   ├── server.go          # Core HTTP server logic
│   └── server_test.go     # Test suite
├── pages/                 # Static files and templates
│   ├── index.html
│   ├── login.html
│   ├── welcome.html
│   └── styles.css
└── main.go               # Example web application
```

## What I Learned

Building from TCP sockets up provided insights into:

- How HTTP requests are structured and parsed at the protocol level
- TCP connection lifecycle and keep-alive mechanics  
- Security considerations (DoS protection, path traversal attacks)
- Go's networking primitives and goroutine-per-connection model
- Template rendering and form data handling from scratch
- The critical importance of proper connection reuse for performance

## Testing

```bash
go test ./server
```

The test suite covers request parsing, routing, response generation, and includes integration tests with real TCP connections.

## Limitations

This is a learning project, not production software:

- **No HTTPS/TLS support**
- **Basic error handling** 
- **Simple routing** (no path parameters or regex matching)
- **No middleware system**
- **Limited HTTP method support**

## Why Build This?

Most web development happens at the framework level (Express, Flask, etc.). Building from TCP sockets up helps understand what's actually happening under the hood - HTTP parsing, connection management, and the networking fundamentals that frameworks abstract away.

The performance optimization journey demonstrates how low-level implementation details can have dramatic impacts on throughput and response times.

## Routes Available

Visit these paths when running the server:

- `/` - Home page with project info
- `/welcome` - Dynamic template rendering demo
- `/login` - Form handling example  
- `/hello` - About page with technical details
- `/ping` - Simple API endpoint

---

*Built with Go 1.21+ • Created by [Uthman](https://github.com/codetesla51) • Learning project focused on HTTP fundamentals*