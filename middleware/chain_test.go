package middleware

import (
	"net/http"
	"testing"

	"github.com/candango/httpok/testrunner"
	"github.com/stretchr/testify/assert"
)

type PlainHandler struct {
	http.Handler
}

func (h *PlainHandler) GetSomething(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Something"))
}

func (h *PlainHandler) GetSomethingElse(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Something else"))
}

func NewPlainServeMux() http.Handler {
	plain := &PlainHandler{}
	h := http.NewServeMux()
	h.HandleFunc("/something", plain.GetSomething)
	h.HandleFunc("/something_else", plain.GetSomethingElse)
	return h
}

func TestChainMiddlewareServer(t *testing.T) {
	plain := NewPlainServeMux()

	runner := testrunner.NewHttpTestRunner(t).WithHandler(plain)

	t.Run("Plain runner", func(t *testing.T) {
		res, err := runner.WithPath("/something").Get()
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, "200 OK", res.Status)
		assert.Equal(t, "Something", testrunner.BodyAsString(t, res))

		res, err = runner.WithPath("/something_else").Get()
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, "200 OK", res.Status)
		assert.Equal(t, "Something else", testrunner.BodyAsString(t, res))
	})

	changeSomething := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "/something" {
				w.Write([]byte("First Middleware with "))
			}
			next.ServeHTTP(w, r)
		})
	}

	blockSomethingElse := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "/something_else" {
				http.Error(w, "Not allowed", http.StatusMethodNotAllowed)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	chain := Chain(plain, changeSomething, blockSomethingElse)
	runner = testrunner.NewHttpTestRunner(t).WithHandler(chain)

	t.Run("Chained runner", func(t *testing.T) {
		res, err := runner.WithPath("/something").Get()
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, "200 OK", res.Status)
		assert.Equal(t, "First Middleware with Something", testrunner.BodyAsString(t, res))

		res, err = runner.WithPath("/something_else").Get()
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, "405 Method Not Allowed", res.Status)
		assert.Equal(t, "Not allowed\n", testrunner.BodyAsString(t, res))
	})
}
