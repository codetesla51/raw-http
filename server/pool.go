package server

import (
	"bytes"
	"sync"
)

// Buffer pools for reducing allocations

// chunkBufferPool holds 4KB buffers for reading from connections
var chunkBufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 4096)
		return &buf
	},
}

// requestBufferPool holds 8KB buffers for accumulating request headers
var requestBufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 8192)
		return &buf
	},
}

// responseBufferPool holds bytes.Buffer for building responses
var responseBufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// Pool size limits - buffers larger than this are discarded
const (
	maxPoolBufferSize = 16384 // 16KB
)
