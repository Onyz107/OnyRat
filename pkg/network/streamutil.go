package network

import (
	"fmt"
	"sync"
	"time"

	"github.com/Onyz107/onyrat/internal/logger"
	"github.com/xtaci/smux"
)

// StreamManager wraps a smux.Session to make Open/Accept thread-safe
type StreamManager struct {
	session *smux.Session
}

const (
	okName  = 1
	errName = 0
)

var confirmPool = sync.Pool{
	New: func() any {
		return make([]byte, 1)
	},
}

func NewStreamManager(sess *smux.Session) *StreamManager {
	return &StreamManager{
		session: sess,
	}
}

// OpenStream opens a named stream and waits for server acknowledgment
func (m *StreamManager) OpenStream(name string, timeout time.Duration) (*smux.Stream, error) {
	logger.Log.Debugf("Opening stream with name: %s", name)
	buf := confirmPool.Get().([]byte)
	defer confirmPool.Put(buf)

	for {
		stream, err := m.session.OpenStream()
		if err != nil {
			return nil, err
		}
		logger.Log.Debugf("Stream opened, sending name: %s", name)

		if timeout > 0 {
			stream.SetDeadline(time.Now().Add(timeout))
		}

		_, err = stream.Write([]byte(name))
		if err != nil {
			stream.Close()
			return nil, err
		}

		_, err = stream.Read(buf)
		if err != nil {
			stream.Close()
			return nil, err
		}

		logger.Log.Debugf("Received acknowledgment: %s", string(buf))
		if string(buf) == fmt.Sprint(okName) {
			stream.SetDeadline(time.Time{})
			logger.Log.Debugf("Stream accepted by server: %s", name)
			return stream, nil
		}

		// server rejected the stream, retry
		stream.Close()
		logger.Log.Debugf("Stream rejected by server, retrying: %s", name)
	}
}

// AcceptStream accepts a named stream from the client
func (m *StreamManager) AcceptStream(name string, timeout time.Duration) (*smux.Stream, error) {
	logger.Log.Debugf("Waiting to accept stream with name: %s", name)
	buf := make([]byte, len(name))

	for {
		stream, err := m.session.AcceptStream()
		if err != nil {
			return nil, err
		}

		if timeout > 0 {
			stream.SetDeadline(time.Now().Add(timeout))
		}

		_, err = stream.Read(buf)
		if err != nil {
			stream.Close()
			return nil, err
		}

		receivedName := string(buf)
		logger.Log.Debugf("Received stream name: %s", receivedName)

		if receivedName == name {
			logger.Log.Debugf("Stream name matches, sending OK")
			_, err = fmt.Fprint(stream, okName)
			if err != nil {
				stream.Close()
				return nil, err
			}

			stream.SetDeadline(time.Time{})
			return stream, nil
		} else {
			logger.Log.Debug("Stream name mismatch, sending ERR and closing stream")
			_, _ = fmt.Fprint(stream, errName)
			stream.Close()
		}
	}
}
