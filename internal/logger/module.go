package logger

import "go.uber.org/fx"

// Module provides logging dependencies.
var Module = fx.Options(
	fx.Provide(New),
	fx.Provide(NewMiddleware),
	fx.Provide(NewGinMiddleware),
)
