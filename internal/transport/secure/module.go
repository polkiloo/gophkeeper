package secure

import "go.uber.org/fx"

// ServerModule provides server-side encryption middleware.
var ServerModule = fx.Options(
	fx.Provide(NewServerMiddleware),
)

// ClientModule provides HTTP clients with encrypted transport.
var ClientModule = fx.Options(
	fx.Provide(NewHTTPClient),
)
