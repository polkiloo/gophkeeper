package secure

import (
	"bytes"
	"testing"
)

func TestParseKey(t *testing.T) {
	key, err := parseKey("0123456789abcdef")
	if err != nil || len(key) != 16 {
		t.Fatalf("expected raw key")
	}

	key, err = parseKey("AAECAwQFBgcICQoLDA0ODw")
	if err != nil || len(key) != 16 {
		t.Fatalf("expected base64 key")
	}

	if _, err := parseKey("short"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key, _ := parseKey("0123456789abcdef")
	plain := []byte("secret")

	enc, err := encrypt(key, plain)
	if err != nil {
		t.Fatalf("encrypt error: %v", err)
	}
	dec, err := decrypt(key, enc)
	if err != nil {
		t.Fatalf("decrypt error: %v", err)
	}
	if !bytes.Equal(dec, plain) {
		t.Fatalf("unexpected plaintext")
	}
}

func TestDecodeEncodePayload(t *testing.T) {
	raw := []byte("payload")
	encoded := encodePayload(raw)
	decoded, err := decodePayload(encoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !bytes.Equal(decoded, raw) {
		t.Fatalf("unexpected decoded data")
	}
}

func TestDecryptInvalidPayload(t *testing.T) {
	key, _ := parseKey("0123456789abcdef")
	if _, err := decrypt(key, []byte("short")); err == nil {
		t.Fatalf("expected error")
	}
}
