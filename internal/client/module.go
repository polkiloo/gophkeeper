package client

import (
	"context"
	"io"
	"os"

	"go.uber.org/fx"
)

// Stdout wraps standard output for DI.
type Stdout io.Writer

// Stderr wraps standard error for DI.
type Stderr io.Writer

// RunResult carries the CLI execution result.
type RunResult chan error

// Module wires the CLI runtime.
var Module = fx.Options(
	fx.Provide(func() Stdout { return Stdout(os.Stdout) }),
	fx.Provide(func() Stderr { return Stderr(os.Stderr) }),
	fx.Provide(func() []string { return os.Args }),
	fx.Provide(ConfigPathFromEnv),
	fx.Provide(LoadConfig),
	fx.Provide(func() RunResult { return make(RunResult, 1) }),
	fx.Provide(NewClientFactory),
	fx.Provide(NewRunner),
	fx.Invoke(runCLI),
)

func runCLI(ctx context.Context, runner *Runner, result RunResult) {
	go func() {
		result <- runner.Run(ctx)
		close(result)
	}()
}
