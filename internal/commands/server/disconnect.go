//go:build server
// +build server

package handlers

import (
	"fmt"

	"github.com/Onyz107/onyrat/pkg/network"
	"github.com/abiosoft/ishell"
)

func RegisterDisconnectCommand(s *network.KCPServer, shell *ishell.Shell) {
	shell.AddCmd(&ishell.Cmd{
		Name:    "disconnect",
		Aliases: []string{"kill", "disconnect"},
		Help:    "disconnect a client",
		LongHelp: `
Usage: disconnect <id|address>

Gracefully disconnects the specified client by ID or address.

Aliases: kill, disconnect`,

		Func: func(c *ishell.Context) {
			if len(c.Args) < 1 {
				c.Println("Type `disconnect help` for more information")
				c.HelpText()
				c.Err(fmt.Errorf("not enough arguments"))
				return
			}

			address := c.Args[0]
			if address == "" {
				c.Println("Usage: disconnect <id|address>")
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

			s.DisconnectClient(address)
		},
	})
}
