package system

import "testing"

func TestClockNow(t *testing.T) {
	clock := Clock{}
	if clock.Now().IsZero() {
		t.Fatalf("expected non-zero time")
	}
}

func TestIDGenerator(t *testing.T) {
	gen := IDGenerator{}
	id := gen.NewID()
	if id == "" {
		t.Fatalf("expected id")
	}
	if len(id) != 32 {
		t.Fatalf("unexpected id length: %d", len(id))
	}
}
