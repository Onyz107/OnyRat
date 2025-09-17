package network

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/xtaci/kcp-go/v5"
	"github.com/xtaci/smux"
)

type KCPServer struct {
	Listener *kcp.Listener
	Ctx      context.Context
	cancel   context.CancelCauseFunc
	clients  map[string]*KCPClient
	mu       sync.RWMutex
}

func NewServer(addr string, ctx context.Context) (*KCPServer, error) {
	inCtx, cancel := context.WithCancelCause(ctx)

	conn, err := kcp.ListenWithOptions(addr, nil, 0, 0)
	if err != nil {
		cancel(fmt.Errorf("failed to listen on %s: %w", addr, err))
		return nil, fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	server := &KCPServer{
		Listener: conn,
		Ctx:      inCtx,
		cancel:   cancel,
		clients:  make(map[string]*KCPClient),
	}

	return server, nil
}

func (s *KCPServer) AcceptClient() (string, error) {
	inCtx, cancel := context.WithCancelCause(s.Ctx)

	conn, err := s.Listener.AcceptKCP()
	if err != nil {
		cancel(fmt.Errorf("failed to accept client: %w", err))
		return "", fmt.Errorf("failed to accept client: %w", err)
	}

	// Performance optimizations
	conn.SetWindowSize(512, 512)
	conn.SetNoDelay(1, 40, 2, 1)

	sess, err := smux.Server(conn, nil)
	if err != nil {
		cancel(fmt.Errorf("failed to create a session: %w", err))
		return "", fmt.Errorf("failed to create a session: %w", err)
	}
	streamMan := NewStreamManager(sess)

	client := &KCPClient{
		Sess:     conn,
		LastSeen: time.Now(),
		ID:       len(s.clients),
		Manager:  streamMan,
		Ctx:      inCtx,
		cancel:   cancel,
	}

	s.mu.Lock()
	s.clients[conn.RemoteAddr().String()] = client
	s.mu.Unlock()

	return conn.RemoteAddr().String(), nil
}

func (s *KCPServer) GetClients() map[string]*KCPClient {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.clients
}

func (s *KCPServer) GetClient(addr string) *KCPClient {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, ok := s.clients[addr]
	if !ok {
		return nil
	}
	return client
}

func (s *KCPServer) DisconnectClient(addr string) {
	client := s.GetClient(addr)
	if client == nil {
		return
	}

	client.Disconnect()
}

func (s *KCPServer) CloseClient(addr string) {
	client := s.GetClient(addr)
	if client == nil {
		return
	}
	s.mu.Lock()
	delete(s.clients, addr)
	s.mu.Unlock()
	client.Close()
}

func (s *KCPServer) Close() error {
	for addr := range s.GetClients() {
		s.CloseClient(addr)
	}
	return s.Listener.Close()
}

func (s *KCPServer) Send(stream *smux.Stream, data []byte, timeout time.Duration) error {
	select {

	case <-s.Ctx.Done():
		return fmt.Errorf("connection closed: %w", context.Cause(s.Ctx))

	default:
		return send(stream, data, timeout)

	}
}

func (s *KCPServer) NewStreamedSender(stream *smux.Stream, timeout time.Duration) (io.WriteCloser, error) {
	select {

	case <-s.Ctx.Done():
		return nil, fmt.Errorf("connection closed: %w", context.Cause(s.Ctx))

	default:
		return newStreamedSender(stream, timeout), nil

	}
}

func (s *KCPServer) SendSerialized(stream *smux.Stream, data []byte, timeout time.Duration) error {
	select {

	case <-s.Ctx.Done():
		return fmt.Errorf("connection closed: %w", context.Cause(s.Ctx))

	default:
		return sendSerialized(stream, data, timeout)

	}
}

func (s *KCPServer) SendEncrypted(stream *smux.Stream, data []byte, timeout time.Duration) error {
	addr := stream.RemoteAddr().String()
	client := s.GetClient(addr)

	select {

	case <-s.Ctx.Done():
		return fmt.Errorf("connection closed: %w", context.Cause(s.Ctx))

	default:
		return sendEncrypted(stream, data, client.AESKey, timeout)

	}
}

func (s *KCPServer) NewStreamedEncryptedSender(stream *smux.Stream, timeout time.Duration) (io.WriteCloser, error) {
	addr := stream.RemoteAddr().String()
	client := s.GetClient(addr)

	select {

	case <-s.Ctx.Done():
		return nil, fmt.Errorf("connection closed: %w", context.Cause(s.Ctx))

	default:
		return newStreamedEncryptedSender(stream, client.AESKey, timeout)

	}
}

func (s *KCPServer) Receive(stream *smux.Stream, buf []byte, timeout time.Duration) error {
	select {

	case <-s.Ctx.Done():
		return fmt.Errorf("connection closed: %w", context.Cause(s.Ctx))

	default:
		return receive(stream, buf, timeout)

	}
}

func (s *KCPServer) NewStreamedReceiver(stream *smux.Stream, timeout time.Duration) (io.ReadCloser, error) {
	select {

	case <-s.Ctx.Done():
		return nil, fmt.Errorf("connection closed: %w", context.Cause(s.Ctx))

	default:
		return newStreamedReceiver(stream, timeout), nil

	}
}

func (s *KCPServer) ReceiveSerialized(stream *smux.Stream, buf []byte, timeout time.Duration) (uint64, error) {
	select {

	case <-s.Ctx.Done():
		return 0, fmt.Errorf("connection closed: %w", context.Cause(s.Ctx))

	default:
		return receiveSerialized(stream, buf, timeout)

	}
}

func (s *KCPServer) ReceiveEncrypted(stream *smux.Stream, buf []byte, timeout time.Duration) (uint64, error) {
	addr := stream.RemoteAddr().String()
	client := s.GetClient(addr)

	select {

	case <-s.Ctx.Done():
		return 0, fmt.Errorf("connection closed: %w", context.Cause(s.Ctx))

	default:
		return receiveEncrypted(stream, buf, client.AESKey, timeout)

	}
}

func (s *KCPServer) NewStreamedEncryptedReceiver(stream *smux.Stream, timeout time.Duration) (io.ReadCloser, error) {
	addr := stream.RemoteAddr().String()
	client := s.GetClient(addr)

	select {

	case <-s.Ctx.Done():
		return nil, fmt.Errorf("connection closed: %w", context.Cause(s.Ctx))

	default:
		return newStreamedEncryptedReceiver(stream, client.AESKey, timeout)

	}
}
