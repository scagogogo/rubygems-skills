package repository

// ------------------------------------------------- --------------------------------------------------------------------
// Mirror sources and custom repositories
//
// The following are pre-configured commonly used RubyGems mirror sources. You can also use the NewCustomRepository function
// to connect to any custom RubyGems compatible repository (such as private gem servers)
// ------------------------------------------------- --------------------------------------------------------------------

const ServerURLRubyChina = "https://gems.ruby-china.com"

// NewRubyChinaRepository Use Ruby China mirror repository, recommended for users in China
//
// Note: This mirror may become unavailable due to policy changes. If connection issues occur, try other mirror sources
// or use NewCustomRepository to configure other available endpoints.
func NewRubyChinaRepository() Repository {
	return NewRepository(NewOptions().SetServerURL(ServerURLRubyChina))
}

// ------------------------------------------------- --------------------------------------------------------------------

// Tsinghua University RubyGems mirror source.
//
// NOTE: The Tsinghua mirror only mirrors gem files, NOT the RubyGems.org API.
// API calls (GetPackage, Search, etc.) will return 404. Use this mirror only
// for gem download URLs; for API queries use the official source or Ruby China.
const ServerURLTSingHua = "https://mirrors.tuna.tsinghua.edu.cn/rubygems"

// NewTSingHuaRepository Use Tsinghua University mirror repository.
//
// Warning: Tsinghua mirror does NOT provide the API. API calls will fail with
// 404. Prefer NewRubyChinaRepository for API access in China, or use the
// official source.
func NewTSingHuaRepository() Repository {
	return NewRepository(NewOptions().SetServerURL(ServerURLTSingHua))
}

// ------------------------------------------------- --------------------------------------------------------------------

// Alibaba Cloud RubyGems mirror source.
//
// NOTE: The Alibaba Cloud mirror only mirrors gem files, NOT the RubyGems.org
// API. API calls will return 404. Use this mirror only for gem download URLs.
const ServerURLAliYun = "https://mirrors.aliyun.com/rubygems"

// NewAliYunRepository Use Alibaba Cloud mirror repository.
//
// Warning: Alibaba Cloud mirror does NOT provide the API. API calls will fail
// with 404. Prefer NewRubyChinaRepository for API access in China, or use the
// official source.
func NewAliYunRepository() Repository {
	return NewRepository(NewOptions().SetServerURL(ServerURLAliYun))
}

// ------------------------------------------------- --------------------------------------------------------------------

// NewCustomRepository Create a connection to custom repository
//
// Use cases:
//   - Self-hosted private gem server (such as Geminabox)
//   - Enterprise internal gem repository
//   - Other RubyGems compatible API endpoints
//
// Usage:
//
//	repo := NewCustomRepository("https://gems.example.com")
//	pkg, err := repo.GetPackage(ctx, "my-gem")
func NewCustomRepository(serverURL string) Repository {
	return NewRepository(NewOptions().SetServerURL(serverURL))
}
