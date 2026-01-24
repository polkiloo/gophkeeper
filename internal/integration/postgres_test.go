//go:build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"

	"gophkeeper/internal/ports/remote/openapi"
)

func TestServerContainerAPIWithPostgres(t *testing.T) {
	ctx, client, db := startServerWithPostgres(t)

	token, userID := registerAndLogin(t, ctx, client, "pg-user", "secret")

	meta := sampleMetadata()
	payload := mustTextPayload(t, "pg-text")
	record := openapi.RecordUpsert{Type: openapi.Text, Payload: payload, Meta: meta}
	upserted := upsertRecord(t, ctx, client, token, record)

	verifyRecord(t, ctx, client, token, upserted.Id, openapi.Text, meta)

	assertTableCount(t, db, "users", 1)
	assertTableCount(t, db, "records", 1)
	assertTableCountAtLeast(t, db, "change_log", 1)

	assertUserRow(t, db, userID, "pg-user")
	assertRecordRow(t, db, upserted.Id, userID)
	assertChangeLogRow(t, db, upserted.Id, userID)

	conflictResp, err := client.RegisterWithResponse(ctx, openapi.RegisterRequest{Login: "pg-user", Password: "secret"})
	if err != nil {
		t.Fatalf("conflict register failed: %v", err)
	}
	if conflictResp.StatusCode() != http.StatusConflict {
		t.Fatalf("expected conflict status, got %d", conflictResp.StatusCode())
	}
}

func startServerWithPostgres(t *testing.T) (context.Context, *openapi.ClientWithResponses, *sql.DB) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	t.Cleanup(cancel)

	net, err := network.New(ctx)
	if err != nil {
		t.Fatalf("failed to create network: %v", err)
	}
	t.Cleanup(func() {
		_ = net.Remove(context.Background())
	})

	dbContainer, hostDSN := startPostgresContainer(t, ctx, net.Name)
	t.Cleanup(func() {
		_ = dbContainer.Terminate(context.Background())
	})

	repoRoot, err := repoRoot()
	if err != nil {
		t.Fatalf("failed to resolve repo root: %v", err)
	}

	serverContainer := startServerContainerWithPostgres(t, ctx, net.Name, repoRoot)
	t.Cleanup(func() {
		_ = serverContainer.Terminate(context.Background())
	})

	host, err := serverContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to resolve server host: %v", err)
	}
	port, err := serverContainer.MappedPort(ctx, "8080/tcp")
	if err != nil {
		t.Fatalf("failed to resolve server port: %v", err)
	}

	baseURL := fmt.Sprintf("http://%s:%s/api", host, port.Port())
	client, err := openapi.NewClientWithResponses(baseURL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	db, err := sql.Open("postgres", hostDSN)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	return ctx, client, db
}

func startPostgresContainer(t *testing.T, ctx context.Context, networkName string) (testcontainers.Container, string) {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "gophkeeper",
			"POSTGRES_PASSWORD": "gophkeeper",
			"POSTGRES_DB":       "gophkeeper",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").
			WithStartupTimeout(2 * time.Minute),
		Networks: []string{networkName},
		NetworkAliases: map[string][]string{
			networkName: {"postgres"},
		},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to resolve postgres host: %v", err)
	}
	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatalf("failed to resolve postgres port: %v", err)
	}

	hostDSN := fmt.Sprintf("postgres://gophkeeper:gophkeeper@%s:%s/gophkeeper?sslmode=disable", host, port.Port())
	return container, hostDSN
}

func startServerContainerWithPostgres(t *testing.T, ctx context.Context, networkName, repoRoot string) testcontainers.Container {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "golang:1.22",
		ExposedPorts: []string{"8080/tcp"},
		Env: map[string]string{
			"GOPHKEEPER_ADDR":          ":8080",
			"GOPHKEEPER_STORAGE":       "postgres",
			"GOPHKEEPER_PG_DSN":        "postgres://gophkeeper:gophkeeper@postgres:5432/gophkeeper?sslmode=disable",
			"GOPHKEEPER_AUTH_PROVIDER": "local",
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
	return container
}

func assertTableCount(t *testing.T, db *sql.DB, table string, expected int) {
	t.Helper()

	var count int
	if err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count); err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != expected {
		t.Fatalf("unexpected %s count: %d", table, count)
	}
}

func assertTableCountAtLeast(t *testing.T, db *sql.DB, table string, expected int) {
	t.Helper()

	var count int
	if err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count); err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count < expected {
		t.Fatalf("expected %s count >= %d, got %d", table, expected, count)
	}
}

func assertUserRow(t *testing.T, db *sql.DB, userID, login string) {
	t.Helper()

	var gotLogin string
	if err := db.QueryRow(`SELECT login FROM users WHERE id = $1`, userID).Scan(&gotLogin); err != nil {
		t.Fatalf("user lookup failed: %v", err)
	}
	if gotLogin != login {
		t.Fatalf("unexpected login: %s", gotLogin)
	}
}

func assertRecordRow(t *testing.T, db *sql.DB, recordID, ownerID string) {
	t.Helper()

	var gotOwner string
	var deleted bool
	if err := db.QueryRow(`SELECT owner_id, deleted FROM records WHERE id = $1`, recordID).Scan(&gotOwner, &deleted); err != nil {
		t.Fatalf("record lookup failed: %v", err)
	}
	if gotOwner != ownerID {
		t.Fatalf("unexpected owner id: %s", gotOwner)
	}
	if deleted {
		t.Fatalf("expected record to be active")
	}
}

func assertChangeLogRow(t *testing.T, db *sql.DB, recordID, ownerID string) {
	t.Helper()

	var gotRecord string
	var gotOwner string
	if err := db.QueryRow(`SELECT record_id, owner_id FROM change_log WHERE record_id = $1 LIMIT 1`, recordID).Scan(&gotRecord, &gotOwner); err != nil {
		t.Fatalf("change_log lookup failed: %v", err)
	}
	if gotRecord != recordID || gotOwner != ownerID {
		t.Fatalf("unexpected change_log row")
	}
}
