package httpok

import (
	"net/http"
	"testing"

	"github.com/candango/httpok/testrunner"
	"github.com/stretchr/testify/assert"
)

func Wrap(next http.Handler, ww *WrappedWriter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*ww = WrappedWriter{
			ResponseWriter: w,
			StatusCode:     http.StatusOK,
		}
		next.ServeHTTP(ww, r)
	})
}

type WrappedHandler struct {
	http.Handler
}

func (h *WrappedHandler) GetOK(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("It's ok"))
}

func (h *WrappedHandler) GetInternalError(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("It's an internal error"))
	w.WriteHeader(http.StatusInternalServerError)
}

func NewWrappedServeMux(ww *WrappedWriter) http.Handler {
	handler := &WrappedHandler{}
	h := http.NewServeMux()
	h.HandleFunc("/ok", handler.GetOK)
	h.HandleFunc("/internal_error", handler.GetInternalError)
	return Wrap(h, ww)
}

func TestWrappedWriter(t *testing.T) {
	ww := &WrappedWriter{}
	h := NewWrappedServeMux(ww)

	runner := testrunner.NewHttpTestRunner(t).WithHandler(h)

	t.Run("Wrapped runner", func(t *testing.T) {
		res, err := runner.WithPath("/ok").Get()
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, http.StatusOK, ww.StatusCode)
		assert.Equal(t, "It's ok", testrunner.BodyAsString(t, res))

		res, err = runner.WithPath("/internal_error").Get()
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, http.StatusInternalServerError, ww.StatusCode)
		assert.Equal(t, "It's an internal error", testrunner.BodyAsString(t, res))
	})

}
