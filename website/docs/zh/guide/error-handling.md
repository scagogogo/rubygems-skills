# 错误处理

每次 API 调用都可能失败。rubygems-skills 提供类型化的、基于谓词的错误检查，让你的程序按**为什么**失败分支 —— 而不只是知道**失败了**。

## 三个谓词

```go
package repository // （导出的辅助函数）

func IsNotFound(err error) bool       // 404 — gem/version/user 不存在
func IsRateLimited(err error) bool    // 429 — 你调用 API 太快
func IsUnauthorized(err error) bool   // 401 — token 缺失/无效
```

## 用法

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

这个 `switch` 是一棵决策树 —— 按**为什么**失败分支，而非只是知道**失败了**：

```mermaid
flowchart TD
    Err(["调用返回 err"]) --> N{"IsNotFound(err)?\n(HTTP 404)"}
    N -->|是| NotFound["gem/version/user 不存在\n→ 跳过、警告或回退"]
    N -->|否| R{"IsRateLimited(err)?\n(HTTP 429)"}
    R -->|是| Rate["需要减速\n→ 应用层退避 / 队列"]
    R -->|否| U{"IsUnauthorized(err)?\n(HTTP 401)"}
    U -->|是| Auth["token 缺失/错误\n→ 提示输入 RUBYGEMS_API_KEY"}
    U -->|否| Other{"err != nil?"}
    Other -->|是| Transient["5xx / 网络 / 未知\n→ 视为瞬态，稍后重试"]
    Other -->|否| OK(["nil — 成功，使用结果"])

    classDef hit fill:#16a34a22,stroke:#16a34a,color:#fff
    classDef miss fill:#cc342d22,stroke:#cc342d,color:#fff
    class OK hit
    class NotFound,Rate,Auth,Transient miss
```

## 底层错误类型

请求失败时，你收到一个 `*APIError`（在 `pkg/repository/errors.go`）：

```go
type APIError struct {
    Cause      error  // 底层错误（如有）
    StatusCode int    // HTTP 状态码（404、429、500 等）
    URL        string // 失败的请求 URL
    Response   string // 原始响应体（字符串形式）
}
```

它的 `Error()` 字符串是 `API error (status: <code>, url: <url>): <cause>`。上面的谓词检查 `StatusCode`：

- `IsNotFound` → `StatusCode == 404`
- `IsRateLimited` → `StatusCode == 429`
- `IsUnauthorized` → `StatusCode == 401`

对其他状态码（如 500），没有谓词匹配 —— 当作通用服务器错误。还有哨兵错误（`ErrNotFound`、`ErrRateLimited`、`ErrUnauthorized`、`ErrServerError`、`ErrTimeout`、`ErrNetworkFailure`、`ErrInvalidRequest`）可用 `errors.Is` —— 见 [Errors API 参考](../api/errors)。

## 检查详情

如果需要完整响应体（如 API 的错误信息）：

```go
pkg, err := repo.GetPackage(ctx, "some-gem")
var apiErr *repository.APIError
if errors.As(err, &apiErr) {
    fmt.Println("status:", apiErr.StatusCode)
    fmt.Println("url:", apiErr.URL)
    fmt.Println("body:", apiErr.Response)
}
```

## 与重试的交互

重试层默认重试**任何**错误（`ShouldRetry = err != nil`），所以 429、5xx、404、401 和网络错误都会重试到 `MaxAttempts`（见 [重试与退避](./retry)）。当错误到达你的代码时，所有重试尝试已经耗尽 —— 所以这里的 `IsRateLimited` 意思是「重试预算用尽**后**仍被限流」。正确的应对通常是应用层退避（更长睡眠、把任务入队、通知运维），而非立即再重试。

因为 404/401 默认也被重试，缺失的 gem 或错误 token 会耗尽完整重试预算才浮现。如果这不是你想要的，请提供排除它们的 `ShouldRetry` 谓词（见 [重试与退避](./retry) 中的警告）。

## 网络错误

非 HTTP 失败（DNS、TLS、连接被拒）以包装了底层原因的普通 `error` 形式浮现。三个谓词都不匹配。当作瞬态网络问题处理 —— 通常可重试，取决于你的重试谓词。

```go
pkg, err := repo.GetPackage(ctx, "rails")
if err != nil && !repository.IsNotFound(err) && !repository.IsUnauthorized(err) {
    // 可能是瞬态（重试后仍被限流、重试后的 5xx，或网络问题）
    log.Printf("transient failure, will retry later: %v", err)
}
```

---

至此 Features 部分结束。下一步前往 [AI Agent 集成指南](../ai-agents/overview) —— 这个 SDK 就是为它而生。
