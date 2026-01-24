package keycloak

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"gophkeeper/internal/config"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/ports/inbound"
)

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time { return c.now }

func TestServiceRegisterLoginValidate(t *testing.T) {
	now := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	token := makeToken("user1")

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case strings.Contains(req.URL.Path, "/realms/gophkeeper/protocol/openid-connect/token/introspect"):
			return jsonResp(http.StatusOK, map[string]any{"active": true, "sub": "user1", "exp": now.Add(time.Hour).Unix()})
		case strings.Contains(req.URL.Path, "/realms/master/protocol/openid-connect/token"):
			return jsonResp(http.StatusOK, map[string]any{"access_token": "admin-token", "expires_in": 60})
		case strings.Contains(req.URL.Path, "/admin/realms/gophkeeper/users") && req.Method == http.MethodPost:
			resp := statusResp(http.StatusCreated)
			resp.Header.Set("Location", "http://example.com/admin/realms/gophkeeper/users/uid123")
			return resp, nil
		case strings.Contains(req.URL.Path, "/admin/realms/gophkeeper/users/uid123") && req.Method == http.MethodGet:
			return jsonResp(http.StatusOK, map[string]any{"id": "uid123", "username": "login"})
		case strings.Contains(req.URL.Path, "/admin/realms/gophkeeper/users/uid123") && req.Method == http.MethodPut:
			return statusResp(http.StatusNoContent), nil
		case strings.Contains(req.URL.Path, "/realms/gophkeeper/protocol/openid-connect/token"):
			form, _ := io.ReadAll(req.Body)
			if !bytes.Contains(form, []byte("username=login")) {
				return statusResp(http.StatusBadRequest), nil
			}
			return jsonResp(http.StatusOK, map[string]any{"access_token": token, "expires_in": 60})
		default:
			return statusResp(http.StatusNotFound), nil
		}
	})}

	svc := NewService(client, fixedClock{now: now}, config.KeycloakConfig{
		BaseURL:  "http://example.com",
		Realm:    "gophkeeper",
		ClientID: "client",
	})

	user, err := svc.Register(context.Background(), inbound.RegisterInput{Login: "login", Password: "pass"})
	if err != nil || user.ID == "" {
		t.Fatalf("register failed: %v", err)
	}

	session, err := svc.Login(context.Background(), inbound.LoginInput{Login: "login", Password: "pass"})
	if err != nil || session.UserID != "user1" {
		t.Fatalf("login failed")
	}

	validated, err := svc.Validate(context.Background(), session.Token)
	if err != nil || validated.UserID != "user1" {
		t.Fatalf("validate failed")
	}
}

func TestValidateUnauthorized(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return statusResp(http.StatusUnauthorized), nil
	})}
	svc := NewService(client, fixedClock{now: time.Now()}, config.KeycloakConfig{
		BaseURL:  "http://example.com",
		Realm:    "gophkeeper",
		ClientID: "client",
	})
	if _, err := svc.Validate(context.Background(), "token"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected unauthorized")
	}
}

func makeToken(sub string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	payload, _ := json.Marshal(map[string]string{"sub": sub})
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}

func jsonResp(status int, payload any) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	resp := statusResp(status)
	resp.Header.Set("Content-Type", "application/json")
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return resp, nil
}

func statusResp(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(nil)),
	}
}
