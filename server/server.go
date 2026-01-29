package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Server represents an HTTP server with support for graceful shutdown and TLS.
type Server struct {
	Router *Router

	// HTTP settings
	Addr string // Address to listen on (e.g., ":8080")

	// TLS settings (optional)
	TLSAddr     string // TLS address (e.g., ":8443")
	TLSCertFile string // Path to TLS certificate file
	TLSKeyFile  string // Path to TLS key file

	// Internal state
	listener    net.Listener
	tlsListener net.Listener
	mu          sync.Mutex
	running     bool
	shutdownCh  chan struct{}
}

// NewServer creates a new server with default settings.
func NewServer(addr string) *Server {
	return &Server{
		Router:     NewRouter(),
		Addr:       addr,
		shutdownCh: make(chan struct{}),
	}
}

// NewServerWithConfig creates a new server with custom config.
func NewServerWithConfig(addr string, config *Config) *Server {
	return &Server{
		Router:     NewRouterWithConfig(config),
		Addr:       addr,
		shutdownCh: make(chan struct{}),
	}
}

// EnableTLS configures TLS/HTTPS support.
func (s *Server) EnableTLS(addr, certFile, keyFile string) *Server {
	s.TLSAddr = addr
	s.TLSCertFile = certFile
	s.TLSKeyFile = keyFile
	return s
}

// Register is a convenience method to register routes on the server's router.
func (s *Server) Register(method, path string, handler RouteHandler) *Server {
	s.Router.Register(method, path, handler)
	return s
}

// ListenAndServe starts the server and blocks until shutdown.
// It handles graceful shutdown on SIGINT/SIGTERM.
func (s *Server) ListenAndServe() error {
	return s.ListenAndServeContext(context.Background())
}

// ListenAndServeContext starts the server with a custom context for shutdown control.
func (s *Server) ListenAndServeContext(ctx context.Context) error {
	// Setup signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start HTTP listener
	var err error
	s.listener, err = net.Listen("tcp", s.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.Addr, err)
	}
	log.Printf("Server listening on http://localhost%s\n", s.Addr)

	// Start TLS listener if configured
	hasTLS := false
	if s.TLSCertFile != "" && s.TLSKeyFile != "" {
		if FileExists(s.TLSCertFile) && FileExists(s.TLSKeyFile) {
			cert, err := tls.LoadX509KeyPair(s.TLSCertFile, s.TLSKeyFile)
			if err != nil {
				log.Printf("Failed to load TLS certificate: %v\n", err)
			} else {
				tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
				s.tlsListener, err = tls.Listen("tcp", s.TLSAddr, tlsConfig)
				if err != nil {
					log.Printf("Failed to listen on TLS %s: %v\n", s.TLSAddr, err)
				} else {
					hasTLS = true
					log.Printf("TLS server listening on https://localhost%s\n", s.TLSAddr)
				}
			}
		}
	}

	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	// HTTP accept loop
	go s.acceptLoop(s.listener, ctx)

	// HTTPS accept loop
	if hasTLS {
		go s.acceptLoop(s.tlsListener, ctx)
	}

	// Wait for shutdown signal
	<-ctx.Done()
	log.Println("Shutting down server...")

	// Close listeners
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()

	if s.listener != nil {
		s.listener.Close()
	}
	if s.tlsListener != nil {
		s.tlsListener.Close()
	}

	// Give active connections time to finish
	time.Sleep(2 * time.Second)
	log.Println("Server stopped.")

	return nil
}

// acceptLoop accepts and handles connections.
func (s *Server) acceptLoop(listener net.Listener, ctx context.Context) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				// Only log if still running
				s.mu.Lock()
				running := s.running
				s.mu.Unlock()
				if running {
					log.Println("Error accepting connection:", err)
				}
				continue
			}
		}
		go s.Router.RunConnection(conn)
	}
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false

	if s.listener != nil {
		s.listener.Close()
	}
	if s.tlsListener != nil {
		s.tlsListener.Close()
	}

	return nil
}
