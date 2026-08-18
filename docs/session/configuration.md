# Session Configuration

`session.EngineProperties` contains the shared configuration for a session
engine. `session.NewStoreEngine` supplies defaults and `WithProperties` merges
non-zero settings.

## Core properties

| Property | Meaning | Default |
|---|---|---|
| `Name` | Client cookie name | `HTTPOKSESSID` |
| `AgeLimit` | Server-side idle lifetime | 30 minutes |
| `CookieMaxAge` | Client cookie lifetime | 31 days |
| `CookieSecret` | Backward-compatible signing key, version 0 | empty |
| `CookieSecrets` | Versioned signing keys | empty |
| `CookieKeyVersion` | Explicit active signing key version | highest configured version |
| `CookieOptions` | HTTP cookie attributes | `HttpOnly`, `SameSite=Lax` |
| `PurgeDuration` | Store purge interval | 2 minutes |
| `Prefix` | Store namespace prefix | `httpok:session` |
| `Enabled` | Enables the engine | true |
| `Encoder` | Session data encoder | JSON |

## Signed cookies and key rotation

A single-key configuration is backward compatible:

```go
session.WithProperties(&session.EngineProperties{
    CookieSecret: []byte("current-secret"),
})
```

For rotation, configure versioned keys and optionally select the active version:

```go
active := 2
session.WithProperties(&session.EngineProperties{
    CookieSecrets: map[int][]byte{
        1: []byte("old-secret"),
        2: []byte("current-secret"),
    },
    CookieKeyVersion: &active,
})
```

Version 2 cookies identify the signing key in the cookie value. The decoder
accepts every configured key so old cookies can remain valid during a planned
rotation. New cookies use the active key. Legacy version 1 cookies do not carry
a key version and are checked against the configured keys.

Keep secrets outside source control and inject them through the application's
secret-management mechanism.

## Cookie attributes

```go
session.WithProperties(&session.EngineProperties{
    CookieOptions: &session.CookieOptions{
        HTTPOnly: true,
        Secure:   true,
        SameSite: http.SameSiteLaxMode,
    },
})
```

The explicit insecure development configuration is possible, but should not
be used in production:

```go
CookieOptions: &session.CookieOptions{
    HTTPOnly: false,
    Secure:   false,
    SameSite: http.SameSiteNoneMode,
}
```

`SameSite=None` normally requires `Secure` in browsers. The session package
does not infer whether the deployment is behind HTTPS; the application must
configure `Secure` correctly.
