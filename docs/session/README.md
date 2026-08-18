# Session Guide

This guide documents the `httpok/session` package and the session middleware.
It is the canonical behavioral guide; Go API comments remain the API reference.

## What a session is

`httpok` keeps session data on the server and sends only an opaque session ID
to the client. The client identifier can be sent as a plain cookie or as a
Tornado-compatible signed cookie when a secret is configured.

Session data is managed through an `Engine` and persisted by a pluggable
`Store`.

```text
HTTP request
  -> read and validate the session cookie
  -> load the server-side session
  -> expose the session through request context
  -> run the application handler
  -> persist or destroy the session
```

## Quick start

```go
store := session.NewMemoryStore()
engine := session.NewStoreEngine(
    store,
    session.WithProperties(&session.EngineProperties{
        Name:         "app-session",
        CookieSecret: []byte("a-long-random-secret"),
        CookieMaxAge: 24 * time.Hour,
    }),
)

handler := middleware.Sessioned(engine)(applicationHandler)
```

Applications retrieve the current session from the request context:

```go
sess, err := session.SessionFromContext(r.Context())
if err != nil {
    http.Error(w, "session unavailable", http.StatusInternalServerError)
    return
}

if err := sess.Set("user_id", userID); err != nil {
    http.Error(w, "session update failed", http.StatusInternalServerError)
    return
}
```

## Documentation map

- [Configuration](configuration.md)
- [Cookies](cookies.md)
- [Lifecycle](lifecycle.md)
- [Stores](stores.md)
- [Security](security.md)
- [Testing](testing.md)
- [Troubleshooting](troubleshooting.md)

## Implementation status

- Signed Tornado-compatible cookies: implemented.
- Cookie attributes and key rotation: implemented in the current worktree.
- Changed-session persistence and destruction: tracked by task #30.
- FileStore filename isolation and ID validation: tracked by task #32.
- Cross-stack Tornado/Go tests: tracked by task #31.
- Full CSRF token protection: not implemented by the session package.
