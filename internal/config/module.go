package config

import (
	"go.uber.org/fx"

	"gophkeeper/internal/transport/secure"
	"gophkeeper/internal/transport/tlsconfig"
)

// Module provides configuration loading.
var Module = fx.Options(
	fx.Provide(ConfigPathFromEnv),
	fx.Provide(Load),
	fx.Provide(func(cfg Config) secure.TransportKey { return secure.TransportKey(cfg.TransportKey) }),
	fx.Provide(func(cfg Config) tlsconfig.ServerSettings {
		return tlsconfig.ServerSettings{
			Enabled:  cfg.TLSEnabled,
			CertFile: cfg.TLSCertFile,
			KeyFile:  cfg.TLSKeyFile,
			ClientCA: cfg.TLSCAFile,
		}
	}),
	tlsconfig.ServerModule,
)
