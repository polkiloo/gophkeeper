package system

import "go.uber.org/fx"

// Module provides system-level adapters.
var Module = fx.Options(
	fx.Provide(func() Clock { return Clock{} }),
	fx.Provide(func() IDGenerator { return IDGenerator{} }),
)
