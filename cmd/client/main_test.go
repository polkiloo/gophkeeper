package main

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/fx"
	"gophkeeper/internal/buildinfo"
	"gophkeeper/internal/client"
)

func TestBuildInfoVars(t *testing.T) {
	info := buildinfo.Info{Version: buildVersion, Date: buildDate, Commit: buildCommit}
	if info.Version == "" {
		t.Fatalf("expected version")
	}
}

func TestRunStops(t *testing.T) {
	ctx := context.Background()
	app := fx.New()
	result := make(client.RunResult, 1)
	result <- nil
	close(result)

	if err := run(ctx, app, result); err != nil {
		t.Fatalf("run error: %v", err)
	}
}

func TestRunStartError(t *testing.T) {
	app := fx.New(fx.Error(errors.New("boom")))
	result := make(client.RunResult, 1)
	if err := run(context.Background(), app, result); err == nil {
		t.Fatalf("expected error")
	}
}

func TestRunResultError(t *testing.T) {
	ctx := context.Background()
	app := fx.New()
	result := make(client.RunResult, 1)
	result <- errors.New("boom")
	close(result)

	if err := run(ctx, app, result); err == nil {
		t.Fatalf("expected error")
	}
}
