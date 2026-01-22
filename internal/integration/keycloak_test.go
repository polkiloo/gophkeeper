//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"

	"gophkeeper/internal/ports/remote/openapi"
)

func TestKeycloakAuthIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	t.Cleanup(cancel)

	net, err := network.New(ctx)
	if err != nil {
		t.Fatalf("failed to create network: %v", err)
	}
	t.Cleanup(func() {
		_ = net.Remove(context.Background())
	})

	keycloakURL, keycloakNetworkURL := startKeycloakContainer(t, ctx, net.Name)
	setupKeycloakRealm(t, ctx, keycloakURL)

	client := startServerWithKeycloak(t, ctx, net.Name, keycloakNetworkURL)

	registerResp, err := client.RegisterWithResponse(ctx, openapi.RegisterRequest{Login: "kc-user", Password: "kc-pass"})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if registerResp.StatusCode() != http.StatusOK || registerResp.JSON200 == nil {
		t.Fatalf("unexpected register status: %d", registerResp.StatusCode())
	}

	loginResp, err := client.LoginWithResponse(ctx, openapi.LoginRequest{Login: "kc-user", Password: "kc-pass"})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if loginResp.StatusCode() != http.StatusOK || loginResp.JSON200 == nil {
		t.Fatalf("unexpected login status: %d", loginResp.StatusCode())
	}

	validateResp, err := client.ValidateWithResponse(ctx, authEditor(loginResp.JSON200.Token))
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if validateResp.StatusCode() != http.StatusOK {
		t.Fatalf("unexpected validate status: %d", validateResp.StatusCode())
	}
}

func startKeycloakContainer(t *testing.T, ctx context.Context, networkName string) (string, string) {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "quay.io/keycloak/keycloak:24.0.5",
		ExposedPorts: []string{"8080/tcp"},
		Env: map[string]string{
			"KEYCLOAK_ADMIN":          "admin",
			"KEYCLOAK_ADMIN_PASSWORD": "admin",
			"KC_HEALTH_ENABLED":       "true",
		},
		Cmd: []string{"start-dev", "--http-port=8080"},
		WaitingFor: wait.ForHTTP("/health/ready").
			WithPort("8080/tcp").
			WithStartupTimeout(3 * time.Minute),
		Networks: []string{networkName},
		NetworkAliases: map[string][]string{
			networkName: {"keycloak"},
		},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start keycloak container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to resolve keycloak host: %v", err)
	}
	port, err := container.MappedPort(ctx, "8080/tcp")
	if err != nil {
		t.Fatalf("failed to resolve keycloak port: %v", err)
	}

	return fmt.Sprintf("http://%s:%s", host, port.Port()), "http://keycloak:8080"
}

func setupKeycloakRealm(t *testing.T, ctx context.Context, baseURL string) {
	t.Helper()

	adminToken := fetchKeycloakAdminToken(t, ctx, baseURL)

	realmPayload := map[string]interface{}{
		"realm":   "gophkeeper",
		"enabled": true,
	}
	createKeycloakEntity(t, ctx, baseURL+"/admin/realms", adminToken, realmPayload)

	clientPayload := map[string]interface{}{
		"clientId":                  "gophkeeper-cli",
		"enabled":                   true,
		"directAccessGrantsEnabled": true,
		"publicClient":              false,
		"secret":                    "secret",
		"protocol":                  "openid-connect",
	}
	createKeycloakEntity(t, ctx, baseURL+"/admin/realms/gophkeeper/clients", adminToken, clientPayload)
}

func fetchKeycloakAdminToken(t *testing.T, ctx context.Context, baseURL string) string {
	t.Helper()

	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("client_id", "admin-cli")
	form.Set("username", "admin")
	form.Set("password", "admin")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/realms/master/protocol/openid-connect/token", bytes.NewBufferString(form.Encode()))
	if err != nil {
		t.Fatalf("admin token request failed: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("admin token request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin token status: %d", resp.StatusCode)
	}

	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("admin token decode failed: %v", err)
	}
	if payload.AccessToken == "" {
		t.Fatalf("empty admin token")
	}
	return payload.AccessToken
}

func createKeycloakEntity(t *testing.T, ctx context.Context, url, token string, payload map[string]interface{}) {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("keycloak request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		t.Fatalf("unexpected keycloak status: %d", resp.StatusCode)
	}
}

func startServerWithKeycloak(t *testing.T, ctx context.Context, networkName, keycloakNetworkURL string) *openapi.ClientWithResponses {
	t.Helper()

	repoRoot, err := repoRoot()
	if err != nil {
		t.Fatalf("failed to resolve repo root: %v", err)
	}

	req := testcontainers.ContainerRequest{
		Image:        "golang:1.22",
		ExposedPorts: []string{"8080/tcp"},
		Env: map[string]string{
			"GOPHKEEPER_ADDR":              ":8080",
			"GOPHKEEPER_AUTH_PROVIDER":     "keycloak",
			"KEYCLOAK_BASE_URL":            keycloakNetworkURL,
			"KEYCLOAK_REALM":               "gophkeeper",
			"KEYCLOAK_CLIENT_ID":           "gophkeeper-cli",
			"KEYCLOAK_CLIENT_SECRET":       "secret",
			"KEYCLOAK_ADMIN_USER":          "admin",
			"KEYCLOAK_ADMIN_PASSWORD":      "admin",
			"KEYCLOAK_ADMIN_CLIENT_ID":     "admin-cli",
			"KEYCLOAK_ADMIN_CLIENT_SECRET": "",
		},
		WorkingDir: "/workspace",
		Cmd:        []string{"sh", "-c", "go run ./cmd/server"},
		Mounts: testcontainers.Mounts(
			testcontainers.BindMount(repoRoot, "/workspace"),
		),
		WaitingFor: wait.ForHTTP("/api/v1/auth/validate").
			WithPort("8080/tcp").
			WithStatusCodeMatcher(func(status int) bool { return status == http.StatusUnauthorized }),
		Networks: []string{networkName},
		NetworkAliases: map[string][]string{
			networkName: {"gophkeeper"},
		},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start server container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to resolve server host: %v", err)
	}
	port, err := container.MappedPort(ctx, "8080/tcp")
	if err != nil {
		t.Fatalf("failed to resolve server port: %v", err)
	}

	baseURL := fmt.Sprintf("http://%s:%s/api", host, port.Port())
	client, err := openapi.NewClientWithResponses(baseURL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return client
}
