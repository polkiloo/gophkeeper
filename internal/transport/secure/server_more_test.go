package secure

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestServerMiddlewareInvalidPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mw, err := NewServerMiddleware("0123456789abcdef")
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}

	engine := gin.New()
	engine.Use(gin.HandlerFunc(mw))
	engine.POST("/echo", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader([]byte("bad")))
	req.Header.Set(headerEncrypted, "1")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestNewServerMiddlewareEmptyKey(t *testing.T) {
	mw, err := NewServerMiddleware("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mw != nil {
		t.Fatalf("expected nil middleware")
	}
}

func TestCaptureWriterWriteString(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	writer := &captureWriter{ResponseWriter: ctx.Writer}
	if _, err := writer.WriteString("hello"); err != nil {
		t.Fatalf("write error: %v", err)
	}
	if writer.body.String() != "hello" {
		t.Fatalf("unexpected body")
	}
}
