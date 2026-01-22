package memory

import (
	"context"
	"strings"
	"sync"

	"gophkeeper/internal/domain"
	"gophkeeper/internal/ports/outbound"
)

type recordEntry struct {
	record  domain.Record
	deleted bool
}

// RecordRepository stores records in memory.
type RecordRepository struct {
	mu     sync.RWMutex
	byID   map[domain.RecordID]recordEntry
	byUser map[domain.UserID]map[domain.RecordID]struct{}
}

// NewRecordRepository constructs an in-memory record repository.
func NewRecordRepository() *RecordRepository {
	return &RecordRepository{
		byID:   make(map[domain.RecordID]recordEntry),
		byUser: make(map[domain.UserID]map[domain.RecordID]struct{}),
	}
}

// Upsert stores a record.
func (r *RecordRepository) Upsert(_ context.Context, record domain.Record) (domain.Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.byID[record.ID] = recordEntry{record: record, deleted: false}
	set := r.byUser[record.OwnerID]
	if set == nil {
		set = make(map[domain.RecordID]struct{})
		r.byUser[record.OwnerID] = set
	}
	set[record.ID] = struct{}{}
	return record, nil
}

// Get returns a record by id.
func (r *RecordRepository) Get(_ context.Context, id domain.RecordID) (domain.Record, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.byID[id]
	if !ok || entry.deleted {
		return domain.Record{}, domain.ErrNotFound
	}
	return entry.record, nil
}

// List returns records for the given owner.
func (r *RecordRepository) List(_ context.Context, ownerID domain.UserID, filter outbound.RecordFilter) ([]domain.Record, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := r.byUser[ownerID]
	if len(ids) == 0 {
		return []domain.Record{}, nil
	}

	records := make([]domain.Record, 0, len(ids))
	for id := range ids {
		entry := r.byID[id]
		if entry.deleted && !filter.IncludeDeleted {
			continue
		}
		if filter.Type != "" && entry.record.Type != filter.Type {
			continue
		}
		if filter.Tag != "" && !hasTag(entry.record.Meta.Tags, filter.Tag) {
			continue
		}
		if filter.Query != "" && !matchesQuery(entry.record, filter.Query) {
			continue
		}
		records = append(records, entry.record)
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
func (r *RecordRepository) Delete(_ context.Context, id domain.RecordID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.byID[id]
	if !ok || entry.deleted {
		return domain.ErrNotFound
	}
	entry.deleted = true
	r.byID[id] = entry
	return nil
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
