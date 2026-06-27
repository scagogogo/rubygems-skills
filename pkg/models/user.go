package models

import "time"

// UserProfile 表示 RubyGems.org 用户/所有者的信息
// GET /api/v1/users/{handle}.json
type UserProfile struct {
	ID        int       `json:"id"`
	Handle    string    `json:"handle"`
	Email     string    `json:"email,omitempty"`
	Name      string    `json:"name,omitempty"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	URL       string    `json:"url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Owner 表示 gem 包的所有者信息
// GET /api/v1/gems/{gem}/owners.json
type Owner struct {
	Handle    string `json:"handle"`
	Email     string `json:"email,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	ID        int    `json:"id"`
}

// OwnerRole 表示所有者的角色（admin 或 owner）
type OwnerRole struct {
	Handle string `json:"handle"`
	Role   string `json:"role"` // "admin" 或 "owner"
}
