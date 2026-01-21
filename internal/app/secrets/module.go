package secrets

import (
	"go.uber.org/fx"

	"gophkeeper/internal/ports/outbound"
)

// Module provides the secrets service.
var Module = fx.Options(
	fx.Provide(func(records outbound.RecordRepository, changes outbound.ChangeLogRepository, clock outbound.Clock, idGen outbound.IDGenerator) *Service {
		return NewService(records, changes, clock, idGen)
	}),
)
