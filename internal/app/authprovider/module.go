package authprovider

import (
	"strings"

	"go.uber.org/fx"

	localauth "gophkeeper/internal/app/auth"
	"gophkeeper/internal/app/auth/keycloak"
	"gophkeeper/internal/config"
	"gophkeeper/internal/ports/inbound"
)

// Module selects the auth use case based on configuration.
var Module = fx.Options(
	fx.Provide(selectAuthUseCase),
)

func selectAuthUseCase(cfg config.Config, local *localauth.Service, kc *keycloak.Service) inbound.AuthUseCase {
	if strings.EqualFold(cfg.AuthProvider, "keycloak") {
		return kc
	}
	return local
}
