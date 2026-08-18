# Session Security

## Threat model

Relevant attacker-controlled inputs include:

- cookie names and values sent in HTTP requests;
- session IDs reaching a storage backend;
- configuration-provided key maps and cookie options;
- files present in a FileStore directory.

The main assets are session data, signing secrets, and the ability to impersonate
a session.

## Cookie integrity

Use a long, random `CookieSecret` or versioned key set. Never commit secrets to
the repository or log cookie values. Signed cookies prevent an attacker from
forging a different session ID without the key, but they do not hide the ID or
session data.

During key rotation:

1. add the new key while retaining the old key;
2. make the new key active for new cookies;
3. accept old configured keys during the migration window;
4. remove the old key after its cookie lifetime and session policy allow it.

## Cookie transport

Recommended production baseline:

```go
CookieOptions: &session.CookieOptions{
    HTTPOnly: true,
    Secure:   true,
    SameSite: http.SameSiteLaxMode,
}
```

- `HttpOnly` limits script access after an XSS flaw;
- `Secure` prevents transmission over cleartext HTTP;
- `SameSite` reduces cross-site request delivery.

These flags do not replace output encoding, XSS defenses, TLS, or CSRF tokens.

## CSRF

A valid session cookie can still be sent automatically by a browser. Applications
that accept state-changing requests should use an explicit CSRF defense, such as
a synchronizer token or a carefully implemented double-submit token, and should
consider validating `Origin`/`Referer`. The session package currently provides
cookie transport and signing, not a complete CSRF framework.

## FileStore boundaries

Never let arbitrary cookie values become filesystem paths. Validate the session
ID before joining it to a storage directory, use a fixed filename namespace, and
operate only on regular files belonging to that namespace. Avoid deleting or
purging unrelated files in the directory.

## Logging and secrets

Do not log:

- `CookieSecret` values;
- versioned signing keys;
- complete signed cookie values;
- session data containing credentials or personal data.

Errors should identify the operation and session lifecycle state without
revealing secret material.
