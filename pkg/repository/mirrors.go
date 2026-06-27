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

// Tsinghua University RubyGems mirror source
// API base path: https://mirrors.tuna.tsinghua.edu.cn/rubygems/api
const ServerURLTSingHua = "https://mirrors.tuna.tsinghua.edu.cn/rubygems/api"

// NewTSingHuaRepository Use Tsinghua University mirror repository
//
// Note: Tsinghua source uses /api path prefix, different from official rubygems.org.
// If mirror service is unavailable, try other mirror sources or use NewCustomRepository.
func NewTSingHuaRepository() Repository {
	return NewRepository(NewOptions().SetServerURL(ServerURLTSingHua))
}

// ------------------------------------------------- --------------------------------------------------------------------

// Alibaba Cloud RubyGems mirror source
// Base path: https://mirrors.aliyun.com/rubygems
const ServerURLAliYun = "https://mirrors.aliyun.com/rubygems"

// NewAliYunRepository Use Alibaba Cloud mirror repository
//
// Note: Alibaba Cloud RubyGems mirror may only provide gem downloads, not complete API service.
// If 404 or API unavailable, use official source or configure other custom repository endpoint.
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
