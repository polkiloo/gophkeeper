package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"gophkeeper/internal/domain"
)

func TestBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	if _, err := bearerToken(ctx); err == nil {
		t.Fatalf("expected error")
	}

	ctx.Request.Header.Set("Authorization", "Token foo")
	if _, err := bearerToken(ctx); err == nil {
		t.Fatalf("expected error")
	}

	ctx.Request.Header.Set("Authorization", "Bearer abc")
	token, err := bearerToken(ctx)
	if err != nil || token != "abc" {
		t.Fatalf("unexpected token")
	}
}

func TestDecodeJSONUnknownField(t *testing.T) {
	body := bytes.NewBufferString(`{"login":"u","extra":"x"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth", body)
	var dst RegisterRequest
	if err := decodeJSON(req, &dst); err == nil {
		t.Fatalf("expected error")
	}
}

func TestWriteDomainError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		err    error
		status int
	}{
		{domain.ErrUnauthorized, http.StatusUnauthorized},
		{domain.ErrNotFound, http.StatusNotFound},
		{domain.ErrConflict, http.StatusConflict},
		{errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tc := range cases {
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		writeDomainError(ctx, tc.err)
		if rec.Code != tc.status {
			t.Fatalf("expected status %d, got %d", tc.status, rec.Code)
		}
		var payload errorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil || payload.Error == "" {
			t.Fatalf("expected error payload")
		}
	}
}

func TestHelpers(t *testing.T) {
	if got := idsToStrings([]domain.RecordID{"a", "b"}); len(got) != 2 || got[0] != "a" {
		t.Fatalf("unexpected ids")
	}
	if derefInt32(nil) != 0 {
		t.Fatalf("expected zero")
	}
	value := int32(7)
	if derefInt32(&value) != 7 {
		t.Fatalf("unexpected deref")
	}
	if derefRecordType(nil) != "" {
		t.Fatalf("expected empty record type")
	}
	rt := RecordType("text")
	if derefRecordType(&rt) != "text" {
		t.Fatalf("unexpected record type")
	}
}
