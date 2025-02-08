package httpok

import "net/http"

type WrappedWriter struct {
	http.ResponseWriter
	StatusCode int
}

func (w *WrappedWriter) WriteHeader(c int) {
	w.ResponseWriter.WriteHeader(c)
	w.StatusCode = c
}
