// Main package comment block:
/*
HTTP Server Example Application

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

This serves as both a demonstration of the HTTP server package capabilities
and a reference implementation for building web applications on top of
the custom server framework.

Run with: go run main.go
Server will start on http://localhost:8080

Routes:
- GET  /welcome - Welcome page with user info
- GET  /login   - Login form
- POST /login   - Login form submission
*/

package main

import (
	"bytes"
	"github.com/codetesla51/raw-http/server"
	"html/template"
	"log"
	"net"
	"time"
)

func main() {
	// Create TCP listener on port 8080
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Println("error listening to server ", err)
		return
	}
	defer listener.Close()
	log.Println("Server listening at Port: 8080")
	router := server.NewRouter()
	router.Register("GET", "/welcome", homeHandler)
	router.Register("GET", "/login", loginHandler)
	router.Register("POST", "/login", loginHandler)
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
