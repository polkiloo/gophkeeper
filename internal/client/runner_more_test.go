package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunnerRecordsListGetDelete(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/api/v1/records":
				return jsonResponse(http.StatusOK, []any{
					map[string]any{
						"id":         "r1",
						"owner_id":   "u1",
						"type":       "text",
						"payload":    map[string]any{"kind": "text", "text": "hello"},
						"meta":       map[string]any{},
						"version":    1,
						"updated_at": now,
					},
				})
			case "/api/v1/records/r1":
				if req.Method == http.MethodDelete {
					return statusResponse(http.StatusNoContent), nil
				}
				return jsonResponse(http.StatusOK, map[string]any{
					"id":         "r1",
					"owner_id":   "u1",
					"type":       "text",
					"payload":    map[string]any{"kind": "text", "text": "hello"},
					"meta":       map[string]any{},
					"version":    1,
					"updated_at": now,
				})
			default:
				return statusResponse(http.StatusNotFound), nil
			}
		}),
	}

	runner := NewRunner([]string{"cli"}, Stdout(new(bytes.Buffer)), Stderr(new(bytes.Buffer)), NewClientFactory(httpClient), Config{BaseURL: "http://example.com/api"}, "")
	if err := runner.handleRecords(context.Background(), []string{"list", "-token", "t"}); err != nil {
		t.Fatalf("list error: %v", err)
	}
	if err := runner.handleRecords(context.Background(), []string{"get", "-token", "t", "-id", "r1"}); err != nil {
		t.Fatalf("get error: %v", err)
	}
	if err := runner.handleRecords(context.Background(), []string{"delete", "-token", "t", "-id", "r1"}); err != nil {
		t.Fatalf("delete error: %v", err)
	}
}

func TestRunnerSyncPullPush(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sync.json")
	data, _ := json.Marshal(map[string]any{
		"changes": []any{},
	})
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write sync file: %v", err)
	}

	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/api/v1/sync":
				if req.Method == http.MethodGet {
					return jsonResponse(http.StatusOK, map[string]any{
						"changes": []any{},
						"cursor":  "1",
					})
				}
				body := map[string]any{
					"applied":   1,
					"rejected":  0,
					"conflicts": []string{},
				}
				return jsonResponse(http.StatusOK, body)
			default:
				return statusResponse(http.StatusNotFound), nil
			}
		}),
	}

	runner := NewRunner([]string{"cli"}, Stdout(new(bytes.Buffer)), Stderr(new(bytes.Buffer)), NewClientFactory(httpClient), Config{BaseURL: "http://example.com/api"}, "")
	if err := runner.handleSync(context.Background(), []string{"pull", "-token", "t"}); err != nil {
		t.Fatalf("pull error: %v", err)
	}
	if err := runner.handleSync(context.Background(), []string{"push", "-token", "t", "-file", path, "-base-url", "http://example.com/api"}); err != nil {
		t.Fatalf("push error: %v", err)
	}
}

func TestRunnerRecordsUpsert(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/api/v1/records" {
				return jsonResponse(http.StatusOK, map[string]any{
					"id":         "r1",
					"owner_id":   "u1",
					"type":       "text",
					"payload":    map[string]any{"kind": "text", "text": "hello"},
					"meta":       map[string]any{},
					"version":    1,
					"updated_at": now,
				})
			}
			return statusResponse(http.StatusNotFound), nil
		}),
	}

	runner := NewRunner([]string{"cli"}, Stdout(new(bytes.Buffer)), Stderr(new(bytes.Buffer)), NewClientFactory(httpClient), Config{BaseURL: "http://example.com/api"}, "")
	err := runner.handleRecords(context.Background(), []string{"upsert", "-token", "t", "-type", "text", "-text", "hello", "-id", "r1"})
	if err != nil {
		t.Fatalf("upsert error: %v", err)
	}
}

func TestRunnerAuthValidate(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/api/v1/auth/validate" {
				return jsonResponse(http.StatusOK, map[string]any{
					"user_id":    "u1",
					"token":      "token",
					"expires_at": now,
				})
			}
			return statusResponse(http.StatusNotFound), nil
		}),
	}

	runner := NewRunner([]string{"cli"}, Stdout(new(bytes.Buffer)), Stderr(new(bytes.Buffer)), NewClientFactory(httpClient), Config{BaseURL: "http://example.com/api"}, "")
	if err := runner.handleAuth(context.Background(), []string{"validate", "-token", "token"}); err != nil {
		t.Fatalf("validate error: %v", err)
	}
}

func TestRunnerAuthLogin(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/api/v1/auth/login" {
				return jsonResponse(http.StatusOK, map[string]any{
					"user_id":    "u1",
					"token":      "token",
					"expires_at": now,
				})
			}
			return statusResponse(http.StatusNotFound), nil
		}),
	}

	runner := NewRunner([]string{"cli"}, Stdout(new(bytes.Buffer)), Stderr(new(bytes.Buffer)), NewClientFactory(httpClient), Config{BaseURL: "http://example.com/api"}, "")
	if err := runner.handleAuth(context.Background(), []string{"login", "-login", "user", "-password", "pass"}); err != nil {
		t.Fatalf("login error: %v", err)
	}
}

func TestRunnerRecordsListWithFilters(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/api/v1/records" {
				return jsonResponse(http.StatusOK, []any{
					map[string]any{
						"id":         "r1",
						"owner_id":   "u1",
						"type":       "text",
						"payload":    map[string]any{"kind": "text", "text": "hello"},
						"meta":       map[string]any{},
						"version":    1,
						"updated_at": now,
					},
				})
			}
			return statusResponse(http.StatusNotFound), nil
		}),
	}

	runner := NewRunner([]string{"cli"}, Stdout(new(bytes.Buffer)), Stderr(new(bytes.Buffer)), NewClientFactory(httpClient), Config{BaseURL: "http://example.com/api"}, "")
	err := runner.handleRecords(context.Background(), []string{"list", "-token", "t", "-type", "text", "-tag", "tag1", "-query", "q", "-limit", "10", "-offset", "1"})
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
}

func TestRunnerRecordsUpsertCredential(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/api/v1/records" {
				return jsonResponse(http.StatusOK, map[string]any{
					"id":         "r1",
					"owner_id":   "u1",
					"type":       "credential",
					"payload":    map[string]any{"kind": "credential", "login": "user", "password": "pass"},
					"meta":       map[string]any{},
					"version":    1,
					"updated_at": now,
				})
			}
			return statusResponse(http.StatusNotFound), nil
		}),
	}

	runner := NewRunner([]string{"cli"}, Stdout(new(bytes.Buffer)), Stderr(new(bytes.Buffer)), NewClientFactory(httpClient), Config{BaseURL: "http://example.com/api"}, "")
	err := runner.handleRecords(context.Background(), []string{
		"upsert", "-token", "t", "-type", "credential", "-login", "user", "-password", "pass", "-tags", "t1,t2",
		"-attrs", "k=v",
	})
	if err != nil {
		t.Fatalf("upsert error: %v", err)
	}
}

func jsonResponse(status int, payload any) (*http.Response, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	resp := statusResponse(status)
	resp.Header.Set("Content-Type", "application/json")
	resp.Body = io.NopCloser(bytes.NewReader(data))
	resp.ContentLength = int64(len(data))
	return resp, nil
}

func statusResponse(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(nil)),
	}
}
