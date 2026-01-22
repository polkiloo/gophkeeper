package system

import (
	"go.uber.org/fx"

	"gophkeeper/internal/ports/outbound"
)

// Module provides system-level adapters.
var Module = fx.Options(
	fx.Provide(fx.Annotate(
		func() Clock { return Clock{} },
		fx.As(new(outbound.Clock)),
	)),
	fx.Provide(fx.Annotate(
		func() IDGenerator { return IDGenerator{} },
		fx.As(new(outbound.IDGenerator)),
	)),
)
