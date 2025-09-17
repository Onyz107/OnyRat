//go:build client
// +build client

package handlers

import "sync"

// Small shared buffer pool
var smallBufPool = sync.Pool{
	New: func() any {
		return make([]byte, 1024)
	},
}
