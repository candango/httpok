# Session Cookies

The session cookie contains an opaque session ID. Session data is not stored in
the cookie.

## Unsigned cookies

When no cookie secret is configured, the cookie value is the raw session ID.
This mode provides no integrity protection: a client can replace the ID with
another value. Use it only when another trusted boundary protects the cookie or
when signing is intentionally disabled for local development.

## Signed cookies

When `CookieSecret` or `CookieSecrets` is configured, the middleware signs the
session ID using the Tornado signed-value format.

Signing provides integrity and authenticity. It does **not** provide:

- confidentiality;
- protection against a browser sending a valid cookie in a forged request;
- a CSRF token;
- server-side session revocation by itself.

## Tornado version 1

Version 1 is the legacy format:

```text
base64(value)|timestamp|hex(HMAC-SHA1(name + value + timestamp))
```

There is no explicit version field. The implementation accepts version 1 for
backward compatibility but does not generate it.

## Tornado version 2

Version 2 is the generated format:

```text
2|key-version|timestamp|name|base64(value)|hex(HMAC-SHA256(signed-fields))
```

The key version, timestamp, name, and value fields are length-prefixed. The
HMAC covers the complete value up to and including the final separator before
the signature.

Example shape:

```text
2|1:2|10:1700000000|11:app-session|16:c2Vzc2lvbi0xMjM=|<64 hex chars>
```

The exact field lengths depend on the configured cookie name and value.

## Validation behavior

A signed cookie is rejected when:

- the secret is empty or unavailable;
- the format is malformed;
- the key version is unknown;
- the cookie name does not match;
- the HMAC does not match;
- the timestamp is outside the configured age window;
- the base64 value is invalid.

A rejected cookie does not become a server-side session lookup. The middleware
creates a fresh session ID and sends a replacement cookie.

## HTTP attributes

| Attribute | Purpose |
|---|---|
| `HttpOnly` | Prevents ordinary JavaScript from reading the cookie |
| `Secure` | Sends the cookie only over HTTPS |
| `SameSite` | Restricts cross-site cookie delivery and reduces CSRF exposure |
| `Max-Age` | Controls browser-side lifetime |
| `Path` | Limits the URL path; session cookies use `/` |

The default options are `HttpOnly=true`, `SameSite=Lax`, and `Secure=false`.
Enable `Secure` in HTTPS deployments.

## SameSite and CSRF

`SameSite=Lax` is a useful browser-level mitigation, not a complete CSRF
defense. State-changing endpoints should still use an application-level CSRF
token and/or validate request origin where appropriate. The session package does
not currently implement a CSRF token protocol.
