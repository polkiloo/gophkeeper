package secure

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"
)

type errReadCloser struct{}

func (errReadCloser) Read([]byte) (int, error) { return 0, errors.New("read error") }
func (errReadCloser) Close() error             { return nil }

func TestNewHTTPClientEmptyKey(t *testing.T) {
	client, err := NewHTTPClient("", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := client.Transport.(*clientTransport); ok {
		t.Fatalf("expected base transport")
	}
}

func TestClientTransportNoEncryptedResponse(t *testing.T) {
	key := TransportKey("0123456789abcdef")
	parsed, _ := parseKey(string(key))
	tr := &clientTransport{
		key: parsed,
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewReader([]byte("plain"))),
			}, nil
		}),
	}

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("roundtrip error: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "plain" {
		t.Fatalf("expected plain body")
	}
}

func TestClientTransportRequestReadError(t *testing.T) {
	key := TransportKey("0123456789abcdef")
	parsed, _ := parseKey(string(key))
	tr := &clientTransport{
		key:  parsed,
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) { return nil, nil }),
	}

	req, _ := http.NewRequest(http.MethodPost, "http://example.com", errReadCloser{})
	if _, err := tr.RoundTrip(req); err == nil {
		t.Fatalf("expected error")
	}
}

func TestDecryptResponseBodyErrors(t *testing.T) {
	if _, err := decryptResponseBody([]byte("key"), errReadCloser{}); err == nil {
		t.Fatalf("expected read error")
	}
	body := io.NopCloser(bytes.NewReader([]byte("bad")))
	if _, err := decryptResponseBody([]byte("key"), body); err == nil {
		t.Fatalf("expected decode error")
	}
}
