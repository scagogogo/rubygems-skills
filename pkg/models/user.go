package models

import "time"

// UserProfile represents the info of a RubyGems.org user/owner
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

// Owner represents the owner info of a gem package
// GET /api/v1/gems/{gem}/owners.json
type Owner struct {
	Handle    string `json:"handle"`
	Email     string `json:"email,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	ID        int    `json:"id"`
}

// OwnerRole represents the role of an owner (admin or owner)
type OwnerRole struct {
	Handle string `json:"handle"`
	Role   string `json:"role"` // "admin" or "owner"
}
