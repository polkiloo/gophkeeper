package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"gophkeeper/internal/domain"
	"gophkeeper/internal/ports/inbound"
	"gophkeeper/internal/ports/outbound"
)

// Service implements authentication use cases.
type Service struct {
	users    outbound.UserRepository
	hasher   outbound.PasswordHasher
	tokens   outbound.TokenIssuer
	clock    outbound.Clock
	idGen    outbound.IDGenerator
	tokenTTL time.Duration
}

// NewService constructs an authentication service.
func NewService(users outbound.UserRepository, hasher outbound.PasswordHasher, tokens outbound.TokenIssuer, clock outbound.Clock, idGen outbound.IDGenerator, tokenTTL time.Duration) *Service {
	return &Service{
		users:    users,
		hasher:   hasher,
		tokens:   tokens,
		clock:    clock,
		idGen:    idGen,
		tokenTTL: tokenTTL,
	}
}

// Register creates a new user account.
func (s *Service) Register(ctx context.Context, input inbound.RegisterInput) (domain.User, error) {
	login := strings.TrimSpace(input.Login)
	password := strings.TrimSpace(input.Password)
	if login == "" || password == "" {
		return domain.User{}, errors.New("login and password are required")
	}

	if existing, _, err := s.users.FindByLogin(ctx, login); err == nil {
		if existing.ID != "" {
			return domain.User{}, domain.ErrConflict
		}
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, err
	}

	hash, err := s.hasher.Hash(ctx, password)
	if err != nil {
		return domain.User{}, err
	}

	user := domain.User{
		ID:        domain.UserID(s.idGen.NewID()),
		Login:     login,
		CreatedAt: s.clock.Now(),
	}

	created, err := s.users.Create(ctx, user, hash)
	if err != nil {
		return domain.User{}, err
	}

	return created, nil
}

// Login authenticates a user and issues a session.
func (s *Service) Login(ctx context.Context, input inbound.LoginInput) (domain.Session, error) {
	login := strings.TrimSpace(input.Login)
	password := strings.TrimSpace(input.Password)
	if login == "" || password == "" {
		return domain.Session{}, errors.New("login and password are required")
	}

	user, hash, err := s.users.FindByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.Session{}, domain.ErrUnauthorized
		}
		return domain.Session{}, err
	}

	if err := s.hasher.Compare(ctx, password, hash); err != nil {
		return domain.Session{}, domain.ErrUnauthorized
	}

	token, expiresAt, err := s.tokens.Issue(ctx, string(user.ID), s.tokenTTL)
	if err != nil {
		return domain.Session{}, err
	}

	return domain.Session{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

// Validate verifies a token and returns its session.
func (s *Service) Validate(ctx context.Context, token string) (domain.Session, error) {
	claims, err := s.tokens.Validate(ctx, token)
	if err != nil {
		return domain.Session{}, domain.ErrUnauthorized
	}

	return domain.Session{
		UserID:    domain.UserID(claims.Subject),
		Token:     token,
		ExpiresAt: claims.ExpiresAt,
	}, nil
}
