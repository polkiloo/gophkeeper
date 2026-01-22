package openapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientWithResponses(t *testing.T) {
	now := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/auth/register":
			writeJSON(w, map[string]any{
				"id":         "u1",
				"login":      "login",
				"created_at": now.Format(time.RFC3339),
			})
		case "/api/v1/auth/login":
			writeJSON(w, map[string]any{
				"user_id":    "u1",
				"token":      "token",
				"expires_at": now.Add(time.Hour).Format(time.RFC3339),
			})
		case "/api/v1/auth/validate":
			if r.Header.Get("Authorization") == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			writeJSON(w, map[string]any{
				"user_id":    "u1",
				"token":      "token",
				"expires_at": now.Add(time.Hour).Format(time.RFC3339),
			})
		case "/api/v1/records":
			if r.Method == http.MethodGet {
				writeJSON(w, []any{sampleRecord(now)})
				return
			}
			writeJSON(w, sampleRecord(now))
		case "/api/v1/records/r1":
			if r.Method == http.MethodDelete {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			writeJSON(w, sampleRecord(now))
		case "/api/v1/sync":
			if r.Method == http.MethodGet {
				writeJSON(w, map[string]any{
					"changes": []any{sampleChange(now)},
					"cursor":  "1",
				})
				return
			}
			writeJSON(w, map[string]any{
				"applied":   1,
				"rejected":  0,
				"conflicts": []string{},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewClientWithResponses(server.URL + "/api")
	if err != nil {
		t.Fatalf("client error: %v", err)
	}

	registerResp, err := client.RegisterWithResponse(context.Background(), RegisterRequest{Login: "login", Password: "pass"})
	if err != nil || registerResp.StatusCode() != http.StatusOK {
		t.Fatalf("register failed")
	}

	loginResp, err := client.LoginWithResponse(context.Background(), LoginRequest{Login: "login", Password: "pass"})
	if err != nil || loginResp.StatusCode() != http.StatusOK {
		t.Fatalf("login failed")
	}

	validateResp, err := client.ValidateWithResponse(context.Background(), authEditor("token"))
	if err != nil || validateResp.StatusCode() != http.StatusOK {
		t.Fatalf("validate failed")
	}

	listResp, err := client.ListRecordsWithResponse(context.Background(), &ListRecordsParams{}, authEditor("token"))
	if err != nil || listResp.StatusCode() != http.StatusOK {
		t.Fatalf("list failed")
	}

	upsertResp, err := client.UpsertRecordWithResponse(context.Background(), RecordUpsert{Type: Text, Payload: mustTextPayload(t), Meta: Metadata{}}, authEditor("token"))
	if err != nil || upsertResp.StatusCode() != http.StatusOK {
		t.Fatalf("upsert failed")
	}

	getResp, err := client.GetRecordWithResponse(context.Background(), "r1", authEditor("token"))
	if err != nil || getResp.StatusCode() != http.StatusOK {
		t.Fatalf("get failed")
	}

	deleteResp, err := client.DeleteRecordWithResponse(context.Background(), "r1", authEditor("token"))
	if err != nil || deleteResp.StatusCode() != http.StatusNoContent {
		t.Fatalf("delete failed")
	}

	pullResp, err := client.PullSyncWithResponse(context.Background(), &PullSyncParams{}, authEditor("token"))
	if err != nil || pullResp.StatusCode() != http.StatusOK {
		t.Fatalf("pull failed")
	}

	pushResp, err := client.PushSyncWithResponse(context.Background(), SyncPush{Changes: []RecordChange{mustChange(t)}}, authEditor("token"))
	if err != nil || pushResp.StatusCode() != http.StatusOK {
		t.Fatalf("push failed")
	}
}

func TestPayloadRoundTrip(t *testing.T) {
	var payload Payload
	if err := payload.FromCredentialPayload(CredentialPayload{Kind: Credential, Login: "l", Password: "p"}); err != nil {
		t.Fatalf("credential error: %v", err)
	}
	if got, err := payload.AsCredentialPayload(); err != nil || got.Login != "l" {
		t.Fatalf("credential roundtrip")
	}

	if err := payload.FromTextPayload(TextPayload{Kind: Text, Text: "t"}); err != nil {
		t.Fatalf("text error: %v", err)
	}
	if got, err := payload.AsTextPayload(); err != nil || got.Text != "t" {
		t.Fatalf("text roundtrip")
	}

	if err := payload.FromBinaryPayload(BinaryPayload{Kind: Binary, Data: []byte("bin")}); err != nil {
		t.Fatalf("binary error: %v", err)
	}
	if got, err := payload.AsBinaryPayload(); err != nil || string(got.Data) != "bin" {
		t.Fatalf("binary roundtrip")
	}

	if err := payload.FromBankCardPayload(BankCardPayload{Kind: BankCard, Cardholder: "c", Number: "1", ExpiresAt: "12/30", Cvv: "123"}); err != nil {
		t.Fatalf("bank card error: %v", err)
	}
	if got, err := payload.AsBankCardPayload(); err != nil || got.Cardholder != "c" {
		t.Fatalf("bank card roundtrip")
	}
}

func authEditor(token string) RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}
}

func sampleRecord(now time.Time) map[string]any {
	return map[string]any{
		"id":         "r1",
		"owner_id":   "u1",
		"type":       "text",
		"payload":    map[string]any{"kind": "text", "text": "hello"},
		"meta":       map[string]any{"title": "t", "description": "d"},
		"version":    1,
		"updated_at": now.Format(time.RFC3339),
	}
}

func sampleChange(now time.Time) map[string]any {
	return map[string]any{
		"record_id":   "r1",
		"owner_id":    "u1",
		"type":        "text",
		"change":      "upsert",
		"payload":     map[string]any{"kind": "text", "text": "hello"},
		"meta":        map[string]any{"title": "t"},
		"version":     1,
		"happened_at": now.Format(time.RFC3339),
	}
}

func mustTextPayload(t *testing.T) Payload {
	var payload Payload
	if err := payload.FromTextPayload(TextPayload{Kind: Text, Text: "hello"}); err != nil {
		t.Fatalf("payload error: %v", err)
	}
	return payload
}

func mustChange(t *testing.T) RecordChange {
	payload := mustTextPayload(t)
	meta := Metadata{}
	return RecordChange{
		RecordId:   "r1",
		OwnerId:    "u1",
		Type:       Text,
		Change:     Upsert,
		Payload:    &payload,
		Meta:       &meta,
		Version:    1,
		HappenedAt: time.Now().UTC(),
	}
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(payload)
}
