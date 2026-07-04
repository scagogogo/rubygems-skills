# How it works

rubygems-skills is a thin, typed layer over the RubyGems.org HTTP API. This page explains the architecture so you (and your AI agent) understand exactly what happens when you call a method.

## The layers

```mermaid
flowchart TD
    subgraph Caller["Your Go program / AI agent"]
        Call["repo.GetPackage(ctx, \"rails\")"]
    end
    subgraph SDK["pkg/repository"]
        direction TB
        Iface["Repository interface — read endpoints (no auth)\nWriteRepository interface — write endpoints (auth)"]
        Generic["getJson[T] / getBytes — one generic HTTP path\nshared by every method"]
        Http["HTTP client + RetryOptions (exponential backoff)\noptional: BasicAuth token · Proxy · custom ServerURL"]
        Iface --> Generic --> Http
    end
    Call -->|"typed call"| Iface
    Http -->|"HTTPS"| Upstream["RubyGems.org\n(or mirror: ruby-china / tsinghua / aliyun)"]

    classDef caller fill:#cc342d22,stroke:#cc342d,color:#fff
    classDef sdk fill:#7c3aed22,stroke:#7c3aed,color:#fff
    classDef net fill:#0ea5e922,stroke:#0ea5e9,color:#fff
    class Call caller
    class Iface,Generic sdk
    class Http,Upstream net
```

Three layers, top to bottom:

1. **Your code** calls a typed method — no URLs, no JSON.
2. **The SDK** turns that call into an HTTP request through one shared generic path (`getJson[T]`), with retry/backoff, auth, and proxy applied centrally.
3. **The upstream** is RubyGems.org or a mirror — the SDK doesn't care which, it's just a `ServerURL`.

## The Repository interface

The core read surface is the `Repository` interface in `pkg/repository/repository.go`. Every method maps 1:1 to a RubyGems.org endpoint:

| Method | Endpoint | Returns |
|---|---|---|
| `GetPackage` | `/api/v1/gems/{name}.json` | `*models.PackageInformation` |
| `Search` | `/api/v1/search.json?query=...` | `[]*models.PackageInformation` |
| `GetGemVersions` | `/api/v1/versions/{name}.json` | `[]*models.Version` |
| `GetDependencies` | `/api/v1/dependencies?gems=...` | `[]*models.DependencyInfo` |
| `TopDownloads` | `/api/v1/downloads/all.json` | `[]*models.TopDownloadedGem` |
| `GetUserProfile` | `/api/v1/profiles/{handle}.json` | `*models.UserProfile` |
| `GetGemOwners` | `/api/v1/gems/{name}/owners.json` | `[]*models.Owner` |
| `GetAttestations` | `/api/v1/attestations/{gem}-{version}.json` | `[]*models.Attestation` |
| ... | ... | ... |

The full list is in the [API Reference](../api/repository).

## Generics: `getJson[T]`

Instead of repeating HTTP-get-then-unmarshal in every method, the SDK uses a single generic helper:

```go
func getJson[T any](ctx context.Context, repository *RepositoryImpl, targetUrl string) (T, error)
```

Each method calls `getJson[*models.PackageInformation](ctx, repo, url)` — one code path for HTTP, retry, auth, and proxy; the type parameter handles the JSON target. This is why the SDK requires **Go 1.21+** (generics landed in 1.18; the SDK pins 1.21 for the broader toolchain).

**Why a generic helper, not per-method HTTP code?** Without generics, every endpoint would either (a) duplicate the HTTP-get → unmarshal → error-wrap boilerplate (~30 methods × ~15 lines = a lot of drift-prone copies), or (b) return `interface{}` and force every caller into a type assertion. `getJson[T]` gives option (a)'s type safety with option (b)'s single code path: the HTTP/retry/auth logic lives in exactly one place, and each method gets a compile-time-typed return value with zero assertions. `runWorkerPool[T]` (in [Bulk Operations](./bulk-operations)) applies the same idea to the worker pool.

## Retry with backoff

Every request flows through `RetryOptions`. By default:

- Up to **3 attempts**.
- **Exponential backoff** starting at 1s, capped at 30s (`waitTime * 2^(attempt-1)`).
- Retries on **any** error — the default `ShouldRetry` is `err != nil`. Non-2xx HTTP status codes (429, 5xx, 404, 401, …) are converted to errors by a response handler before the retry decision, so they all trigger a retry unless you override `ShouldRetry`.

Configure via:

```go
retry := repository.NewDefaultRetryOptions().
    WithMaxAttempts(6).
    WithWaitTime(200 * time.Millisecond).
    WithMaxWaitTime(5 * time.Second)

opts := repository.NewOptions().
    SetRetryOptions(retry)

repo := repository.NewRepository(opts)
```

See [Retry & Backoff](./retry).

## The cache decorator

`CachedRepository` wraps any `Repository` and memoizes results with a TTL:

```go
repo := repository.NewRubyChinaRepository()
cached := repository.NewCachedRepository(repo, 10*time.Minute, nil)
```

Same interface — but repeat calls hit the in-memory cache instead of the network. Drop-in, no code changes elsewhere. See [Caching](./caching).

## Write operations

`WriteRepository` covers the authenticated endpoints — push, yank, owners, webhooks, API keys. Gem/owner/webhook ops send the token as an `Authorization` header; API-key & profile ops use HTTP Basic Auth with username + password. Gem pushes use `multipart/form-data`. See [WriteRepository](../api/write-repository) for the auth breakdown.

## Auto-install layer

Separately, `pkg/install` can provision Ruby/RubyGems on the host machine — useful when your workflow needs to actually run `gem`/`ruby`, not just call the HTTP API. It detects the OS and dispatches to the right package manager. See [Auto-Install](../auto-install/overview).

---

Next: [Installation](./installation).
