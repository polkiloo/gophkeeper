package inbound

import (
	"context"

	"gophkeeper/internal/domain"
)

// RecordFilter narrows record queries.
type RecordFilter struct {
	Type    domain.RecordType
	Tag     string
	Query   string
	Limit   int
	Offset  int
	IncludeDeleted bool
}

// SecretsUseCase defines operations over private records.
type SecretsUseCase interface {
	Upsert(ctx context.Context, record domain.Record) (domain.Record, error)
	Create(ctx context.Context, record domain.Record) (domain.Record, error)
	Update(ctx context.Context, record domain.Record) (domain.Record, error)
	Get(ctx context.Context, id domain.RecordID) (domain.Record, error)
	List(ctx context.Context, filter RecordFilter) ([]domain.Record, error)
	Delete(ctx context.Context, id domain.RecordID) error
}
