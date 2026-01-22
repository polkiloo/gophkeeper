package sync

import (
	"go.uber.org/fx"

	"gophkeeper/internal/ports/inbound"
	"gophkeeper/internal/ports/outbound"
)

// Module provides the sync service.
var Module = fx.Options(
	fx.Provide(fx.Annotate(
		func(records outbound.RecordRepository, changes outbound.ChangeLogRepository, clock outbound.Clock) *Service {
			return NewService(records, changes, clock)
		},
		fx.As(new(inbound.SyncUseCase)),
	)),
)
