package middleware

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/candango/httpok/security"
	"github.com/candango/httpok/session"
	"github.com/candango/schedulerok/clocktest"
)

const (
	crossStackCookieName = "sid"
	crossStackSecret     = "cross-stack-test-secret"
	crossStackSessionID  = "session-123"
)

func TestTornadoAndGoSessionCookieInterop(t *testing.T) {
	clock := clocktest.New(time.Now().UTC().Truncate(time.Second))
	cookieMaxAge := 24 * time.Hour
	engine := session.NewStoreEngine(
		session.NewMemoryStore(),
		session.WithProperties(&session.EngineProperties{
			Name:         crossStackCookieName,
			CookieSecret: []byte(crossStackSecret),
			CookieMaxAge: cookieMaxAge,
			AgeLimit:     time.Hour,
		}),
	)
	if err := engine.Start(context.Background()); err != nil {
		t.Fatalf("engine.Start() error = %v", err)
	}
	defer engine.Stop(context.Background())

	if err := engine.SaveSession(context.Background(), crossStackSessionID,
		session.Session{
			Id: crossStackSessionID,
			Data: map[string]any{
				"UserRole": "Member",
			},
		}); err != nil {
		t.Fatalf("seed session error = %v", err)
	}

	server := httptest.NewServer(Sessioned(engine)(http.HandlerFunc(
		crossStackSessionHandler,
	)))
	defer server.Close()

	for _, version := range []int{1, 2} {
		t.Run(fmt.Sprintf("tornado v%d to go", version), func(t *testing.T) {
			cookieValue := tornadoGenerate(t, version, crossStackSessionID,
				clock.Now())
			response, body := requestWithCookie(t, server.URL, cookieValue)
			defer response.Body.Close()

			if response.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.StatusCode,
					http.StatusOK)
			}
			if got, want := body, crossStackSessionID+":Member"; got != want {
				t.Fatalf("body = %q, want %q", got, want)
			}
			if cookies := response.Header.Values("Set-Cookie"); len(cookies) != 0 {
				t.Fatalf("valid cookie was replaced: %v", cookies)
			}
		})
	}

	t.Run("go v2 to tornado", func(t *testing.T) {
		goCookie := security.CreateSignedValue(
			[]byte(crossStackSecret),
			crossStackCookieName,
			crossStackSessionID,
			clock.Now(),
		)
		decoded := tornadoDecode(t, goCookie, clock.Now())
		if decoded != crossStackSessionID {
			t.Fatalf("Tornado decoded %q, want %q", decoded,
				crossStackSessionID)
		}
	})
}

func TestTornadoInvalidCookiesRenewSession(t *testing.T) {
	clock := clocktest.New(time.Now().UTC().Truncate(time.Second))
	cookieMaxAge := time.Hour
	engine := session.NewStoreEngine(
		session.NewMemoryStore(),
		session.WithProperties(&session.EngineProperties{
			Name:         crossStackCookieName,
			CookieSecret: []byte(crossStackSecret),
			CookieMaxAge: cookieMaxAge,
		}),
	)
	if err := engine.Start(context.Background()); err != nil {
		t.Fatalf("engine.Start() error = %v", err)
	}
	defer engine.Stop(context.Background())

	server := httptest.NewServer(Sessioned(engine)(http.HandlerFunc(
		crossStackSessionHandler,
	)))
	defer server.Close()

	validCookie := tornadoGenerate(t, 2, crossStackSessionID, clock.Now())
	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "tampered signature",
			value: tamperCookie(validCookie),
		},
		{
			name: "expired timestamp",
			value: tornadoGenerate(
				t,
				2,
				crossStackSessionID,
				clock.Now().Add(-2*cookieMaxAge),
			),
		},
		{
			name:  "unknown session ID",
			value: tornadoGenerate(t, 2, "unknown-session", clock.Now()),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, body := requestWithCookie(t, server.URL, test.value)
			defer response.Body.Close()

			if response.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.StatusCode,
					http.StatusOK)
			}
			if strings.HasPrefix(body, crossStackSessionID+":") ||
				strings.HasPrefix(body, "unknown-session:") {
				t.Fatalf("invalid cookie reused session: %q", body)
			}
			if cookies := response.Cookies(); len(cookies) != 1 {
				t.Fatalf("replacement cookies = %d, want 1", len(cookies))
			}
		})
	}
}

func crossStackSessionHandler(w http.ResponseWriter, r *http.Request) {
	current, err := session.SessionFromContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	role, err := current.Get("UserRole")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if role == nil {
		role = "anonymous"
	}
	_, _ = fmt.Fprintf(w, "%s:%v", current.Id, role)
}

func requestWithCookie(
	t *testing.T,
	serverURL string,
	cookieValue string,
) (*http.Response, string) {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, serverURL, nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	request.AddCookie(&http.Cookie{
		Name:  crossStackCookieName,
		Value: cookieValue,
	})
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("HTTP request error = %v", err)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		response.Body.Close()
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	return response, string(body)
}

func tornadoGenerate(
	t *testing.T,
	version int,
	value string,
	now time.Time,
) string {
	t.Helper()
	return runTornado(t,
		"generate",
		"--secret", crossStackSecret,
		"--name", crossStackCookieName,
		"--value", value,
		"--version", strconv.Itoa(version),
		"--now", strconv.FormatInt(now.Unix(), 10),
	)
}

func tornadoDecode(t *testing.T, value string, now time.Time) string {
	t.Helper()
	return runTornado(t,
		"decode",
		"--secret", crossStackSecret,
		"--name", crossStackCookieName,
		"--value", value,
		"--max-age-days", "1",
		"--now", strconv.FormatInt(now.Unix(), 10),
	)
}

func runTornado(t *testing.T, args ...string) string {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Fatalf("python3 is required for cross-stack tests: %v", err)
	}
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	script := filepath.Join(filepath.Dir(sourceFile), "testdata",
		"tornado_cookie.py")
	commandArgs := append([]string{script}, args...)
	command := exec.Command(python, commandArgs...)
	output, err := command.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("Tornado helper failed: %v\n%s", err, exitErr.Stderr)
		}
		t.Fatalf("Tornado helper failed: %v", err)
	}
	return strings.TrimSpace(string(output))
}

func tamperCookie(value string) string {
	if value[len(value)-1] == '0' {
		return value[:len(value)-1] + "1"
	}
	return value[:len(value)-1] + "0"
}
