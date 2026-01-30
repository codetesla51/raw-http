package main

import (
	"fmt"
	"log"
	"os"

	"github.com/codetesla51/raw-http/server"
)

func main() {
	// Create server with HTTPS support
	srv := server.NewServer(":8080")
	srv.EnableTLS(":8443", "server.crt", "server.key")

	// API endpoint with query parameter
	srv.Register("GET", "/data", func(req *server.Request) ([]byte, string) {
		id := req.Query["id"]
		if id == "" {
			return server.Serve400("missing 'id' query parameter")
		}
		data := fmt.Sprintf(`{"id":"%s","info":"Data for id %s"}`, id, id)
		return server.CreateResponseBytes("200", "application/json", "OK", []byte(data))
	})

	// User endpoint with path parameter
	srv.Register("GET", "/users/:id", func(req *server.Request) ([]byte, string) {
		userID := req.PathParams["id"]
		response := []byte("User: " + userID)
		return server.CreateResponseBytes("200", "text/plain", "OK", response)
	})

	// POST endpoint with body parsing
	srv.Register("POST", "/api/create", func(req *server.Request) ([]byte, string) {
		name := req.Body["name"]
		if name == "" {
			return server.Serve400("name field required")
		}
		response := []byte(`{"status":"created","name":"` + name + `"}`)
		return server.CreateResponseBytes("201", "application/json", "Created", response)
	})

	// Health check endpoint
	srv.Register("GET", "/ping", func(req *server.Request) ([]byte, string) {
		return server.CreateResponseBytes("200", "text/plain", "OK", []byte("pong"))
	})

	// Serve index.html at root
	srv.Register("GET", "/", func(req *server.Request) ([]byte, string) {
		content, err := os.ReadFile("pages/index.html")
		if err != nil {
			return server.Serve500("could not load index.html")
		}
		return server.CreateResponseBytes("200", "text/html", "OK", content)
	})

	// Error handling example (panic recovery)
	srv.Register("GET", "/panic", func(req *server.Request) ([]byte, string) {
		panic("test panic - server will recover")
	})

	// Start server (graceful shutdown on Ctrl+C)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
