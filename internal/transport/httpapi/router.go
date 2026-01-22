package httpapi

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"gophkeeper/internal/logger"
)

type routerParams struct {
	fx.In
	Server     *Server
	Middleware logger.GinMiddleware
}

// NewRouter constructs the Gin router with API routes.
func NewRouter(p routerParams) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery())
	if p.Middleware != nil {
		engine.Use(p.Middleware)
	}

	api := engine.Group(apiPrefix)
	RegisterHandlers(api, p.Server)

	return engine
}
