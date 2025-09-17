//go:build !windows && client
// +build !windows,client

package remoteshell

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/creack/pty"
)

func getHandles() (io.ReadCloser, io.WriteCloser, *exec.Cmd, error) {
	cmd := exec.Command("bash")

	r, err := pty.Start(cmd)
	if err != nil {
		r.Close()
		return nil, nil, nil, fmt.Errorf("failed to start pty: %w", err)
	}

	return r, r, cmd, nil
}
