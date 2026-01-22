package main

import (
	"testing"

	"gophkeeper/internal/buildinfo"
)

func TestBuildInfoVars(t *testing.T) {
	info := buildinfo.Info{Version: buildVersion, Date: buildDate, Commit: buildCommit}
	if info.Version == "" {
		t.Fatalf("expected version")
	}
}
