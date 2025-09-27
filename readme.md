# HTTP Server from Scratch

A lightweight HTTP server built from raw TCP sockets in Go, created to understand HTTP protocol internals and network programming fundamentals.

## Features

- **Raw TCP handling** - Parses HTTP requests directly from socket connections
- **Custom routing** - Simple router with method and path matching
- **Static file serving** - Serves files from a `pages` directory with proper MIME types
- **Template rendering** - Supports Go's `html/template` for dynamic content
- **Connection management** - Keep-alive support with request limits
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

## Limitations

This is a learning project, not production software. It lacks many features of real web servers:
- No HTTPS/TLS support
- Basic error handling
- Simple routing (no path parameters or regex)
- No middleware system
- Limited HTTP method support

## Why Build This?

Most web development happens at the framework level (Express, Flask, etc.). Building from TCP sockets up helps understand what's actually happening under the hood - HTTP parsing, connection management, and the networking fundamentals that frameworks abstract away.

---

*Built with Go 1.21+*