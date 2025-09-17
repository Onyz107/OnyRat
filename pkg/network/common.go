package network

import (
	"sync"
	"time"

	"github.com/xtaci/smux"
)

const (
	// Stream name for heartbeat messages
	heartbeatStream = "heartbeatStream"
)

// deadlineWriter sets a per-write dedline on the underlying stream.
type deadlineWriter struct {
	s   *smux.Stream
	dur time.Duration
}

// Write writes bytes to the underlying stream with a timeout set by dur.
func (w *deadlineWriter) Write(p []byte) (int, error) {
	if w.dur > 0 {
		_ = w.s.SetWriteDeadline(time.Now().Add(w.dur))
	}
	defer w.s.SetWriteDeadline(time.Time{})
	return w.s.Write(p)
}

// Close closes the underlying stream.
func (w *deadlineWriter) Close() error { return w.s.Close() }

// deadlineReader sets a per-read deadline on the underlying stream.
type deadlineReader struct {
	s   *smux.Stream
	dur time.Duration
}

// Read reads bytes from the underlying stream with a timeout set by dur.
func (r *deadlineReader) Read(p []byte) (int, error) {
	if r.dur > 0 {
		_ = r.s.SetReadDeadline(time.Now().Add(r.dur))
	}
	defer r.s.SetReadDeadline(time.Time{})
	return r.s.Read(p)
}

// Close closes the underlying stream.
func (r *deadlineReader) Close() error { return r.s.Close() }

// headerPool is a sync.Pool for reusable 8-byte buffers for length headers.
var headerPool = sync.Pool{
	New: func() any {
		return make([]byte, 8)
	},
}
