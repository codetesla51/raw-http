package server

import (
	"log"

	"github.com/fatih/color"
)

// logRequest logs an HTTP request with color-coded status
func logRequest(method, path, status string) {
	switch status {
	case "200":
		log.Print(color.GreenString("%s %s %s", method, path, status))
	case "404", "403", "405":
		log.Print(color.RedString("%s %s %s", method, path, status))
	default:
		log.Printf("%s %s %s", method, path, status)
	}
}
