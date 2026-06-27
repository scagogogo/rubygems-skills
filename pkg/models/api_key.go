package models

import "time"

// APIKey 表示 RubyGems.org 的 API Key 信息
// GET /api/v1/api_key
// POST /api/v1/api_key
type APIKey struct {
	// API Key 的唯一标识
	ID int `json:"id"`

	// API Key 的名称
	Name string `json:"name"`

	// API Key 的权限范围列表
	// 可能的值: "index_rubygems", "push_rubygem", "yank_rubygem", "add_owner", "remove_owner",
	// "access_webhooks", "dashboard", "read_settings", "write_settings"
	Scopes []string `json:"scopes"`

	// API Key 的值（仅在创建时返回）
	Key string `json:"key,omitempty"`

	// 是否启用了 MFA
	MFA string `json:"mfa,omitempty"`

	// 关联的 gem 包名称（如果有）
	RubygemName string `json:"rubygem_name,omitempty"`

	// 过期时间
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// 创建时间
	CreatedAt time.Time `json:"created_at"`

	// 更新时间
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateAPIKeyRequest 创建 API Key 的请求参数
// POST /api/v1/api_key
type CreateAPIKeyRequest struct {
	// API Key 的名称（必填）
	Name string `json:"name"`

	// API Key 的权限范围（必填）
	Scopes []string `json:"scopes"`

	// MFA 设置: "enabled", "disabled"
	MFA string `json:"mfa,omitempty"`

	// 关联的 gem 包名称（可选）
	RubygemName string `json:"rubygem_name,omitempty"`

	// 过期时间（可选，ISO8601 格式）
	ExpiresAt string `json:"expires_at,omitempty"`
}

// UpdateAPIKeyRequest 更新 API Key 的请求参数
// PATCH /api/v1/api_key
type UpdateAPIKeyRequest struct {
	// 要更新的 API Key 值（必填，用于标识哪个 key）
	APIKey string `json:"api_key"`

	// 新的权限范围
	Scopes []string `json:"scopes,omitempty"`

	// MFA 设置: "enabled", "disabled"
	MFA string `json:"mfa,omitempty"`
}

// MFAStatus 表示用户的多因素认证状态
// GET /api/v1/multifactor_auth
type MFAStatus struct {
	// 是否已启用 MFA
	Enabled bool `json:"enabled"`

	// MFA 级别: "disabled", "enabled", "required"
	Level string `json:"level"`

	// 可用的 MFA 方法
	Methods []string `json:"methods,omitempty"`
}
