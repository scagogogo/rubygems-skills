package models

import "time"

// APIKey represents the API key info of RubyGems.org
// GET /api/v1/api_key
// POST /api/v1/api_key
type APIKey struct {
	// unique identifier of the API key
	ID int `json:"id"`

	// name of the API key
	Name string `json:"name"`

	// permission scope list of the API key
	// possible values: "index_rubygems", "push_rubygem", "yank_rubygem", "add_owner", "remove_owner",
	// "access_webhooks", "dashboard", "read_settings", "write_settings"
	Scopes []string `json:"scopes"`

	// the API key value (only returned on creation)
	Key string `json:"key,omitempty"`

	// whether MFA is enabled
	MFA string `json:"mfa,omitempty"`

	// associated gem package name (if any)
	RubygemName string `json:"rubygem_name,omitempty"`

	// expiration time
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// created time
	CreatedAt time.Time `json:"created_at"`

	// updated time
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateAPIKeyRequest request parameters for creating an API key
// POST /api/v1/api_key
type CreateAPIKeyRequest struct {
	// name of the API key (required)
	Name string `json:"name"`

	// permission scopes of the API key (required)
	Scopes []string `json:"scopes"`

	// MFA setting: "enabled", "disabled"
	MFA string `json:"mfa,omitempty"`

	// associated gem package name (optional)
	RubygemName string `json:"rubygem_name,omitempty"`

	// expiration time (optional, ISO8601 format)
	ExpiresAt string `json:"expires_at,omitempty"`
}

// UpdateAPIKeyRequest request parameters for updating an API key
// PATCH /api/v1/api_key
type UpdateAPIKeyRequest struct {
	// the API key value to update (required, used to identify which key)
	APIKey string `json:"api_key"`

	// new permission scopes
	Scopes []string `json:"scopes,omitempty"`

	// MFA setting: "enabled", "disabled"
	MFA string `json:"mfa,omitempty"`
}

// MFAStatus represents the user's multi-factor authentication status
// GET /api/v1/multifactor_auth
type MFAStatus struct {
	// whether MFA is enabled
	Enabled bool `json:"enabled"`

	// MFA level: "disabled", "enabled", "required"
	Level string `json:"level"`

	// available MFA methods
	Methods []string `json:"methods,omitempty"`
}
