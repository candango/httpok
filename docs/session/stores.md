# Session Stores

The `Store` interface owns server-side persistence. An engine delegates storage
operations to a store implementation.

## Store contract

| Method | Contract |
|---|---|
| `Start` | Prepare storage resources |
| `Stop` | Release resources |
| `Exists` | Check whether an ID is present |
| `Get` / `GetString` | Read stored data |
| `Set` / `SetString` | Write data and refresh TTL |
| `Touch` | Refresh TTL without changing data |
| `Delete` | Invalidate one session immediately; physical cleanup may be deferred |
| `Purge` | Remove expired entries |
| `RequiresPurge` | Tell the engine whether scheduled purge is needed |

A store must treat IDs as untrusted input. Path-backed stores must validate IDs
and keep all derived paths inside their configured directory.

## MemoryStore

`MemoryStore` stores encoded data in memory and tracks expiration with a
`LastTouched` timestamp. It is useful for tests and single-process deployments.
It does not provide cross-process persistence.

## FileStore

`FileStore` stores one encoded session per file and uses file modification time
for sliding expiration. The store uses a per-instance mutex for file access and
runs scheduled purge through the engine.

The FileStore hardening implemented for task #32 provides:

- a configurable filename prefix, defaulting to `httpok_`;
- a strict allowlist for session IDs;
- regular-file checks;
- purge limited to namespaced session files;
- idempotent deletion;
- private `0700` storage directories and `0600` session files.

Do not use a raw user-controlled string as a filename.

## Storage encoding

The engine encodes `Session.Data` through the configured `Encoder`. The default
`JsonEncoder` stores JSON bytes. Stores should remain agnostic about the data
format and treat the value as opaque bytes.

## Custom stores

A custom store must implement every method in `Store`, preserve the `Set`
refresh-TTL contract, and define safe behavior for missing IDs. If the backend
has native expiration, `RequiresPurge` can return false.
