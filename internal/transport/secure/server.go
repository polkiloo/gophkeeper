package secure

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Middleware encrypts and decrypts HTTP payloads.
type Middleware func(*gin.Context)

// NewServerMiddleware builds a Gin middleware for request/response encryption.
func NewServerMiddleware(key TransportKey) (Middleware, error) {
	parsed, err := parseKey(string(key))
	if err != nil {
		return nil, err
	}
	if len(parsed) == 0 {
		return nil, nil
	}

	return Middleware(func(c *gin.Context) {
		if c.GetHeader(headerEncrypted) == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "encrypted transport required"})
			return
		}

		if c.Request.Body != nil && c.Request.ContentLength != 0 {
			raw, err := io.ReadAll(c.Request.Body)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
				return
			}
			decoded, err := decodePayload(raw)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid encrypted payload"})
				return
			}
			plain, err := decrypt(parsed, decoded)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid encrypted payload"})
				return
			}
			c.Request.Body = io.NopCloser(bytes.NewReader(plain))
			c.Request.ContentLength = int64(len(plain))
		}

		original := c.Writer
		capture := &captureWriter{ResponseWriter: original}
		c.Writer = capture

		c.Next()

		encrypted, err := encrypt(parsed, capture.body.Bytes())
		if err != nil {
			original.Header().Set("Content-Type", "application/json")
			original.WriteHeader(http.StatusInternalServerError)
			_, _ = original.Write([]byte(`{"error":"failed to encrypt response"}`))
			return
		}
		encoded := encodePayload(encrypted)
		original.Header().Set(headerEncrypted, "1")
		if len(encoded) > 0 {
			original.Header().Set("Content-Length", fmt.Sprintf("%d", len(encoded)))
		}
		status := capture.status
		if status == 0 {
			status = http.StatusOK
		}
		original.WriteHeader(status)
		if len(encoded) > 0 {
			_, _ = original.Write(encoded)
		}
	}), nil
}

type captureWriter struct {
	gin.ResponseWriter
	status int
	body   bytes.Buffer
}

func (w *captureWriter) WriteHeader(status int) {
	w.status = status
}

func (w *captureWriter) Write(data []byte) (int, error) {
	return w.body.Write(data)
}

func (w *captureWriter) WriteString(s string) (int, error) {
	return w.body.WriteString(s)
}
