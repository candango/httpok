package httpok

import "net/http"

// WrappedWriter wraps an http.ResponseWriter to capture the status code of the
// response.
type WrappedWriter struct {
	http.ResponseWriter
	StatusCode int
}

// WriteHeader records the status code and then calls the underlying
// WriteHeader method.
// This allows tracking of the response status without altering how the
// original ResponseWriter behaves.
func (w *WrappedWriter) WriteHeader(c int) {
	w.ResponseWriter.WriteHeader(c)
	w.StatusCode = c
}
