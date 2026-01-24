package keycloak

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gophkeeper/internal/config"
	"gophkeeper/internal/domain"
	"gophkeeper/internal/ports/inbound"
	"gophkeeper/internal/ports/outbound"
)

const adminRealm = "master"

// Service implements authentication via Keycloak.
type Service struct {
	client  *http.Client
	clock   outbound.Clock
	cfg     config.KeycloakConfig
	baseURL string
}

// NewService creates a Keycloak-backed authentication service.
func NewService(client *http.Client, clock outbound.Clock, cfg config.KeycloakConfig) *Service {
	if client == nil {
		client = http.DefaultClient
	}
	return &Service{
		client:  client,
		clock:   clock,
		cfg:     cfg,
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
	}
}

// Register creates a new user account in Keycloak.
func (s *Service) Register(ctx context.Context, input inbound.RegisterInput) (domain.User, error) {
	login := strings.TrimSpace(input.Login)
	password := strings.TrimSpace(input.Password)
	if login == "" || password == "" {
		return domain.User{}, errors.New("login and password are required")
	}

	adminToken, err := s.adminToken(ctx)
	if err != nil {
		return domain.User{}, err
	}

	createPayload := struct {
		Username        string   `json:"username"`
		Enabled         bool     `json:"enabled"`
		Email           string   `json:"email,omitempty"`
		FirstName       string   `json:"firstName,omitempty"`
		LastName        string   `json:"lastName,omitempty"`
		EmailVerified   bool     `json:"emailVerified"`
		RequiredActions []string `json:"requiredActions"`
		Credentials     []struct {
			Type      string `json:"type"`
			Value     string `json:"value"`
			Temporary bool   `json:"temporary"`
		} `json:"credentials"`
	}{
		Username:        login,
		Enabled:         true,
		Email:           login + "@example.com",
		FirstName:       "Gophkeeper",
		LastName:        "User",
		EmailVerified:   true,
		RequiredActions: []string{},
		Credentials: []struct {
			Type      string `json:"type"`
			Value     string `json:"value"`
			Temporary bool   `json:"temporary"`
		}{
			{
				Type:      "password",
				Value:     password,
				Temporary: false,
			},
		},
	}

	resp, err := s.doJSON(ctx, http.MethodPost, s.realmURL("/admin/realms/%s/users", s.cfg.Realm), adminToken, createPayload)
	if err != nil {
		return domain.User{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return domain.User{}, domain.ErrConflict
	}
	if resp.StatusCode != http.StatusCreated {
		return domain.User{}, readHTTPError(resp)
	}

	userID, err := idFromLocation(resp.Header.Get("Location"))
	if err != nil {
		return domain.User{}, err
	}

	userPayload, err := s.fetchUser(ctx, adminToken, userID)
	if err != nil {
		return domain.User{}, err
	}
	userPayload["requiredActions"] = []string{}
	userPayload["emailVerified"] = true
	userPayload["enabled"] = true

	patchResp, err := s.doJSON(ctx, http.MethodPut, s.realmURL("/admin/realms/%s/users/%s", s.cfg.Realm, userID), adminToken, userPayload)
	if err != nil {
		return domain.User{}, err
	}
	defer patchResp.Body.Close()
	if patchResp.StatusCode != http.StatusNoContent {
		return domain.User{}, readHTTPError(patchResp)
	}

	return domain.User{
		ID:        domain.UserID(userID),
		Login:     login,
		CreatedAt: s.clock.Now(),
	}, nil
}

// Login authenticates a user against Keycloak.
func (s *Service) Login(ctx context.Context, input inbound.LoginInput) (domain.Session, error) {
	login := strings.TrimSpace(input.Login)
	password := strings.TrimSpace(input.Password)
	if login == "" || password == "" {
		return domain.Session{}, errors.New("login and password are required")
	}

	values := url.Values{}
	values.Set("grant_type", "password")
	values.Set("username", login)
	values.Set("password", password)

	token, expiresIn, err := s.requestToken(ctx, s.cfg.Realm, s.cfg.ClientID, s.cfg.ClientSecret, values, true)
	if err != nil {
		return domain.Session{}, err
	}

	subject, err := parseJWTSubject(token)
	if err != nil {
		return domain.Session{}, err
	}

	return domain.Session{
		UserID:    domain.UserID(subject),
		Token:     token,
		ExpiresAt: s.clock.Now().Add(time.Duration(expiresIn) * time.Second),
	}, nil
}

// Validate checks a token against Keycloak introspection.
func (s *Service) Validate(ctx context.Context, token string) (domain.Session, error) {
	values := url.Values{}
	values.Set("token", token)

	resp, err := s.doForm(ctx, http.MethodPost, s.realmURL("/realms/%s/protocol/openid-connect/token/introspect", s.cfg.Realm), values, s.cfg.ClientID, s.cfg.ClientSecret)
	if err != nil {
		return domain.Session{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			return domain.Session{}, domain.ErrUnauthorized
		}
		return domain.Session{}, readHTTPError(resp)
	}

	var body struct {
		Active bool   `json:"active"`
		Sub    string `json:"sub"`
		Exp    int64  `json:"exp"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return domain.Session{}, err
	}
	if !body.Active || body.Sub == "" {
		return domain.Session{}, domain.ErrUnauthorized
	}

	expiresAt := s.clock.Now()
	if body.Exp > 0 {
		expiresAt = time.Unix(body.Exp, 0)
	}

	return domain.Session{
		UserID:    domain.UserID(body.Sub),
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) adminToken(ctx context.Context) (string, error) {
	values := url.Values{}
	values.Set("grant_type", "password")
	values.Set("username", s.cfg.AdminUser)
	values.Set("password", s.cfg.AdminPassword)
	token, _, err := s.requestToken(ctx, adminRealm, s.cfg.AdminClientID, s.cfg.AdminClientSecret, values, false)
	return token, err
}

func (s *Service) requestToken(ctx context.Context, realm, clientID, clientSecret string, values url.Values, mapAuthError bool) (string, int64, error) {
	resp, err := s.doForm(ctx, http.MethodPost, s.realmURL("/realms/%s/protocol/openid-connect/token", realm), values, clientID, clientSecret)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if mapAuthError && (resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized) {
			return "", 0, domain.ErrUnauthorized
		}
		return "", 0, readHTTPError(resp)
	}

	var body struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", 0, err
	}
	if body.AccessToken == "" {
		return "", 0, errors.New("missing access token")
	}
	return body.AccessToken, body.ExpiresIn, nil
}

func (s *Service) doJSON(ctx context.Context, method, url string, token string, payload interface{}) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return s.client.Do(req)
}

func (s *Service) fetchUser(ctx context.Context, token, userID string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.realmURL("/admin/realms/%s/users/%s", s.cfg.Realm, userID), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, readHTTPError(resp)
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (s *Service) doForm(ctx context.Context, method, url string, values url.Values, clientID, clientSecret string) (*http.Response, error) {
	if clientID != "" {
		values.Set("client_id", clientID)
	}
	if clientSecret != "" {
		values.Set("client_secret", clientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return s.client.Do(req)
}

func (s *Service) realmURL(format string, args ...interface{}) string {
	return s.baseURL + fmt.Sprintf(format, args...)
}

func readHTTPError(resp *http.Response) error {
	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if len(payload) == 0 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
}

func idFromLocation(location string) (string, error) {
	if location == "" {
		return "", errors.New("missing location header")
	}
	idx := strings.LastIndex(location, "/")
	if idx == -1 || idx == len(location)-1 {
		return "", fmt.Errorf("invalid location header: %s", location)
	}
	return location[idx+1:], nil
}

func parseJWTSubject(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return "", errors.New("invalid token format")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}
	var claims struct {
		Sub string `json:"sub"`
	}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return "", err
	}
	if claims.Sub == "" {
		return "", errors.New("missing sub claim")
	}
	return claims.Sub, nil
}
