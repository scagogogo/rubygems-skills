# Architecture

A deep dive into how rubygems-skills is structured end to end — the package layout, the request pipeline, the auth model, and the concurrency design. Read this once and you'll know where every feature lives.

## Package layout

```mermaid
flowchart TB
    subgraph RepoRoot["rubygems-skills"]
        Cmd["cmd/rubygems<br/>cobra CLI"]
        Examples["examples/<br/>basic_usage · bulk · cache"]
        Pkg["pkg/"]
        Tests["tests/<br/>unit · integration · mirrors"]
    end

    subgraph PkgSub["pkg/ subpackages"]
        Repository["repository/<br/>client · write · mirrors<br/>options · errors · retry<br/>bulk_operations<br/>cached_repository"]
        Models["models/<br/>PackageInformation · Version<br/>VersionDetail · APIKey<br/>Webhook · Owner · UserProfile<br/>MFAStatus · Attestation · ..."]
        Cache["cache/<br/>Cache interface<br/>MemoryCache (TTL)"]
        Install["install/<br/>platform detection<br/>apt/yum/dnf/apk/pacman/<br/>brew/choco/scoop/zypper"]
    end

    Pkg --> Repository
    Pkg --> Models
    Pkg --> Cache
    Pkg --> Install
    Repository --> Models
    Repository --> Cache
    Repository --> Retry["retry.go<br/>SendRequestWithRetry"]
    Cmd --> Repository
    Cmd --> Install
    Examples --> Repository

    classDef root fill:#cc342d22,stroke:#cc342d,color:#fff
    classDef pkg fill:#7c3aed22,stroke:#7c3aed,color:#fff
    classDef leaf fill:#0ea5e922,stroke:#0ea5e9,color:#fff
    class Cmd,Examples,Pkg,Tests root
    class Repository,Models,Cache,Install pkg
    class Retry leaf
```

Four packages, one responsibility each: `repository` is the HTTP client, `models` holds typed JSON structs, `cache` is the pluggable TTL store, `install` is the cross-platform Ruby provisioner. The CLI and examples are thin callers over `repository` (and `install`).

## The request pipeline

Every read call flows through the same five stages. Write calls add auth and form/multipart encoding but otherwise share the pipeline.

```mermaid
flowchart LR
    Call["1. typed call<br/>repo.GetPackage(ctx, gem)"] --> Build["2. build URL<br/>PathEscape / QueryEscape"]
    Build --> Auth["3. apply options<br/>token · proxy · retry"]
    Auth --> Send["4. HTTP GET<br/>via go-requests"]
    Send --> Handle["5. response handler<br/>2xx → bytes<br/>non-2xx → APIError"]
    Handle --> Decode["6. getJson[T]<br/>json.Unmarshal → T"]
    Decode --> Return["return (T, nil)"]

    Handle -.->|"429 / 5xx / net err"| RetryCheck{"retry enabled?"}
    RetryCheck -->|"yes"| Backoff["backoff wait<br/>waitTime * 2^(n-1)"]
    Backoff --> Send
    RetryCheck -->|"no / exhausted"| Err["return APIError"]

    classDef step fill:#0ea5e922,stroke:#0ea5e9,color:#fff
    classDef err fill:#ef444422,stroke:#ef4444,color:#fff
    classDef retry fill:#f59e0b22,stroke:#f59e0b,color:#fff
    class Call,Build,Auth,Send,Handle,Decode,Return step
    class Err err
    class RetryCheck,Backoff retry
```

Two things to notice: (1) **URL encoding is centralized** — every path segment goes through `url.PathEscape`, every query value through `url.QueryEscape`, so special characters never break a request. (2) **Retry is opt-in per client**, configured once via `Options.SetRetryOptions` and applied to every request that client makes (GET, POST, DELETE, form, multipart).

## Authentication model

The SDK uses two auth strategies, chosen by endpoint. Read endpoints take an optional token to raise the rate-limit quota; write endpoints split between a bearer token and HTTP Basic.

```mermaid
flowchart TD
    Op{"operation?"} -->|"read (most)"| ReadToken{"token set?"}
    Op -->|"push / yank / owners / webhooks"| Bearer["Authorization: Bearer {token}"]
    Op -->|"api_key CRUD / my-profile"| Basic["Authorization: Basic\nbase64(user:pass)"]

    ReadToken -->|"yes"| Quota["raises rate-limit quota"]
    ReadToken -->|"no"| Anon["anonymous<br/>(lower quota)"]
    Quota --> Bearer
    Anon --> Plain["plain GET"]

    Bearer --> Req["HTTP request"]
    Basic --> Req
    Plain --> Req

    Op -->|"GetOwnedGems / GetMFAStatus"| Required["token REQUIRED<br/>else 401"]

    classDef op fill:#7c3aed22,stroke:#7c3aed,color:#fff
    classDef auth fill:#0ea5e922,stroke:#0ea5e9,color:#fff
    classDef warn fill:#ef444422,stroke:#ef4444,color:#fff
    class Op op
    class Bearer,Basic,Quota,Anon,Plain auth
    class Required warn
```

**Why two strategies?** RubyGems.org exposes API-key management and the full authenticated profile behind HTTP Basic Auth (username + password), while gem publishing, yanking, owner, and webhook operations accept a bearer API token. The SDK mirrors this exactly: `WriteRepository` carries the bearer token in its options, and the `*APIKey*` / `GetMyProfile` methods take `username, password` arguments and apply Basic Auth per-request.

## Cache decorator

`CachedRepository` wraps any `Repository` and short-circuits repeat reads. Same interface, drop-in.

```mermaid
sequenceDiagram
    participant Caller
    participant Cached as CachedRepository
    participant Cache as MemoryCache (TTL)
    participant Inner as Repository (upstream)

    Caller->>Cached: GetPackage(ctx, "rails")
    Cached->>Cache: get("rails")
    alt cache hit (not expired)
        Cache-->>Cached: *PackageInformation
        Cached-->>Caller: cached value (no HTTP)
    else cache miss
        Cache-->>Cached: nil
        Cached->>Inner: GetPackage(ctx, "rails")
        Inner-->>Cached: *PackageInformation
        Cached->>Cache: set("rails", value, ttl)
        Cached-->>Caller: fresh value
    end
```

The cache is keyed by method + arguments, with a per-entry TTL and a background cleanup goroutine. Pass any `cache.Cache` implementation — `MemoryCache` is the in-process default; you can back it with Redis, filesystem, or anything else by implementing the interface.

## Retry & backoff

When `RetryOptions` is set, transient failures retry with exponential backoff before surfacing.

```mermaid
sequenceDiagram
    participant Caller
    participant Retry as Retry layer
    participant HTTP
    participant Upstream as rubygems.org

    Caller->>Retry: GetPackage(ctx, "rails")
    Retry->>HTTP: attempt 1
    HTTP->>Upstream: GET /api/v1/gems/rails.json
    Upstream-->>HTTP: 429 Too Many Requests
    HTTP-->>Retry: APIError (429)
    Retry->>Retry: ShouldRetry(err)? yes
    Note right of Retry: wait 1s (waitTime * 2^0)
    Retry->>HTTP: attempt 2
    HTTP->>Upstream: GET /api/v1/gems/rails.json
    Upstream-->>HTTP: 200 OK
    HTTP-->>Retry: bytes
    Retry-->>Caller: *PackageInformation, nil
```

Defaults: 3 attempts, 1s initial wait, exponential backoff (`waitTime * 2^(attempt-1)`, capped at 30s), retry on any error. Tune with `NewDefaultRetryOptions().WithMaxAttempts(n).WithWaitTime(d)`. The same retry layer covers GET, POST, DELETE, form, and multipart requests — so `PushGem` retries too.

## Bulk concurrency

`BulkGet*` methods fan out over a fixed-size worker pool. Each worker owns a distinct result slot, so the pool is lock-free on the result slice.

```mermaid
flowchart TB
    Input["names[0..N-1]"] --> Dispatcher["dispatcher\nwrites indices 0..N-1\nto a buffered channel"]
    Dispatcher --> Ch(("index channel"))
    Ch --> W0["worker 0\n→ results[0]"])
    Ch --> W1["worker 1\n→ results[1]"])
    Ch --> W2["worker 2\n→ results[2]"])
    Ch --> WN["worker N-1\n→ results[N-1]"])
    W0 & W1 & W2 & WN --> Gather["pre-sized results[]\norder matches input"]

    W0 -.->|"429?"| Retry["per-request retry\n(if enabled)"]
    W0 -.->|"fail"| Slot["results[i].Error = err\n(others continue)"]

    classDef io fill:#0ea5e922,stroke:#0ea5e9,color:#fff
    classDef worker fill:#7c3aed22,stroke:#7c3aed,color:#fff
    classDef alt fill:#f59e0b22,stroke:#f59e0b,color:#fff
    class Input,Dispatcher,Ch,Gather io
    class W0,W1,W2,WN worker
    class Retry,Slot alt
```

Because the dispatcher hands out **indices** (not names) and each worker writes only to its own `results[i]`, there's no data race on the result slice — no mutex, no copy. `ContinueOnError(false)` stops dispatching after the first failure; the default `true` runs every item and collects per-slot errors. See [Bulk Operations](./bulk-operations).

## End-to-end from an AI agent's view

```mermaid
flowchart LR
    Agent["AI agent<br/>(Claude Code / Codex)"] --> Decide{"needs Ruby runtime?"}
    Decide -->|"no, just data"| SDK["go get SDK<br/>NewRepository()"]
    Decide -->|"yes, run gem/ruby"| Inst["install.NewInstaller()<br/>provision Ruby"]
    Inst --> SDK
    SDK --> Pick{"what data?"}
    Pick -->|"one gem"| Get["GetPackage"]
    Pick -->|"search"| Search["Search"]
    Pick -->|"versions / detail"| Ver["GetGemVersions / GetGemVersionDetail"]
    Pick -->|"dependents"| RDep["GetReverseDependencies"]
    Pick -->|"many gems"| Bulk["BulkGetPackages"]
    Pick -->|"publish / manage"| Write["WriteRepository (token)"]
    Get & Search & Ver & RDep & Bulk & Write --> Out["typed Go structs<br/>→ agent reasons & acts"]

    classDef agent fill:#7c3aed22,stroke:#7c3aed,color:#fff
    classDef sdk fill:#0ea5e922,stroke:#0ea5e9,color:#fff
    classDef out fill:#10b98122,stroke:#10b981,color:#fff
    class Agent,Decide agent
    class SDK,Inst,Pick,Get,Search,Ver,RDep,Bulk,Write sdk
    class Out out
```

The agent never touches HTTP, JSON, or URL encoding — it calls typed methods and reasons over typed results. When it needs the actual `gem`/`ruby` binaries (e.g. to build a `.gem` before `PushGem`), the installer provisions them first.

---

← Back: [How it works](./how-it-works) · Next: [Installation](./installation)
