package middleware

import (
	"net/http"
	"slices"
)

type Middleware func(http.Handler) http.Handler

func CreateStack(mw ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := range slices.Backward(mw) {
			middleware := mw[i]
			next = middleware(next)
		}

		return next
	}
}
