package logger

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// GinMiddleware logs Gin HTTP requests.
type GinMiddleware = gin.HandlerFunc

// NewGinMiddleware constructs a Gin-compatible logging middleware.
func NewGinMiddleware(logger *log.Logger) GinMiddleware {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Printf("%s %s %d %s", c.Request.Method, c.Request.URL.Path, c.Writer.Status(), time.Since(start))
	}
}
