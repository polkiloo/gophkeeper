package httpapi

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"gophkeeper/internal/logger"
	"gophkeeper/internal/transport/secure"
)

type routerParams struct {
	fx.In
	Server     *Server
	Middleware logger.GinMiddleware
	Secure     secure.Middleware `optional:"true"`
}

// NewRouter constructs the Gin router with API routes.
func NewRouter(p routerParams) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery())
	if p.Secure != nil {
		engine.Use(gin.HandlerFunc(p.Secure))
	}
	if p.Middleware != nil {
		engine.Use(p.Middleware)
	}

	api := engine.Group(apiPrefix)
	RegisterHandlers(api, p.Server)

	return engine
}
