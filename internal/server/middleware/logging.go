package middleware

import (
	"log"
	"net/http"
	"time"

	"github.com/nickarnoldi/funnel-service/internal/utils"
)

type WrappedWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *WrappedWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.statusCode = statusCode
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now().UTC()

		wrapped := &WrappedWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(wrapped, r)

		clientIP := utils.GetRequestIPAddress(r)

		log.Println(wrapped.statusCode, r.Method, r.URL.Path, time.Since(start), clientIP, r.Header.Get("Referer"))
	})
}
