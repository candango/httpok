package middleware

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/candango/httpok/session"
)

// newCookie creates and returns a new HTTP cookie with the specified
// parameters.
func newCookie(name string, value string, age time.Duration) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   int(age),
		HttpOnly: false,
		Secure:   false,
		// SameSite: http.SameSiteLaxMode, // Protection mode against CSRF
	}
}

// Sessioned returns a middleware that manages session cookies using the
// provided session Engine.
// It checks for existing cookies, creates new ones if necessary, and
// associates session data with
// the request context before passing it to the next handler.
func Sessioned(e session.Engine) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fromServer := false
			ctxEngine := context.WithValue(r.Context(), session.ContextEngValue, e)
			cookie, err := r.Cookie(e.Name())
			if err != nil {
				fromServer = true
				cookie = newCookie(e.Name(), e.NewId(), 1*time.Hour)
				log.Printf("cookie %s does not exists", cookie.Name)
				http.SetCookie(w, cookie)
			}
			log.Printf("BEM AQUI: from server %v", fromServer)
			nok, err := e.SessionNotExists(cookie.Value)
			if !fromServer && nok {
				fromServer = true
				cookie = newCookie(e.Name(), e.NewId(), 1*time.Hour)
				log.Printf("cookie %s does exists but session does not exists", cookie.Name)
				http.SetCookie(w, cookie)
			}
			s, err := e.GetSession(cookie.Value, ctxEngine)
			if err != nil {
				// TODO: Log this error
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			ctxSess := context.WithValue(ctxEngine, session.ContextSessValue, &s)
			req := r.WithContext(ctxSess)
			log.Printf("From server: %v\n", fromServer)
			next.ServeHTTP(w, req)
			e.StoreSession(s.Id, s)
			log.Printf("Session Data at the end: %v\n", s.Data)
			// TODO: Store the session
		})
	}
}
