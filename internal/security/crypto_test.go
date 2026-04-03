package security_test

import (
	"encoding/hex"
	"testing"

	"github.com/sam-liem/quizbot/internal/security"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	plaintext := "Life in the UK test answer"
	ciphertext, err := security.Encrypt(plaintext, key)
	require.NoError(t, err)
	require.NotEmpty(t, ciphertext)

	got, err := security.Decrypt(ciphertext, key)
	require.NoError(t, err)
	require.Equal(t, plaintext, got)
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	for i := range key1 {
		key1[i] = byte(i)
		key2[i] = byte(i + 1)
	}

	ciphertext, err := security.Encrypt("secret", key1)
	require.NoError(t, err)

	_, err = security.Decrypt(ciphertext, key2)
	require.Error(t, err)
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 5)
	}

	ciphertext, err := security.Encrypt("tamper me", key)
	require.NoError(t, err)

	// Decode, flip a byte in the payload area, re-encode
	raw, err := hex.DecodeString(ciphertext)
	require.NoError(t, err)

	// Flip the last byte (part of ciphertext+tag, not nonce)
	raw[len(raw)-1] ^= 0xFF
	tampered := hex.EncodeToString(raw)

	_, err = security.Decrypt(tampered, key)
	require.Error(t, err)
}

func TestEncrypt_NonceUniqueness(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 10)
	}

	plaintext := "same plaintext"
	ct1, err := security.Encrypt(plaintext, key)
	require.NoError(t, err)

	ct2, err := security.Encrypt(plaintext, key)
	require.NoError(t, err)

	require.NotEqual(t, ct1, ct2, "each encryption should use a unique nonce")
}

func TestEncrypt_InvalidKeySize(t *testing.T) {
	tests := []struct {
		name string
		key  []byte
	}{
		{"empty key", []byte{}},
		{"16-byte key", make([]byte, 16)},
		{"31-byte key", make([]byte, 31)},
		{"33-byte key", make([]byte, 33)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := security.Encrypt("hello", tc.key)
			require.Error(t, err)
		})
	}
}
