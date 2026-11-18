package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"regexp"
	"strings"
)

const keyringPrefix = "ENC(v2:"

var keyIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// Keyring encrypts with one active AES-256-GCM key and decrypts with the
// configured key named in each ciphertext. A Keyring created by [NewKeyring]
// is safe for concurrent use. To rotate keys, create a new Keyring.
type Keyring struct {
	activeKeyID string
	keys        map[string]cipher.AEAD
}

// NewKeyring creates a Keyring whose activeKeyID must be present in keys.
// Every key ID must match [A-Za-z0-9_-]{1,64}, and every key must be 32 bytes.
// The returned Keyring does not retain the caller's map or key slices.
// When rotating keys, never bind an existing key ID to different key material.
func NewKeyring(activeKeyID string, keys map[string][]byte) (*Keyring, error) {
	if _, ok := keys[activeKeyID]; !ok {
		return nil, errors.New("crypto: active key ID is not configured")
	}

	aeads := make(map[string]cipher.AEAD, len(keys))
	for id, key := range keys {
		if !keyIDPattern.MatchString(id) {
			return nil, errors.New("crypto: invalid key ID")
		}
		if len(key) != 32 {
			return nil, errors.New("crypto: AES-256 keys must be 32 bytes")
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		aead, err := cipher.NewGCM(block)
		if err != nil {
			return nil, err
		}
		aeads[id] = aead
	}
	return &Keyring{activeKeyID: activeKeyID, keys: aeads}, nil
}

// Encrypt uses the active key and a fresh random nonce to return a ciphertext
// in the form "ENC(v2:<keyID>:<base64(nonce || ciphertext || tag)>)".
// The key ID and aad are authenticated; aad is neither encrypted nor stored.
// Encrypt does not modify plaintext or aad.
func (k *Keyring) Encrypt(plaintext, aad []byte) (string, error) {
	id := k.activeKeyID
	aead := k.keys[id]
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	data := aead.Seal(nonce, nonce, plaintext, keyringAAD(id, aad))
	return keyringPrefix + id + ":" + base64.StdEncoding.EncodeToString(data) + ")", nil
}

// Decrypt accepts only the v2 format produced by [Keyring.Encrypt] and uses
// only the key identified in the header. The same aad must be supplied as
// at encryption. Decrypt does not modify aad and returns nil and an error
// if the format, key ID, or authentication is invalid.
func (k *Keyring) Decrypt(encoded string, aad []byte) ([]byte, error) {
	if !strings.HasPrefix(encoded, keyringPrefix) || !strings.HasSuffix(encoded, ")") {
		return nil, errors.New("crypto: invalid v2 ciphertext envelope")
	}
	id, payload, ok := strings.Cut(encoded[len(keyringPrefix):len(encoded)-1], ":")
	if !ok || !keyIDPattern.MatchString(id) {
		return nil, errors.New("crypto: invalid v2 ciphertext header")
	}
	aead, ok := k.keys[id]
	if !ok {
		return nil, errors.New("crypto: unknown key ID")
	}
	// Even Strict base64 decoding ignores CR and LF; the envelope forbids them.
	if strings.ContainsAny(payload, "\r\n") {
		return nil, errors.New("crypto: invalid v2 ciphertext payload")
	}
	data, err := base64.StdEncoding.Strict().DecodeString(payload)
	if err != nil || len(data) < nonceSize+aead.Overhead() {
		return nil, errors.New("crypto: invalid v2 ciphertext payload")
	}
	plaintext, err := aead.Open(nil, data[:nonceSize], data[nonceSize:], keyringAAD(id, aad))
	if err != nil {
		return nil, errors.New("crypto: v2 ciphertext authentication failed")
	}
	return plaintext, nil
}

func keyringAAD(id string, aad []byte) []byte {
	return append([]byte("saltfishpr/pkg/crypto:v2:"+id+"\x00"), aad...)
}
