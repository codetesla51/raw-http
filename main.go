
/*
HTTP/HTTPS Server Example Application

Author: Uthman Dev
GitHub: https://github.com/codetesla51
Repository: https://github.com/codetesla51/raw-http

Example web application built on top of the custom HTTP server package.
Demonstrates practical usage including template rendering, form handling,
and basic authentication flow.

Features:
- Welcome page with dynamic content
- Login form with POST handling
- Template rendering with current time
- Browser detection display
- HTTPS support with TLS/SSL encryption

The server supports both HTTP and HTTPS protocols. To enable HTTPS, place
your TLS certificate (server.crt) and private key (server.key) in the
application directory. The server will automatically detect these files
and enable HTTPS on port 8443.

This serves as both a demonstration of the HTTP server package capabilities
and a reference implementation for building web applications on top of
the custom server framework.

Run with: go run main.go
Server will start on http://localhost:8080
If TLS certificates are present: https://localhost:8443

Routes:
- GET  /welcome - Welcome page with user info
- GET  /login   - Login form
- POST /login   - Login form submission
*/

package main

import (
	"bytes"
	"crypto/tls"
	"github.com/codetesla51/raw-http/server"
	"html/template"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

func main() {
	// Create TCP listener on port 8080
	port := 8080
	var listener net.Listener
	tlsPort := ":8443"
	hasTLS := false
	var err error
	for {
		addr := ":" + strconv.Itoa(port)
		listener, err = net.Listen("tcp", addr)
		if err == nil {
			break
		}
		log.Printf("port %d currently in use trying %d..\n", port, port+1)
		port++
	}
	defer listener.Close()

	var tlsListener net.Listener
	if fileExists("server.crt") && fileExists("server.key") {
		cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
		if err != nil {
			log.Println("Failed to load TLS certificate:", err)
		} else {
			config := &tls.Config{Certificates: []tls.Certificate{cert}}
			tlsListener, err = tls.Listen("tcp", tlsPort, config)
			if err != nil {
				log.Println("Failed to listen on TLS port:", err)
			} else {
				hasTLS = true
				defer tlsListener.Close()
				log.Println("HTTPS enabled on port 8443")
			}
		}
	}
	log.Printf("Server listening at Port: %d\n", port)
	router := server.NewRouter()
	router.Register("GET", "/welcome", homeHandler)
	router.Register("GET", "/hello", handleHello)
	router.Register("GET", "/login", loginHandler)
	router.Register("POST", "/login", loginHandler)
	router.Register("GET", "/ping", func(req *server.Request) (string, string) {
		return server.CreateResponse("200", "text/plain", "OK", "pong")
	})

if hasTLS {
		log.Println("TLS listener successfully started on 8443")
		go func() {
			for {
				conn, err := tlsListener.Accept()
				if err != nil {
					log.Println("Error accepting TLS connection:", err)
					continue
				}
				go router.RunConnection(conn)
			}
		}()
	}
	// Accept connections in infinite loop
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		// Handle each connection in its own goroutine

		go router.RunConnection(conn)

	}
	

}

func homeHandler(req *server.Request) (response, status string) {
	t, err := template.ParseFiles("pages/welcome.html")
	if err != nil {
		return server.CreateResponse("500", "text/plain", "Error", "Could not load template")
	}
	currentTime := time.Now()
	formattedTime := currentTime.Format("15:04:05")
	data := struct {
		Title   string
		Name    string
		Browser string
		Time    string
	}{
		Title:   "My Home Page",
		Name:    "John Doe",
		Browser: req.Browser,
		Time:    formattedTime,
	}
	var result bytes.Buffer
	err = t.Execute(&result, data)
	if err != nil {
		return server.CreateResponse("500", "text/plain", "Error", "Template error")
	}
	return server.CreateResponse("200", "text/html", "OK",
		result.String())
}

func loginHandler(req *server.Request) (response, status string) {
	var result bytes.Buffer
	if req.Method == "GET" {
		t, err := template.ParseFiles("pages/login.html")
		if err != nil {
			return server.CreateResponse("500", "text/plain", "Error", "Could not load template")
		}
		err = t.Execute(&result, nil)
		return server.CreateResponse("200", "text/html", "OK", result.String())

	} else if req.Method == "POST" {
		username := req.Body["username"]
		password := req.Body["password"]
		if username == "admin" && password == "secret" {
			return server.CreateResponse("200", "text/html", "OK", "<h1>Login Successful!</h1><p>Welcome "+username+"!</p>")
		} else {
			return server.CreateResponse("200", "text/html", "OK", "<h1>Login Failed</h1><p>Wrong username or password</p>")
		}
	}
	return server.CreateResponse("200", "text/html", "OK",
		result.String())
}
func handleHello(req *server.Request) (response, status string) {
	var result bytes.Buffer
	t, err := template.ParseFiles("pages/hello.html")
	if err != nil {
		return server.CreateResponse("500", "text/plain", "Error", "Could not load template")
	}
	err = t.Execute(&result, nil)
	if err != nil {
		return server.CreateResponse("500", "text/plain", "Error", "Template error")
	}
	return server.CreateResponse("200", "text/html", "OK",
		result.String())
}
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}