//go:build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"gophkeeper/internal/ports/remote/openapi"
)

func startServer(t *testing.T) (context.Context, *openapi.ClientWithResponses) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)

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

	baseURL := fmt.Sprintf("http://%s:%s/api", host, port.Port())
	client, err := openapi.NewClientWithResponses(baseURL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	return ctx, client
}

func registerAndLogin(t *testing.T, ctx context.Context, client *openapi.ClientWithResponses, login, password string) (string, string) {
	t.Helper()

	registerResp, err := client.RegisterWithResponse(ctx, openapi.RegisterRequest{Login: login, Password: password})
	if err != nil {
		t.Fatalf("register request failed: %v", err)
	}
	if registerResp.StatusCode() != http.StatusOK {
		t.Fatalf("unexpected register status: %d", registerResp.StatusCode())
	}

	loginResp, err := client.LoginWithResponse(ctx, openapi.LoginRequest{Login: login, Password: password})
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	if loginResp.StatusCode() != http.StatusOK || loginResp.JSON200 == nil {
		t.Fatalf("unexpected login status: %d", loginResp.StatusCode())
	}
	if loginResp.JSON200.Token == "" {
		t.Fatalf("expected login token")
	}

	return loginResp.JSON200.Token, loginResp.JSON200.UserId
}

func authEditor(token string) openapi.RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}
}

func mustTextPayload(t *testing.T, text string) openapi.Payload {
	var payload openapi.Payload
	if err := payload.FromTextPayload(openapi.TextPayload{Kind: openapi.Text, Text: text}); err != nil {
		t.Fatalf("failed to build payload: %v", err)
	}
	return payload
}

func mustCredentialPayload(t *testing.T, login, password string) openapi.Payload {
	var payload openapi.Payload
	if err := payload.FromCredentialPayload(openapi.CredentialPayload{Kind: openapi.Credential, Login: login, Password: password}); err != nil {
		t.Fatalf("failed to build credential payload: %v", err)
	}
	return payload
}

func mustBinaryPayload(t *testing.T, data []byte) openapi.Payload {
	var payload openapi.Payload
	if err := payload.FromBinaryPayload(openapi.BinaryPayload{Kind: openapi.Binary, Data: data}); err != nil {
		t.Fatalf("failed to build binary payload: %v", err)
	}
	return payload
}

func mustBankCardPayload(t *testing.T, cardholder, number, expiresAt, cvv string) openapi.Payload {
	var payload openapi.Payload
	if err := payload.FromBankCardPayload(openapi.BankCardPayload{Kind: openapi.BankCard, Cardholder: cardholder, Number: number, ExpiresAt: expiresAt, Cvv: cvv}); err != nil {
		t.Fatalf("failed to build bank card payload: %v", err)
	}
	return payload
}

func recordPresent(records *[]openapi.Record, id string) bool {
	if records == nil {
		return false
	}
	for _, record := range *records {
		if record.Id == id {
			return true
		}
	}
	return false
}

func upsertRecord(t *testing.T, ctx context.Context, client *openapi.ClientWithResponses, token string, req openapi.RecordUpsert) openapi.Record {
	resp, err := client.UpsertRecordWithResponse(ctx, req, authEditor(token))
	if err != nil {
		t.Fatalf("upsert request failed: %v", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		t.Fatalf("unexpected upsert status: %d", resp.StatusCode())
	}
	return *resp.JSON200
}

func verifyRecord(t *testing.T, ctx context.Context, client *openapi.ClientWithResponses, token, id string, recordType openapi.RecordType, meta openapi.Metadata) {
	resp, err := client.GetRecordWithResponse(ctx, id, authEditor(token))
	if err != nil {
		t.Fatalf("get request failed: %v", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		t.Fatalf("unexpected get status: %d", resp.StatusCode())
	}
	record := resp.JSON200
	if record.Type != recordType {
		t.Fatalf("unexpected record type: %s", record.Type)
	}
	assertMetadata(t, record.Meta, meta)
}

func sampleMetadata() openapi.Metadata {
	title := "example"
	description := "description"
	tags := []string{"one", "two"}
	attrs := map[string]string{"site": "example.com", "note": "otp list"}
	return openapi.Metadata{
		Title:       &title,
		Description: &description,
		Tags:        &tags,
		Attributes:  &attrs,
	}
}

func assertMetadata(t *testing.T, got openapi.Metadata, expected openapi.Metadata) {
	if derefString(got.Title) != derefString(expected.Title) {
		t.Fatalf("unexpected title")
	}
	if derefString(got.Description) != derefString(expected.Description) {
		t.Fatalf("unexpected description")
	}
	if !equalStrings(derefStringSlice(got.Tags), derefStringSlice(expected.Tags)) {
		t.Fatalf("unexpected tags")
	}
	if !equalMap(derefStringMap(got.Attributes), derefStringMap(expected.Attributes)) {
		t.Fatalf("unexpected attributes")
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func derefStringSlice(value *[]string) []string {
	if value == nil {
		return nil
	}
	return *value
}

func derefStringMap(value *map[string]string) map[string]string {
	if value == nil {
		return nil
	}
	return *value
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalMap(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}

func findChange(changes []openapi.RecordChange, recordID string) (openapi.RecordChange, bool) {
	for _, change := range changes {
		if change.RecordId == recordID {
			return change, true
		}
	}
	return openapi.RecordChange{}, false
}

func repoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Dir(filepath.Dir(cwd)), nil
}
