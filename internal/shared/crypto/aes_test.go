package crypto_test

import (
	"testing"

	"github.com/skenzeriq/patchiq/internal/shared/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := crypto.GenerateKey()
	plaintext := []byte("smtp://user:password@smtp.example.com:587")
	ciphertext, err := crypto.Encrypt(key, plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)
	decrypted, err := crypto.Decrypt(key, ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := crypto.GenerateKey()
	key2 := crypto.GenerateKey()
	plaintext := []byte("secret-webhook-url")
	ciphertext, err := crypto.Encrypt(key1, plaintext)
	require.NoError(t, err)
	_, err = crypto.Decrypt(key2, ciphertext)
	assert.Error(t, err)
}

func TestEncrypt_DifferentCiphertextEachTime(t *testing.T) {
	key := crypto.GenerateKey()
	plaintext := []byte("same input")
	c1, err := crypto.Encrypt(key, plaintext)
	require.NoError(t, err)
	c2, err := crypto.Encrypt(key, plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, c1, c2, "AES-GCM should produce different ciphertext due to random nonce")
}
