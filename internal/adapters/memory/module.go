package memory

import (
	"go.uber.org/fx"

	"gophkeeper/internal/ports/outbound"
)

// Module provides in-memory repositories.
var Module = fx.Options(
	fx.Provide(fx.Annotate(
		NewUserRepository,
		fx.As(new(outbound.UserRepository)),
	)),
	fx.Provide(fx.Annotate(
		NewRecordRepository,
		fx.As(new(outbound.RecordRepository)),
	)),
	fx.Provide(fx.Annotate(
		NewChangeLogRepository,
		fx.As(new(outbound.ChangeLogRepository)),
	)),
)
