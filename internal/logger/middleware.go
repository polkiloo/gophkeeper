package logger

import (
	"log"
	"net/http"
	"time"
)

// Middleware logs HTTP request metadata.
type Middleware func(http.Handler) http.Handler

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// NewMiddleware constructs a logging middleware.
func NewMiddleware(logger *log.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)
			duration := time.Since(start)
			logger.Printf("%s %s %d %s", r.Method, r.URL.Path, recorder.status, duration)
		})
	}
}
