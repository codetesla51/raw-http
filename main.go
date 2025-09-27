package main 

import (
  "log"
  "net"
  
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

	// Accept connections in infinite loop
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		// Handle each connection in its own goroutine
		go runConnection(conn)
	}
}
