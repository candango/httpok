package middleware

import (
	"net/http"
)

// Middleware represents a function that can wrap a http.Handler with
// additional functionality. It takes a http.Handler and returns a new
// http.Handler that includes the middleware's behavior.
type Middleware func(http.Handler) http.Handler

// Chain creates a chain of HTTP middleware functions to wrap around a
// http.Handler.
// It applies each middleware in the order they are provided, allowing for
// layered processing of HTTP requests and responses.
func Chain(next http.Handler, ms ...Middleware) http.Handler {
	for i := len(ms) - 1; i >= 0; i-- {
		m := ms[i]
		next = m(next)
	}
	return next
}
