package httpapi

import (
	"errors"
	"fmt"

	"gophkeeper/internal/domain"
)

func userToResponse(user domain.User) User {
	return User{Id: string(user.ID), Login: user.Login, CreatedAt: user.CreatedAt}
}

func sessionToResponse(session domain.Session) Session {
	return Session{UserId: string(session.UserID), Token: session.Token, ExpiresAt: session.ExpiresAt}
}

func metadataFromOpenAPI(meta Metadata) domain.Metadata {
	return domain.Metadata{
		Title:       derefString(meta.Title),
		Description: derefString(meta.Description),
		Tags:        derefStringSlice(meta.Tags),
		Attributes:  derefStringMap(meta.Attributes),
	}
}

func metadataToOpenAPI(meta domain.Metadata) Metadata {
	return Metadata{
		Title:       stringPtr(meta.Title),
		Description: stringPtr(meta.Description),
		Tags:        stringSlicePtr(meta.Tags),
		Attributes:  stringMapPtr(meta.Attributes),
	}
}

func payloadFromOpenAPI(payload Payload, recordType RecordType) (domain.Payload, error) {
	switch recordType {
	case Credential:
		value, err := payload.AsCredentialPayload()
		if err != nil {
			return nil, err
		}
		if value.Login == "" || value.Password == "" {
			return nil, errors.New("credential payload requires login and password")
		}
		return domain.CredentialPayload{Login: value.Login, Password: value.Password}, nil
	case Text:
		value, err := payload.AsTextPayload()
		if err != nil {
			return nil, err
		}
		return domain.TextPayload{Text: value.Text}, nil
	case Binary:
		value, err := payload.AsBinaryPayload()
		if err != nil {
			return nil, err
		}
		return domain.BinaryPayload{Data: value.Data}, nil
	case BankCard:
		value, err := payload.AsBankCardPayload()
		if err != nil {
			return nil, err
		}
		if value.Cardholder == "" || value.Number == "" || value.ExpiresAt == "" || value.Cvv == "" {
			return nil, errors.New("bank card payload requires card details")
		}
		return domain.BankCardPayload{Cardholder: value.Cardholder, Number: value.Number, ExpiresAt: value.ExpiresAt, CVV: value.Cvv}, nil
	default:
		return nil, fmt.Errorf("unknown payload kind: %s", recordType)
	}
}

func payloadToOpenAPI(payload domain.Payload) (Payload, error) {
	var result Payload
	switch value := payload.(type) {
	case domain.CredentialPayload:
		err := result.FromCredentialPayload(CredentialPayload{Kind: Credential, Login: value.Login, Password: value.Password})
		return result, err
	case domain.TextPayload:
		err := result.FromTextPayload(TextPayload{Kind: Text, Text: value.Text})
		return result, err
	case domain.BinaryPayload:
		err := result.FromBinaryPayload(BinaryPayload{Kind: Binary, Data: value.Data})
		return result, err
	case domain.BankCardPayload:
		err := result.FromBankCardPayload(BankCardPayload{Kind: BankCard, Cardholder: value.Cardholder, Number: value.Number, ExpiresAt: value.ExpiresAt, Cvv: value.CVV})
		return result, err
	default:
		return Payload{}, errors.New("unsupported payload type")
	}
}

func recordFromOpenAPI(req RecordUpsert) (domain.Record, error) {
	if req.Type == "" {
		return domain.Record{}, errors.New("record type is required")
	}
	payload, err := payloadFromOpenAPI(req.Payload, req.Type)
	if err != nil {
		return domain.Record{}, err
	}

	return domain.Record{
		ID:      domain.RecordID(derefString(req.Id)),
		Type:    domain.RecordType(req.Type),
		Payload: payload,
		Meta:    metadataFromOpenAPI(req.Meta),
		Version: domain.Version(derefInt64(req.Version)),
	}, nil
}

func recordToOpenAPI(record domain.Record) (Record, error) {
	payload, err := payloadToOpenAPI(record.Payload)
	if err != nil {
		return Record{}, err
	}

	return Record{
		Id:        string(record.ID),
		OwnerId:   string(record.OwnerID),
		Type:      RecordType(record.Type),
		Payload:   payload,
		Meta:      metadataToOpenAPI(record.Meta),
		Version:   int64(record.Version),
		UpdatedAt: record.UpdatedAt,
	}, nil
}

func changeFromOpenAPI(req RecordChange) (domain.RecordChange, error) {
	var payload domain.Payload
	var err error
	if req.Payload != nil {
		payload, err = payloadFromOpenAPI(*req.Payload, req.Type)
		if err != nil {
			return domain.RecordChange{}, err
		}
	}

	return domain.RecordChange{
		RecordID:   domain.RecordID(req.RecordId),
		OwnerID:    domain.UserID(req.OwnerId),
		Type:       domain.RecordType(req.Type),
		Change:     domain.ChangeType(req.Change),
		Payload:    payload,
		Meta:       metadataFromOpenAPI(derefMetadata(req.Meta)),
		Version:    domain.Version(req.Version),
		HappenedAt: req.HappenedAt,
	}, nil
}

func changeToOpenAPI(change domain.RecordChange) (RecordChange, error) {
	var payload *Payload
	if change.Payload != nil {
		converted, err := payloadToOpenAPI(change.Payload)
		if err != nil {
			return RecordChange{}, err
		}
		payload = &converted
	}

	meta := metadataToOpenAPI(change.Meta)
	return RecordChange{
		RecordId:   string(change.RecordID),
		OwnerId:    string(change.OwnerID),
		Type:       RecordType(change.Type),
		Change:     ChangeType(change.Change),
		Payload:    payload,
		Meta:       &meta,
		Version:    int64(change.Version),
		HappenedAt: change.HappenedAt,
	}, nil
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func stringSlicePtr(value []string) *[]string {
	if len(value) == 0 {
		return nil
	}
	return &value
}

func stringMapPtr(value map[string]string) *map[string]string {
	if len(value) == 0 {
		return nil
	}
	return &value
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func derefStringSlice(value *[]string) []string {
	if value == nil {
		return nil
	}
	return *value
}

func derefStringMap(value *map[string]string) map[string]string {
	if value == nil {
		return nil
	}
	return *value
}

func derefInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func derefMetadata(value *Metadata) Metadata {
	if value == nil {
		return Metadata{}
	}
	return *value
}
