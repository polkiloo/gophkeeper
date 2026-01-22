package keycloak

import (
	"net/http"
	"time"

	"go.uber.org/fx"

	"gophkeeper/internal/config"
	"gophkeeper/internal/ports/outbound"
)

// Module provides Keycloak authentication services.
var Module = fx.Options(
	fx.Provide(func() *http.Client {
		return &http.Client{Timeout: 10 * time.Second}
	}),
	fx.Provide(func(client *http.Client, clock outbound.Clock, cfg config.Config) *Service {
		return NewService(client, clock, cfg.Keycloak)
	}),
)
