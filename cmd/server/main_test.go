package main

import "testing"

func TestBuildInfoVars(t *testing.T) {
	if buildVersion == "" {
		t.Fatalf("expected version")
	}
}
