//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type sessionResponse struct {
	Token string `json:"token"`
}

func TestServerContainerAPI(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	repoRoot, err := repoRoot()
	if err != nil {
		t.Fatalf("failed to resolve repo root: %v", err)
	}

	req := testcontainers.ContainerRequest{
		Image:        "golang:1.22",
		ExposedPorts: []string{"8080/tcp"},
		Env: map[string]string{
			"GOPHKEEPER_ADDR": ":8080",
		},
		WorkingDir: "/workspace",
		Cmd:        []string{"sh", "-c", "go run ./cmd/server"},
		Mounts: testcontainers.Mounts(
			testcontainers.BindMount(repoRoot, "/workspace"),
		),
		WaitingFor: wait.ForHTTP("/api/v1/auth/validate").
			WithPort("8080/tcp").
			WithStatusCodeMatcher(func(status int) bool { return status == http.StatusUnauthorized }),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to resolve container host: %v", err)
	}
	port, err := container.MappedPort(ctx, "8080/tcp")
	if err != nil {
		t.Fatalf("failed to resolve container port: %v", err)
	}

	baseURL := fmt.Sprintf("http://%s:%s/api/v1", host, port.Port())
	client := &http.Client{Timeout: 10 * time.Second}

	registerBody := map[string]string{"login": "alice", "password": "secret"}
	registerStatus := doJSONRequest(t, client, http.MethodPost, baseURL+"/auth/register", registerBody, nil)
	if registerStatus != http.StatusOK {
		t.Fatalf("unexpected register status: %d", registerStatus)
	}

	loginBody := map[string]string{"login": "alice", "password": "secret"}
	var session sessionResponse
	loginStatus := doJSONRequest(t, client, http.MethodPost, baseURL+"/auth/login", loginBody, &session)
	if loginStatus != http.StatusOK {
		t.Fatalf("unexpected login status: %d", loginStatus)
	}
	if session.Token == "" {
		t.Fatalf("expected session token")
	}

	validateReq, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/auth/validate", nil)
	if err != nil {
		t.Fatalf("failed to create validate request: %v", err)
	}
	validateReq.Header.Set("Authorization", "Bearer "+session.Token)

	resp, err := client.Do(validateReq)
	if err != nil {
		t.Fatalf("validate request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected validate status: %d", resp.StatusCode)
	}
}

func doJSONRequest(t *testing.T, client *http.Client, method, url string, body any, out any) int {
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if out != nil {
		decoder := json.NewDecoder(resp.Body)
		if err := decoder.Decode(out); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
	}

	return resp.StatusCode
}

func repoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Dir(filepath.Dir(cwd)), nil
}
