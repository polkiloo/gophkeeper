package postgres

import (
	"testing"

	"gophkeeper/internal/domain"
)

type customPayload struct{}

func (customPayload) RecordType() domain.RecordType { return "custom" }

func TestMetadataCodec(t *testing.T) {
	meta := domain.Metadata{Title: "t", Tags: []string{"a"}}
	raw, err := encodeMetadata(meta)
	if err != nil {
		t.Fatalf("encode meta error: %v", err)
	}
	decoded, err := decodeMetadata(raw)
	if err != nil || decoded.Title != "t" || len(decoded.Tags) != 1 {
		t.Fatalf("decode meta error")
	}
	decoded, err = decodeMetadata(nil)
	if err != nil || decoded.Title != "" {
		t.Fatalf("decode empty meta error")
	}
}

func TestPayloadCodec(t *testing.T) {
	if raw, err := encodePayload(nil, ""); err != nil || raw != nil {
		t.Fatalf("expected nil payload")
	}
	if _, err := encodePayload(customPayload{}, ""); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := decodePayload("unknown", []byte(`{}`)); err == nil {
		t.Fatalf("expected decode error")
	}

	raw, err := encodePayload(domain.TextPayload{Text: "hi"}, domain.RecordTypeText)
	if err != nil {
		t.Fatalf("encode text error: %v", err)
	}
	payload, err := decodePayload(domain.RecordTypeText, raw)
	if err != nil {
		t.Fatalf("decode text error: %v", err)
	}
	if payload.(domain.TextPayload).Text != "hi" {
		t.Fatalf("unexpected payload")
	}
	if payload, err = decodePayload(domain.RecordTypeBinary, nil); err != nil || payload != nil {
		t.Fatalf("expected nil payload")
	}
}
