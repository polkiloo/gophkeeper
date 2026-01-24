package secure

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

func TestClientTransportEncryptsRequest(t *testing.T) {
	key := TransportKey("0123456789abcdef")
	parsed, _ := parseKey(string(key))

	client, err := NewHTTPClient(key, nil)
	if err != nil {
		t.Fatalf("client error: %v", err)
	}

	base := client.Transport.(*clientTransport)
	base.base = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get(headerEncrypted) != "1" {
			t.Fatalf("missing encryption header")
		}
		raw, _ := io.ReadAll(req.Body)
		decoded, err := decodePayload(raw)
		if err != nil {
			t.Fatalf("decode error: %v", err)
		}
		plain, err := decrypt(parsed, decoded)
		if err != nil {
			t.Fatalf("decrypt error: %v", err)
		}
		if string(plain) != "ping" {
			t.Fatalf("unexpected request body")
		}

		respPlain := []byte(`{"ok":true}`)
		enc, _ := encrypt(parsed, respPlain)
		respPayload := encodePayload(enc)
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(respPayload)),
		}
		resp.Header.Set(headerEncrypted, "1")
		return resp, nil
	})

	req, _ := http.NewRequest(http.MethodPost, "http://example.com", bytes.NewBufferString("ping"))
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"ok":true}` {
		t.Fatalf("unexpected response body")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
