package configutil

import (
	"testing"
	"time"
)

func TestOverrideEnvString(t *testing.T) {
	value := "default"
	if OverrideEnv(&value, "MISSING_ENV", ParseString) {
		t.Fatalf("unexpected override")
	}
	t.Setenv("TEST_STRING", "value")
	if !OverrideEnv(&value, "TEST_STRING", ParseString) || value != "value" {
		t.Fatalf("expected override")
	}
}

func TestOverrideEnvBool(t *testing.T) {
	value := false
	t.Setenv("TEST_BOOL", "true")
	if !OverrideEnv(&value, "TEST_BOOL", ParseBool) || !value {
		t.Fatalf("expected bool override")
	}
}

func TestOverrideEnvDuration(t *testing.T) {
	value := time.Second
	t.Setenv("TEST_DURATION", "2m")
	if !OverrideEnv(&value, "TEST_DURATION", ParseDuration) || value != 2*time.Minute {
		t.Fatalf("expected duration override")
	}
}

func TestOverrideEnvInvalid(t *testing.T) {
	value := false
	t.Setenv("TEST_INVALID_BOOL", "nope")
	if OverrideEnv(&value, "TEST_INVALID_BOOL", ParseBool) {
		t.Fatalf("expected invalid override")
	}
}
