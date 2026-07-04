# 工作原理

rubygems-skills 是 RubyGems.org HTTP API 之上的一层薄类型化封装。本页解释架构，让你（和你的 AI agent）理解调用方法时到底发生了什么。

## 分层

```mermaid
flowchart TD
    subgraph Caller["你的 Go 程序 / AI agent"]
        Call["repo.GetPackage(ctx, \"rails\")"]
    end
    subgraph SDK["pkg/repository"]
        direction TB
        Iface["Repository 接口 — 读端点（无需认证）\nWriteRepository 接口 — 写端点（需要认证）"]
        Generic["getJson[T] / getBytes — 一个泛型 HTTP 路径\n所有方法共用"]
        Http["HTTP client + RetryOptions（指数退避）\n可选：BasicAuth token · Proxy · 自定义 ServerURL"]
        Iface --> Generic --> Http
    end
    Call -->|"类型化调用"| Iface
    Http -->|"HTTPS"| Upstream["RubyGems.org\n（或镜像：ruby-china / tsinghua / aliyun）"]

    classDef caller fill:#cc342d22,stroke:#cc342d,color:#fff
    classDef sdk fill:#7c3aed22,stroke:#7c3aed,color:#fff
    classDef net fill:#0ea5e922,stroke:#0ea5e9,color:#fff
    class Call caller
    class Iface,Generic sdk
    class Http,Upstream net
```

三层，自上而下：

1. **你的代码** 调用类型化方法 —— 无需关心 URL、JSON。
2. **SDK** 通过一个共享的泛型路径（`getJson[T]`）把调用转为 HTTP 请求，重试/退避、认证、代理都在这里集中处理。
3. **上游** 是 RubyGems.org 或镜像 —— SDK 不关心是哪个，只是一个 `ServerURL`。

## Repository 接口

核心读接口是 `pkg/repository/repository.go` 中的 `Repository` 接口。每个方法与 RubyGems.org 端点 1:1 对应：

| 方法 | 端点 | 返回 |
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

完整列表见 [API 参考](../api/repository)。

## 泛型：`getJson[T]`

SDK 不在每个方法里重复 HTTP-然后-反序列化，而是用一个泛型辅助函数：

```go
func getJson[T any](ctx context.Context, repository *RepositoryImpl, targetUrl string) (T, error)
```

每个方法调用 `getJson[*models.PackageInformation](ctx, repo, url)` —— HTTP、重试、认证、代理只有一条代码路径；类型参数处理 JSON 目标。这就是为什么 SDK 需要 **Go 1.21+**（泛型在 1.18 引入；SDK 固定 1.21 以适配更广泛的工具链）。

**为什么用泛型辅助函数，而不是每个方法单独写 HTTP 代码？** 没有泛型的话，每个端点要么 (a) 重复 HTTP-get → 反序列化 → 错误包装的样板代码（~30 个方法 × ~15 行 = 大量易漂移的副本），要么 (b) 返回 `interface{}` 强制每个调用者做类型断言。`getJson[T]` 给出 (a) 的类型安全和 (b) 的单一代码路径：HTTP/重试/认证逻辑只在一个地方，每个方法获得编译时类型化的返回值，零断言。`runWorkerPool[T]`（见 [批量操作](./bulk-operations)）对 worker pool 用同样的思路。

## 重试与退避

每个请求都经过 `RetryOptions`。默认：

- 最多 **3 次尝试**。
- **指数退避**，起始 1s，上限 30s（`waitTime * 2^(attempt-1)`）。
- **任何**错误都重试 —— 默认 `ShouldRetry` 是 `err != nil`。非 2xx HTTP 状态码（429、5xx、404、401 等）在重试决策前被响应处理器转为错误，所以都会触发重试，除非你覆盖 `ShouldRetry`。

配置方式：

```go
retry := repository.NewDefaultRetryOptions().
    WithMaxAttempts(6).
    WithWaitTime(200 * time.Millisecond).
    WithMaxWaitTime(5 * time.Second)

opts := repository.NewOptions().
    SetRetryOptions(retry)

repo := repository.NewRepository(opts)
```

参见 [重试与退避](./retry)。

## 缓存装饰器

`CachedRepository` 包装任意 `Repository`，用 TTL 缓存结果：

```go
repo := repository.NewRubyChinaRepository()
cached := repository.NewCachedRepository(repo, 10*time.Minute, nil)
```

接口相同 —— 但重复调用命中内存缓存而非网络。直接替换，其他代码无需改动。参见 [缓存](./caching)。

## 写操作

`WriteRepository` 覆盖需要认证的端点 —— push、yank、owners、webhooks、API keys。gem/owner/webhook 操作用 `Authorization` header 发送 token；API-key 和 profile 操作用 HTTP Basic Auth（用户名 + 密码）。gem push 用 `multipart/form-data`。认证细节见 [WriteRepository](../api/write-repository)。

## 自动安装层

另外，`pkg/install` 可以在主机上安装 Ruby/RubyGems —— 当你的工作流需要实际运行 `gem`/`ruby` 而不只是调用 HTTP API 时有用。它检测 OS 并派发到正确的包管理器。参见 [自动安装](../auto-install/overview)。

---

下一步：[安装](./installation)。
