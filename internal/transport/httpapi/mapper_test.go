package httpapi

import (
	"testing"
	"time"

	"gophkeeper/internal/domain"
)

type unknownPayload struct{}

func (unknownPayload) RecordType() domain.RecordType { return "unknown" }

func TestMetadataRoundTrip(t *testing.T) {
	meta := domain.Metadata{
		Title:       "title",
		Description: "description",
		Tags:        []string{"t1", "t2"},
		Attributes:  map[string]string{"site": "example.com"},
	}

	open := metadataToOpenAPI(meta)
	back := metadataFromOpenAPI(open)

	if back.Title != meta.Title || back.Description != meta.Description {
		t.Fatalf("metadata mismatch")
	}
	if len(back.Tags) != len(meta.Tags) || back.Attributes["site"] != "example.com" {
		t.Fatalf("metadata mismatch")
	}
}

func TestPayloadMapping(t *testing.T) {
	payload, err := payloadToOpenAPI(domain.CredentialPayload{Login: "l", Password: "p"})
	if err != nil {
		t.Fatalf("credential to openapi error: %v", err)
	}
	back, err := payloadFromOpenAPI(payload, Credential)
	if err != nil {
		t.Fatalf("credential from openapi error: %v", err)
	}
	if back.(domain.CredentialPayload).Login != "l" {
		t.Fatalf("credential mismatch")
	}

	payload, err = payloadToOpenAPI(domain.TextPayload{Text: "txt"})
	if err != nil {
		t.Fatalf("text to openapi error: %v", err)
	}
	back, err = payloadFromOpenAPI(payload, Text)
	if err != nil {
		t.Fatalf("text from openapi error: %v", err)
	}
	if back.(domain.TextPayload).Text != "txt" {
		t.Fatalf("text mismatch")
	}

	payload, err = payloadToOpenAPI(domain.BinaryPayload{Data: []byte("bin")})
	if err != nil {
		t.Fatalf("binary to openapi error: %v", err)
	}
	back, err = payloadFromOpenAPI(payload, Binary)
	if err != nil {
		t.Fatalf("binary from openapi error: %v", err)
	}
	if string(back.(domain.BinaryPayload).Data) != "bin" {
		t.Fatalf("binary mismatch")
	}

	payload, err = payloadToOpenAPI(domain.BankCardPayload{Cardholder: "c", Number: "1", ExpiresAt: "12/30", CVV: "123"})
	if err != nil {
		t.Fatalf("bank card to openapi error: %v", err)
	}
	back, err = payloadFromOpenAPI(payload, BankCard)
	if err != nil {
		t.Fatalf("bank card from openapi error: %v", err)
	}
	if back.(domain.BankCardPayload).Cardholder != "c" {
		t.Fatalf("bank card mismatch")
	}
}

func TestPayloadFromOpenAPIErrors(t *testing.T) {
	var payload Payload
	_ = payload.FromCredentialPayload(CredentialPayload{Kind: Credential, Login: "", Password: ""})
	if _, err := payloadFromOpenAPI(payload, Credential); err == nil {
		t.Fatalf("expected credential error")
	}

	payload = Payload{}
	_ = payload.FromBankCardPayload(BankCardPayload{Kind: BankCard, Cardholder: ""})
	if _, err := payloadFromOpenAPI(payload, BankCard); err == nil {
		t.Fatalf("expected bank card error")
	}

	payload = Payload{}
	if _, err := payloadFromOpenAPI(payload, RecordType("unknown")); err == nil {
		t.Fatalf("expected unknown type error")
	}
}

func TestPayloadToOpenAPIErrors(t *testing.T) {
	if _, err := payloadToOpenAPI(unknownPayload{}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestChangeFromOpenAPINilMeta(t *testing.T) {
	change := RecordChange{
		RecordId:   "r1",
		OwnerId:    "u1",
		Type:       Text,
		Change:     Upsert,
		Version:    1,
		HappenedAt: time.Now(),
	}

	if _, err := changeFromOpenAPI(change); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRecordMapping(t *testing.T) {
	payload, err := payloadToOpenAPI(domain.TextPayload{Text: "hello"})
	if err != nil {
		t.Fatalf("payload error: %v", err)
	}

	upsert := RecordUpsert{Type: Text, Payload: payload, Meta: Metadata{}}
	record, err := recordFromOpenAPI(upsert)
	if err != nil {
		t.Fatalf("record from openapi error: %v", err)
	}
	if record.Type != domain.RecordTypeText {
		t.Fatalf("record type mismatch")
	}

	mapped, err := recordToOpenAPI(domain.Record{ID: "r1", OwnerID: "u1", Type: domain.RecordTypeText, Payload: domain.TextPayload{Text: "hello"}, Meta: domain.Metadata{}, Version: 1, UpdatedAt: time.Now()})
	if err != nil {
		t.Fatalf("record to openapi error: %v", err)
	}
	if mapped.Id != "r1" || mapped.OwnerId != "u1" {
		t.Fatalf("record mapping mismatch")
	}
}

func TestChangeMapping(t *testing.T) {
	change := domain.RecordChange{
		RecordID:   "r1",
		OwnerID:    "u1",
		Type:       domain.RecordTypeText,
		Change:     domain.ChangeUpsert,
		Payload:    domain.TextPayload{Text: "hello"},
		Meta:       domain.Metadata{},
		Version:    1,
		HappenedAt: time.Now(),
	}

	mapped, err := changeToOpenAPI(change)
	if err != nil {
		t.Fatalf("change to openapi error: %v", err)
	}
	back, err := changeFromOpenAPI(mapped)
	if err != nil {
		t.Fatalf("change from openapi error: %v", err)
	}
	if back.RecordID != change.RecordID || back.Change != change.Change {
		t.Fatalf("change mapping mismatch")
	}
}

func TestRecordFromOpenAPIErrors(t *testing.T) {
	_, err := recordFromOpenAPI(RecordUpsert{})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestDerefHelpers(t *testing.T) {
	if derefInt64(nil) != 0 {
		t.Fatalf("expected zero")
	}
	value := int64(7)
	if derefInt64(&value) != 7 {
		t.Fatalf("unexpected value")
	}
	if derefMetadata(nil).Title != nil {
		t.Fatalf("expected empty metadata")
	}
}
