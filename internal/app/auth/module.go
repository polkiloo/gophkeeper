package auth

import (
	"go.uber.org/fx"

	"gophkeeper/internal/config"
	"gophkeeper/internal/ports/inbound"
	"gophkeeper/internal/ports/outbound"
)

// Module provides the auth service.
var Module = fx.Options(
	fx.Provide(fx.Annotate(
		func(users outbound.UserRepository, hasher outbound.PasswordHasher, tokens outbound.TokenIssuer, clock outbound.Clock, idGen outbound.IDGenerator, cfg config.Config) *Service {
			return NewService(users, hasher, tokens, clock, idGen, cfg.TokenTTL)
		},
		fx.As(new(inbound.AuthUseCase)),
	)),
)
