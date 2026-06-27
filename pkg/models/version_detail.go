package models

import "time"

// VersionDetail 是 API v2 端点返回的详细版本信息
// GET /api/v2/rubygems/{gem}/versions/{version}.json
// 比V1的Version结构体包含更多字段（spec_sha, yanked, 完整依赖信息等）
type VersionDetail struct {
	Authors         string       `json:"authors"`
	BuiltAt         time.Time    `json:"built_at"`
	CreatedAt       time.Time    `json:"created_at"`
	Description     string       `json:"description"`
	DownloadsCount  int          `json:"downloads_count"`
	Number          string       `json:"number"`
	Summary         string       `json:"summary"`
	Platform        string       `json:"platform"`
	RubygemsVersion string       `json:"rubygems_version"`
	RubyVersion     string       `json:"ruby_version"`
	Prerelease      bool         `json:"prerelease"`
	Licenses        []string     `json:"licenses"`
	Requirements    []string     `json:"requirements"`
	Sha             string       `json:"sha"`
	SpecSha         string       `json:"spec_sha"`
	Yanked          bool         `json:"yanked"`
	Metadata        *Metadata    `json:"metadata,omitempty"`
	Dependencies    Dependencies `json:"dependencies"`
}

// VersionContent 表示 gem 版本的文件校验和/清单内容
// GET /api/v2/rubygems/{gem}/versions/{version}/contents.json
type VersionContent struct {
	// 文件路径到SHA256校验和的映射
	Files map[string]string `json:"files"`
}

// Attestation 表示 gem 版本的 sigstore 证明
// GET /api/v1/attestations/{gem}-{version}.json
type Attestation struct {
	// 证明主体内容
	Body string `json:"body"`
}
