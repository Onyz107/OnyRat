package videostreaming

import (
	"bytes"
	_ "embed"
	"sync"
)

// The stream for screen sharing
const screenStream = "screenStream"

//go:embed templates/videostream.min.html
var html string

var bufPool = sync.Pool{
	New: func() any {
		return make([]byte, 128*1024)
	},
}

var lazyBufPool = sync.Pool{
	New: func() any {
		return make([]byte, 0, 128*1024)
	},
}

var bytesPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}
