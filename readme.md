# raw-http

HTTP/1.1 server built from raw TCP sockets in Go.

**This is a learning project.** It works for small applications but is not battle-tested. For production, use net/http or nginx.

## Install

```bash
git clone https://github.com/codetesla51/raw-http
cd raw-http
go build -o server main.go
./server
```

Listens on `http://localhost:8080`.

## Usage

```go
package main

import "github.com/codetesla51/raw-http/server"

func main() {
    router := server.NewRouter()

    router.Register("GET", "/ping", func(req *server.Request) ([]byte, string) {
        return server.CreateResponseBytes("200", "text/plain", "OK", []byte("pong"))
    })

    router.Register("GET", "/users/:id", func(req *server.Request) ([]byte, string) {
        userID := req.PathParams["id"]
        return server.CreateResponseBytes("200", "text/plain", "OK", []byte("User: "+userID))
    })

    router.Register("POST", "/api/data", func(req *server.Request) ([]byte, string) {
        name := req.Body["name"]
        if name == "" {
            return server.Serve400("name required")
        }
        return server.CreateResponseBytes("201", "application/json", "Created", 
            []byte(`{"name":"`+name+`"}`))
    })

    // Start server (see main.go for full example)
}
```

## Features

- HTTP/1.1 request parsing
- Route matching with path parameters (`/users/:id`)
- Query string and body parsing (form + JSON)
- Static file serving from `pages/` directory
- Keep-alive connections
- Panic recovery
- Graceful shutdown
- TLS/HTTPS support

## Static Files & Custom 404

Create a `pages/` directory for static files:

```
pages/
  404.html     ← Custom 404 page (optional)
```

**Setting up 404.html:**

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

When a route isn't found, the server returns `pages/404.html`. If the file doesn't exist, it returns plain text "Route Not Found".

Static files are served automatically:
- `pages/style.css` → `GET /style.css`
- `pages/app.js` → `GET /app.js`

## Configuration

```go
cfg := server.DefaultConfig()
cfg.ReadTimeout = 30 * time.Second
cfg.MaxBodySize = 10 * 1024 * 1024  // 10MB
cfg.EnableKeepAlive = true

router := server.NewRouterWithConfig(cfg)
```

## Response Helpers

```go
server.Serve400("bad request")     // 400
server.Serve401("unauthorized")    // 401
server.Serve403("forbidden")       // 403
server.Serve500("error")           // 500
server.Serve201("created")         // 201
server.Serve204()                  // 204
```

## Testing

```bash
go test ./server/... -v
```

## Limitations

- Not battle-tested in production
- Single process only
- No middleware system
- No built-in logging/metrics
- Performance degrades beyond 5k concurrent connections

Use net/http for production applications.

## License

MIT



