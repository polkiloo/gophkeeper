package postgres

import (
	"encoding/json"
	"errors"

	"gophkeeper/internal/domain"
)

func encodeMetadata(meta domain.Metadata) ([]byte, error) {
	return json.Marshal(meta)
}

func decodeMetadata(raw []byte) (domain.Metadata, error) {
	if len(raw) == 0 {
		return domain.Metadata{}, nil
	}
	var meta domain.Metadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		return domain.Metadata{}, err
	}
	return meta, nil
}

func encodePayload(payload domain.Payload, recordType domain.RecordType) ([]byte, error) {
	if payload == nil {
		return nil, nil
	}

	switch value := payload.(type) {
	case domain.CredentialPayload:
		return json.Marshal(value)
	case *domain.CredentialPayload:
		return json.Marshal(value)
	case domain.TextPayload:
		return json.Marshal(value)
	case *domain.TextPayload:
		return json.Marshal(value)
	case domain.BinaryPayload:
		return json.Marshal(value)
	case *domain.BinaryPayload:
		return json.Marshal(value)
	case domain.BankCardPayload:
		return json.Marshal(value)
	case *domain.BankCardPayload:
		return json.Marshal(value)
	default:
		if recordType == "" {
			return nil, errors.New("unknown payload type")
		}
		return json.Marshal(payload)
	}
}

func decodePayload(recordType domain.RecordType, raw []byte) (domain.Payload, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	switch recordType {
	case domain.RecordTypeCredential:
		var payload domain.CredentialPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, err
		}
		return payload, nil
	case domain.RecordTypeText:
		var payload domain.TextPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, err
		}
		return payload, nil
	case domain.RecordTypeBinary:
		var payload domain.BinaryPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, err
		}
		return payload, nil
	case domain.RecordTypeBankCard:
		var payload domain.BankCardPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, err
		}
		return payload, nil
	default:
		return nil, errors.New("unknown record type")
	}
}
