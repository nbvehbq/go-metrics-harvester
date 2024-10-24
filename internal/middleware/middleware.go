package middleware

import "net/http"

type Middleware func(http.HandlerFunc) http.HandlerFunc

// Combine combines middlewares
func Combine(h http.HandlerFunc, m ...Middleware) http.HandlerFunc {
	if len(m) < 1 {
		return h
	}

	wrapped := h
	for i := len(m) - 1; i >= 0; i-- {
		wrapped = m[i](wrapped)
	}

	return wrapped
}
