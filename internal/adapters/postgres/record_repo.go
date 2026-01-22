package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"gophkeeper/internal/domain"
	"gophkeeper/internal/ports/outbound"
)

// RecordRepository stores records in PostgreSQL.
type RecordRepository struct {
	db *sql.DB
}

// NewRecordRepository constructs a PostgreSQL-backed record repository.
func NewRecordRepository(db *sql.DB) *RecordRepository {
	return &RecordRepository{db: db}
}

// Upsert stores a record.
func (r *RecordRepository) Upsert(ctx context.Context, record domain.Record) (domain.Record, error) {
	payload, err := encodePayload(record.Payload, record.Type)
	if err != nil {
		return domain.Record{}, err
	}
	meta, err := encodeMetadata(record.Meta)
	if err != nil {
		return domain.Record{}, err
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO records (id, owner_id, type, payload, meta, version, updated_at, deleted)
		VALUES ($1, $2, $3, $4, $5, $6, $7, false)
		ON CONFLICT (id) DO UPDATE
		SET owner_id = EXCLUDED.owner_id,
			type = EXCLUDED.type,
			payload = EXCLUDED.payload,
			meta = EXCLUDED.meta,
			version = EXCLUDED.version,
			updated_at = EXCLUDED.updated_at,
			deleted = false
	`, string(record.ID), string(record.OwnerID), string(record.Type), payload, meta, int64(record.Version), record.UpdatedAt)
	if err != nil {
		return domain.Record{}, err
	}
	return record, nil
}

// Get returns a record by id.
func (r *RecordRepository) Get(ctx context.Context, id domain.RecordID) (domain.Record, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, owner_id, type, payload, meta, version, updated_at, deleted
		FROM records
		WHERE id = $1
	`, string(id))

	var record domain.Record
	var payload, meta []byte
	var deleted bool
	var version int64
	var recordType string
	if err := row.Scan(&record.ID, &record.OwnerID, &recordType, &payload, &meta, &version, &record.UpdatedAt, &deleted); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Record{}, domain.ErrNotFound
		}
		return domain.Record{}, err
	}
	if deleted {
		return domain.Record{}, domain.ErrNotFound
	}

	record.Type = domain.RecordType(recordType)
	record.Version = domain.Version(version)

	decodedPayload, err := decodePayload(record.Type, payload)
	if err != nil {
		return domain.Record{}, err
	}
	record.Payload = decodedPayload

	record.Meta, err = decodeMetadata(meta)
	if err != nil {
		return domain.Record{}, err
	}

	return record, nil
}

// List returns records for the given owner.
func (r *RecordRepository) List(ctx context.Context, ownerID domain.UserID, filter outbound.RecordFilter) ([]domain.Record, error) {
	query := `
		SELECT id, owner_id, type, payload, meta, version, updated_at, deleted
		FROM records
		WHERE owner_id = $1
	`
	args := []interface{}{string(ownerID)}
	argPos := 2
	if !filter.IncludeDeleted {
		query += " AND deleted = false"
	}
	if filter.Type != "" {
		query += " AND type = $" + itoa(argPos)
		args = append(args, string(filter.Type))
		argPos++
	}
	query += " ORDER BY updated_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]domain.Record, 0)
	for rows.Next() {
		var record domain.Record
		var payload, meta []byte
		var deleted bool
		var version int64
		var recordType string
		if err := rows.Scan(&record.ID, &record.OwnerID, &recordType, &payload, &meta, &version, &record.UpdatedAt, &deleted); err != nil {
			return nil, err
		}
		if deleted && !filter.IncludeDeleted {
			continue
		}
		record.Type = domain.RecordType(recordType)
		record.Version = domain.Version(version)
		decodedPayload, err := decodePayload(record.Type, payload)
		if err != nil {
			return nil, err
		}
		record.Payload = decodedPayload
		record.Meta, err = decodeMetadata(meta)
		if err != nil {
			return nil, err
		}
		if filter.Tag != "" && !hasTag(record.Meta.Tags, filter.Tag) {
			continue
		}
		if filter.Query != "" && !matchesQuery(record, filter.Query) {
			continue
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	start := filter.Offset
	if start < 0 {
		start = 0
	}
	if start >= len(records) {
		return []domain.Record{}, nil
	}

	end := len(records)
	if filter.Limit > 0 && start+filter.Limit < end {
		end = start + filter.Limit
	}

	return records[start:end], nil
}

// Delete removes a record.
func (r *RecordRepository) Delete(ctx context.Context, id domain.RecordID) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE records SET deleted = true WHERE id = $1 AND deleted = false
	`, string(id))
	if err != nil {
		return err
	}
	updated, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if updated == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func itoa(value int) string {
	return fmt.Sprintf("%d", value)
}

func hasTag(tags []string, tag string) bool {
	for _, value := range tags {
		if value == tag {
			return true
		}
	}
	return false
}

func matchesQuery(record domain.Record, query string) bool {
	query = strings.ToLower(query)
	if strings.Contains(strings.ToLower(record.Meta.Title), query) {
		return true
	}
	if strings.Contains(strings.ToLower(record.Meta.Description), query) {
		return true
	}
	for _, tag := range record.Meta.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	for key, value := range record.Meta.Attributes {
		if strings.Contains(strings.ToLower(key), query) || strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}
