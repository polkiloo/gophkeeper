package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"gophkeeper/internal/domain"
	"gophkeeper/internal/ports/inbound"
)

type authStub struct {
	register func(inbound.RegisterInput) (domain.User, error)
	login    func(inbound.LoginInput) (domain.Session, error)
	validate func(string) (domain.Session, error)
}

func (a authStub) Register(ctx context.Context, input inbound.RegisterInput) (domain.User, error) {
	return a.register(input)
}

func (a authStub) Login(ctx context.Context, input inbound.LoginInput) (domain.Session, error) {
	return a.login(input)
}

func (a authStub) Validate(ctx context.Context, token string) (domain.Session, error) {
	return a.validate(token)
}

type secretsStub struct {
	upsert func(domain.Record) (domain.Record, error)
	get    func(domain.RecordID) (domain.Record, error)
	list   func(inbound.RecordFilter) ([]domain.Record, error)
	delete func(domain.RecordID) error
}

func (s secretsStub) Upsert(ctx context.Context, record domain.Record) (domain.Record, error) {
	return s.upsert(record)
}

func (s secretsStub) Get(ctx context.Context, id domain.RecordID) (domain.Record, error) {
	return s.get(id)
}

func (s secretsStub) List(ctx context.Context, filter inbound.RecordFilter) ([]domain.Record, error) {
	return s.list(filter)
}

func (s secretsStub) Delete(ctx context.Context, id domain.RecordID) error {
	return s.delete(id)
}

type syncStub struct {
	pull func(domain.SyncCursor, int) (domain.SyncBatch, error)
	push func([]domain.RecordChange) (domain.SyncResult, error)
}

func (s syncStub) Pull(ctx context.Context, cursor domain.SyncCursor, limit int) (domain.SyncBatch, error) {
	return s.pull(cursor, limit)
}

func (s syncStub) Push(ctx context.Context, changes []domain.RecordChange) (domain.SyncResult, error) {
	return s.push(changes)
}

func TestAuthHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := NewServer(
		authStub{
			register: func(input inbound.RegisterInput) (domain.User, error) {
				return domain.User{ID: "u1", Login: input.Login, CreatedAt: time.Now()}, nil
			},
			login: func(input inbound.LoginInput) (domain.Session, error) {
				return domain.Session{UserID: "u1", Token: "t", ExpiresAt: time.Now().Add(time.Hour)}, nil
			},
			validate: func(token string) (domain.Session, error) {
				if token == "" {
					return domain.Session{}, domain.ErrUnauthorized
				}
				return domain.Session{UserID: "u1", Token: token, ExpiresAt: time.Now().Add(time.Hour)}, nil
			},
		},
		secretsStub{},
		syncStub{},
	)

	rec := performJSON(server.Register, http.MethodPost, "/auth/register", map[string]any{
		"login":    "demo",
		"password": "pass",
	}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("register status: %d", rec.Code)
	}

	rec = performJSON(server.Login, http.MethodPost, "/auth/login", map[string]any{
		"login":    "demo",
		"password": "pass",
	}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status: %d", rec.Code)
	}

	rec = performRequest(server.Validate, http.MethodGet, "/auth/validate", nil, map[string]string{
		"Authorization": "Bearer token",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("validate status: %d", rec.Code)
	}

	rec = performRequest(server.Validate, http.MethodGet, "/auth/validate", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized")
	}
}

func TestRecordHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	record := domain.Record{
		ID:        "r1",
		OwnerID:   "u1",
		Type:      domain.RecordTypeText,
		Payload:   domain.TextPayload{Text: "hello"},
		Meta:      domain.Metadata{Title: "title"},
		Version:   1,
		UpdatedAt: time.Now(),
	}

	server := NewServer(
		authStub{
			validate: func(token string) (domain.Session, error) {
				return domain.Session{UserID: "u1", Token: token, ExpiresAt: time.Now().Add(time.Hour)}, nil
			},
		},
		secretsStub{
			upsert: func(r domain.Record) (domain.Record, error) { return record, nil },
			get:    func(id domain.RecordID) (domain.Record, error) { return record, nil },
			list:   func(filter inbound.RecordFilter) ([]domain.Record, error) { return []domain.Record{record}, nil },
			delete: func(id domain.RecordID) error { return nil },
		},
		syncStub{},
	)

	rec := performJSON(func(c *gin.Context) { server.UpsertRecord(c) }, http.MethodPost, "/records", map[string]any{
		"type":    "text",
		"payload": map[string]any{"kind": "text", "text": "hello"},
		"meta":    map[string]any{"title": "title"},
	}, map[string]string{"Authorization": "Bearer token"})
	if rec.Code != http.StatusOK {
		t.Fatalf("upsert status: %d", rec.Code)
	}

	rec = performRequest(func(c *gin.Context) { server.GetRecord(c, "r1") }, http.MethodGet, "/records/r1", nil, map[string]string{"Authorization": "Bearer token"})
	if rec.Code != http.StatusOK {
		t.Fatalf("get status: %d", rec.Code)
	}

	rec = performRequest(func(c *gin.Context) { server.DeleteRecord(c, "r1") }, http.MethodDelete, "/records/r1", nil, map[string]string{"Authorization": "Bearer token"})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status: %d", rec.Code)
	}

	rec = performRequest(func(c *gin.Context) { server.ListRecords(c, ListRecordsParams{}) }, http.MethodGet, "/records", nil, map[string]string{"Authorization": "Bearer token"})
	if rec.Code != http.StatusOK {
		t.Fatalf("list status: %d", rec.Code)
	}
}

func TestSyncHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := NewServer(
		authStub{
			validate: func(token string) (domain.Session, error) {
				return domain.Session{UserID: "u1", Token: token, ExpiresAt: time.Now().Add(time.Hour)}, nil
			},
		},
		secretsStub{},
		syncStub{
			pull: func(cursor domain.SyncCursor, limit int) (domain.SyncBatch, error) {
				return domain.SyncBatch{Changes: []domain.RecordChange{}, Cursor: "1"}, nil
			},
			push: func(changes []domain.RecordChange) (domain.SyncResult, error) {
				return domain.SyncResult{Applied: len(changes)}, nil
			},
		},
	)

	rec := performRequest(func(c *gin.Context) { server.PullSync(c, PullSyncParams{}) }, http.MethodGet, "/sync", nil, map[string]string{"Authorization": "Bearer token"})
	if rec.Code != http.StatusOK {
		t.Fatalf("pull status: %d", rec.Code)
	}

	rec = performJSON(func(c *gin.Context) { server.PushSync(c) }, http.MethodPost, "/sync", map[string]any{
		"changes": []any{},
	}, map[string]string{"Authorization": "Bearer token"})
	if rec.Code != http.StatusOK {
		t.Fatalf("push status: %d", rec.Code)
	}
}

func performJSON(handler func(*gin.Context), method, path string, payload any, headers map[string]string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(payload)
	return performRequest(handler, method, path, body, headers)
}

func performRequest(handler func(*gin.Context), method, path string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	c.Request = req
	handler(c)
	c.Writer.WriteHeaderNow()
	return w
}
