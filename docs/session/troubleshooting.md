# Session Troubleshooting

## The browser receives a new cookie on every request

Check:

- whether the cookie name matches `EngineProperties.Name`;
- whether `CookieSecret` or `CookieSecrets` changed between requests;
- whether the active key version exists in the configured map;
- whether the session store is running and can find the ID;
- whether `CookieMaxAge` and the signed timestamp are valid.

## A signed cookie is rejected

Check the secret, cookie name, key version, timestamp, base64 value, and HMAC.
Do not disable signing to hide the error. Use a test vector or a controlled
local secret to isolate configuration from protocol problems.

## The cookie is missing in JavaScript

That is expected when `HttpOnly=true`. Application code should not use the
session cookie as a JavaScript data channel.

## The cookie is not sent over HTTPS

Check `CookieOptions.Secure` and the deployment's TLS termination model. If TLS
terminates at a reverse proxy, ensure the application configuration still marks
session cookies as secure.

## Cross-site requests fail

Check `SameSite`. `Strict` can block legitimate navigation flows, `Lax` is a
common default, and `None` requires HTTPS-capable secure cookies in modern
browsers.

## Session data disappears after a restart

`MemoryStore` is process-local. Use a persistent or shared store for deployments
that require restart or multi-process survival.

## File sessions behave unexpectedly

Check the configured directory, regular-file permissions, namespace, and session
ID validation. Do not manually create arbitrary files in the session directory;
use store APIs and the FileStore tests to reproduce behavior.
