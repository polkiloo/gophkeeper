package secure

import (
	"bytes"
	"crypto/tls"
	"io"
	"net/http"
)

// NewHTTPClient constructs an HTTP client with encrypted transport.
func NewHTTPClient(key TransportKey, tlsConfig *tls.Config) (*http.Client, error) {
	parsed, err := parseKey(string(key))
	if err != nil {
		return nil, err
	}
	base := http.DefaultTransport.(*http.Transport).Clone()
	if tlsConfig != nil {
		base.TLSClientConfig = tlsConfig
	}
	if len(parsed) == 0 {
		return &http.Client{Transport: base}, nil
	}
	return &http.Client{
		Transport: &clientTransport{key: parsed, base: base},
	}, nil
}

type clientTransport struct {
	key  []byte
	base http.RoundTripper
}

func (t *clientTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, err
		}
		encrypted, err := encrypt(t.key, body)
		if err != nil {
			return nil, err
		}
		encoded := encodePayload(encrypted)
		req.Body = io.NopCloser(bytes.NewReader(encoded))
		req.ContentLength = int64(len(encoded))
	}
	req.Header.Set(headerEncrypted, "1")

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if resp.Header.Get(headerEncrypted) != "1" {
		return resp, nil
	}
	plain, err := decryptResponseBody(t.key, resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		return nil, err
	}
	resp.Body = io.NopCloser(bytes.NewReader(plain))
	resp.ContentLength = int64(len(plain))
	return resp, nil
}

func decryptResponseBody(key []byte, body io.ReadCloser) ([]byte, error) {
	raw, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	decoded, err := decodePayload(raw)
	if err != nil {
		return nil, err
	}
	return decrypt(key, decoded)
}
