//go:build server
// +build server

package auth

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/Onyz107/onyrat/internal/logger"
	"github.com/Onyz107/onyrat/pkg/network"
	"github.com/xtaci/smux"
)

// Server-side client authorization
func ClientAuthorization(s *network.KCPServer, manager *network.StreamManager, privateKey string) error {
	blockPriv, _ := pem.Decode([]byte(privateKey))
	if blockPriv == nil {
		return fmt.Errorf("failed to parse private key")
	}

	priv, err := x509.ParsePKCS8PrivateKey(blockPriv.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}
	key, ok := priv.(*rsa.PrivateKey)
	if !ok {
		return fmt.Errorf("private key is not RSA")
	}

	var errs []error
	for addr, client := range s.GetClients() {
		if client.Authorized {
			continue
		}

		aesKey, err := authorizeClient(s, client, key)
		if err != nil {
			s.CloseClient(client.Sess.RemoteAddr().String())
			errs = append(errs, fmt.Errorf("failed to authorize client: %w", err))
			continue
		}

		client.Authorized = true
		client.AESKey = aesKey
		logger.Log.Infof("Client: %s is successfully authorized.", addr)
	}

	return errors.Join(errs...)
}

// This function receives an encrypted AES key from the client, decrypts it, receives a challenge from the client
// signs the challenge and sends it back. If the server fails to decrypt the AES key that means that RSA key
// pair between the server and the client do not match therefore an error is returned.
func authorizeClient(s *network.KCPServer, c *network.KCPClient, key *rsa.PrivateKey) ([]byte, error) {
	addr := c.Sess.RemoteAddr().String()

	stream, err := c.Manager.AcceptStream(authStream, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to accept the authorization stream from client: %s: %w", addr, err)
	}
	defer stream.Close()

	aesKey, err := handleClientAESKey(s, c, stream, key)
	if err != nil {
		return nil, fmt.Errorf("failed to handle AES key from client: %s: %w", addr, err)
	}

	if err := handleSignature(s, c, stream, key); err != nil {
		return nil, fmt.Errorf("failed to sign challenge from client: %s: %w", addr, err)
	}

	return aesKey, nil
}

func handleClientAESKey(s *network.KCPServer, c *network.KCPClient, stream *smux.Stream, key *rsa.PrivateKey) ([]byte, error) {
	buf := bufPool.Get().([]byte)
	defer bufPool.Put(buf)

	addr := c.Sess.RemoteAddr().String()

	n, err := s.ReceiveSerialized(stream, buf, 15*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to recieve AES key from client: %s: %w", addr, err)
	}
	challenge := buf[:n]

	plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, key, challenge, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt AES key from client: %s: %w", addr, err)
	}

	return plaintext, nil
}

func handleSignature(s *network.KCPServer, c *network.KCPClient, stream *smux.Stream, key *rsa.PrivateKey) error {
	buf := bufPool.Get().([]byte)
	defer bufPool.Put(buf)

	addr := c.Sess.RemoteAddr().String()

	n, err := s.ReceiveSerialized(stream, buf, 15*time.Second)
	if err != nil {
		return fmt.Errorf("failed to receive challenge from client: %s: %w", addr, err)
	}
	challenge := buf[:n]

	hash := sha256.Sum256(challenge)
	signature, err := rsa.SignPKCS1v15(nil, key, crypto.SHA256, hash[:]) // The random parameter is legacy and can be ignored according to docs
	if err != nil {
		return fmt.Errorf("failed to sign challenge from client: %s: %w", addr, err)
	}

	if err := s.SendSerialized(stream, signature, 15*time.Second); err != nil {
		return fmt.Errorf("failed to send signed challenge to client: %s: %w", addr, err)
	}

	return nil
}
