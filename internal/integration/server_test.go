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

	baseURL := fmt.Sprintf("http://%s:%s/api", host, port.Port())
	client, err := openapi.NewClientWithResponses(baseURL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	registerResp, err := client.RegisterWithResponse(ctx, openapi.RegisterRequest{Login: "alice", Password: "secret"})
	if err != nil {
		t.Fatalf("register request failed: %v", err)
	}
	if registerResp.StatusCode() != http.StatusOK {
		t.Fatalf("unexpected register status: %d", registerResp.StatusCode())
	}

	loginResp, err := client.LoginWithResponse(ctx, openapi.LoginRequest{Login: "alice", Password: "secret"})
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	if loginResp.StatusCode() != http.StatusOK || loginResp.JSON200 == nil {
		t.Fatalf("unexpected login status: %d", loginResp.StatusCode())
	}
	if loginResp.JSON200.Token == "" {
		t.Fatalf("expected login token")
	}

	token := loginResp.JSON200.Token
	userID := loginResp.JSON200.UserId

	validateResp, err := client.ValidateWithResponse(ctx, authEditor(token))
	if err != nil {
		t.Fatalf("validate request failed: %v", err)
	}
	if validateResp.StatusCode() != http.StatusOK {
		t.Fatalf("unexpected validate status: %d", validateResp.StatusCode())
	}

	payload := mustTextPayload(t, "secret-data")
	upsertReq := openapi.RecordUpsert{Type: openapi.Text, Payload: payload, Meta: openapi.Metadata{}}
	upsertResp, err := client.UpsertRecordWithResponse(ctx, upsertReq, authEditor(token))
	if err != nil {
		t.Fatalf("upsert request failed: %v", err)
	}
	if upsertResp.StatusCode() != http.StatusOK || upsertResp.JSON200 == nil {
		t.Fatalf("unexpected upsert status: %d", upsertResp.StatusCode())
	}
	recordID := upsertResp.JSON200.Id

	getResp, err := client.GetRecordWithResponse(ctx, recordID, authEditor(token))
	if err != nil {
		t.Fatalf("get request failed: %v", err)
	}
	if getResp.StatusCode() != http.StatusOK {
		t.Fatalf("unexpected get status: %d", getResp.StatusCode())
	}

	listResp, err := client.ListRecordsWithResponse(ctx, &openapi.ListRecordsParams{}, authEditor(token))
	if err != nil {
		t.Fatalf("list request failed: %v", err)
	}
	if listResp.StatusCode() != http.StatusOK || listResp.JSON200 == nil {
		t.Fatalf("unexpected list status: %d", listResp.StatusCode())
	}
	if !recordPresent(listResp.JSON200, recordID) {
		t.Fatalf("record not found in list")
	}

	pullResp, err := client.PullSyncWithResponse(ctx, &openapi.PullSyncParams{}, authEditor(token))
	if err != nil {
		t.Fatalf("pull request failed: %v", err)
	}
	if pullResp.StatusCode() != http.StatusOK || pullResp.JSON200 == nil {
		t.Fatalf("unexpected pull status: %d", pullResp.StatusCode())
	}
	if len(pullResp.JSON200.Changes) == 0 {
		t.Fatalf("expected sync changes")
	}

	syncPayload := mustTextPayload(t, "sync-data")
	meta := openapi.Metadata{}
	change := openapi.RecordChange{
		RecordId:   fmt.Sprintf("sync-%d", time.Now().UnixNano()),
		OwnerId:    userID,
		Type:       openapi.Text,
		Change:     openapi.Upsert,
		Payload:    &syncPayload,
		Meta:       &meta,
		Version:    1,
		HappenedAt: time.Now().UTC(),
	}
	pushResp, err := client.PushSyncWithResponse(ctx, openapi.SyncPush{Changes: []openapi.RecordChange{change}}, authEditor(token))
	if err != nil {
		t.Fatalf("push request failed: %v", err)
	}
	if pushResp.StatusCode() != http.StatusOK || pushResp.JSON200 == nil {
		t.Fatalf("unexpected push status: %d", pushResp.StatusCode())
	}
	if pushResp.JSON200.Applied != 1 {
		t.Fatalf("expected applied=1")
	}

	getSyncedResp, err := client.GetRecordWithResponse(ctx, change.RecordId, authEditor(token))
	if err != nil {
		t.Fatalf("get synced record failed: %v", err)
	}
	if getSyncedResp.StatusCode() != http.StatusOK {
		t.Fatalf("unexpected get synced status: %d", getSyncedResp.StatusCode())
	}

	deleteResp, err := client.DeleteRecordWithResponse(ctx, recordID, authEditor(token))
	if err != nil {
		t.Fatalf("delete request failed: %v", err)
	}
	if deleteResp.StatusCode() != http.StatusNoContent {
		t.Fatalf("unexpected delete status: %d", deleteResp.StatusCode())
	}

	getDeletedResp, err := client.GetRecordWithResponse(ctx, recordID, authEditor(token))
	if err != nil {
		t.Fatalf("get deleted record failed: %v", err)
	}
	if getDeletedResp.StatusCode() != http.StatusNotFound {
		t.Fatalf("unexpected deleted get status: %d", getDeletedResp.StatusCode())
	}
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

func repoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Dir(filepath.Dir(cwd)), nil
}
