package models

// Webhook 表示 RubyGems.org 上的 Webhook 配置
// GET /api/v1/web_hooks.json
type Webhook struct {
	// Webhook 回调 URL
	URL string `json:"url"`

	// 失败次数
	FailureCount int `json:"failure_count"`
}

// TopDownloadedGem 表示下载量排名前50的 gem 包
// GET /api/v1/downloads/all.json
type TopDownloadedGem struct {
	// gem 包名
	Name string `json:"name"`

	// 总下载量
	Downloads int `json:"downloads"`
}
