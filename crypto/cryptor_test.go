package crypto

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

var key, _ = base64.StdEncoding.DecodeString("f1gKitOJ3Embg8zM6DejnEafFI7gsIFeXwFlSHZCuf0=")

func TestCryptor(t *testing.T) {
	cryptor := New(key)
	plaintext := []byte("lebensoft")
	ciphertext, err := cryptor.Encrypt(plaintext)
	assert.NoError(t, err)
	decrypted, err := cryptor.Decrypt(ciphertext)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}
