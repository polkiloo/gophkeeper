package secure

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestServerMiddlewareEncryptsAndDecrypts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	key := TransportKey("0123456789abcdef")
	mw, err := NewServerMiddleware(key)
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	if mw == nil {
		t.Fatalf("expected middleware")
	}

	engine := gin.New()
	engine.Use(gin.HandlerFunc(mw))
	engine.POST("/echo", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		c.JSON(http.StatusOK, gin.H{"echo": string(body)})
	})

	parsed, _ := parseKey(string(key))
	plain := []byte(`{"value":"ping"}`)
	enc, _ := encrypt(parsed, plain)
	payload := encodePayload(enc)

	req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader(payload))
	req.Header.Set(headerEncrypted, "1")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if rec.Header().Get(headerEncrypted) != "1" {
		t.Fatalf("expected encrypted response header")
	}
	decoded, err := decodePayload(rec.Body.Bytes())
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	respBody, err := decrypt(parsed, decoded)
	if err != nil {
		t.Fatalf("decrypt error: %v", err)
	}
	if !bytes.Contains(respBody, []byte("ping")) {
		t.Fatalf("unexpected response body")
	}
}

func TestServerMiddlewareRequiresHeader(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader([]byte(`{}`)))
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}
