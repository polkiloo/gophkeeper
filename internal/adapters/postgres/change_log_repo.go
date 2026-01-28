package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"gophkeeper/internal/domain"
)

// ChangeLogRepository stores sync changes in PostgreSQL.
type ChangeLogRepository struct {
	db *sql.DB
}

// NewChangeLogRepository constructs a PostgreSQL-backed changelog repository.
func NewChangeLogRepository(db *sql.DB) *ChangeLogRepository {
	return &ChangeLogRepository{db: db}
}

// Append adds a change to the stream.
func (r *ChangeLogRepository) Append(ctx context.Context, change domain.RecordChange) error {
	payload, err := encodePayload(change.Payload, change.Type)
	if err != nil {
		return err
	}
	meta, err := encodeMetadata(change.Meta)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO change_log (record_id, owner_id, type, change, payload, meta, version, happened_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, string(change.RecordID), string(change.OwnerID), string(change.Type), string(change.Change), payload, meta, int64(change.Version), change.HappenedAt)
	return err
}

// List returns changes after the cursor.
func (r *ChangeLogRepository) List(ctx context.Context, ownerID domain.UserID, cursor domain.SyncCursor, limit int) ([]domain.RecordChange, domain.SyncCursor, error) {
	startID := int64(0)
	if cursor != "" {
		if parsed, err := strconv.ParseInt(string(cursor), 10, 64); err == nil && parsed > 0 {
			startID = parsed
		}
	}

	query := `
		SELECT id, record_id, owner_id, type, change, payload, meta, version, happened_at
		FROM change_log
		WHERE owner_id = $1 AND id > $2
		ORDER BY id ASC
	`
	args := []interface{}{string(ownerID), startID}
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	changes := make([]domain.RecordChange, 0)
	var lastID int64 = startID
	for rows.Next() {
		var change domain.RecordChange
		var payload, meta []byte
		var version int64
		var recordType string
		var changeType string
		if err := rows.Scan(&lastID, &change.RecordID, &change.OwnerID, &recordType, &changeType, &payload, &meta, &version, &change.HappenedAt); err != nil {
			return nil, "", err
		}
		change.Type = domain.RecordType(recordType)
		change.Change = domain.ChangeType(changeType)
		change.Version = domain.Version(version)
		decodedPayload, err := decodePayload(change.Type, payload)
		if err != nil {
			return nil, "", err
		}
		change.Payload = decodedPayload
		change.Meta, err = decodeMetadata(meta)
		if err != nil {
			return nil, "", err
		}
		changes = append(changes, change)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	nextCursor := domain.SyncCursor(strconv.FormatInt(lastID, 10))
	if len(changes) == 0 {
		nextCursor = domain.SyncCursor(strconv.FormatInt(startID, 10))
	}
	return changes, nextCursor, nil
}
