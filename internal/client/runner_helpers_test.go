package client

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"testing"

	"gophkeeper/internal/ports/remote/openapi"
)

func TestRunnerUsageErrors(t *testing.T) {
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	runner := NewRunner([]string{"cli"}, Stdout(out), Stderr(errOut), NewClientFactory(nil), Config{BaseURL: DefaultBaseURL}, "")

	if err := runner.Run(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
	if errOut.Len() == 0 {
		t.Fatalf("expected usage output")
	}
}

func TestBuildPayloadErrors(t *testing.T) {
	if _, err := buildPayload(openapi.Credential, "", "", "", "", "", "", "", ""); err == nil {
		t.Fatalf("expected credential error")
	}
	if _, err := buildPayload(openapi.Text, "", "", "", "", "", "", "", ""); err == nil {
		t.Fatalf("expected text error")
	}
	if _, err := buildPayload(openapi.Binary, "", "", "", "not-base64", "", "", "", ""); err == nil {
		t.Fatalf("expected binary error")
	}
	if _, err := buildPayload(openapi.BankCard, "", "", "", "", "", "", "", ""); err == nil {
		t.Fatalf("expected bank card error")
	}
	if _, err := buildPayload(openapi.RecordType("other"), "", "", "", "", "", "", "", ""); err == nil {
		t.Fatalf("expected unknown type error")
	}
}

func TestHelpers(t *testing.T) {
	if _, err := decodeBase64(""); err == nil {
		t.Fatalf("expected decode error")
	}
	if got := splitComma(" a, , b "); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("unexpected split")
	}
	if stringPtr("") != nil {
		t.Fatalf("expected nil string ptr")
	}
	if stringPtr("x") == nil {
		t.Fatalf("expected string ptr")
	}
	if stringSlicePtr(nil) != nil {
		t.Fatalf("expected nil slice ptr")
	}
	if stringMapPtr(nil) != nil {
		t.Fatalf("expected nil map ptr")
	}
}

func TestTokenOrEnv(t *testing.T) {
	t.Setenv("GOPHKEEPER_TOKEN", "env-token")
	if tokenOrEnv("") != "env-token" {
		t.Fatalf("expected env token")
	}
	if tokenOrEnv("inline") != "inline" {
		t.Fatalf("expected inline token")
	}
}

func TestPrintResponseSuccess(t *testing.T) {
	out := new(bytes.Buffer)
	runner := NewRunner([]string{"cli"}, Stdout(out), Stderr(new(bytes.Buffer)), NewClientFactory(nil), Config{BaseURL: DefaultBaseURL}, "")
	resp := &http.Response{StatusCode: http.StatusNoContent, Status: "204 No Content"}
	if err := runner.printResponse(resp, nil, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Len() == 0 {
		t.Fatalf("expected output")
	}
}

func TestWriteConfigErrors(t *testing.T) {
	runner := NewRunner([]string{"cli"}, Stdout(new(bytes.Buffer)), Stderr(new(bytes.Buffer)), NewClientFactory(nil), Config{BaseURL: DefaultBaseURL}, "")
	if err := runner.writeConfig("", Config{}); err == nil {
		t.Fatalf("expected error")
	}
	if err := runner.writeConfig(os.TempDir(), Config{BaseURL: ""}); err == nil {
		t.Fatalf("expected write error")
	}
}

func TestRunnerUnknownSubcommands(t *testing.T) {
	runner := NewRunner([]string{"cli"}, Stdout(new(bytes.Buffer)), Stderr(new(bytes.Buffer)), NewClientFactory(nil), Config{BaseURL: DefaultBaseURL}, "")
	if err := runner.handleAuth(context.Background(), []string{"unknown"}); err == nil {
		t.Fatalf("expected auth error")
	}
	if err := runner.handleRecords(context.Background(), []string{"unknown"}); err == nil {
		t.Fatalf("expected records error")
	}
	if err := runner.handleSync(context.Background(), []string{"unknown"}); err == nil {
		t.Fatalf("expected sync error")
	}
}

func TestResolveConfigPath(t *testing.T) {
	runner := NewRunner([]string{"cli"}, Stdout(new(bytes.Buffer)), Stderr(new(bytes.Buffer)), NewClientFactory(nil), Config{BaseURL: DefaultBaseURL}, ConfigPath("/tmp/client.yaml"))
	if runner.resolveConfigPath() != "/tmp/client.yaml" {
		t.Fatalf("unexpected config path")
	}
}

func TestRunnerRunConfigInit(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/client.yaml"
	runner := NewRunner([]string{"cli", "config", "init", "-path", path, "-base-url", "http://example.com/api"}, Stdout(new(bytes.Buffer)), Stderr(new(bytes.Buffer)), NewClientFactory(nil), Config{BaseURL: DefaultBaseURL}, "")
	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("run error: %v", err)
	}
}

func TestNewClientFactory(t *testing.T) {
	factory := NewClientFactory(nil)
	if _, err := factory("http://example.com"); err != nil {
		t.Fatalf("expected client: %v", err)
	}
	factory = NewClientFactory(&http.Client{})
	if _, err := factory("http://example.com"); err != nil {
		t.Fatalf("expected client: %v", err)
	}
}

func TestRunnerRunBranches(t *testing.T) {
	cases := [][]string{
		{"cli", "auth"},
		{"cli", "records"},
		{"cli", "sync"},
	}

	for _, args := range cases {
		runner := NewRunner(args, Stdout(new(bytes.Buffer)), Stderr(new(bytes.Buffer)), NewClientFactory(nil), Config{BaseURL: DefaultBaseURL}, "")
		if err := runner.Run(context.Background()); err == nil {
			t.Fatalf("expected error for %v", args)
		}
	}
}
