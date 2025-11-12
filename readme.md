# HTTP/HTTPS Server from Scratch

A lightweight HTTP/HTTPS server built from raw TCP sockets in Go to understand HTTP protocol internals and network programming fundamentals. No frameworks - just socket programming, HTTP parsing, and TLS encryption.

**Peak performance: 4,000 RPS** | **Built for learning, not production**

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
- **Security basics** - Path traversal protection and request limits
- **HTTPS/TLS support** - Optional encrypted connections with certificate support
- **Graceful shutdown** - Clean server termination with signal handling

## Graceful Shutdown

The server now supports graceful shutdown, allowing in-flight requests to complete before the server stops. The server listens for interrupt signals (Ctrl+C) and SIGTERM signals, making it safe to stop in production-like environments.

### How It Works

1. **Signal Detection** - Captures `SIGINT` (Ctrl+C) and `SIGTERM` signals
2. **Stop Accepting Connections** - Listeners stop accepting new connections immediately
3. **Grace Period** - Waits 2 seconds for active connections to complete
4. **Clean Shutdown** - Properly closes both HTTP and HTTPS listeners

### Usage

```bash
# Start the server
go run main.go
# Server listening on http://localhost:8080
# TLS listener successfully started on https://localhost:8443

# Press Ctrl+C to trigger graceful shutdown
^C
# Shutting down server...
# Server stopped.
```

### Implementation Details

The server uses Go's `signal.NotifyContext` to create a context that cancels when shutdown signals are received. Both HTTP and HTTPS listeners check this context and stop accepting new connections gracefully:

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

// Listeners check ctx.Done() to stop accepting connections
<-ctx.Done()
log.Println("Shutting down server...")
```

This ensures that your application can be safely stopped without abruptly terminating active requests, preventing data loss or incomplete transactions.

## HTTPS Configuration

### Included Self-Signed Certificates

This repository includes self-signed certificates for local development and testing:
- `server.crt` - Self-signed certificate
- `server.key` - Private key

The server automatically detects these files and enables HTTPS on port 8443. These certificates are suitable for development and learning purposes only.

### Using Production Certificates

To use standard certificates from a Certificate Authority (like Let's Encrypt):

1. **Obtain certificates** from a trusted CA:
   ```bash
   # Using Let's Encrypt (via Certbot)
   certbot certonly --standalone -d yourdomain.com
   
   # Certificates will be in: /etc/letsencrypt/live/yourdomain.com/
   ```

2. **Copy certificate files** to your project directory:
   ```bash
   cp /etc/letsencrypt/live/yourdomain.com/fullchain.pem server.crt
   cp /etc/letsencrypt/live/yourdomain.com/privkey.pem server.key
   chmod 600 server.key
   ```

3. **Restart the server** - it will automatically use the new certificates:
   ```bash
   go run main.go
   ```

### Generating Self-Signed Certificates (for testing)

If you want to create your own self-signed certificates:

```bash
openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt -days 365 -nodes
```

This creates:
- `server.key` - 4096-bit RSA private key
- `server.crt` - Self-signed certificate valid for 365 days

### Certificate Requirements

- Both `server.crt` and `server.key` must exist in the project root
- Files must be readable by the application
- If either file is missing or invalid, HTTPS will be disabled automatically (HTTP continues to work)
- Private key permissions should be restricted: `chmod 600 server.key`

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
- **TLS/SSL Layer:** Optional HTTPS encryption for secure connections
- **Signal Handling:** Context-based graceful shutdown with proper cleanup

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
├── server.crt             # Self-signed certificate (for HTTPS)
├── server.key             # Private key (for HTTPS)
├── main.go                # Example web application
└── README.md
```

## What I Learned

Building from TCP sockets up provided insights into:

- How HTTP requests are structured and parsed at the protocol level
- TCP connection lifecycle and keep-alive mechanics  
- TLS/SSL encryption and certificate management
- Security considerations (DoS protection, path traversal attacks)
- Go's networking primitives and goroutine-per-connection model
- Template rendering and form data handling from scratch
- The critical importance of proper connection reuse for performance
- Signal handling and graceful shutdown patterns for reliable server termination

## Testing

```bash
go test ./server
```

The test suite covers request parsing, routing, response generation, and includes integration tests with real TCP connections.

## Limitations

This is a learning project, not production software:

- **Basic error handling** 
- **Simple routing** (no path parameters or regex matching)
- **No middleware system**
- **Limited HTTP method support**
- **Self-signed certificates only** (use proper CAs for production)

## Why Build This?

Most web development happens at the framework level (Express, Flask, etc.). Building from TCP sockets up helps understand what's actually happening under the hood - HTTP parsing, connection management, TLS encryption, and the networking fundamentals that frameworks abstract away.

The performance optimization journey demonstrates how low-level implementation details can have dramatic impacts on throughput and response times.

## Routes Available

Visit these paths when running the server:

- `/` - Home page with project info
- `/welcome` - Dynamic template rendering demo
- `/login` - Form handling example  
- `/hello` - About page with technical details
- `/ping` - Simple API endpoint

---

*Built with Go 1.21+ • Created by [Uthman](https://github.com/codetesla51) • Learning project focused on HTTP/HTTPS fundamentals*