package memory

import "go.uber.org/fx"

// Module provides in-memory repositories.
var Module = fx.Options(
	fx.Provide(NewUserRepository),
	fx.Provide(NewRecordRepository),
	fx.Provide(NewChangeLogRepository),
)
