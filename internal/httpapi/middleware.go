package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

const headerRequestID = "X-Request-ID"

// withMiddleware wraps h with panic recovery and request logging.
func withMiddleware(h http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get(headerRequestID)
		if reqID == "" {
			reqID = newRequestID()
		}
		w.Header().Set(headerRequestID, reqID)

		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		defer func() {
			if v := recover(); v != nil {
				logger.Error("panic recovered", "request_id", reqID, "panic", v)
				rw.status = http.StatusInternalServerError
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			}
			logger.Info("request",
				"request_id", reqID,
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.status,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		}()

		h.ServeHTTP(rw, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func newRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}
