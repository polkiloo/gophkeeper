package keycloak

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"gophkeeper/internal/config"
)

func TestParseJWTSubject(t *testing.T) {
	payload, _ := json.Marshal(map[string]string{"sub": "user1"})
	token := "a." + base64.RawURLEncoding.EncodeToString(payload) + ".c"

	sub, err := parseJWTSubject(token)
	if err != nil || sub != "user1" {
		t.Fatalf("unexpected subject")
	}

	if _, err := parseJWTSubject("invalid"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestIDFromLocation(t *testing.T) {
	id, err := idFromLocation("http://example.com/users/123")
	if err != nil || id != "123" {
		t.Fatalf("unexpected id")
	}
	if _, err := idFromLocation(""); err == nil {
		t.Fatalf("expected error")
	}
}

func TestReadHTTPError(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(bytes.NewBufferString("bad")),
	}
	err := readHTTPError(resp)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestRequestToken(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(req.Body)
		if !bytes.Contains(body, []byte("client_id=cid")) {
			return nil, errors.New("missing client_id")
		}
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{"access_token":"token","expires_in":60}`)),
			Header:     make(http.Header),
		}
		resp.Header.Set("Content-Type", "application/json")
		return resp, nil
	})}

	svc := &Service{client: client, baseURL: "http://example.com", cfg: configStub()}
	token, exp, err := svc.requestToken(contextStub{}, "realm", "cid", "secret", urlValues("grant_type", "password"), false)
	if err != nil || token != "token" || exp != 60 {
		t.Fatalf("unexpected token result")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

type contextStub struct{}

func (contextStub) Deadline() (deadline time.Time, ok bool) { return time.Time{}, false }
func (contextStub) Done() <-chan struct{}                   { return nil }
func (contextStub) Err() error                              { return nil }
func (contextStub) Value(key interface{}) interface{}       { return nil }

func urlValues(key, value string) url.Values {
	values := url.Values{}
	values.Set(key, value)
	return values
}

func configStub() config.KeycloakConfig {
	return config.KeycloakConfig{
		BaseURL: "http://example.com",
		Realm:   "realm",
	}
}
