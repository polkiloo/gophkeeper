package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gophkeeper/internal/ports/remote/openapi"
	"gopkg.in/yaml.v3"
)

func TestBuildPayload(t *testing.T) {
	payload, err := buildPayload(openapi.Credential, "login", "pass", "", "", "", "", "", "")
	if err != nil {
		t.Fatalf("credential payload error: %v", err)
	}
	cred, err := payload.AsCredentialPayload()
	if err != nil || cred.Login != "login" {
		t.Fatalf("credential payload invalid")
	}

	payload, err = buildPayload(openapi.Text, "", "", "text", "", "", "", "", "")
	if err != nil {
		t.Fatalf("text payload error: %v", err)
	}
	text, err := payload.AsTextPayload()
	if err != nil || text.Text != "text" {
		t.Fatalf("text payload invalid")
	}

	encoded := base64.StdEncoding.EncodeToString([]byte("bin"))
	payload, err = buildPayload(openapi.Binary, "", "", "", encoded, "", "", "", "")
	if err != nil {
		t.Fatalf("binary payload error: %v", err)
	}
	binary, err := payload.AsBinaryPayload()
	if err != nil || string(binary.Data) != "bin" {
		t.Fatalf("binary payload invalid")
	}

	payload, err = buildPayload(openapi.BankCard, "", "", "", "", "User", "4111", "10/28", "123")
	if err != nil {
		t.Fatalf("bank card payload error: %v", err)
	}
	card, err := payload.AsBankCardPayload()
	if err != nil || card.Number != "4111" {
		t.Fatalf("bank card payload invalid")
	}
}

func TestParseAttrs(t *testing.T) {
	attrs, err := parseAttrs("a=b, c=d")
	if err != nil {
		t.Fatalf("parse attrs error: %v", err)
	}
	if attrs["a"] != "b" || attrs["c"] != "d" {
		t.Fatalf("unexpected attrs")
	}

	if _, err := parseAttrs("invalid"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestRunnerUnknownCommand(t *testing.T) {
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	runner := NewRunner([]string{"cli", "unknown"}, Stdout(out), Stderr(errOut), NewClientFactory(nil), Config{BaseURL: DefaultBaseURL}, "")

	if err := runner.Run(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
	if errOut.Len() == 0 {
		t.Fatalf("expected usage output")
	}
}

func TestRunnerAuthRegister(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/api/v1/auth/register" {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(bytes.NewReader(nil)),
					Header:     make(http.Header),
				}, nil
			}
			body, _ := json.Marshal(map[string]any{
				"id":         "u1",
				"login":      "login",
				"created_at": time.Now().Format(time.RFC3339),
			})
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(body)),
				Header:     make(http.Header),
			}
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil
		}),
	}

	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cfg := Config{BaseURL: "http://example.com/api"}
	runner := NewRunner([]string{"cli"}, Stdout(out), Stderr(errOut), NewClientFactory(httpClient), cfg, "")

	if err := runner.handleAuth(context.Background(), []string{"register", "-login", "login", "-password", "pass"}); err != nil {
		t.Fatalf("register error: %v", err)
	}
	if out.Len() == 0 {
		t.Fatalf("expected output")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestRunnerConfigInit(t *testing.T) {
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	dir := t.TempDir()
	path := filepath.Join(dir, "client.yaml")

	runner := NewRunner([]string{"cli"}, Stdout(out), Stderr(errOut), NewClientFactory(nil), Config{BaseURL: DefaultBaseURL}, ConfigPath(path))
	if err := runner.handleConfig(context.Background(), []string{"init", "-path", path, "-base-url", "http://example.com/api"}); err != nil {
		t.Fatalf("config init error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("config read error: %v", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("config unmarshal error: %v", err)
	}
	if cfg.BaseURL != "http://example.com/api" {
		t.Fatalf("unexpected baseurl: %s", cfg.BaseURL)
	}
}

func TestRunnerConfigUnknown(t *testing.T) {
	runner := NewRunner([]string{"cli"}, Stdout(new(bytes.Buffer)), Stderr(new(bytes.Buffer)), NewClientFactory(nil), Config{BaseURL: DefaultBaseURL}, "")
	if err := runner.handleConfig(context.Background(), []string{"unknown"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestRunnerAuthValidateMissingToken(t *testing.T) {
	runner := NewRunner([]string{"cli"}, Stdout(new(bytes.Buffer)), Stderr(new(bytes.Buffer)), NewClientFactory(nil), Config{BaseURL: DefaultBaseURL}, "")
	if err := runner.handleAuth(context.Background(), []string{"validate"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestRunnerRecordsMissingToken(t *testing.T) {
	runner := NewRunner([]string{"cli"}, Stdout(new(bytes.Buffer)), Stderr(new(bytes.Buffer)), NewClientFactory(nil), Config{BaseURL: DefaultBaseURL}, "")
	if err := runner.handleRecords(context.Background(), []string{"list"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestRunnerSyncPushMissingFile(t *testing.T) {
	runner := NewRunner([]string{"cli"}, Stdout(new(bytes.Buffer)), Stderr(new(bytes.Buffer)), NewClientFactory(nil), Config{BaseURL: DefaultBaseURL}, "")
	if err := runner.handleSync(context.Background(), []string{"push"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestPrintResponseError(t *testing.T) {
	runner := NewRunner([]string{"cli"}, Stdout(new(bytes.Buffer)), Stderr(new(bytes.Buffer)), NewClientFactory(nil), Config{BaseURL: DefaultBaseURL}, "")
	resp := &http.Response{StatusCode: http.StatusInternalServerError, Status: "500 Internal Server Error"}
	if err := runner.printResponse(resp, []byte("boom"), nil); err == nil {
		t.Fatalf("expected error")
	}
}
