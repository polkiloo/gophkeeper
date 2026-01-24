package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"gophkeeper/internal/domain"
	"gophkeeper/internal/ports/inbound"
)

type badPayload struct{}

func (badPayload) RecordType() domain.RecordType { return "bad" }

func TestRegisterErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := NewServer(
		authStub{
			register: func(input inbound.RegisterInput) (domain.User, error) {
				return domain.User{}, domain.ErrConflict
			},
		},
		secretsStub{},
		syncStub{},
	)

	rec := performRequest(server.Register, http.MethodPost, "/auth/register", []byte("{"), nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request")
	}

	rec = performJSON(server.Register, http.MethodPost, "/auth/register", map[string]any{
		"login":    "demo",
		"password": "pass",
	}, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected conflict")
	}
}

func TestLoginUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := NewServer(
		authStub{
			login: func(input inbound.LoginInput) (domain.Session, error) {
				return domain.Session{}, domain.ErrUnauthorized
			},
		},
		secretsStub{},
		syncStub{},
	)

	rec := performJSON(server.Login, http.MethodPost, "/auth/login", map[string]any{
		"login":    "demo",
		"password": "pass",
	}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized")
	}
}

func TestListRecordsUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := NewServer(
		authStub{
			validate: func(token string) (domain.Session, error) {
				return domain.Session{}, domain.ErrUnauthorized
			},
		},
		secretsStub{},
		syncStub{},
	)

	rec := performRequest(func(c *gin.Context) { server.ListRecords(c, ListRecordsParams{}) }, http.MethodGet, "/records", nil, map[string]string{"Authorization": "Bearer token"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized")
	}
}

func TestListRecordsPayloadError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := NewServer(
		authStub{
			validate: func(token string) (domain.Session, error) {
				return domain.Session{UserID: "u1", Token: token, ExpiresAt: time.Now().Add(time.Hour)}, nil
			},
		},
		secretsStub{
			list: func(filter inbound.RecordFilter) ([]domain.Record, error) {
				return []domain.Record{{
					ID:        "r1",
					OwnerID:   "u1",
					Type:      domain.RecordTypeText,
					Payload:   badPayload{},
					Meta:      domain.Metadata{},
					Version:   1,
					UpdatedAt: time.Now(),
				}}, nil
			},
		},
		syncStub{},
	)

	rec := performRequest(func(c *gin.Context) { server.ListRecords(c, ListRecordsParams{}) }, http.MethodGet, "/records", nil, map[string]string{"Authorization": "Bearer token"})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected error status")
	}
}

func TestUpsertRecordErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := NewServer(
		authStub{
			validate: func(token string) (domain.Session, error) {
				return domain.Session{UserID: "u1", Token: token, ExpiresAt: time.Now().Add(time.Hour)}, nil
			},
		},
		secretsStub{
			upsert: func(record domain.Record) (domain.Record, error) {
				return record, nil
			},
		},
		syncStub{},
	)

	rec := performRequest(func(c *gin.Context) { server.UpsertRecord(c) }, http.MethodPost, "/records", []byte("{"), map[string]string{"Authorization": "Bearer token"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request")
	}

	rec = performJSON(func(c *gin.Context) { server.UpsertRecord(c) }, http.MethodPost, "/records", map[string]any{
		"type":    "credential",
		"payload": map[string]any{"kind": "credential"},
	}, map[string]string{"Authorization": "Bearer token"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad payload")
	}
}

func TestGetDeleteNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := NewServer(
		authStub{
			validate: func(token string) (domain.Session, error) {
				return domain.Session{UserID: "u1", Token: token, ExpiresAt: time.Now().Add(time.Hour)}, nil
			},
		},
		secretsStub{
			get: func(id domain.RecordID) (domain.Record, error) {
				return domain.Record{}, domain.ErrNotFound
			},
			delete: func(id domain.RecordID) error {
				return domain.ErrNotFound
			},
		},
		syncStub{},
	)

	rec := performRequest(func(c *gin.Context) { server.GetRecord(c, "r1") }, http.MethodGet, "/records/r1", nil, map[string]string{"Authorization": "Bearer token"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected not found")
	}

	rec = performRequest(func(c *gin.Context) { server.DeleteRecord(c, "r1") }, http.MethodDelete, "/records/r1", nil, map[string]string{"Authorization": "Bearer token"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected not found")
	}
}

func TestSyncErrors(t *testing.T) {
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
				return domain.SyncBatch{}, domain.ErrUnauthorized
			},
			push: func(changes []domain.RecordChange) (domain.SyncResult, error) {
				return domain.SyncResult{}, domain.ErrConflict
			},
		},
	)

	rec := performRequest(func(c *gin.Context) { server.PullSync(c, PullSyncParams{}) }, http.MethodGet, "/sync", nil, map[string]string{"Authorization": "Bearer token"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized")
	}

	rec = performJSON(func(c *gin.Context) { server.PushSync(c) }, http.MethodPost, "/sync", map[string]any{
		"changes": []any{
			map[string]any{
				"record_id":   "r1",
				"owner_id":    "u1",
				"type":        "credential",
				"change":      "upsert",
				"payload":     map[string]any{"kind": "credential"},
				"meta":        map[string]any{},
				"version":     1,
				"happened_at": time.Now().UTC().Format(time.RFC3339),
			},
		},
	}, map[string]string{"Authorization": "Bearer token"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad payload")
	}

	rec = performJSON(func(c *gin.Context) { server.PushSync(c) }, http.MethodPost, "/sync", map[string]any{
		"changes": []any{},
	}, map[string]string{"Authorization": "Bearer token"})
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected conflict")
	}
}

func TestListRecordsInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := NewServer(
		authStub{
			validate: func(token string) (domain.Session, error) {
				return domain.Session{UserID: "u1", Token: token, ExpiresAt: time.Now().Add(time.Hour)}, nil
			},
		},
		secretsStub{
			list: func(filter inbound.RecordFilter) ([]domain.Record, error) {
				return nil, bytes.ErrTooLarge
			},
		},
		syncStub{},
	)

	rec := performRequest(func(c *gin.Context) { server.ListRecords(c, ListRecordsParams{}) }, http.MethodGet, "/records", nil, map[string]string{"Authorization": "Bearer token"})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected error status")
	}
}

func TestPullSyncChangeError(t *testing.T) {
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
				return domain.SyncBatch{
					Changes: []domain.RecordChange{{
						RecordID:   "r1",
						OwnerID:    "u1",
						Type:       domain.RecordTypeText,
						Change:     domain.ChangeUpsert,
						Payload:    badPayload{},
						Meta:       domain.Metadata{},
						Version:    1,
						HappenedAt: time.Now(),
					}},
					Cursor: "1",
				}, nil
			},
		},
	)

	rec := performRequest(func(c *gin.Context) { server.PullSync(c, PullSyncParams{}) }, http.MethodGet, "/sync", nil, map[string]string{"Authorization": "Bearer token"})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected error status")
	}
}

func TestPushSyncUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := NewServer(
		authStub{
			validate: func(token string) (domain.Session, error) {
				return domain.Session{}, domain.ErrUnauthorized
			},
		},
		secretsStub{},
		syncStub{},
	)

	rec := performJSON(func(c *gin.Context) { server.PushSync(c) }, http.MethodPost, "/sync", map[string]any{
		"changes": []any{},
	}, map[string]string{"Authorization": "Bearer token"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized")
	}
}

func TestAuthenticateErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := NewServer(
		authStub{
			validate: func(token string) (domain.Session, error) {
				return domain.Session{}, domain.ErrUnauthorized
			},
		},
		secretsStub{},
		syncStub{},
	)

	rec := performRequest(func(c *gin.Context) { server.GetRecord(c, "r1") }, http.MethodGet, "/records/r1", nil, map[string]string{"Authorization": "Bearer token"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized")
	}
}

func TestDecodeJSONError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/auth", bytes.NewReader([]byte("{")))
	var dst RegisterRequest
	if err := decodeJSON(req, &dst); err == nil {
		t.Fatalf("expected error")
	}
}
