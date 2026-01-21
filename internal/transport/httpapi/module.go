package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"gophkeeper/internal/ports/inbound"
)

type serverParams struct {
	fx.In
	Auth    inbound.AuthUseCase
	Secrets inbound.SecretsUseCase
	Sync    inbound.SyncUseCase
}

// Module provides HTTP transport handlers.
var Module = fx.Options(
	fx.Provide(func(p serverParams) *Server {
		return NewServer(p.Auth, p.Secrets, p.Sync)
	}),
	fx.Provide(NewRouter),
	fx.Provide(func(engine *gin.Engine) http.Handler {
		return engine
	}),
)
