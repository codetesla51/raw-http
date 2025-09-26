package main

import (
	"errors"
	"fmt"
	"net"
	"os"
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
	requestParts := strings.SplitN(request, "\r\n\r\n", 2)
	headerSection := requestParts[0]
	body := ""
	bodyMap := make(map[string]string)

	if len(requestParts) > 1 {
		body = requestParts[1]
		pairs := strings.Split(body, "&")
		for _, pair := range pairs {
			parts := strings.Split(pair, "=")
			if len(parts) == 2 {
				key := parts[0]
				value := parts[1]
				bodyMap[key] = value
			}
		}
	}
	lines := strings.Split(headerSection, "\r\n")
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
	var filePath string
	if path == "/" {
		filePath = "pages/index.html"
	} else {
		filePath = "pages" + path
	}
	switch method {
	case "GET":
		if fileExists(filePath) {
			content, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Printf("Error reading file: %v\n", err)
			}
			response = createResponse("200", "OK", string(content))
		} else {
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
		}
	case "POST":
		switch path {
		case "/test":
			username := getFormValue(bodyMap, "username", "anonymous")
			password := getFormValue(bodyMap, "password", "")
			response = createResponse("200", "OK", username+password)
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
func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	fmt.Printf("Error checking file %s: %v\n", filePath, err)
	return false
}
func getFormValue(bodyMap map[string]string, key string, defaultValue string) string {
	if value, exists := bodyMap[key]; exists {
		return value
	}
	return defaultValue
}
