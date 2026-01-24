package openapi

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestClientWithResponsesBodyMethods(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/api/v1/auth/login":
				return jsonResponse(http.StatusOK, map[string]any{"user_id": "u1", "token": "t", "expires_at": now})
			case "/api/v1/auth/register":
				return jsonResponse(http.StatusOK, map[string]any{"id": "u1", "login": "l", "created_at": now})
			case "/api/v1/records":
				return jsonResponse(http.StatusOK, sampleRecord(time.Now()))
			case "/api/v1/sync":
				return jsonResponse(http.StatusOK, map[string]any{"applied": 1, "rejected": 0, "conflicts": []string{}})
			default:
				return statusResponse(http.StatusNotFound), nil
			}
		}),
	}

	client, err := NewClientWithResponses("http://example.com/api", WithHTTPClient(httpClient))
	if err != nil {
		t.Fatalf("client error: %v", err)
	}

	if _, err := client.LoginWithBodyWithResponse(context.Background(), "application/json", bytes.NewReader([]byte(`{}`))); err != nil {
		t.Fatalf("login with body error: %v", err)
	}
	if _, err := client.RegisterWithBodyWithResponse(context.Background(), "application/json", bytes.NewReader([]byte(`{}`))); err != nil {
		t.Fatalf("register with body error: %v", err)
	}
	if _, err := client.UpsertRecordWithBodyWithResponse(context.Background(), "application/json", bytes.NewReader([]byte(`{}`))); err != nil {
		t.Fatalf("upsert with body error: %v", err)
	}
	if _, err := client.PushSyncWithBodyWithResponse(context.Background(), "application/json", bytes.NewReader([]byte(`{}`))); err != nil {
		t.Fatalf("push with body error: %v", err)
	}
}

func TestPayloadMerge(t *testing.T) {
	var payload Payload
	if err := payload.MergeCredentialPayload(CredentialPayload{Kind: Credential, Login: "l", Password: "p"}); err != nil {
		t.Fatalf("credential merge error: %v", err)
	}
	if err := payload.MergeTextPayload(TextPayload{Kind: Text, Text: "t"}); err != nil {
		t.Fatalf("text merge error: %v", err)
	}
	if err := payload.MergeBinaryPayload(BinaryPayload{Kind: Binary, Data: []byte("b")}); err != nil {
		t.Fatalf("binary merge error: %v", err)
	}
	if err := payload.MergeBankCardPayload(BankCardPayload{Kind: BankCard, Cardholder: "c", Number: "1", ExpiresAt: "12/30", Cvv: "123"}); err != nil {
		t.Fatalf("bank card merge error: %v", err)
	}
}

func TestResponseStatuses(t *testing.T) {
	resp := &http.Response{Status: "200 OK", StatusCode: http.StatusOK}

	if (LoginResponse{HTTPResponse: resp}).Status() != "200 OK" {
		t.Fatalf("unexpected status")
	}
	if (RegisterResponse{HTTPResponse: resp}).Status() != "200 OK" {
		t.Fatalf("unexpected status")
	}
	if (ValidateResponse{HTTPResponse: resp}).Status() != "200 OK" {
		t.Fatalf("unexpected status")
	}
	if (ListRecordsResponse{HTTPResponse: resp}).Status() != "200 OK" {
		t.Fatalf("unexpected status")
	}
	if (UpsertRecordResponse{HTTPResponse: resp}).Status() != "200 OK" {
		t.Fatalf("unexpected status")
	}
	if (DeleteRecordResponse{HTTPResponse: resp}).Status() != "200 OK" {
		t.Fatalf("unexpected status")
	}
	if (GetRecordResponse{HTTPResponse: resp}).Status() != "200 OK" {
		t.Fatalf("unexpected status")
	}
	if (PullSyncResponse{HTTPResponse: resp}).Status() != "200 OK" {
		t.Fatalf("unexpected status")
	}
	if (PushSyncResponse{HTTPResponse: resp}).Status() != "200 OK" {
		t.Fatalf("unexpected status")
	}
}

func TestWithBaseURL(t *testing.T) {
	if _, err := NewClient("http://example.com", WithBaseURL("://bad")); err == nil {
		t.Fatalf("expected base url error")
	}
}

func TestClientMethodsEditorError(t *testing.T) {
	client, err := NewClient("http://example.com/api")
	if err != nil {
		t.Fatalf("client error: %v", err)
	}

	editorErr := func(ctx context.Context, req *http.Request) error {
		return errors.New("editor")
	}

	if _, err := client.Login(context.Background(), LoginRequest{}, editorErr); err == nil {
		t.Fatalf("expected login error")
	}
	if _, err := client.Register(context.Background(), RegisterRequest{}, editorErr); err == nil {
		t.Fatalf("expected register error")
	}
	if _, err := client.Validate(context.Background(), editorErr); err == nil {
		t.Fatalf("expected validate error")
	}
	if _, err := client.ListRecords(context.Background(), &ListRecordsParams{}, editorErr); err == nil {
		t.Fatalf("expected list error")
	}
	if _, err := client.UpsertRecord(context.Background(), RecordUpsert{}, editorErr); err == nil {
		t.Fatalf("expected upsert error")
	}
	if _, err := client.DeleteRecord(context.Background(), "id", editorErr); err == nil {
		t.Fatalf("expected delete error")
	}
	if _, err := client.GetRecord(context.Background(), "id", editorErr); err == nil {
		t.Fatalf("expected get error")
	}
	if _, err := client.PullSync(context.Background(), &PullSyncParams{}, editorErr); err == nil {
		t.Fatalf("expected pull error")
	}
	if _, err := client.PushSync(context.Background(), SyncPush{}, editorErr); err == nil {
		t.Fatalf("expected push error")
	}
}

func TestClientMethodsInvalidServer(t *testing.T) {
	client := &Client{Server: "://bad", Client: &http.Client{}}

	if _, err := client.LoginWithBody(context.Background(), "application/json", bytes.NewReader([]byte(`{}`))); err == nil {
		t.Fatalf("expected login error")
	}
	if _, err := client.RegisterWithBody(context.Background(), "application/json", bytes.NewReader([]byte(`{}`))); err == nil {
		t.Fatalf("expected register error")
	}
	if _, err := client.UpsertRecordWithBody(context.Background(), "application/json", bytes.NewReader([]byte(`{}`))); err == nil {
		t.Fatalf("expected upsert error")
	}
	if _, err := client.PushSyncWithBody(context.Background(), "application/json", bytes.NewReader([]byte(`{}`))); err == nil {
		t.Fatalf("expected push error")
	}
	if _, err := client.ListRecords(context.Background(), &ListRecordsParams{}); err == nil {
		t.Fatalf("expected list error")
	}
	if _, err := client.DeleteRecord(context.Background(), "id"); err == nil {
		t.Fatalf("expected delete error")
	}
	if _, err := client.GetRecord(context.Background(), "id"); err == nil {
		t.Fatalf("expected get error")
	}
	if _, err := client.PullSync(context.Background(), &PullSyncParams{}); err == nil {
		t.Fatalf("expected pull error")
	}
}

func TestClientWithResponsesError(t *testing.T) {
	_, err := NewClientWithResponses("http://example.com", func(c *Client) error {
		return errors.New("option")
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestClientWithResponsesMethodsError(t *testing.T) {
	client := &ClientWithResponses{&Client{Server: "://bad", Client: &http.Client{}}}

	if _, err := client.LoginWithResponse(context.Background(), LoginRequest{}); err == nil {
		t.Fatalf("expected login error")
	}
	if _, err := client.LoginWithBodyWithResponse(context.Background(), "application/json", bytes.NewReader([]byte(`{}`))); err == nil {
		t.Fatalf("expected login body error")
	}
	if _, err := client.RegisterWithResponse(context.Background(), RegisterRequest{}); err == nil {
		t.Fatalf("expected register error")
	}
	if _, err := client.RegisterWithBodyWithResponse(context.Background(), "application/json", bytes.NewReader([]byte(`{}`))); err == nil {
		t.Fatalf("expected register body error")
	}
	if _, err := client.ValidateWithResponse(context.Background()); err == nil {
		t.Fatalf("expected validate error")
	}
	if _, err := client.ListRecordsWithResponse(context.Background(), &ListRecordsParams{}); err == nil {
		t.Fatalf("expected list error")
	}
	if _, err := client.UpsertRecordWithResponse(context.Background(), RecordUpsert{}); err == nil {
		t.Fatalf("expected upsert error")
	}
	if _, err := client.UpsertRecordWithBodyWithResponse(context.Background(), "application/json", bytes.NewReader([]byte(`{}`))); err == nil {
		t.Fatalf("expected upsert body error")
	}
	if _, err := client.DeleteRecordWithResponse(context.Background(), "id"); err == nil {
		t.Fatalf("expected delete error")
	}
	if _, err := client.GetRecordWithResponse(context.Background(), "id"); err == nil {
		t.Fatalf("expected get error")
	}
	if _, err := client.PullSyncWithResponse(context.Background(), &PullSyncParams{}); err == nil {
		t.Fatalf("expected pull error")
	}
	if _, err := client.PushSyncWithResponse(context.Background(), SyncPush{}); err == nil {
		t.Fatalf("expected push error")
	}
	if _, err := client.PushSyncWithBodyWithResponse(context.Background(), "application/json", bytes.NewReader([]byte(`{}`))); err == nil {
		t.Fatalf("expected push body error")
	}
}
