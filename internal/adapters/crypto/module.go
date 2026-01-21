package crypto

import (
	"go.uber.org/fx"

	"gophkeeper/internal/config"
	"gophkeeper/internal/ports/outbound"
)

// Module provides crypto-related adapters.
var Module = fx.Options(
	fx.Provide(func() PasswordHasher { return PasswordHasher{} }),
	fx.Provide(func(cfg config.Config, clock outbound.Clock) outbound.TokenIssuer {
		return NewTokenIssuer(cfg.TokenSecret, clock.Now)
	}),
)
