package client

import (
	"bytes"
	"context"
	"os"
	"testing"
)

func TestModuleProviders(t *testing.T) {
	if provideStdout() == nil {
		t.Fatalf("expected stdout")
	}
	if provideStderr() == nil {
		t.Fatalf("expected stderr")
	}
	if len(provideArgs()) != len(os.Args) {
		t.Fatalf("expected args")
	}
	result := provideRunResult()
	if cap(result) == 0 {
		t.Fatalf("expected buffered result")
	}
	if provideTransportKey(Config{TransportKey: "k"}) != "k" {
		t.Fatalf("unexpected transport key")
	}
	settings := provideTLSSettings(Config{TLSCAFile: "/tmp/ca", TLSInsecureSkipVerify: true})
	if settings.CAFile != "/tmp/ca" || !settings.InsecureSkipVerify {
		t.Fatalf("unexpected tls settings")
	}
}

func TestRunCLI(t *testing.T) {
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	runner := NewRunner([]string{"cli"}, Stdout(out), Stderr(errOut), NewClientFactory(nil), Config{BaseURL: DefaultBaseURL}, "")
	result := make(RunResult, 1)

	runCLI(context.Background(), runner, result)
	if err := <-result; err == nil {
		t.Fatalf("expected error")
	}
}
