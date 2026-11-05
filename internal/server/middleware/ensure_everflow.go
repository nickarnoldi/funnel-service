package middleware

import (
	"net/http"
	"strings"
)

func EnsureIsEverflow(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bearer := r.Header.Get("Authorization")
		if bearer != "" {
			token := strings.TrimPrefix(bearer, "Bearer ")
			if token == "b2ab6db739a9a1b8e4f28cce3d6fb158" {
				next.ServeHTTP(w, r)
				return
			}
		}

		http.Error(w, "Not Found", http.StatusNotFound)
	})
}
