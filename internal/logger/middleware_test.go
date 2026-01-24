package logger

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMiddlewareLogs(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := log.New(buf, "", 0)
	mw := NewMiddleware(logger)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	output := buf.String()
	if !strings.Contains(output, "POST /test") || !strings.Contains(output, "201") {
		t.Fatalf("expected log output, got: %s", output)
	}
}
