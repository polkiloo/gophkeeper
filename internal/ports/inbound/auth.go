package inbound

import (
	"context"

	"gophkeeper/internal/domain"
)

// RegisterInput captures registration data.
type RegisterInput struct {
	Login    string
	Password string
}

// LoginInput captures authentication data.
type LoginInput struct {
	Login    string
	Password string
}

// AuthUseCase defines authentication flows.
type AuthUseCase interface {
	Register(ctx context.Context, input RegisterInput) (domain.User, error)
	Login(ctx context.Context, input LoginInput) (domain.Session, error)
	Validate(ctx context.Context, token string) (domain.Session, error)
}
