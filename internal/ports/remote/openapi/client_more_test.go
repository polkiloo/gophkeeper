package openapi

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"
)

func TestClientMethods(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/api/v1/auth/register":
				return jsonResponse(http.StatusOK, map[string]any{"id": "u1", "login": "login", "created_at": now})
			case "/api/v1/auth/login":
				return jsonResponse(http.StatusOK, map[string]any{"user_id": "u1", "token": "token", "expires_at": now})
			case "/api/v1/auth/validate":
				return jsonResponse(http.StatusOK, map[string]any{"user_id": "u1", "token": "token", "expires_at": now})
			case "/api/v1/records":
				if req.Method == http.MethodGet {
					return jsonResponse(http.StatusOK, []any{})
				}
				return jsonResponse(http.StatusOK, sampleRecord(time.Now()))
			case "/api/v1/records/r1":
				if req.Method == http.MethodDelete {
					return statusResponse(http.StatusNoContent), nil
				}
				return jsonResponse(http.StatusOK, sampleRecord(time.Now()))
			case "/api/v1/sync":
				if req.Method == http.MethodGet {
					return jsonResponse(http.StatusOK, map[string]any{"changes": []any{}, "cursor": "1"})
				}
				return jsonResponse(http.StatusOK, map[string]any{"applied": 1, "rejected": 0, "conflicts": []string{}})
			default:
				return statusResponse(http.StatusNotFound), nil
			}
		}),
	}

	client, err := NewClient("http://example.com/api", WithHTTPClient(httpClient))
	if err != nil {
		t.Fatalf("client error: %v", err)
	}

	_, _ = client.Register(context.Background(), RegisterRequest{Login: "login", Password: "pass"})
	_, _ = client.RegisterWithBody(context.Background(), "application/json", bytes.NewReader([]byte(`{}`)))
	_, _ = client.Login(context.Background(), LoginRequest{Login: "login", Password: "pass"})
	_, _ = client.LoginWithBody(context.Background(), "application/json", bytes.NewReader([]byte(`{}`)))
	_, _ = client.Validate(context.Background())
	_, _ = client.ListRecords(context.Background(), &ListRecordsParams{})
	_, _ = client.UpsertRecord(context.Background(), RecordUpsert{Type: Text, Payload: mustTextPayload(t), Meta: Metadata{}})
	_, _ = client.UpsertRecordWithBody(context.Background(), "application/json", bytes.NewReader([]byte(`{}`)))
	_, _ = client.GetRecord(context.Background(), "r1")
	_, _ = client.DeleteRecord(context.Background(), "r1")
	_, _ = client.PullSync(context.Background(), &PullSyncParams{})
	_, _ = client.PushSync(context.Background(), SyncPush{Changes: []RecordChange{mustChange(t)}})
	_, _ = client.PushSyncWithBody(context.Background(), "application/json", bytes.NewReader([]byte(`{}`)))
}
