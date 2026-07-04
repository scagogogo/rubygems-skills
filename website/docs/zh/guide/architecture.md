# 架构

深入剖析 rubygems-skills 的端到端结构——包布局、请求管线、认证模型与并发设计。读一遍，你就知道每个特性住在哪里。

## 包布局

```mermaid
flowchart TB
    subgraph RepoRoot["rubygems-skills"]
        Cmd["cmd/rubygems<br/>cobra CLI"]
        Examples["examples/<br/>basic_usage · bulk · cache"]
        Pkg["pkg/"]
        Tests["tests/<br/>unit · integration · mirrors"]
    end

    subgraph PkgSub["pkg/ 子包"]
        Repository["repository/<br/>client · write · mirrors<br/>options · errors · retry<br/>bulk_operations<br/>cached_repository"]
        Models["models/<br/>PackageInformation · Version<br/>VersionDetail · APIKey<br/>Webhook · Owner · UserProfile<br/>MFAStatus · Attestation · ..."]
        Cache["cache/<br/>Cache 接口<br/>MemoryCache (TTL)"]
        Install["install/<br/>平台探测<br/>apt/yum/dnf/apk/pacman/<br/>brew/choco/scoop/zypper"]
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

四个包各司其职：`repository` 是 HTTP 客户端，`models` 存放类型化的 JSON 结构体，`cache` 是可插拔的 TTL 存储，`install` 是跨平台 Ruby 安装器。CLI 和 examples 都是 `repository`（及 `install`）的薄封装。

## 请求管线

每次读调用都经过相同的五个阶段。写调用在此基础上增加认证和 form/multipart 编码，其余共享同一管线。

```mermaid
flowchart LR
    Call["1. 类型化调用<br/>repo.GetPackage(ctx, gem)"] --> Build["2. 构造 URL<br/>PathEscape / QueryEscape"]
    Build --> Auth["3. 应用 options<br/>token · proxy · retry"]
    Auth --> Send["4. HTTP GET<br/>经 go-requests"]
    Send --> Handle["5. 响应处理<br/>2xx → bytes<br/>非 2xx → APIError"]
    Handle --> Decode["6. getJson[T]<br/>json.Unmarshal → T"]
    Decode --> Return["return (T, nil)"]

    Handle -.->|"429 / 5xx / 网络错误"| RetryCheck{"开启重试?"}
    RetryCheck -->|"是"| Backoff["退避等待<br/>waitTime * 2^(n-1)"]
    Backoff --> Send
    RetryCheck -->|"否 / 用尽"| Err["返回 APIError"]

    classDef step fill:#0ea5e922,stroke:#0ea5e9,color:#fff
    classDef err fill:#ef444422,stroke:#ef4444,color:#fff
    classDef retry fill:#f59e0b22,stroke:#f59e0b,color:#fff
    class Call,Build,Auth,Send,Handle,Decode,Return step
    class Err err
    class RetryCheck,Backoff retry
```

两点注意：(1) **URL 编码集中处理**——每个路径段过 `url.PathEscape`，每个查询值过 `url.QueryEscape`，特殊字符不会破坏请求。(2) **重试按客户端可选**，通过 `Options.SetRetryOptions` 一次配置，对该客户端的每个请求（GET、POST、DELETE、form、multipart）生效。

## 认证模型

SDK 按端点选用两种认证策略。读端点可选传 token 以提升限流配额；写端点在 bearer token 与 HTTP Basic 之间二分。

```mermaid
flowchart TD
    Op{"操作?"} -->|"读（多数）"| ReadToken{"已设 token?"}
    Op -->|"push / yank / owners / webhooks"| Bearer["Authorization: Bearer {token}"]
    Op -->|"api_key 增删改查 / my-profile"| Basic["Authorization: Basic\nbase64(user:pass)"]

    ReadToken -->|"是"| Quota["提升限流配额"]
    ReadToken -->|"否"| Anon["匿名<br/>（更低配额）"]
    Quota --> Bearer
    Anon --> Plain["普通 GET"]

    Bearer --> Req["HTTP 请求"]
    Basic --> Req
    Plain --> Req

    Op -->|"GetOwnedGems / GetMFAStatus"| Required["必须传 token<br/>否则 401"]

    classDef op fill:#7c3aed22,stroke:#7c3aed,color:#fff
    classDef auth fill:#0ea5e922,stroke:#0ea5e9,color:#fff
    classDef warn fill:#ef444422,stroke:#ef4444,color:#fff
    class Op op
    class Bearer,Basic,Quota,Anon,Plain auth
    class Required warn
```

**为何两种策略？** RubyGems.org 把 API Key 管理和完整认证资料放在 HTTP Basic Auth（用户名+密码）之后，而 gem 发布、撤回、所有者、webhook 操作接受 bearer API token。SDK 完全照此实现：`WriteRepository` 在 options 中携带 bearer token，`*APIKey*` / `GetMyProfile` 方法接受 `username, password` 参数并按请求应用 Basic Auth。

## 缓存装饰器

`CachedRepository` 包装任意 `Repository`，对重复读短路。同一接口，直接替换。

```mermaid
sequenceDiagram
    participant Caller as 调用方
    participant Cached as CachedRepository
    participant Cache as MemoryCache (TTL)
    participant Inner as Repository（上游）

    Caller->>Cached: GetPackage(ctx, "rails")
    Cached->>Cache: get("rails")
    alt 缓存命中（未过期）
        Cache-->>Cached: *PackageInformation
        Cached-->>Caller: 缓存值（无 HTTP）
    else 缓存未命中
        Cache-->>Cached: nil
        Cached->>Inner: GetPackage(ctx, "rails")
        Inner-->>Cached: *PackageInformation
        Cached->>Cache: set("rails", value, ttl)
        Cached-->>Caller: 新鲜值
    end
```

缓存以方法 + 参数为键，每项有 TTL，并有后台清理 goroutine。传入任意 `cache.Cache` 实现——`MemoryCache` 是进程内默认实现；你也可以实现该接口，用 Redis、文件系统等做后端。

## 重试与退避

设置了 `RetryOptions` 时，瞬态失败会以指数退避重试后再向上抛出。

```mermaid
sequenceDiagram
    participant Caller as 调用方
    participant Retry as 重试层
    participant HTTP
    participant Upstream as rubygems.org

    Caller->>Retry: GetPackage(ctx, "rails")
    Retry->>HTTP: 第 1 次尝试
    HTTP->>Upstream: GET /api/v1/gems/rails.json
    Upstream-->>HTTP: 429 Too Many Requests
    HTTP-->>Retry: APIError (429)
    Retry->>Retry: ShouldRetry(err)? 是
    Note right of Retry: 等待 1s (waitTime * 2^0)
    Retry->>HTTP: 第 2 次尝试
    HTTP->>Upstream: GET /api/v1/gems/rails.json
    Upstream-->>HTTP: 200 OK
    HTTP-->>Retry: bytes
    Retry-->>Caller: *PackageInformation, nil
```

默认：3 次尝试、1s 初始等待、指数退避（`waitTime * 2^(attempt-1)`，上限 30s）、任意错误都重试。可用 `NewDefaultRetryOptions().WithMaxAttempts(n).WithWaitTime(d)` 调整。同一重试层覆盖 GET、POST、DELETE、form、multipart 请求——`PushGem` 也会重试。

## 批量并发

`BulkGet*` 方法在固定大小的 worker 池上扇出。每个 worker 拥有独立的结果槽位，因此结果切片无需加锁。

```mermaid
flowchart TB
    Input["names[0..N-1]"] --> Dispatcher["分发器\n把索引 0..N-1 写入带缓冲 channel"]
    Dispatcher --> Ch(("索引 channel"))
    Ch --> W0["worker 0\n→ results[0]"])
    Ch --> W1["worker 1\n→ results[1]"])
    Ch --> W2["worker 2\n→ results[2]"])
    Ch --> WN["worker N-1\n→ results[N-1]"])
    W0 & W1 & W2 & WN --> Gather["预分配 results[]\n顺序与输入一致"]

    W0 -.->|"429?"| Retry["按请求重试\n（若开启）"]
    W0 -.->|"失败"| Slot["results[i].Error = err\n（其他继续）"]

    classDef io fill:#0ea5e922,stroke:#0ea5e9,color:#fff
    classDef worker fill:#7c3aed22,stroke:#7c3aed,color:#fff
    classDef alt fill:#f59e0b22,stroke:#f59e0b,color:#fff
    class Input,Dispatcher,Ch,Gather io
    class W0,W1,W2,WN worker
    class Retry,Slot alt
```

由于分发器派发的是**索引**（而非名字），每个 worker 只写自己的 `results[i]`，结果切片上没有数据竞争——无需互斥锁，无需拷贝。`ContinueOnError(false)` 在首次失败后停止派发；默认 `true` 则跑完所有项并收集各槽位的错误。详见[批量操作](./bulk-operations)。

## AI Agent 视角的端到端流程

```mermaid
flowchart LR
    Agent["AI Agent<br/>(Claude Code / Codex)"] --> Decide{"需要 Ruby 运行时?"}
    Decide -->|"否，只要数据"| SDK["go get SDK<br/>NewRepository()"]
    Decide -->|"是，要跑 gem/ruby"| Inst["install.NewInstaller()<br/>安装 Ruby"]
    Inst --> SDK
    SDK --> Pick{"要什么数据?"}
    Pick -->|"单个 gem"| Get["GetPackage"]
    Pick -->|"搜索"| Search["Search"]
    Pick -->|"版本 / 详情"| Ver["GetGemVersions / GetGemVersionDetail"]
    Pick -->|"反向依赖"| RDep["GetReverseDependencies"]
    Pick -->|"多个 gem"| Bulk["BulkGetPackages"]
    Pick -->|"发布 / 管理"| Write["WriteRepository (token)"]
    Get & Search & Ver & RDep & Bulk & Write --> Out["类型化 Go 结构体<br/>→ Agent 推理与行动"]

    classDef agent fill:#7c3aed22,stroke:#7c3aed,color:#fff
    classDef sdk fill:#0ea5e922,stroke:#0ea5e9,color:#fff
    classDef out fill:#10b98122,stroke:#10b981,color:#fff
    class Agent,Decide agent
    class SDK,Inst,Pick,Get,Search,Ver,RDep,Bulk,Write sdk
    class Out out
```

Agent 完全不碰 HTTP、JSON 或 URL 编码——它调用类型化方法，对类型化结果做推理。当它需要实际的 `gem`/`ruby` 二进制（例如在 `PushGem` 前构建 `.gem`）时，先由安装器装好。

---

← 返回：[工作原理](./how-it-works) · 下一篇：[安装](./installation)
