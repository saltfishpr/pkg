// Package crypto provides AES-256-GCM authenticated encryption with a
// simple Encrypt/Decrypt API.
//
// Ciphertext is formatted as "ENC(<base64>)" where the base64 payload
// contains a 12-byte random nonce followed by the GCM ciphertext and tag.
// [Cryptor.Decrypt] transparently handles both wrapped ("ENC(...)") and
// plain-text inputs, returning plain text unchanged.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
)

// nonceSize is the recommended nonce length for AES-GCM (96 bits).
const nonceSize = 12

// Cryptor performs AES-256-GCM encryption and decryption using a fixed key.
type Cryptor struct {
	key []byte
}

// New creates a [Cryptor] with the given AES key.
// The key must be 16, 24, or 32 bytes for AES-128, AES-192, or AES-256.
func New(key []byte) *Cryptor {
	return &Cryptor{key: key}
}

// Encrypt encrypts plaintext with AES-GCM and returns the result as
// "ENC(<base64(nonce + ciphertext)>)".
func (c *Cryptor) Encrypt(plaintext []byte) (string, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", errors.WithStack(err)
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", errors.WithStack(err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", errors.WithStack(err)
	}

	ciphertext := aesGCM.Seal(nil, nonce, plaintext, nil)

	result := make([]byte, len(nonce)+len(ciphertext))
	copy(result[:len(nonce)], nonce)
	copy(result[len(nonce):], ciphertext)

	return c.wrap(base64.StdEncoding.EncodeToString(result)), nil
}

// Decrypt decrypts an "ENC(...)" encoded string produced by [Cryptor.Encrypt].
// If the input is not wrapped, it is returned as-is, allowing transparent
// handling of unencrypted legacy data.
func (c *Cryptor) Decrypt(encoded string) ([]byte, error) {
	encoded, ok := c.unwrap(encoded)
	if !ok {
		return []byte(encoded), nil
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return plaintext, nil
}

// wrap envelopes a base64 string in the "ENC(...)" marker.
func (c *Cryptor) wrap(s string) string {
	return fmt.Sprintf("ENC(%s)", s)
}

// unwrap strips the "ENC(...)" envelope. It returns the inner string and true
// on success, or the original string and false if the envelope is absent.
func (c *Cryptor) unwrap(s string) (string, bool) {
	if strings.HasPrefix(s, "ENC(") && strings.HasSuffix(s, ")") {
		s = s[4 : len(s)-1]
		return s, true
	}
	return s, false
}
