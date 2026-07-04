# Error Handling

Every API call can fail. rubygems-skills gives you typed, predicate-based error checks so your program can branch on *why* it failed — not just *that* it failed.

## The three predicates

```go
package repository // (exported helpers)

func IsNotFound(err error) bool       // 404 — gem/version/user doesn't exist
func IsRateLimited(err error) bool    // 429 — you're hitting the API too fast
func IsUnauthorized(err error) bool   // 401 — missing/invalid token
```

## Usage

```go
pkg, err := repo.GetPackage(ctx, "some-gem")
switch {
case repository.IsNotFound(err):
    fmt.Println("No such gem — check the name.")
    return
case repository.IsRateLimited(err):
    fmt.Println("Slow down — backing off.")
    time.Sleep(30 * time.Second)
    return
case repository.IsUnauthorized(err):
    fmt.Println("This endpoint needs a token. Set one via Options.SetToken.")
    return
case err != nil:
    log.Fatalf("unexpected error: %v", err)
}
fmt.Println(pkg.Name, pkg.Version)
```

That `switch` is a decision tree — branch on *why* it failed, not just *that* it failed:

```mermaid
flowchart TD
    Err(["err returned from a call"]) --> N{"IsNotFound(err)?\n(HTTP 404)"}
    N -->|yes| NotFound["gem/version/user doesn't exist\n→ skip, warn, or fallback"]
    N -->|no| R{"IsRateLimited(err)?\n(HTTP 429)"}
    R -->|yes| Rate["slow down\n→ app-level backoff / queue"]
    R -->|no| U{"IsUnauthorized(err)?\n(HTTP 401)"}
    U -->|yes| Auth["missing/bad token\n→ prompt for RUBYGEMS_API_KEY"}
    U -->|no| Other{"err != nil?"}
    Other -->|yes| Transient["5xx / network / unknown\n→ treat as transient, retry later"]
    Other -->|no| OK(["nil — success, use the result"])

    classDef hit fill:#16a34a22,stroke:#16a34a,color:#fff
    classDef miss fill:#cc342d22,stroke:#cc342d,color:#fff
    class OK hit
    class NotFound,Rate,Auth,Transient miss
```

## The underlying error type

When a request fails, you receive a `*APIError` (in `pkg/repository/errors.go`):

```go
type APIError struct {
    Cause      error  // the underlying error, if any
    StatusCode int    // HTTP status (404, 429, 500, ...)
    URL        string // request URL that failed
    Response   string // raw response body (as a string)
}
```

Its `Error()` string is `API error (status: <code>, url: <url>): <cause>`. The predicates above inspect `StatusCode`:

- `IsNotFound` → `StatusCode == 404`
- `IsRateLimited` → `StatusCode == 429`
- `IsUnauthorized` → `StatusCode == 401`

For other status codes (e.g. 500), none of the predicates match — treat it as a generic server error. There are also sentinel errors (`ErrNotFound`, `ErrRateLimited`, `ErrUnauthorized`, `ErrServerError`, `ErrTimeout`, `ErrNetworkFailure`, `ErrInvalidRequest`) usable with `errors.Is` — see the [Errors API reference](../api/errors).

## Inspecting details

If you need the full response body (e.g. the API's error message):

```go
pkg, err := repo.GetPackage(ctx, "some-gem")
var apiErr *repository.APIError
if errors.As(err, &apiErr) {
    fmt.Println("status:", apiErr.StatusCode)
    fmt.Println("url:", apiErr.URL)
    fmt.Println("body:", apiErr.Response)
}
```

## Retry interaction

The retry layer retries **any** error by default (`ShouldRetry = err != nil`), so 429, 5xx, 404, 401, and network errors are all retried up to `MaxAttempts` (see [Retry & Backoff](./retry)). By the time an error reaches your code, all retry attempts have already been exhausted — so `IsRateLimited` here means "rate-limited *after* the retry budget ran out." The right response is usually to back off at the application level (longer sleep, queue the work, alert an operator), not to retry again immediately.

Because 404/401 are also retried by default, a missing gem or bad token will burn the full retry budget before surfacing. If that's not what you want, supply a `ShouldRetry` predicate that excludes them (see the warning in [Retry & Backoff](./retry)).

## Network errors

A non-HTTP failure (DNS, TLS, connection refused) surfaces as a plain `error` wrapping the underlying cause. None of the three predicates match it. Treat it as a transient network issue — often retryable, depending on your retry predicate.

```go
pkg, err := repo.GetPackage(ctx, "rails")
if err != nil && !repository.IsNotFound(err) && !repository.IsUnauthorized(err) {
    // likely transient (rate-limited-after-retries, 5xx-after-retries, or network)
    log.Printf("transient failure, will retry later: %v", err)
}
```

---

That wraps the Features section. Next, head to the [AI Agent Integration guide](../ai-agents/overview) — the part this SDK was built for.
