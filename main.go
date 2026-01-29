package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/codetesla51/raw-http/server"
)

func main() {
	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create TCP listener on port 8080 (fallback if busy)
	port := 8080
	var listener net.Listener
	var err error
	for {
		addr := ":" + strconv.Itoa(port)
		listener, err = net.Listen("tcp", addr)
		if err == nil {
			break
		}
		log.Printf("Port %d currently in use, trying %d...\n", port, port+1)
		port++
	}
	defer listener.Close()
	log.Printf("Server listening on http://localhost:%d\n", port)

	// Setup HTTPS listener if certs exist
	hasTLS := false
	var tlsListener net.Listener
	if server.FileExists("server.crt") && server.FileExists("server.key") {
		cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
		if err != nil {
			log.Println("Failed to load TLS certificate:", err)
		} else {
			config := &tls.Config{Certificates: []tls.Certificate{cert}}
			tlsListener, err = tls.Listen("tcp", ":8443", config)
			if err != nil {
				log.Println("Failed to listen on TLS port 8443:", err)
			} else {
				hasTLS = true
				defer tlsListener.Close()
				log.Println("TLS listener successfully started on https://localhost:8443")
			}
		}
	}

	// Create router and register routes
	router := server.NewRouter()

	// API endpoint with query parameter
	router.Register("GET", "/data", func(req *server.Request) ([]byte, string) {
		id := req.Query["id"]
		if id == "" {
			return server.Serve400("missing 'id' query parameter")
		}
		data := fmt.Sprintf(`{"id":"%s","info":"Data for id %s"}`, id, id)
		return server.CreateResponseBytes("200", "application/json", "OK", []byte(data))
	})

	// User endpoint with path parameter
	router.Register("GET", "/users/:id", func(req *server.Request) ([]byte, string) {
		userID := req.PathParams["id"]
		response := []byte("User: " + userID)
		return server.CreateResponseBytes("200", "text/plain", "OK", response)
	})

	// POST endpoint with body parsing
	router.Register("POST", "/api/create", func(req *server.Request) ([]byte, string) {
		name := req.Body["name"]
		if name == "" {
			return server.Serve400("name field required")
		}
		response := []byte(`{"status":"created","name":"` + name + `"}`)
		return server.CreateResponseBytes("201", "application/json", "Created", response)
	})

	// Health check endpoint
	router.Register("GET", "/ping", func(req *server.Request) ([]byte, string) {
		return server.CreateResponseBytes("200", "text/plain", "OK", []byte("pong"))
	})

	// Error handling example (panic recovery)
	router.Register("GET", "/panic", func(req *server.Request) ([]byte, string) {
		panic("test panic - server will recover")
	})

	// HTTP listener
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					log.Println("Error accepting connection:", err)
					continue
				}
			}
			go router.RunConnection(conn)
		}
	}()

	// HTTPS listener
	if hasTLS {
		go func() {
			for {
				conn, err := tlsListener.Accept()
				if err != nil {
					select {
					case <-ctx.Done():
						return
					default:
						log.Println("Error accepting TLS connection:", err)
						continue
					}
				}
				go router.RunConnection(conn)
			}
		}()
	}

	// Wait for shutdown
	<-ctx.Done()
	log.Println("Shutting down server...")
	listener.Close()
	if hasTLS {
		tlsListener.Close()
	}
	time.Sleep(2 * time.Second)
	log.Println("Server stopped.")
}
