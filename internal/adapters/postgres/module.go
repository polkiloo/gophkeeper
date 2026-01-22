package postgres

import "go.uber.org/fx"

// Module provides PostgreSQL database connections.
var Module = fx.Options(
	fx.Provide(NewDB),
)
