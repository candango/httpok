package middleware

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/candango/httpok/security"
	"github.com/candango/httpok/session"
)

// newCookie creates and returns a new HTTP cookie with the specified
// parameters.
func newCookie(name string, value string, age time.Duration) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   int(age / time.Second),
		HttpOnly: false,
		Secure:   false,
		// SameSite: http.SameSiteLaxMode, // Protection mode against CSRF
	}
}

func sessionCookie(e session.Engine, id string) *http.Cookie {
	properties := e.Properties()
	value := id
	if len(properties.CookieSecret) != 0 {
		value = security.CreateSignedValue(
			properties.CookieSecret,
			properties.Name,
			id,
			time.Now(),
		)
	}
	return newCookie(properties.Name, value, properties.CookieMaxAge)
}

func sessionIDFromCookie(e session.Engine, value string) (string, bool) {
	properties := e.Properties()
	if len(properties.CookieSecret) == 0 {
		return value, true
	}
	return security.DecodeSignedValue(
		properties.CookieSecret,
		properties.Name,
		value,
		properties.CookieMaxAge,
		time.Now(),
	)
}

// Sessioned returns middleware that manages session cookies using the
// provided session Engine.
//
// When CookieSecret is configured, cookies contain Tornado-compatible signed
// session IDs. Missing, invalid, expired, or unknown session IDs are replaced
// with new sessions. The current session is added to the request context before
// the next handler runs and is persisted after the handler returns.
func Sessioned(e session.Engine) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctxEngine := context.WithValue(r.Context(), session.ContextEngValue, e)
			id, ok := sessionIDFromRequest(e, r)
			if ok {
				exists, err := e.SessionExists(ctxEngine, id)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				ok = exists
			}
			if !ok {
				id = e.NewId(r.Context())
				setSessionCookie(w, e, id)
			}

			s, err := e.GetSession(ctxEngine, id)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			ctxSess := context.WithValue(ctxEngine, session.ContextSessValue, &s)
			next.ServeHTTP(w, r.WithContext(ctxSess))
			if err := e.SaveSession(ctxEngine, s.Id, s); err != nil {
				log.Printf("failed to save session %s: %v", s.Id, err)
			}
		})
	}
}

func sessionIDFromRequest(e session.Engine, r *http.Request) (string, bool) {
	cookie, err := r.Cookie(e.Properties().Name)
	if err != nil {
		return "", false
	}
	return sessionIDFromCookie(e, cookie.Value)
}

func setSessionCookie(w http.ResponseWriter, e session.Engine, id string) {
	http.SetCookie(w, sessionCookie(e, id))
}
