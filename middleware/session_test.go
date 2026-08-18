package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/candango/httpok/security"
	"github.com/candango/httpok/session"
	"github.com/candango/httpok/testrunner"
	"github.com/stretchr/testify/assert"
)

func TestSessionedSignsAndVerifiesSessionCookies(t *testing.T) {
	engine := session.NewStoreEngine(
		session.NewMemoryStore(),
		session.WithProperties(&session.EngineProperties{
			Name:         "sid",
			CookieSecret: []byte("secret"),
			CookieMaxAge: 24 * time.Hour,
		}),
	)
	handler := Sessioned(engine)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, err := session.SessionFromContext(r.Context())
		if err != nil {
			t.Error(err)
			return
		}
		if _, err := sess.Get("initialized"); err != nil {
			t.Error(err)
			return
		}
		if _, ok := sess.Data["initialized"]; !ok {
			if err := sess.Set("initialized", true); err != nil {
				t.Error(err)
				return
			}
		}
		_, _ = w.Write([]byte(sess.Id))
	}))

	firstRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	firstResponse := httptest.NewRecorder()
	handler.ServeHTTP(firstResponse, firstRequest)

	firstCookies := firstResponse.Result().Cookies()
	if assert.Len(t, firstCookies, 1) {
		firstCookie := firstCookies[0]
		firstID, ok := security.DecodeSignedValue(
			[]byte("secret"),
			"sid",
			firstCookie.Value,
			24*time.Hour,
			time.Now(),
		)
		assert.True(t, ok)
		assert.NotEmpty(t, firstID)

		secondRequest := httptest.NewRequest(http.MethodGet, "/", nil)
		secondRequest.AddCookie(firstCookie)
		secondResponse := httptest.NewRecorder()
		handler.ServeHTTP(secondResponse, secondRequest)
		assert.Empty(t, secondResponse.Header().Values("Set-Cookie"))
		assert.Equal(t, firstID, secondResponse.Body.String())

		tamperedCookie := *firstCookie
		tamperedCookie.Value += "x"
		tamperedRequest := httptest.NewRequest(http.MethodGet, "/", nil)
		tamperedRequest.AddCookie(&tamperedCookie)
		tamperedResponse := httptest.NewRecorder()
		handler.ServeHTTP(tamperedResponse, tamperedRequest)
		tamperedCookies := tamperedResponse.Result().Cookies()
		if assert.Len(t, tamperedCookies, 1) {
			tamperedID, ok := security.DecodeSignedValue(
				[]byte("secret"),
				"sid",
				tamperedCookies[0].Value,
				24*time.Hour,
				time.Now(),
			)
			assert.True(t, ok)
			assert.NotEqual(t, firstID, tamperedID)
		}
	}
}

func TestSessionMiddlewareServer(t *testing.T) {
	plain := NewPlainServeMux()

	runner := testrunner.NewHttpTestRunner(t).WithHandler(plain)

	t.Run("Session engine", func(t *testing.T) {
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

}
