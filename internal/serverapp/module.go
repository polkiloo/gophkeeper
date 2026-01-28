package serverapp

import (
	"go.uber.org/fx"

	"gophkeeper/internal/adapters/crypto"
	"gophkeeper/internal/adapters/persistence"
	"gophkeeper/internal/adapters/system"
	"gophkeeper/internal/app/auth"
	"gophkeeper/internal/app/auth/keycloak"
	"gophkeeper/internal/app/authprovider"
	"gophkeeper/internal/app/secrets"
	"gophkeeper/internal/app/sync"
	"gophkeeper/internal/config"
	"gophkeeper/internal/logger"
	"gophkeeper/internal/migrate"
	"gophkeeper/internal/server"
	"gophkeeper/internal/transport/httpapi"
	"gophkeeper/internal/transport/secure"
)

// DataModule wires configuration, logging, and storage dependencies.
var DataModule = fx.Options(
	logger.Module,
	config.Module,
	system.Module,
	persistence.Module,
	crypto.Module,
)

// ServiceModule wires application services and use cases.
var ServiceModule = fx.Options(
	auth.Module,
	keycloak.Module,
	authprovider.Module,
	secrets.Module,
	sync.Module,
)

// TransportModule wires HTTP transport and server lifecycle.
var TransportModule = fx.Options(
	httpapi.Module,
	secure.ServerModule,
	migrate.Module,
	server.Module,
)

// AppModule bundles the full server dependency graph.
var AppModule = fx.Options(
	DataModule,
	ServiceModule,
	TransportModule,
)
