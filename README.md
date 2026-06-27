# rubygems-skills

[![Go Reference](https://pkg.go.dev/badge/github.com/scagogogo/rubygems-skills.svg)](https://pkg.go.dev/github.com/scagogogo/rubygems-skills)
[![Go Report Card](https://goreportcard.com/badge/github.com/scagogogo/rubygems-skills)](https://goreportcard.com/report/github.com/scagogogo/rubygems-skills)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.21-blue)](https://go.dev/)

[🇨🇳 简体中文](README.zh-CN.md)

A production-ready Go SDK for the [RubyGems.org](https://rubygems.org) API. It provides a complete, type-safe client that covers **all public API v1 and v2 endpoints** — including package queries, search, versions, downloads, dependencies, user/owner management, API key management, MFA status, webhooks, attestations, and gem publishing — with built-in caching, concurrent bulk operations, retry with exponential backoff, mirror repository support, and a full-featured CLI.

## Why This SDK?

If you're building Go tooling that interacts with the Ruby gem ecosystem — dependency analysis, security auditing, registry mirroring, CI/CD integration, or data pipelines — you need a reliable, typed API client. This SDK eliminates the need to hand-craft HTTP calls, parse JSON, handle rate limits, and manage retries. It wraps every RubyGems.org endpoint into idiomatic Go with proper error types, URL-safe parameter encoding, and optional caching out of the box.

---

## Features

- **Complete API Coverage** — All RubyGems API v1/v2 endpoints: packages, search, versions, downloads, dependencies, reverse dependencies, user profiles, owners, API keys, MFA, webhooks, attestations, and gem publishing
- **Multi-Repository Support** — Built-in mirrors (Ruby China, Tsinghua, Alibaba Cloud) plus `NewCustomRepository()` for private/custom gem servers
- **Smart Error Handling** — Typed errors (`IsNotFound`, `IsRateLimited`, `IsUnauthorized`) with structured `APIError` for programmatic handling
- **Automatic Retry** — Configurable retry with exponential backoff for transient failures (network errors, 429, 5xx). All request types (GET, POST, DELETE, form, multipart) support retry
- **URL-Safe Encoding** — All path and query parameters are properly encoded via `url.PathEscape` / `url.QueryEscape` to handle special characters
- **In-Memory Cache** — Thread-safe cache with TTL support, auto-cleanup, and `Cache` interface for custom implementations
- **Bulk Operations** — Concurrent batch requests with configurable concurrency for high-throughput data collection
- **Auto-Install** — Cross-platform automatic Ruby/RubyGems installation supporting apt, yum, dnf, apk, pacman, brew, choco, scoop, and zypper
- **HTTP Proxy & Auth** — Full support for corporate proxy environments, API token authentication, and HTTP Basic authentication
- **CLI Tool** — Command-line interface for quick queries with JSON output, mirror selection, and auto-install
- **Type-Safe Models** — Complete Go struct definitions matching the RubyGems API JSON schema
- **Comprehensive Tests** — Unit tests for all packages, Docker-based cross-platform integration tests, and race detector coverage

---

## Installation

```bash
go get github.com/scagogogo/rubygems-skills
```

**Requirements:** Go 1.21+

---

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"

    "github.com/scagogogo/rubygems-skills/pkg/repository"
)

func main() {
    repo := repository.NewRepository()

    pkg, err := repo.GetPackage(context.Background(), "rails")
    if err != nil {
        panic(err)
    }

    fmt.Printf("Name: %s\n", pkg.Name)
    fmt.Printf("Version: %s\n", pkg.Version)
    fmt.Printf("Downloads: %d\n", pkg.Downloads)
    fmt.Printf("Authors: %s\n", pkg.Authors)
}
```

### Using Mirror Repositories

```go
// Ruby China Mirror (recommended for users in China)
repo := repository.NewRubyChinaRepository()

// Tsinghua University Mirror
repo := repository.NewTSingHuaRepository()

// Alibaba Cloud Mirror
repo := repository.NewAliYunRepository()

// Custom / Private gem server
repo := repository.NewCustomRepository("https://gems.example.com")
```

### Caching

```go
import (
    "time"
    "github.com/scagogogo/rubygems-skills/pkg/cache"
    "github.com/scagogogo/rubygems-skills/pkg/repository"
)

memCache := cache.NewMemoryCache(10*time.Minute, 30*time.Minute)
cachedRepo := repository.NewCachedRepository(repo, 5*time.Minute, memCache)

// First call hits the API
pkg, _ := cachedRepo.GetPackage(ctx, "rails")

// Second call returns from cache
pkg, _ = cachedRepo.GetPackage(ctx, "rails")

cachedRepo.ClearCache()
cachedRepo.Close()
```

### Bulk Concurrent Requests

```go
gems := []string{"rails", "rack", "activesupport", "rake", "bundler"}
options := repository.NewBulkOptions().WithMaxConcurrency(5)
results := repo.BulkGetPackages(ctx, gems, options)

for _, result := range results {
    if result.Error != nil {
        fmt.Printf("%s failed: %v\n", result.Key, result.Error)
        continue
    }
    fmt.Printf("%s: v%s (%d downloads)\n", result.Value.Name, result.Value.Version, result.Value.Downloads)
}
```

### Error Handling

```go
pkg, err := repo.GetPackage(ctx, "non-existent-gem")
if err != nil {
    if repository.IsNotFound(err) {
        fmt.Println("Package not found")
    } else if repository.IsRateLimited(err) {
        fmt.Println("Rate limited — back off and retry")
    } else if repository.IsUnauthorized(err) {
        fmt.Println("Authentication failed")
    } else {
        var apiErr *repository.APIError
        if errors.As(err, &apiErr) {
            fmt.Printf("HTTP %d at %s: %s\n", apiErr.StatusCode, apiErr.URL, apiErr.Response)
        }
    }
}
```

### Authentication & Proxy

```go
// API Token (increases rate limits)
options := repository.NewOptions().SetToken("your-api-token")
repo := repository.NewRepository(options)

// HTTP Proxy (corporate environments)
options := repository.NewOptions().SetProxy("http://127.0.0.1:7890")
repo := repository.NewRepository(options)
```

### Custom Retry Strategy

```go
retryOpts := repository.NewDefaultRetryOptions().
    WithMaxAttempts(5).
    WithWaitTime(2 * time.Second).
    WithExponentialBackoff(true)

options := repository.NewOptions().SetRetryOptions(retryOpts)
repo := repository.NewRepository(options)
```

### Write Operations (Auth Required)

```go
options := repository.NewOptions().SetToken("your-api-token")
writeRepo := repository.NewWriteRepository(options)

// Yank (unpublish) a gem version
result, err := writeRepo.YankGem(ctx, "my-gem", "1.0.0")

// Manage gem owners
err = writeRepo.AddGemOwner(ctx, "my-gem", "user@example.com", "owner")
err = writeRepo.RemoveGemOwner(ctx, "my-gem", "user@example.com")

// Manage webhooks
err = writeRepo.CreateWebhook(ctx, "my-gem", "https://example.com/webhook")
webhooks, err := writeRepo.ListWebhooks(ctx)
err = writeRepo.DeleteWebhook(ctx, "my-gem", "https://example.com/webhook")
```

### API Key Management (HTTP Basic Auth)

```go
// Retrieve a legacy API key
apiKey, err := writeRepo.GetAPIKey(ctx, "username", "password")

// Create a new scoped API key
req := &models.CreateAPIKeyRequest{
    Name:   "ci-key",
    Scopes: []string{"push_rubygem", "yank_rubygem"},
    MFA:    "enabled",
}
apiKey, err = writeRepo.CreateAPIKey(ctx, "username", "password", req)

// Update an API key's scopes
updateReq := &models.UpdateAPIKeyRequest{
    APIKey: "existing-key-value",
    Scopes: []string{"index_rubygems"},
}
apiKey, err = writeRepo.UpdateAPIKey(ctx, "username", "password", updateReq)
```

### MFA Status

```go
// Check MFA status for the authenticated user (requires API Token)
status, err := repo.GetMFAStatus(ctx)
fmt.Printf("MFA enabled: %v, level: %s\n", status.Enabled, status.Level)
```

### Authenticated User Profile

```go
// Get your full profile (including private fields, requires HTTP Basic Auth)
profile, err := writeRepo.GetMyProfile(ctx, "username", "password")
```

### V2 API — Richer Version Details

```go
// Detailed version info via API v2 (includes spec_sha, yanked, full deps)
detail, err := repo.GetGemVersionDetail(ctx, "rails", "7.0.5")
fmt.Printf("Yanked: %v\n", detail.Yanked)
fmt.Printf("Spec SHA: %s\n", detail.SpecSha)

// File checksums for a version
contents, err := repo.GetGemVersionContents(ctx, "rails", "7.0.5")
for file, sha := range contents.Files {
    fmt.Printf("  %s: %s\n", file, sha)
}
```

### User & Owner Info

```go
profile, err := repo.GetUserProfile(ctx, "qrush")
gems, err := repo.GetGemsByOwner(ctx, "qrush")
owners, err := repo.GetGemOwners(ctx, "rails")
```

### Version-Level Reverse Dependencies

```go
// Get packages that depend on a specific version (fullName = "gemname-version")
deps, err := repo.GetVersionReverseDependencies(ctx, "rack-2.2.7")
```

### Top Downloads & Autocomplete

```go
topGems, err := repo.TopDownloads(ctx)
suggestions, err := repo.SearchAutocomplete(ctx, "rails")
```

---

## CLI Tool

```bash
go build -o rubygems-cli ./cmd/rubygems/

./rubygems-cli -get -gem rails          # Get package info
./rubygems-cli -search -query rails     # Search packages
./rubygems-cli -versions -gem rails     # List versions
./rubygems-cli -deps -gem rails         # Show dependencies
./rubygems-cli -rdeps -gem rails        # Show reverse dependencies
./rubygems-cli -get -gem rails -json    # JSON output
./rubygems-cli -get -gem rails -mirror ruby-china  # Use mirror
./rubygems-cli -install                 # Auto-install Ruby/RubyGems
./rubygems-cli -help                    # Help
```

---

## API Reference

### Repository Interface (Read Operations)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GetPackage(ctx, gem)` | `GET /api/v1/gems/{gem}.json` | Get detailed package info |
| `Search(ctx, query, page)` | `GET /api/v1/search.json?query=` | Search packages |
| `SearchAutocomplete(ctx, query)` | `GET /api/v1/search/autocomplete.json` | Search autocomplete suggestions |
| `GetGemVersions(ctx, gem)` | `GET /api/v1/versions/{gem}.json` | List all versions |
| `GetGemLatestVersion(ctx, gem)` | `GET /api/v1/versions/{gem}/latest.json` | Get latest version |
| `GetGemVersionDetail(ctx, gem, ver)` | `GET /api/v2/rubygems/{gem}/versions/{ver}.json` | **V2** Detailed version info |
| `GetTimeFrameVersions(ctx, from, to)` | `GET /api/v1/timeframe_versions.json` | Versions in time range |
| `Downloads(ctx)` | `GET /api/v1/downloads.json` | Total repository downloads |
| `VersionDownloads(ctx, gem, ver)` | `GET /api/v1/downloads/{gem}-{ver}.json` | Version download count |
| `TopDownloads(ctx)` | `GET /api/v1/downloads/all.json` | Top 50 most downloaded gems |
| `GetDependencies(ctx, gems...)` | `GET /api/v1/dependencies?gems=` | Dependency info |
| `GetReverseDependencies(ctx, gem)` | `GET /api/v1/gems/{gem}/reverse_dependencies.json` | Reverse dependencies |
| `GetVersionReverseDependencies(ctx, fullName)` | `GET /api/v1/versions/{fullName}/reverse_dependencies.json` | Version-level reverse dependencies |
| `LatestGems(ctx)` | `GET /api/v1/activity/latest.json` | Recently published gems |
| `JustUpdatedGems(ctx)` | `GET /api/v1/activity/just_updated.json` | Recently updated gems |
| `GetUserProfile(ctx, handle)` | `GET /api/v1/profiles/{handle}.json` | User profile info |
| `GetOwnedGems(ctx)` | `GET /api/v1/gems.json` | List your gems (auth required) |
| `GetGemsByOwner(ctx, handle)` | `GET /api/v1/owners/{handle}/gems.json` | Gems by user |
| `GetGemOwners(ctx, gem)` | `GET /api/v1/gems/{gem}/owners.json` | Gem owners |
| `GetAttestations(ctx, gem, ver)` | `GET /api/v1/attestations/{gem}-{ver}.json` | Sigstore attestations |
| `GetGemVersionContents(ctx, gem, ver)` | `GET /api/v2/rubygems/{gem}/versions/{ver}/contents.json` | **V2** Version file checksums |
| `GetMFAStatus(ctx)` | `GET /api/v1/multifactor_auth` | MFA status (auth required) |
| `BulkGetPackages(ctx, gems, opts)` | (concurrent) | Bulk package fetch |
| `BulkGetVersions(ctx, gems, opts)` | (concurrent) | Bulk version fetch |
| `BulkGetDependencies(ctx, gems, opts)` | (concurrent) | Bulk dependency fetch |
| `BulkGetReverseDependencies(ctx, gems, opts)` | (concurrent) | Bulk reverse dependency fetch |

### WriteRepository Interface (Auth Required)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `PushGem(ctx, file)` | `POST /api/v1/gems` | Publish a gem |
| `YankGem(ctx, gem, ver)` | `DELETE /api/v1/gems/yank` | Yank (unpublish) a version |
| `YankGemWithPlatform(ctx, gem, ver, platform)` | `DELETE /api/v1/gems/yank` | Yank with platform |
| `AddGemOwner(ctx, gem, email, role)` | `POST /api/v1/gems/{gem}/owners` | Add gem owner |
| `RemoveGemOwner(ctx, gem, email)` | `DELETE /api/v1/gems/{gem}/owners` | Remove gem owner |
| `UpdateGemOwnerRole(ctx, gem, email, role)` | `PATCH /api/v1/gems/{gem}/owners` | Update owner role |
| `ListWebhooks(ctx)` | `GET /api/v1/web_hooks.json` | List webhooks |
| `CreateWebhook(ctx, gem, url)` | `POST /api/v1/web_hooks` | Create webhook |
| `DeleteWebhook(ctx, gem, url)` | `DELETE /api/v1/web_hooks/remove` | Delete webhook |
| `FireWebhook(ctx, gem, url)` | `POST /api/v1/web_hooks/fire` | Test fire webhook |
| `GetAPIKey(ctx, user, pass)` | `GET /api/v1/api_key` | Retrieve API key (Basic Auth) |
| `CreateAPIKey(ctx, user, pass, req)` | `POST /api/v1/api_key` | Create scoped API key (Basic Auth) |
| `UpdateAPIKey(ctx, user, pass, req)` | `PATCH /api/v1/api_key` | Update API key scopes (Basic Auth) |
| `GetMyProfile(ctx, user, pass)` | `GET /api/v1/profiles/me.json` | Full authenticated profile (Basic Auth) |

---

## Project Structure

```
rubygems-skills/
├── cmd/rubygems/              # CLI tool
├── examples/                  # Usage examples
│   ├── basic_usage.go
│   ├── bulk/main.go
│   └── cache/main.go
├── pkg/
│   ├── cache/                 # Cache interface & memory implementation
│   ├── install/               # Cross-platform auto-install
│   ├── models/                # JSON data models (APIKey, MFAStatus, etc.)
│   └── repository/            # Repository client
│       ├── repository.go      # Core client & read interface
│       ├── write_repository.go # Write operations & auth interface
│       ├── mirrors.go         # Mirror & custom repository factories
│       ├── options.go         # Client configuration
│       ├── errors.go          # Typed API errors
│       ├── retry.go           # Retry logic with backoff
│       ├── bulk_operations.go # Concurrent batch operations
│       └── cached_repository.go # Cache decorator
├── tests/
│   └── integration/           # Integration tests
├── go.mod
└── LICENSE
```

---

## Rate Limits

RubyGems.org enforces API rate limits. See the [official documentation](https://guides.rubygems.org/rubygems-org-rate-limits/) for details. Using an API token increases your request quota significantly.

---

## Testing

```bash
# Run all unit tests (no network)
go test -short -v ./...

# Run all tests including live API tests
go test -v ./...

# Run with race detector
go test -short -race -v ./...
```

---

## Contributing

Contributions are welcome! Before submitting a PR, please ensure:

1. All tests pass: `go test -short -race ./...`
2. No vet warnings: `go vet ./...`
3. New code includes tests
4. Code is formatted: `gofmt -s -w .`
5. Documentation is updated

---

## License

MIT — see [LICENSE](LICENSE) for details.

---

## References

- [RubyGems API v2 Guide](https://guides.rubygems.org/rubygems-org-api-v2/)
- [RubyGems API v1 Guide](https://guides.rubygems.org/rubygems-org-api/)
- [RubyGems Rate Limits](https://guides.rubygems.org/rubygems-org-rate-limits/)
