package server

import "time"

type Config struct {
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	MaxHeaderSize   int
	MaxBodySize     int64
	EnableKeepAlive bool
	EnableLogging   bool
}

func DefaultConfig() *Config {
	return &Config{
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     120 * time.Second,
		MaxHeaderSize:   8192,
		MaxBodySize:     10 * 1024 * 1024, // 10MB
		EnableKeepAlive: true,
		EnableLogging:   false,
	}
}
