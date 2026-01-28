package buildinfo

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrint(t *testing.T) {
	buf := new(bytes.Buffer)
	Print(buf, Info{Version: "v1", Date: "2024-01-01", Commit: "abc"})

	out := buf.String()
	if !strings.Contains(out, "version=v1") || !strings.Contains(out, "commit=abc") {
		t.Fatalf("unexpected output: %s", out)
	}
}
