// Package middleware provides HTTP middleware for httpok servers.
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

// sessionCookie creates the session cookie for id. When the engine has a
// cookie secret, the value is encoded as a Tornado-compatible signed value.
func sessionCookie(e session.Engine, id string) *http.Cookie {
	properties := e.Properties()
	value := id
	keys := cookieSecrets(properties)
	if len(keys) != 0 {
		version, secret, ok := activeCookieSecret(properties, keys)
		if ok {
			value = security.CreateSignedValueWithKeyVersion(
				secret,
				version,
				properties.Name,
				id,
				time.Now(),
			)
		} else {
			log.Printf("no active session cookie key configured")
			value = ""
		}
	}
	cookie := newCookie(properties.Name, value, properties.CookieMaxAge)
	options := cookieOptions(properties)
	cookie.HttpOnly = options.HTTPOnly
	cookie.Secure = options.Secure
	cookie.SameSite = options.SameSite
	return cookie
}

// sessionIDFromCookie verifies value when signed cookies are configured and
// returns the underlying session ID.
func sessionIDFromCookie(e session.Engine, value string) (string, bool) {
	properties := e.Properties()
	keys := cookieSecrets(properties)
	if len(keys) == 0 {
		return value, true
	}
	return security.DecodeSignedValueWithKeys(
		keys,
		properties.Name,
		value,
		properties.CookieMaxAge,
		time.Now(),
	)
}

func cookieSecrets(properties *session.EngineProperties) map[int][]byte {
	keys := make(map[int][]byte, len(properties.CookieSecrets)+1)
	for version, secret := range properties.CookieSecrets {
		keys[version] = append([]byte(nil), secret...)
	}
	if len(properties.CookieSecret) != 0 {
		if _, ok := keys[0]; !ok {
			keys[0] = append([]byte(nil), properties.CookieSecret...)
		}
	}
	return keys
}

func activeCookieSecret(
	properties *session.EngineProperties,
	keys map[int][]byte,
) (int, []byte, bool) {
	if properties.CookieKeyVersion != nil {
		version := *properties.CookieKeyVersion
		secret, ok := keys[version]
		return version, secret, ok && len(secret) != 0
	}

	activeVersion := 0
	var activeSecret []byte
	for version, secret := range keys {
		if len(secret) != 0 && (activeSecret == nil || version > activeVersion) {
			activeVersion = version
			activeSecret = secret
		}
	}
	return activeVersion, activeSecret, len(activeSecret) != 0
}

func cookieOptions(properties *session.EngineProperties) session.CookieOptions {
	if properties.CookieOptions == nil {
		return session.CookieOptions{
			HTTPOnly: true,
			SameSite: http.SameSiteLaxMode,
		}
	}
	return *properties.CookieOptions
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

// sessionIDFromRequest extracts and validates the configured session cookie
// from r.
func sessionIDFromRequest(e session.Engine, r *http.Request) (string, bool) {
	cookie, err := r.Cookie(e.Properties().Name)
	if err != nil {
		return "", false
	}
	return sessionIDFromCookie(e, cookie.Value)
}

// setSessionCookie writes a new session cookie to w.
func setSessionCookie(w http.ResponseWriter, e session.Engine, id string) {
	http.SetCookie(w, sessionCookie(e, id))
}
