# Session Testing

## Baseline checks

Run the repository checks after session changes:

```text
gofmt -w <touched Go files>
go test ./...
go vet ./...
go test -race ./security ./middleware ./session
```

Use `go doc -all ./session` and `go doc -all ./middleware` when changing public
configuration or middleware behavior.

## Required behavior coverage

Tests should cover:

- a missing cookie creates one session;
- a valid signed cookie reuses its session;
- v1 and v2 Tornado vectors decode correctly;
- rotation accepts old configured keys and signs with the active key;
- invalid, expired, or unknown cookies receive a new session;
- default and explicit cookie attributes are emitted;
- changed sessions persist;
- unchanged sessions are not rewritten;
- destroyed sessions are deleted;
- FileStore IDs cannot escape the storage directory;
- purge ignores unrelated files;
- concurrent store access is safe.

## Cross-stack E2E

Task #31 covers Python/Tornado and Go interoperability. The integration test
in `middleware/session_crossstack_test.go`:

1. starts a Go test server on loopback and an ephemeral port;
2. seeds a known server-side session;
3. generates Python/Tornado v1 and v2 cookies and sends them to Go;
4. decodes a Go-generated v2 cookie with Tornado;
5. exercises tampering, expiration, and unknown IDs;
6. cleans up the server and session engine on every result.

The helper uses the pinned dependency in `testdata/requirements.txt`. Run the
cross-stack test directly with:

```bash
python3 -m pip install --requirement testdata/requirements.txt
go test ./middleware -run 'TestTornado'
```

Secrets used by the test are fixed test-only values and must never be reused
in deployments.
