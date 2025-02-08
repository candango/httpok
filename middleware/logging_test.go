package middleware

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type bufferedLogger struct {
	*bytes.Buffer
	t *testing.T
}

func (l *bufferedLogger) Infof(_ string, v ...any) {
	line := fmt.Sprintf("%s %d %s", []any{v[1], v[2], v[3]}...)
	fmt.Fprintf(l, "info: %s ", line)
	fmt.Fprintln(l)
}

func (l *bufferedLogger) Errorf(format string, v ...any) {
	line := fmt.Sprintf("%s %d %s", []any{v[1], v[2], v[3]}...)
	fmt.Fprintf(l, "error: %s ", line)
	fmt.Fprintln(l)
}

func (l *bufferedLogger) Fatalf(format string, v ...any) {
	line := fmt.Sprintf("%s %d %s", []any{v[1], v[2], v[3]}...)
	fmt.Fprintf(l, "fatal: %s ", line)
	fmt.Fprintln(l)
}

func (l *bufferedLogger) Printf(format string, v ...any) {
	l.Infof(format, v...)
}

func (l *bufferedLogger) Warnf(format string, v ...any) {
	line := fmt.Sprintf("%s %d %s", []any{v[1], v[2], v[3]}...)
	fmt.Fprintf(l, "warn: %s ", line)
	fmt.Fprintln(l)
}

func TestLoggingMiddleware(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		expected string
	}{
		{"Success", http.StatusOK, "info: GET 200 / "},
		{"Client Error", http.StatusBadRequest, "warn: GET 400 / "},
		{"Server Error", http.StatusInternalServerError, "error: GET 500 / "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := &bufferedLogger{&buf, t}

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
			})

			loggedHandler := Logging(logger)(handler)
			req, err := http.NewRequest("GET", "/", nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			loggedHandler.ServeHTTP(rr, req)
			// Check if the log output matches the expected format
			if got := buf.String(); !contains(got, tt.expected) {
				t.Errorf("Log contains %q, expected to contain %q", got, tt.expected)
			}

		})
	}
}

// contains checks if a string is present in another string
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
