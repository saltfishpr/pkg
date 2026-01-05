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

const nonceSize = 12 // GCM 推荐的 nonce 长度为 12 字节

// Cryptor 加密解密器
type Cryptor struct {
	key []byte
}

// New 创建一个新的加密解密器
func New(key []byte) *Cryptor {
	return &Cryptor{key: key}
}

// Encrypt 使用 AES-GCM 加密
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

	// 拼接 nonce 和密文
	result := make([]byte, len(nonce)+len(ciphertext))
	copy(result[:len(nonce)], nonce)
	copy(result[len(nonce):], ciphertext)

	return c.wrap(base64.StdEncoding.EncodeToString(result)), nil
}

// Decrypt 使用 AES-GCM 解密
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

func (c *Cryptor) wrap(s string) string {
	return fmt.Sprintf("ENC(%s)", s)
}

func (c *Cryptor) unwrap(s string) (string, bool) {
	if strings.HasPrefix(s, "ENC(") && strings.HasSuffix(s, ")") {
		s = s[4 : len(s)-1]
		return s, true
	}
	return s, false
}
