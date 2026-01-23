package domain

import "time"

// UserID identifies a user in the system.
type UserID string

// RecordID identifies a private record in the system.
type RecordID string

// RecordType defines the kind of private data stored.
type RecordType string

const (
	// RecordTypeCredential stores a login/password pair.
	RecordTypeCredential RecordType = "credential"
	// RecordTypeText stores arbitrary text data.
	RecordTypeText RecordType = "text"
	// RecordTypeBinary stores arbitrary binary data.
	RecordTypeBinary RecordType = "binary"
	// RecordTypeBankCard stores bank card details.
	RecordTypeBankCard RecordType = "bank_card"
)

// Metadata holds user-provided annotations for a record.
type Metadata struct {
	Title       string
	Description string
	Tags        []string
	Attributes  map[string]string
}

// Version represents a monotonically increasing version value.
type Version int64

// Record represents a single piece of private data.
type Record struct {
	ID        RecordID
	OwnerID   UserID
	Type      RecordType
	Payload   Payload
	Meta      Metadata
	Version   Version
	UpdatedAt time.Time
}

// Payload represents record contents with a stable type.
type Payload interface {
	RecordType() RecordType
}
