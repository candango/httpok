# Session compatibility analysis

## Scope

This document records the session hardening delivered for the `httpok` v0.3.0
release and the compatibility boundaries that remain. Firenado is the internal
behavioral reference for shared server-side sessions, while Tornado is the
external format reference for signed cookies.

## v0.3.0 release baseline

The session package now provides:

- `Engine` and `Store` abstractions;
- memory and file-backed stores;
- JSON encoding;
- sliding server-side expiration through `Touch`;
- periodic purge through `schedulerok` for stores that require it;
- Tornado-compatible signed session ID cookies;
- secure cookie attributes and versioned signing-key rotation;
- changed-only persistence and explicit session destruction;
- FileStore namespace isolation and strict session ID validation.

The client stores only an opaque session ID. Session data remains in the
configured server-side store.

## Closed compatibility gaps

### Signed session ID cookies

When a cookie secret is configured, `httpok` emits Tornado version 2 signed
values using HMAC-SHA256. It accepts valid Tornado version 1 and version 2
values, validates timestamps and signatures, and replaces invalid or unknown
session identifiers.

Versioned key maps support planned signing-key rotation. The legacy
`CookieSecret` configuration remains available as key version 0.

### Secure cookie attributes

Session cookies default to `HttpOnly=true` and `SameSite=Lax`. Applications
must explicitly enable `Secure` when deployed over HTTPS because the framework
cannot infer the external transport used by a reverse proxy.

### FileStore isolation

FileStore applies a validated filename prefix and accepts session IDs from a
strict character allowlist. Reads, writes, touches, and deletes reject unsafe
IDs. Purge operates only on regular files in the configured namespace, and
session directories and files use private permissions.

### Changed-only persistence

The middleware writes session data only when `Session.Changed` is true. Reads
still call `Touch`, allowing stores to refresh expiration without rewriting an
unchanged payload. Store implementations must refresh expiration as part of
`Set` as well.

### Session destruction

`Session.Destroy` marks the session as destroyed. After the handler returns,
the middleware calls `Engine.DeleteSession` instead of saving the cleared
payload. Deletion must invalidate subsequent reads immediately, even when a
backend defers physical cleanup.

The old cookie may remain in a response whose headers were already written.
The missing server-side ID causes the next request to receive a new session.

### Scheduler ownership

`schedulerok` owns purge scheduling, cancellation, overlap policy, and graceful
shutdown. Stores with native expiration report `RequiresPurge=false`; stores
with manual expiration report true and receive scheduled purge calls.
Backoff primitives belong to `intervalok`, not to the `httpok` repository.

## Remaining work

### Cross-stack integration coverage

Task #31 tracks executable Tornado/Go integration tests. Unit tests already
cover captured version 1 and version 2 vectors, malformed values, timestamp
validation, key rotation, middleware reuse, and tampering. The remaining test
must prove live encoding and decoding across both runtimes.

### Configurable ID generation

`StoreEngine.NewId` still uses `security.RandomString` directly. A configurable
ID generator remains a possible extension for applications that require a
specific compatible identifier format. The secure random default must remain.

### Optional Redis integration

Redis is intentionally excluded from the core module. The separate
`github.com/candango/httpok-redis` module implements `session.Store`, owns the
Redis client dependency, and may provide a compatibility preset for the legacy
PHP/Firenado contract:

```text
cookie: PHPSESSID
Redis database: 1
key: GLOBAL_SESSION:<PHPSESSID>
payload: JSON
```

Applications that do not use Redis therefore do not acquire Redis code or
transitive runtime dependencies through `httpok`.

## Release validation

Session changes are validated with:

```bash
go test ./...
go vet ./...
go test -race ./...
```

The cross-stack integration test remains an explicit release-confidence item
until task #31 is completed.

## Licensing and provenance

Firenado is an internal Candango implementation reference. Tornado's source is
distributed under the Apache License 2.0 and is used only as a behavioral and
wire-format reference. `httpok` remains distributed under the MIT License.
