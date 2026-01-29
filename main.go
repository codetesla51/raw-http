package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
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

	router.Register("GET", "/welcome", homeHandler)
	router.Register("GET", "/hello", handleHello)
	router.Register("GET", "/login", loginHandler)
	router.Register("POST", "/login", loginHandler)
	router.Register("GET", "/ping", func(req *server.Request) ([]byte, string) {
		return server.CreateResponseBytes("200", "text/plain", "OK", []byte("pong"))
	})
	router.Register("GET", "/users/:id", func(req *server.Request) ([]byte, string) {
		userId := req.PathParams["id"]
		response := []byte("user id:" + userId)
		return server.CreateResponseBytes("200", "text/plain", "OK", response)
	})
	router.Register("GET", "/panic", func(req *server.Request) ([]byte, string) {
		panic("test panic")
	})
	router.Register("GET", "/data", func(req *server.Request) (response []byte, status string) {
		id := req.Query["id"]
		if id == "" {
			return server.Serve400("Missing 'id' query parameter")
		}
		data := processData(id)
		return server.CreateResponseBytes("200", "application/json", "OK", data)
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

func processData(id string) []byte {
	return []byte(fmt.Sprintf(`{"id":"%s","info":"This is some data related to id %s"}`, id, id))
}

func homeHandler(req *server.Request) ([]byte, string) {
	t, err := template.ParseFiles("pages/welcome.html")
	if err != nil {
		return server.CreateResponseBytes("500", "text/plain", "Error", []byte("Could not load template"))
	}

	currentTime := time.Now()
	formattedTime := currentTime.Format("15:04:05")
	currentDate := currentTime.Weekday().String()

	data := struct {
		Title   string
		Name    string
		Browser string
		Time    string
		Day     string
	}{
		Title:   "My Home Page",
		Name:    "John Doe",
		Browser: req.Browser,
		Time:    formattedTime,
		Day:     currentDate,
	}

	var result bytes.Buffer
	err = t.Execute(&result, data)
	if err != nil {
		return server.CreateResponseBytes("500", "text/plain", "Error", []byte("Template error"))
	}
	return server.CreateResponseBytes("200", "text/html", "OK", result.Bytes())
}

func loginHandler(req *server.Request) ([]byte, string) {
	var result bytes.Buffer
	if req.Method == "GET" {
		t, err := template.ParseFiles("pages/login.html")
		if err != nil {
			return server.CreateResponseBytes("500", "text/plain", "Error", []byte("Could not load template"))
		}
		t.Execute(&result, nil)
		return server.CreateResponseBytes("200", "text/html", "OK", result.Bytes())
	}

	if req.Method == "POST" {
		username := req.Body["username"]
		password := req.Body["password"]
		if username == "admin" && password == "secret" {
			response := "<h1>Login Successful!</h1><p>Welcome " + username + "!</p>"
			return server.CreateResponseBytes("200", "text/html", "OK", []byte(response))
		}
		return server.CreateResponseBytes("200", "text/html", "OK",
			[]byte("<h1>Login Failed</h1><p>Wrong username or password</p>"))
	}
	return server.CreateResponseBytes("200", "text/html", "OK", result.Bytes())
}

func handleHello(req *server.Request) ([]byte, string) {
	var result bytes.Buffer
	t, err := template.ParseFiles("pages/hello.html")
	if err != nil {
		return server.CreateResponseBytes("500", "text/plain", "Error", []byte("Could not load template"))
	}
	err = t.Execute(&result, nil)
	if err != nil {
		return server.CreateResponseBytes("500", "text/plain", "Error", []byte("Template error"))
	}
	return server.CreateResponseBytes("200", "text/html", "OK", result.Bytes())
}
