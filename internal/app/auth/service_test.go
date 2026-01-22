package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"gophkeeper/internal/domain"
	"gophkeeper/internal/ports/inbound"
	"gophkeeper/internal/ports/outbound"
)

type mockUserRepo struct{ mock.Mock }

type mockHasher struct{ mock.Mock }

type mockTokenIssuer struct{ mock.Mock }

type mockClock struct{ mock.Mock }

type mockIDGen struct{ mock.Mock }

func (m *mockUserRepo) Create(ctx context.Context, user domain.User, passwordHash string) (domain.User, error) {
	args := m.Called(ctx, user, passwordHash)
	return args.Get(0).(domain.User), args.Error(1)
}

func (m *mockUserRepo) FindByLogin(ctx context.Context, login string) (domain.User, string, error) {
	args := m.Called(ctx, login)
	return args.Get(0).(domain.User), args.String(1), args.Error(2)
}

func (m *mockUserRepo) FindByID(ctx context.Context, id domain.UserID) (domain.User, string, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.User), args.String(1), args.Error(2)
}

func (m *mockUserRepo) UpdatePasswordHash(ctx context.Context, id domain.UserID, passwordHash string) error {
	args := m.Called(ctx, id, passwordHash)
	return args.Error(0)
}

func (m *mockHasher) Hash(ctx context.Context, password string) (string, error) {
	args := m.Called(ctx, password)
	return args.String(0), args.Error(1)
}

func (m *mockHasher) Compare(ctx context.Context, password, hash string) error {
	args := m.Called(ctx, password, hash)
	return args.Error(0)
}

func (m *mockTokenIssuer) Issue(ctx context.Context, subject string, ttl time.Duration) (string, time.Time, error) {
	args := m.Called(ctx, subject, ttl)
	return args.String(0), args.Get(1).(time.Time), args.Error(2)
}

func (m *mockTokenIssuer) Validate(ctx context.Context, token string) (outbound.TokenClaims, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(outbound.TokenClaims), args.Error(1)
}

func (m *mockClock) Now() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

func (m *mockIDGen) NewID() string {
	args := m.Called()
	return args.String(0)
}

func TestServiceRegisterSuccess(t *testing.T) {
	users := new(mockUserRepo)
	hasher := new(mockHasher)
	tokens := new(mockTokenIssuer)
	clock := new(mockClock)
	idGen := new(mockIDGen)

	now := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	input := inbound.RegisterInput{Login: "user", Password: "secret"}

	users.On("FindByLogin", mock.Anything, "user").Return(domain.User{}, "", domain.ErrNotFound)
	hasher.On("Hash", mock.Anything, "secret").Return("hash", nil)
	idGen.On("NewID").Return("id-1")
	clock.On("Now").Return(now)
	users.On("Create", mock.Anything, domain.User{ID: "id-1", Login: "user", CreatedAt: now}, "hash").Return(domain.User{ID: "id-1", Login: "user", CreatedAt: now}, nil)

	svc := NewService(users, hasher, tokens, clock, idGen, time.Hour)
	user, err := svc.Register(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != "id-1" {
		t.Fatalf("unexpected user id: %s", user.ID)
	}

	users.AssertExpectations(t)
	hasher.AssertExpectations(t)
	clock.AssertExpectations(t)
	idGen.AssertExpectations(t)
}

func TestServiceRegisterConflict(t *testing.T) {
	users := new(mockUserRepo)
	svc := NewService(users, new(mockHasher), new(mockTokenIssuer), new(mockClock), new(mockIDGen), time.Hour)

	users.On("FindByLogin", mock.Anything, "user").Return(domain.User{ID: "exists"}, "", nil)

	_, err := svc.Register(context.Background(), inbound.RegisterInput{Login: "user", Password: "secret"})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

func TestServiceRegisterValidation(t *testing.T) {
	svc := NewService(new(mockUserRepo), new(mockHasher), new(mockTokenIssuer), new(mockClock), new(mockIDGen), time.Hour)
	_, err := svc.Register(context.Background(), inbound.RegisterInput{Login: "", Password: ""})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestServiceRegisterLookupError(t *testing.T) {
	users := new(mockUserRepo)
	svc := NewService(users, new(mockHasher), new(mockTokenIssuer), new(mockClock), new(mockIDGen), time.Hour)
	users.On("FindByLogin", mock.Anything, "user").Return(domain.User{}, "", errors.New("boom"))
	_, err := svc.Register(context.Background(), inbound.RegisterInput{Login: "user", Password: "secret"})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected lookup error")
	}
}

func TestServiceLoginSuccess(t *testing.T) {
	users := new(mockUserRepo)
	hasher := new(mockHasher)
	tokens := new(mockTokenIssuer)
	clock := new(mockClock)
	idGen := new(mockIDGen)
	expires := time.Date(2024, 2, 2, 12, 0, 0, 0, time.UTC)

	users.On("FindByLogin", mock.Anything, "user").Return(domain.User{ID: "id-1"}, "hash", nil)
	hasher.On("Compare", mock.Anything, "secret", "hash").Return(nil)
	tokens.On("Issue", mock.Anything, "id-1", time.Hour).Return("token", expires, nil)

	svc := NewService(users, hasher, tokens, clock, idGen, time.Hour)
	session, err := svc.Login(context.Background(), inbound.LoginInput{Login: "user", Password: "secret"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.Token != "token" || session.UserID != "id-1" {
		t.Fatalf("unexpected session")
	}
}

func TestServiceLoginUnauthorized(t *testing.T) {
	users := new(mockUserRepo)
	hasher := new(mockHasher)
	tokens := new(mockTokenIssuer)

	users.On("FindByLogin", mock.Anything, "user").Return(domain.User{}, "", domain.ErrNotFound)

	svc := NewService(users, hasher, tokens, new(mockClock), new(mockIDGen), time.Hour)
	_, err := svc.Login(context.Background(), inbound.LoginInput{Login: "user", Password: "secret"})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestServiceLoginInvalidPassword(t *testing.T) {
	users := new(mockUserRepo)
	hasher := new(mockHasher)
	tokens := new(mockTokenIssuer)

	users.On("FindByLogin", mock.Anything, "user").Return(domain.User{ID: "id-1"}, "hash", nil)
	hasher.On("Compare", mock.Anything, "secret", "hash").Return(errors.New("bad"))

	svc := NewService(users, hasher, tokens, new(mockClock), new(mockIDGen), time.Hour)
	_, err := svc.Login(context.Background(), inbound.LoginInput{Login: "user", Password: "secret"})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestServiceLoginValidation(t *testing.T) {
	svc := NewService(new(mockUserRepo), new(mockHasher), new(mockTokenIssuer), new(mockClock), new(mockIDGen), time.Hour)
	_, err := svc.Login(context.Background(), inbound.LoginInput{Login: "", Password: ""})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestServiceValidate(t *testing.T) {
	tokens := new(mockTokenIssuer)
	claims := outbound.TokenClaims{Subject: "id-1", ExpiresAt: time.Date(2024, 3, 3, 9, 0, 0, 0, time.UTC)}
	tokens.On("Validate", mock.Anything, "token").Return(claims, nil)

	svc := NewService(new(mockUserRepo), new(mockHasher), tokens, new(mockClock), new(mockIDGen), time.Hour)
	session, err := svc.Validate(context.Background(), "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.UserID != "id-1" {
		t.Fatalf("unexpected session user")
	}
}

func TestServiceValidateUnauthorized(t *testing.T) {
	tokens := new(mockTokenIssuer)
	tokens.On("Validate", mock.Anything, "token").Return(outbound.TokenClaims{}, errors.New("bad"))

	svc := NewService(new(mockUserRepo), new(mockHasher), tokens, new(mockClock), new(mockIDGen), time.Hour)
	_, err := svc.Validate(context.Background(), "token")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestServiceRegisterHashError(t *testing.T) {
	users := new(mockUserRepo)
	hasher := new(mockHasher)

	users.On("FindByLogin", mock.Anything, "user").Return(domain.User{}, "", domain.ErrNotFound)
	hasher.On("Hash", mock.Anything, "secret").Return("", errors.New("hash failed"))

	svc := NewService(users, hasher, new(mockTokenIssuer), new(mockClock), new(mockIDGen), time.Hour)
	_, err := svc.Register(context.Background(), inbound.RegisterInput{Login: "user", Password: "secret"})
	if err == nil || err.Error() != "hash failed" {
		t.Fatalf("expected hash error, got %v", err)
	}
}

func TestServiceRegisterCreateError(t *testing.T) {
	users := new(mockUserRepo)
	hasher := new(mockHasher)
	clock := new(mockClock)
	idGen := new(mockIDGen)

	now := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	users.On("FindByLogin", mock.Anything, "user").Return(domain.User{}, "", domain.ErrNotFound)
	hasher.On("Hash", mock.Anything, "secret").Return("hash", nil)
	idGen.On("NewID").Return("id-1")
	clock.On("Now").Return(now)
	users.On("Create", mock.Anything, domain.User{ID: "id-1", Login: "user", CreatedAt: now}, "hash").Return(domain.User{}, errors.New("create failed"))

	svc := NewService(users, hasher, new(mockTokenIssuer), clock, idGen, time.Hour)
	_, err := svc.Register(context.Background(), inbound.RegisterInput{Login: "user", Password: "secret"})
	if err == nil || err.Error() != "create failed" {
		t.Fatalf("expected create error, got %v", err)
	}
}

func TestServiceRegisterExistingEmptyID(t *testing.T) {
	users := new(mockUserRepo)
	hasher := new(mockHasher)
	clock := new(mockClock)
	idGen := new(mockIDGen)

	now := time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)
	users.On("FindByLogin", mock.Anything, "user").Return(domain.User{}, "", nil)
	hasher.On("Hash", mock.Anything, "secret").Return("hash", nil)
	idGen.On("NewID").Return("id-2")
	clock.On("Now").Return(now)
	users.On("Create", mock.Anything, domain.User{ID: "id-2", Login: "user", CreatedAt: now}, "hash").Return(domain.User{ID: "id-2", Login: "user", CreatedAt: now}, nil)

	svc := NewService(users, hasher, new(mockTokenIssuer), clock, idGen, time.Hour)
	if _, err := svc.Register(context.Background(), inbound.RegisterInput{Login: "user", Password: "secret"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceLoginLookupError(t *testing.T) {
	users := new(mockUserRepo)
	users.On("FindByLogin", mock.Anything, "user").Return(domain.User{}, "", errors.New("db down"))

	svc := NewService(users, new(mockHasher), new(mockTokenIssuer), new(mockClock), new(mockIDGen), time.Hour)
	_, err := svc.Login(context.Background(), inbound.LoginInput{Login: "user", Password: "secret"})
	if err == nil || err.Error() != "db down" {
		t.Fatalf("expected lookup error, got %v", err)
	}
}

func TestServiceLoginTokenIssueError(t *testing.T) {
	users := new(mockUserRepo)
	hasher := new(mockHasher)
	tokens := new(mockTokenIssuer)

	users.On("FindByLogin", mock.Anything, "user").Return(domain.User{ID: "id-1"}, "hash", nil)
	hasher.On("Compare", mock.Anything, "secret", "hash").Return(nil)
	tokens.On("Issue", mock.Anything, "id-1", time.Hour).Return("", time.Time{}, errors.New("issue failed"))

	svc := NewService(users, hasher, tokens, new(mockClock), new(mockIDGen), time.Hour)
	_, err := svc.Login(context.Background(), inbound.LoginInput{Login: "user", Password: "secret"})
	if err == nil || err.Error() != "issue failed" {
		t.Fatalf("expected issue error, got %v", err)
	}
}
