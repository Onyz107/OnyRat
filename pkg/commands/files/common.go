package files

import "sync"

// The stream for file operations (listing files, etc.)
const fileStream = "fileStream"

// Small shared buffer pool
var bufPool = sync.Pool{
	New: func() any {
		return make([]byte, 16*1024)
	},
}
