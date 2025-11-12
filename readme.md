# HTTP/HTTPS Server from Scratch

A lightweight HTTP/HTTPS server built from raw TCP sockets in Go to understand HTTP protocol internals and network programming fundamentals. No frameworks - just socket programming, HTTP parsing, and TLS encryption.

**Peak performance: 7,721 RPS** | **Built for learning, not production**

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
- **Buffer pooling** - Memory-optimized request/response handling with `sync.Pool`

## Buffer Pooling Optimization

I recently learned about **buffer pooling** - a technique for reusing memory buffers instead of constantly allocating and deallocating them. The core idea is simple: instead of creating new buffers for every request, maintain a pool of reusable buffers.

### Implementation

The server uses three buffer pools:

1. **Request Buffer Pool** (8KB) - Reuses buffers for reading incoming HTTP requests
2. **Chunk Buffer Pool** (256 bytes) - Reuses small buffers for chunked reading
3. **Response Buffer Pool** - Reuses `bytes.Buffer` for building HTTP responses

```go
var requestBufferPool = sync.Pool{
    New: func() interface{} {
        buf := make([]byte, 8192)
        return &buf
    },
}
```

### How It Works

```
Get buffer from pool → Reset it → Use it → Return to pool
                           ↓
                    (Reused by next request)
```

Instead of:
```
Allocate new buffer → Use it → Garbage collector cleans up
                                      ↓
                               (Memory churn + GC pressure)
```

### Performance Impact

| Metric | Before Buffer Pools | After Buffer Pools | Improvement |
|--------|--------------------|--------------------|-------------|
| **Peak RPS** | 4,000 | 7,721 | **+93%** 🔥 |
| **Memory Allocations** | High | Minimal | Significantly reduced |
| **GC Pressure** | Constant | Minimal | Much lower |

**Result:** Nearly doubled throughput by eliminating memory allocation overhead!

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

### Peak Performance: 7,721 RPS (with buffer pooling)

| Concurrency Level | RPS | Response Time | Performance Tier |
|-------------------|-----|---------------|------------------|
| **10** | **7,721** | **0.130ms** | **Peak Performance** 🔥 |
| 50 | 2,926 | 0.342ms | Excellent |
| 100 | 2,067 | 0.484ms | Very Good |
| 200 | 2,082 | 0.480ms | Very Good |
| 500 | 2,286 | 0.437ms | Good |
| 1000 | 1,232 | 0.811ms | Moderate Load |
| 1500 | 1,981 | 0.505ms | High Load |
| 2000 | 1,507 | 0.664ms | Stress Test |

**Key insights:** 
- **Optimal range:** 10-50 concurrent connections (2,900-7,700 RPS)
- **Sweet spot:** Low concurrency scenarios achieve sub-millisecond response times
- **Scaling:** Maintains good performance up to 500 concurrent connections
- **Reliability:** 0% failure rate across all load levels tested

### Historical Performance Journey
- **Initial buggy version:** ~250-282 RPS (Connection: close with processing bug)
- **Bug fix:** 1,389 RPS (Connection: close, but fixed request handling) 
- **Keep-alive optimization:** 1,710 RPS (enabled proper connection reuse)
- **Pre-buffer pooling:** 4,000 RPS (optimal concurrency discovered)
- **Buffer pooling added:** 7,721 RPS (memory optimization breakthrough) 🚀

**Total improvement:** ~27x from initial version | **Buffer pool impact:** +93% improvement

### Important Note on Real-World Performance

These benchmarks test a simple `/ping` endpoint that returns "pong". Real-world applications with database queries, business logic, file I/O, and external API calls will see significantly lower RPS (typically 100-500 RPS for typical web applications). The buffer pooling optimization helps with the networking layer, but your application logic will be the primary bottleneck in production scenarios.

## Under the Hood

- **TCP Connection Pooling:** HTTP/1.1 keep-alive implementation
- **Custom HTTP Parser:** Zero-dependency request parsing with chunked reading
- **Goroutine-per-connection:** Leverages Go's concurrency model
- **Memory Pooling:** `sync.Pool` for buffer reuse across requests
- **MIME Detection:** Comprehensive content-type handling for static files
- **Memory Efficient:** Streams request bodies instead of full buffering
- **TLS/SSL Layer:** Optional HTTPS encryption for secure connections
- **Signal Handling:** Context-based graceful shutdown with proper cleanup

## Project Structure

```
├── server/
│   ├── server.go          # Core HTTP server logic with buffer pools
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
- **Buffer pooling and memory optimization** - Reusing buffers with `sync.Pool` to eliminate allocation overhead and reduce GC pressure

## Testing

```bash
go test ./server
```

The test suite covers request parsing, routing, response generation, and includes integration tests with real TCP connections.

### Load Testing

Test the server with ApacheBench:

```bash
# Peak performance test (low concurrency)
ab -n 10000 -c 10 -k http://localhost:8080/ping

# High concurrency test
ab -n 5000 -c 1000 -k http://localhost:8080/ping

# Without keep-alive (shows the impact of connection reuse)
ab -n 5000 -c 100 http://localhost:8080/ping
```

## Limitations

This is a learning project, not production software:

- **Basic error handling** 
- **Simple routing** (no path parameters or regex matching)
- **No middleware system**
- **Limited HTTP method support**
- **Self-signed certificates only** (use proper CAs for production)
- **No rate limiting or DDoS protection**
- **No database connection pooling**
- **No caching layer**
- **Synthetic benchmarks** (real apps with DB/logic will be slower)

## Why Build This?

Most web development happens at the framework level (Express, Flask, etc.). Building from TCP sockets up helps understand what's actually happening under the hood - HTTP parsing, connection management, TLS encryption, memory optimization, and the networking fundamentals that frameworks abstract away.

The performance optimization journey demonstrates how low-level implementation details (like buffer pooling and connection reuse) can have dramatic impacts on throughput and response times.

## Routes Available

Visit these paths when running the server:

- `/` - Home page with project info
- `/welcome` - Dynamic template rendering demo
- `/login` - Form handling example  
- `/hello` - About page with technical details
- `/ping` - Simple API endpoint

---

*Built with Go 1.21+ • Created by [Uthman](https://github.com/codetesla51) • Learning project focused on HTTP/HTTPS fundamentals and performance optimization*