package httpapi

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"gophkeeper/internal/domain"
	"gophkeeper/internal/logger"
	"gophkeeper/internal/ports/inbound"
)

type stubAuth struct{}

type stubSecrets struct{}

type stubSync struct{}

func (stubAuth) Register(ctx context.Context, input inbound.RegisterInput) (domain.User, error) {
	return domain.User{}, nil
}

func (stubAuth) Login(ctx context.Context, input inbound.LoginInput) (domain.Session, error) {
	return domain.Session{}, nil
}

func (stubAuth) Validate(ctx context.Context, token string) (domain.Session, error) {
	return domain.Session{}, nil
}

func (stubSecrets) Upsert(ctx context.Context, record domain.Record) (domain.Record, error) {
	return domain.Record{}, nil
}

func (stubSecrets) Get(ctx context.Context, id domain.RecordID) (domain.Record, error) {
	return domain.Record{}, nil
}

func (stubSecrets) List(ctx context.Context, filter inbound.RecordFilter) ([]domain.Record, error) {
	return []domain.Record{}, nil
}

func (stubSecrets) Delete(ctx context.Context, id domain.RecordID) error {
	return nil
}

func (stubSync) Pull(ctx context.Context, cursor domain.SyncCursor, limit int) (domain.SyncBatch, error) {
	return domain.SyncBatch{}, nil
}

func (stubSync) Push(ctx context.Context, changes []domain.RecordChange) (domain.SyncResult, error) {
	return domain.SyncResult{}, nil
}

func TestModuleBuildsRouter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	app := fx.New(
		fx.Provide(func() inbound.AuthUseCase { return stubAuth{} }),
		fx.Provide(func() inbound.SecretsUseCase { return stubSecrets{} }),
		fx.Provide(func() inbound.SyncUseCase { return stubSync{} }),
		fx.Provide(func() logger.GinMiddleware { return func(c *gin.Context) {} }),
		Module,
		fx.Populate(new(*gin.Engine)),
	)

	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("failed to start fx app: %v", err)
	}
	if err := app.Stop(ctx); err != nil {
		t.Fatalf("failed to stop fx app: %v", err)
	}
}
