package client

import (
	"context"
	"io"
	"os"

	"go.uber.org/fx"

	"gophkeeper/internal/transport/secure"
	"gophkeeper/internal/transport/tlsconfig"
)

// Stdout wraps standard output for DI.
type Stdout io.Writer

// Stderr wraps standard error for DI.
type Stderr io.Writer

// RunResult carries the CLI execution result.
type RunResult chan error

// Module wires the CLI runtime.
var Module = fx.Options(
	fx.Provide(provideStdout),
	fx.Provide(provideStderr),
	fx.Provide(provideArgs),
	fx.Provide(ConfigPathFromEnv),
	fx.Provide(LoadConfig),
	fx.Provide(provideRunResult),
	fx.Provide(provideTransportKey),
	fx.Provide(provideTLSSettings),
	fx.Provide(tlsconfig.NewClientTLSConfig),
	fx.Provide(secure.NewHTTPClient),
	fx.Provide(NewClientFactory),
	fx.Provide(NewRunner),
	fx.Invoke(runCLI),
)

func provideStdout() Stdout {
	return Stdout(os.Stdout)
}

func provideStderr() Stderr {
	return Stderr(os.Stderr)
}

func provideArgs() []string {
	return os.Args
}

func provideRunResult() RunResult {
	return make(RunResult, 1)
}

func provideTransportKey(cfg Config) secure.TransportKey {
	return secure.TransportKey(cfg.TransportKey)
}

func provideTLSSettings(cfg Config) tlsconfig.ClientSettings {
	return tlsconfig.ClientSettings{
		CAFile:             cfg.TLSCAFile,
		InsecureSkipVerify: cfg.TLSInsecureSkipVerify,
	}
}

func runCLI(ctx context.Context, runner *Runner, result RunResult) {
	go func() {
		result <- runner.Run(ctx)
		close(result)
	}()
}
