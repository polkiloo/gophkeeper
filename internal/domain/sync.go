package domain

import "time"

// ChangeType describes how a record changed.
type ChangeType string

const (
	// ChangeUpsert indicates the record was created or updated.
	ChangeUpsert ChangeType = "upsert"
	// ChangeDelete indicates the record was deleted.
	ChangeDelete ChangeType = "delete"
)

// RecordChange captures a delta for synchronization.
type RecordChange struct {
	RecordID   RecordID
	OwnerID    UserID
	Type       RecordType
	Change     ChangeType
	Payload    Payload
	Meta       Metadata
	Version    Version
	HappenedAt time.Time
}

// SyncBatch groups changes for a pull response.
type SyncBatch struct {
	Changes []RecordChange
	Cursor  SyncCursor
}

// SyncCursor represents a position in the change stream.
type SyncCursor string

// SyncResult summarizes an applied push request.
type SyncResult struct {
	Applied   int
	Rejected  int
	Conflicts []RecordID
}
