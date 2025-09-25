package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

func main() {
	// create listner
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("error listening to server ", err)
		return
	}
	defer listener.Close()
	fmt.Println("Server listening at Port: 8080")
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go runConnection(conn)
	}
}
func runConnection(conn net.Conn) {
	defer conn.Close()
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading:", err)
		return
	}
	request := string(buffer[:n])
	fmt.Println("Received request:", request)
	lines := strings.Split(request, "\r\n")
	if len(lines) == 0 {
		fmt.Println("Invalid request")
		return
	}
	firstLine := lines[0]
	headerLines := lines[1:]
	headerMap := make(map[string]string)

	for _, headerLine := range headerLines {
		parts := strings.SplitN(headerLine, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			headerMap[key] = value
		}
	}
	var browserName string
	userAgent := headerMap["User-Agent"]
	if strings.Contains(userAgent, "Chrome") {
		browserName = "Chrome"
	} else if strings.Contains(userAgent, "Firefox") {
		browserName = "Firefox"
	} else if strings.Contains(userAgent, "Safari") {
		browserName = "Safari"
	} else {
		browserName = "Unknown Browser"
	}
	parts := strings.Split(firstLine, " ")
	if len(parts) < 3 {
		fmt.Println("Invalid request line")
		return
	}
	method := parts[0]
	path := parts[1]
	version := parts[2]
	fmt.Printf("Method: %s, Path: %s, Version: %s\n", method, path, version)
	var response string
	switch method {
	case "GET":
		switch path {
		case "/hello":
			response = createResponse("200", "OK", "Hello "+browserName+" user!")
		case "/time":
			currentTime := time.Now()
			formattedTime := currentTime.Format("15:04:05")
			response = createResponse("200", "OK", formattedTime)
		default:
			response = createResponse("404", "Not Found", "Route Not found")
		}
	case "POST":
		switch path {
		case "/test":
			response = createResponse("200", "OK", "Hello from test Post")
		default:
			response = createResponse("404", "Not Found", "Route Not found")
		}
	default:
		response = createResponse("405", "Method Not Allowed", "Method not supported")
	}
	_, err = conn.Write([]byte(response))
	if err != nil {
		fmt.Println("Error writing response:", err)
		return
	}

	fmt.Println("Response sent successfully")
}
func createResponse(statusCode, statusMessage, body string) string {
	return "HTTP/1.1 " + statusCode + " " + statusMessage +
		"\r\nContent-Type: text/html" +
		"\r\nContent-Length: " + strconv.Itoa(len(body)) +
		"\r\n\r\n" + body
}
