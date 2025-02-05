package middleware

import (
	"net/http"
	"time"

	"github.com/candango/httpok"
	"github.com/candango/httpok/logger"
)

// Logging creates a logging middleware with a custom logger
func Logging(log logger.Logger) func(http.Handler) http.Handler {
	if log == nil {
		log = &logger.StandardLogger{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := &httpok.WrappedWriter{
				ResponseWriter: w,
				StatusCode:     http.StatusOK,
			}
			f := "[02/01/2006:03:04:05 07]"
			next.ServeHTTP(wrapped, r)
			s := time.Now().Format(f)
			switch {
			case wrapped.StatusCode >= 500:
				log.Errorf("%s %s %d %s %d", s, r.Method, wrapped.StatusCode,
					r.URL.Path, time.Since(start).Microseconds())
			case wrapped.StatusCode >= 400:
				log.Warnf("%s %s %d %s %d", s, r.Method, wrapped.StatusCode,
					r.URL.Path, time.Since(start).Microseconds())
			default:
				log.Printf("%s %s %d %s %d", s, r.Method, wrapped.StatusCode,
					r.URL.Path, time.Since(start).Microseconds())
			}
		})
	}
}
