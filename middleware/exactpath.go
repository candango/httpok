package middleware

import (
	"net/http"
)

// ExactPath creates a middleware that checks if the request's path exactly
// matches the given path.
// If the path does not match, it returns a 404 Not Found response. Otherwise,
// it passes the request to the next handler in the chain.
func ExactPath(path string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
