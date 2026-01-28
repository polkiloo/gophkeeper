package secure

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

const headerEncrypted = "X-Gophkeeper-Enc"

// TransportKey is the shared encryption key.
type TransportKey string

func parseKey(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	decoded, err := base64.RawStdEncoding.DecodeString(value)
	if err == nil {
		if validKeyLength(len(decoded)) {
			return decoded, nil
		}
	}

	raw := []byte(value)
	if validKeyLength(len(raw)) {
		return raw, nil
	}
	return nil, errors.New("invalid transport key length")
}

func validKeyLength(length int) bool {
	return length == 16 || length == 24 || length == 32
}

func encrypt(key []byte, plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, nil
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ciphertext...), nil
}

func decrypt(key []byte, data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(data) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}
	nonce := data[:gcm.NonceSize()]
	ciphertext := data[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func encodePayload(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	out := make([]byte, base64.RawStdEncoding.EncodedLen(len(data)))
	base64.RawStdEncoding.Encode(out, data)
	return out
}

func decodePayload(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}
	out := make([]byte, base64.RawStdEncoding.DecodedLen(len(data)))
	n, err := base64.RawStdEncoding.Decode(out, data)
	if err != nil {
		return nil, err
	}
	return out[:n], nil
}
