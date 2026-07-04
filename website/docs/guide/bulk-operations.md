# Bulk Operations

Fetching one gem at a time is slow when you need dozens or hundreds. rubygems-skills ships a concurrent worker pool for bulk reads — `BulkGet*` methods fan out in parallel and collect typed results.

## The bulk methods

| Method | Returns per gem |
|---|---|
| `BulkGetPackages(ctx, names, opts)` | `*models.PackageInformation` |
| `BulkGetVersions(ctx, names, opts)` | `[]*models.Version` |
| `BulkGetDependencies(ctx, names, opts)` | `[]*models.DependencyInfo` |
| `BulkGetReverseDependencies(ctx, names, opts)` | `[]string` |

Each returns a `[]*BulkResult[T]` aligned to your input slice — index `i` of the result corresponds to `names[i]`.

## Quick start

```go
repo := repository.NewRepository()

names := []string{"rails", "puma", "sidekiq", "redis", "rake"}

results := repo.BulkGetPackages(ctx, names, nil) // nil = default options
for i, r := range results {
    if r.Error != nil {
        fmt.Printf("%s: error — %v\n", names[i], r.Error)
        continue
    }
    fmt.Printf("%s: %s (%d downloads)\n", r.Value.Name, r.Value.Version, r.Value.Downloads)
}
```

## Options

```go
opts := repository.NewBulkOptions().
    WithMaxConcurrency(8).       // parallel workers
    WithContinueOnError(true)    // keep going if one gem fails

results := repo.BulkGetPackages(ctx, names, opts)
```

| Method | Default | Purpose |
|---|---|---|
| `WithMaxConcurrency(n)` | 10 | Number of in-flight requests. |
| `WithContinueOnError(bool)` | true | If false, the pool stops on the first error. |

## The result type

```go
type BulkResult[T any] struct {
    Key   string // the request key, usually the gem name
    Value T      // operation result
    Error error  // possible error during the operation
}
```

Because errors are per-item rather than a single fatal error, one missing gem doesn't sink the whole batch. Always check `r.Error` before using `r.Value`; `r.Key` lets you correlate a result back to its input name without relying on slice index.

## Concurrency limits

RubyGems.org rate-limits aggressively. Set `MaxConcurrency` conservatively:

- **Unauthenticated:** 2–5 workers. Higher risks 429s.
- **Authenticated (token):** 5–10 workers is usually safe.
- **With retry enabled:** you can be a bit more aggressive — transient 429s will be retried automatically.

If you see frequent `IsRateLimited` errors, lower `MaxConcurrency` or add a token via `Options.SetToken`.

## With caching

Bulk methods bypass the `CachedRepository` decorator and hit the underlying repo directly (the per-item concurrency makes the cache layer redundant for bulk reads). For single-gem repeat reads, use the wrapped methods on `CachedRepository` instead.

## Implementation note

The pool is generic — `runWorkerPool[T]` — so all four bulk methods share one concurrency implementation.

```mermaid
flowchart LR
    subgraph In["input: gemNames[]"]
        N0["names[0]"]
        N1["names[1]"]
        N2["names[2]")
        NDots["..."]
    end
    Ch(("index channel\n0,1,2,...")):::chan
    subgraph Workers["N workers (MaxConcurrency)"]
        W0["worker 0"]
        W1["worker 1"]
        W2["worker 2"]
    end
    subgraph Out["pre-sized results[] (no locks)"]
        R0["results[0] = BulkResult{T}"]
        R1["results[1] = BulkResult{T}"]
        R2["results[2] = BulkResult{T}"]
    end
    In --> Ch
    Ch --> W0 & W1 & W2
    W0 --> R0
    W1 --> R1
    W2 --> R2

    classDef chan fill:#0ea5e922,stroke:#0ea5e9,color:#fff
```

How it stays correct without locks:

- The dispatcher sends **indices** (not gem names) onto a buffered channel — each index `i` corresponds to `names[i]`.
- Each worker pulls an index, calls the underlying method for `names[i]`, and writes its `BulkResult[T]` **only into `results[i]`**. Because every write targets a distinct slice slot, there's no data race and no mutex needed on the result slice.
- The result slice is pre-sized to `len(names)`, so order matches the input exactly — `results[i]` always answers `names[i]`.

With `ContinueOnError(false)`, the pool stops dispatching new indices after the first error (workers finish their in-flight item, then exit) — so a single failure short-circuits the rest of the batch. With the default `true`, every item runs regardless, and failures land in their slot's `Error` field.

---

Next: [Error Handling](./error-handling).
