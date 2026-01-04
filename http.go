package httpok

import (
	"encoding/json"
	"io"
	"net/http"
)

// BodyAsString reads the entire body of an HTTP response and returns it as a
// string.
// It consumes the response body, so the caller should not attempt to read from
// it again.
func BodyAsString(res *http.Response) (string, error) {
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// BodyAsJson reads the entire body of an HTTP response and unmarshals it into
// the provided jsonBody.
// It consumes the response body, so the caller should not attempt to read from
// it again.
// The jsonBody parameter should be a pointer to a struct or a slice where JSON
// data will be unmarshaled.
func BodyAsJson(res *http.Response, jsonBody any) error {
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, jsonBody)
	if err != nil {
		return err
	}
	return nil
}

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
