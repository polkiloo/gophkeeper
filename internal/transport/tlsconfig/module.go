package tlsconfig

import "go.uber.org/fx"

// ServerModule provides TLS configuration for the server.
var ServerModule = fx.Options(
	fx.Provide(NewServerTLSConfig),
)

// ClientModule provides TLS configuration for the client.
var ClientModule = fx.Options(
	fx.Provide(NewClientTLSConfig),
)
