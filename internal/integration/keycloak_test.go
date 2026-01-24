//go:build integration

package integration

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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

	keycloakContainer, keycloakNetworkURL, keycloakHostURL := startKeycloakContainer(t, ctx, net.Name)
	setupKeycloakRealm(t, ctx, keycloakContainer)

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
		status, body := debugTokenRequest(t, ctx, keycloakHostURL, "kc-user", "kc-pass")
		dumpKeycloakUser(t, ctx, keycloakContainer, "kc-user")
		t.Fatalf("unexpected login status: %d (token status %d: %s)", loginResp.StatusCode(), status, body)
	}

	validateResp, err := client.ValidateWithResponse(ctx, authEditor(loginResp.JSON200.Token))
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if validateResp.StatusCode() != http.StatusOK {
		t.Fatalf("unexpected validate status: %d", validateResp.StatusCode())
	}
}

func startKeycloakContainer(t *testing.T, ctx context.Context, networkName string) (testcontainers.Container, string, string) {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "quay.io/keycloak/keycloak:24.0.5",
		ExposedPorts: []string{"8080/tcp"},
		Env: map[string]string{
			"KEYCLOAK_ADMIN":              "admin",
			"KEYCLOAK_ADMIN_PASSWORD":     "admin",
			"KC_BOOTSTRAP_ADMIN_USERNAME": "admin",
			"KC_BOOTSTRAP_ADMIN_PASSWORD": "admin",
			"KC_HTTP_ENABLED":             "true",
			"KC_HEALTH_ENABLED":           "true",
			"KC_HOSTNAME_STRICT":          "false",
			"KC_HOSTNAME_STRICT_HTTPS":    "false",
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

	return container, "http://keycloak:8080", fmt.Sprintf("http://%s:%s", host, port.Port())
}

func setupKeycloakRealm(t *testing.T, ctx context.Context, container testcontainers.Container) {
	t.Helper()

	runKeycloakCmd(t, ctx, container, "config", "credentials", "--server", "http://localhost:8080", "--realm", "master", "--user", "admin", "--password", "admin")
	runKeycloakCmd(t, ctx, container, "update", "realms/master", "-s", "sslRequired=NONE")
	runKeycloakCmd(t, ctx, container, "create", "realms", "-s", "realm=gophkeeper", "-s", "enabled=true", "-s", "sslRequired=NONE")
	runKeycloakCmd(t, ctx, container, "update", "realms/gophkeeper", "-s", "verifyEmail=false", "-s", "registrationAllowed=true", "-s", "resetPasswordAllowed=true", "-s", "rememberMe=true")
	runKeycloakCmd(t, ctx, container, "create", "clients", "-r", "gophkeeper", "-s", "clientId=gophkeeper-cli", "-s", "enabled=true", "-s", "directAccessGrantsEnabled=true", "-s", "standardFlowEnabled=true", "-s", "publicClient=false", "-s", "clientAuthenticatorType=client-secret", "-s", "secret=secret", "-s", "protocol=openid-connect")
}

func runKeycloakCmd(t *testing.T, ctx context.Context, container testcontainers.Container, args ...string) {
	t.Helper()

	cmd := append([]string{"/opt/keycloak/bin/kcadm.sh"}, args...)
	code, reader, err := container.Exec(ctx, cmd)
	if err != nil {
		t.Fatalf("keycloak command failed: %v", err)
	}
	output, _ := io.ReadAll(reader)
	outputText := strings.TrimSpace(string(output))
	if code != 0 {
		t.Fatalf("keycloak command failed (%d): %s", code, outputText)
	}
	if outputText != "" {
		t.Logf("keycloak: %s", outputText)
	}
}

func dumpKeycloakUser(t *testing.T, ctx context.Context, container testcontainers.Container, username string) {
	t.Helper()

	cmd := []string{"/opt/keycloak/bin/kcadm.sh", "get", "users", "-r", "gophkeeper", "-q", "username=" + username}
	code, reader, err := container.Exec(ctx, cmd)
	if err != nil {
		t.Logf("keycloak user dump failed: %v", err)
		return
	}
	output, _ := io.ReadAll(reader)
	outputText := strings.TrimSpace(string(output))
	if code != 0 {
		t.Logf("keycloak user dump failed (%d): %s", code, outputText)
		return
	}
	t.Logf("keycloak user: %s", outputText)
}

func debugTokenRequest(t *testing.T, ctx context.Context, baseURL, username, password string) (int, string) {
	t.Helper()

	form := strings.NewReader("grant_type=password&client_id=gophkeeper-cli&client_secret=secret&username=" + url.QueryEscape(username) + "&password=" + url.QueryEscape(password))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/realms/gophkeeper/protocol/openid-connect/token", form)
	if err != nil {
		t.Fatalf("token request failed: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("token request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, strings.TrimSpace(string(body))
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
