# Session Lifecycle

## Request flow

`middleware.Sessioned` performs the following operations:

1. Reads the configured cookie.
2. Verifies the signed value when signing is enabled.
3. Checks whether the decoded session ID exists in the store.
4. Creates a new ID when the cookie is missing, invalid, expired, or unknown.
5. Loads the session and places it in the request context.
6. Runs the downstream handler.
7. Persists or destroys the session after the handler returns.

The session is available through `session.SessionFromContext`.

## Session state

A `Session` has these relevant state fields:

- `Changed`: data was modified by `Set`, `Delete`, or `Clear`;
- `Destroyed`: the session must no longer be persisted;
- `Data`: server-side session values;
- `Id`: the opaque storage and cookie identifier.

`Get`, `Has`, `Set`, and `Delete` reject normal operations after destruction.

## Mutation

Use the session methods instead of mutating `Data` without marking the session:

```go
if err := sess.Set("role", "admin"); err != nil {
    return err
}
```

`Set`, `Delete`, and `Clear` mark the session as changed. Direct map mutation
can bypass that signal and should be avoided.

## Save contract

The intended lifecycle contract is:

```text
Changed=false, Destroyed=false -> do not rewrite session data
Changed=true,  Destroyed=false -> encode and persist session data
Destroyed=true                 -> delete the server-side session
```

The changed-session persistence and destruction work is tracked by task #30.
Until that task is complete, callers must treat persistence behavior as an
active implementation boundary rather than depending on write suppression.

## Destruction

`Destroy` clears data and marks the session destroyed. The target lifecycle is:

```text
Destroy -> delete from Store -> old cookie becomes unknown -> next request gets a new ID
```

A response that has already written its headers cannot reliably add a replacement
cookie after the handler returns. The server-side deletion is therefore the
security boundary; the next request renews the client identifier.

## Expiration

The server-side `AgeLimit` is a sliding idle lifetime. Reads touch the store,
and store writes must also refresh the store TTL. Client-side `CookieMaxAge` is
a separate browser lifetime and does not replace server-side expiration.
