package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/fx"
)

func TestBuildInfoVars(t *testing.T) {
	if buildVersion == "" {
		t.Fatalf("expected version")
	}
}

func TestRunStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	app := fx.New()
	if err := run(ctx, app); err != nil {
		t.Fatalf("run error: %v", err)
	}
}

func TestRunStartError(t *testing.T) {
	app := fx.New(fx.Error(errors.New("boom")))
	if err := run(context.Background(), app); err == nil {
		t.Fatalf("expected error")
	}
}
