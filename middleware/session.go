package middleware

// Copyright 2023-2025 Flavio Garcia
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/candango/httpok/session"
)

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
