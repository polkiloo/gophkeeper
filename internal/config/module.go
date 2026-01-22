package config

import "go.uber.org/fx"

// Module provides configuration loading.
var Module = fx.Options(
	fx.Provide(Load),
)
