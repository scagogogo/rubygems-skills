# 批量操作

需要几十或几百个 gem 时，一次取一个太慢。rubygems-skills 提供并发 worker pool 做批量读取 —— `BulkGet*` 方法并行扇出并收集类型化结果。

## 批量方法

| 方法 | 每个 gem 返回 |
|---|---|
| `BulkGetPackages(ctx, names, opts)` | `*models.PackageInformation` |
| `BulkGetVersions(ctx, names, opts)` | `[]*models.Version` |
| `BulkGetDependencies(ctx, names, opts)` | `[]*models.DependencyInfo` |
| `BulkGetReverseDependencies(ctx, names, opts)` | `[]string` |

每个返回 `[]*BulkResult[T]`，与输入切片对齐 —— 结果的索引 `i` 对应 `names[i]`。

## 快速开始

```go
repo := repository.NewRepository()

names := []string{"rails", "puma", "sidekiq", "redis", "rake"}

results := repo.BulkGetPackages(ctx, names, nil) // nil = 默认 options
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
    WithMaxConcurrency(8).       // 并行 worker 数
    WithContinueOnError(true)    // 单个 gem 失败也继续

results := repo.BulkGetPackages(ctx, names, opts)
```

| 方法 | 默认值 | 用途 |
|---|---|---|
| `WithMaxConcurrency(n)` | 10 | 在飞请求数。 |
| `WithContinueOnError(bool)` | true | 为 false 时，pool 在首个错误后停止。 |

## 结果类型

```go
type BulkResult[T any] struct {
    Key   string // 请求键，通常是 gem 名
    Value T      // 操作结果
    Error error  // 操作中可能的错误
}
```

因为错误是按条目而非单个致命错误，一个 gem 缺失不会拖垮整批。使用 `r.Value` 前务必检查 `r.Error`；`r.Key` 让你不依赖切片索引就能把结果关联回输入名。

## 并发上限

RubyGems.org 限流较激进。保守设置 `MaxConcurrency`：

- **未认证：** 2–5 个 worker。更高易触发 429。
- **已认证（有 token）：** 5–10 个 worker 通常安全。
- **开启重试：** 可以激进一些 —— 瞬态 429 会自动重试。

如果频繁遇到 `IsRateLimited` 错误，降低 `MaxConcurrency` 或通过 `Options.SetToken` 加 token。

## 配合缓存

批量方法绕过 `CachedRepository` 装饰器，直接打底层 repo（按条目并发使缓存层对批量读取多余）。对单个 gem 的重复读取，改用 `CachedRepository` 上的包装方法。

## 实现说明

pool 是泛型的 —— `runWorkerPool[T]` —— 所以四个批量方法共用同一套并发实现。

```mermaid
flowchart LR
    subgraph In["输入：gemNames[]"]
        N0["names[0]"]
        N1["names[1]"]
        N2["names[2]")
        NDots["..."]
    end
    Ch(("索引 channel\n0,1,2,...")):::chan
    subgraph Workers["N 个 worker（MaxConcurrency）"]
        W0["worker 0"]
        W1["worker 1"]
        W2["worker 2"]
    end
    subgraph Out["预分配的 results[]（无锁）"]
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

无锁如何保持正确：

- 分发器把**索引**（不是 gem 名）发到带缓冲 channel —— 每个索引 `i` 对应 `names[i]`。
- 每个 worker 拉取一个索引，为 `names[i]` 调用底层方法，把 `BulkResult[T]` **只写入 `results[i]`**。因为每次写指向不同的切片槽位，无数据竞争，结果切片上无需 mutex。
- 结果切片预分配为 `len(names)`，所以顺序与输入完全匹配 —— `results[i]` 总是回答 `names[i]`。

`ContinueOnError(false)` 时，pool 在首个错误后停止分发新索引（worker 完成在飞的条目后退出）—— 所以单个失败会短路整批的后续。默认 `true` 时，每个条目都执行，失败落在各自槽位的 `Error` 字段。

---

下一步：[错误处理](./error-handling)。
