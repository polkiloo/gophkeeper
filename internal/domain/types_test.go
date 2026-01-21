package domain

import (
	"errors"
	"testing"
)

func TestPayloadRecordTypes(t *testing.T) {
	cases := []struct {
		payload Payload
		want    RecordType
	}{
		{payload: CredentialPayload{}, want: RecordTypeCredential},
		{payload: TextPayload{}, want: RecordTypeText},
		{payload: BinaryPayload{}, want: RecordTypeBinary},
		{payload: BankCardPayload{}, want: RecordTypeBankCard},
	}

	for _, tt := range cases {
		if got := tt.payload.RecordType(); got != tt.want {
			t.Fatalf("expected %s, got %s", tt.want, got)
		}
	}
}

func TestDomainErrors(t *testing.T) {
	if ErrNotFound == nil || ErrUnauthorized == nil || ErrConflict == nil {
		t.Fatalf("expected domain errors to be set")
	}
	if !errors.Is(ErrNotFound, ErrNotFound) {
		t.Fatalf("expected ErrNotFound to match itself")
	}
}
