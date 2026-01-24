package logger

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNewGinMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	buf := new(bytes.Buffer)
	logger := log.New(buf, "", 0)
	engine := gin.New()
	engine.Use(NewGinMiddleware(logger))
	engine.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	output := buf.String()
	if !strings.Contains(output, "GET /ping") || !strings.Contains(output, "204") {
		t.Fatalf("expected log output, got: %s", output)
	}
}
