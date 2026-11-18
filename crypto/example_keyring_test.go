package crypto_test

import (
	"bytes"
	"fmt"

	"github.com/saltfishpr/pkg/crypto"
)

func ExampleKeyring_rotation() {
	// Use securely generated keys in production, not these example values.
	k1 := bytes.Repeat([]byte{1}, 32)
	k2 := bytes.Repeat([]byte{2}, 32)
	aad := []byte("archive/records/2026/report")

	// First, deploy both keys to every reader while writes still use k1.
	keys := map[string][]byte{"k1": k1, "k2": k2}
	reader, err := crypto.NewKeyring("k1", keys)
	if err != nil {
		panic(err)
	}
	oldCiphertext, err := reader.Encrypt([]byte("written with k1"), aad)
	if err != nil {
		panic(err)
	}

	// Once every reader knows k2, switch writers to k2. Keep k1 for old data
	// and never reuse either ID for different key material.
	writer, err := crypto.NewKeyring("k2", keys)
	if err != nil {
		panic(err)
	}
	newCiphertext, err := writer.Encrypt([]byte("written with k2"), aad)
	if err != nil {
		panic(err)
	}
	for _, ciphertext := range []string{oldCiphertext, newCiphertext} {
		plaintext, err := reader.Decrypt(ciphertext, aad)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(plaintext))
	}

	// Output:
	// written with k1
	// written with k2
}
