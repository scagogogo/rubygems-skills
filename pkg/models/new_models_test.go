package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestVersionDetail_MarshalUnmarshal(t *testing.T) {
	builtAt, _ := time.Parse(time.RFC3339, "2023-05-24T19:21:28.229Z")
	createdAt, _ := time.Parse(time.RFC3339, "2023-05-24T19:21:28.229Z")

	detail := VersionDetail{
		Authors:         "David Heinemeier Hansson",
		BuiltAt:         builtAt,
		CreatedAt:       createdAt,
		Description:     "Ruby on Rails is a full-stack web framework",
		DownloadsCount:  54428,
		Number:          "7.0.5",
		Summary:         "Full-stack web application framework",
		Platform:        "ruby",
		RubygemsVersion: "3.3.26",
		RubyVersion:     "3.2.2",
		Prerelease:      false,
		Licenses:        []string{"MIT"},
		Requirements:    []string{},
		Sha:             "57ef2baa4a1f5f954bc6e5a019b1fac8486ece36f79c1cf366e6de33210637fe",
		SpecSha:         "abc123def456",
		Yanked:          false,
		Metadata: &Metadata{
			DocumentationURI: "https://api.rubyonrails.org/v7.0.5/",
			BugTrackerURI:    "https://github.com/rails/rails/issues",
		},
		Dependencies: Dependencies{
			Runtime: []*Dependency{
				{Name: "actioncable", Requirements: "= 7.0.5"},
			},
		},
	}

	jsonData, err := json.Marshal(detail)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaled VersionDetail
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, detail.Number, unmarshaled.Number)
	assert.Equal(t, detail.SpecSha, unmarshaled.SpecSha)
	assert.Equal(t, detail.Yanked, unmarshaled.Yanked)
	assert.Equal(t, detail.Sha, unmarshaled.Sha)
	assert.Len(t, unmarshaled.Dependencies.Runtime, 1)
	assert.Equal(t, "actioncable", unmarshaled.Dependencies.Runtime[0].Name)
}

func TestVersionDetail_JsonUnmarshal(t *testing.T) {
	jsonData := `{
		"authors": "David Heinemeier Hansson",
		"built_at": "2023-05-24T19:21:28.229Z",
		"created_at": "2023-05-24T19:21:28.229Z",
		"description": "Ruby on Rails is a full-stack web framework",
		"downloads_count": 54428,
		"number": "7.0.5",
		"summary": "Full-stack web application framework",
		"platform": "ruby",
		"rubygems_version": "3.3.26",
		"ruby_version": "3.2.2",
		"prerelease": false,
		"licenses": ["MIT"],
		"requirements": [],
		"sha": "57ef2baa4a1f5f954bc6e5a019b1fac8486ece36f79c1cf366e6de33210637fe",
		"spec_sha": "abc123def456",
		"yanked": false,
		"metadata": {
			"documentation_uri": "https://api.rubyonrails.org/v7.0.5/"
		},
		"dependencies": {
			"development": [],
			"runtime": [
				{"name": "actioncable", "requirements": "= 7.0.5"}
			]
		}
	}`

	var detail VersionDetail
	err := json.Unmarshal([]byte(jsonData), &detail)
	assert.NoError(t, err)

	assert.Equal(t, "7.0.5", detail.Number)
	assert.Equal(t, "abc123def456", detail.SpecSha)
	assert.False(t, detail.Yanked)
	assert.Len(t, detail.Dependencies.Runtime, 1)
	assert.Equal(t, "actioncable", detail.Dependencies.Runtime[0].Name)
}

func TestUserProfile_JsonUnmarshal(t *testing.T) {
	jsonData := `{
		"id": 1,
		"handle": "qrush",
		"email": "qrush@example.com",
		"name": "Nick Quaranto",
		"avatar_url": "https://avatar.example.com/qrush.png",
		"url": "https://rubygems.org/profiles/qrush",
		"created_at": "2023-01-01T00:00:00Z",
		"updated_at": "2023-06-01T00:00:00Z"
	}`

	var profile UserProfile
	err := json.Unmarshal([]byte(jsonData), &profile)
	assert.NoError(t, err)

	assert.Equal(t, 1, profile.ID)
	assert.Equal(t, "qrush", profile.Handle)
	assert.Equal(t, "qrush@example.com", profile.Email)
	assert.Equal(t, "Nick Quaranto", profile.Name)
}

func TestOwner_JsonUnmarshal(t *testing.T) {
	jsonData := `{
		"handle": "qrush",
		"email": "qrush@example.com",
		"avatar_url": "https://avatar.example.com/qrush.png",
		"id": 1
	}`

	var owner Owner
	err := json.Unmarshal([]byte(jsonData), &owner)
	assert.NoError(t, err)

	assert.Equal(t, "qrush", owner.Handle)
	assert.Equal(t, 1, owner.ID)
}

func TestWebhook_JsonUnmarshal(t *testing.T) {
	jsonData := `{
		"url": "https://example.com/webhook",
		"failure_count": 3
	}`

	var webhook Webhook
	err := json.Unmarshal([]byte(jsonData), &webhook)
	assert.NoError(t, err)

	assert.Equal(t, "https://example.com/webhook", webhook.URL)
	assert.Equal(t, 3, webhook.FailureCount)
}

func TestTopDownloadedGem_JsonUnmarshal(t *testing.T) {
	jsonData := `{
		"name": "rails",
		"downloads": 436090160
	}`

	var gem TopDownloadedGem
	err := json.Unmarshal([]byte(jsonData), &gem)
	assert.NoError(t, err)

	assert.Equal(t, "rails", gem.Name)
	assert.Equal(t, 436090160, gem.Downloads)
}

func TestAttestation_JsonUnmarshal(t *testing.T) {
	jsonData := `{
		"body": "sigstore attestation body content"
	}`

	var att Attestation
	err := json.Unmarshal([]byte(jsonData), &att)
	assert.NoError(t, err)

	assert.Equal(t, "sigstore attestation body content", att.Body)
}

func TestVersionContent_JsonUnmarshal(t *testing.T) {
	jsonData := `{
		"files": {
			"lib/rails.rb": "abc123",
			"lib/rails/engine.rb": "def456"
		}
	}`

	var content VersionContent
	err := json.Unmarshal([]byte(jsonData), &content)
	assert.NoError(t, err)

	assert.Len(t, content.Files, 2)
	assert.Equal(t, "abc123", content.Files["lib/rails.rb"])
	assert.Equal(t, "def456", content.Files["lib/rails/engine.rb"])
}

// ===== Tests for new models =====

func TestAPIKey_JsonUnmarshal(t *testing.T) {
	jsonData := `{
		"id": 42,
		"name": "my-api-key",
		"scopes": ["index_rubygems", "push_rubygem"],
		"key": "abc123secret",
		"mfa": "enabled",
		"rubygem_name": "rails",
		"created_at": "2023-01-15T10:30:00Z",
		"updated_at": "2023-06-20T14:00:00Z"
	}`

	var apiKey APIKey
	err := json.Unmarshal([]byte(jsonData), &apiKey)
	assert.NoError(t, err)

	assert.Equal(t, 42, apiKey.ID)
	assert.Equal(t, "my-api-key", apiKey.Name)
	assert.Equal(t, []string{"index_rubygems", "push_rubygem"}, apiKey.Scopes)
	assert.Equal(t, "abc123secret", apiKey.Key)
	assert.Equal(t, "enabled", apiKey.MFA)
	assert.Equal(t, "rails", apiKey.RubygemName)
	assert.False(t, apiKey.CreatedAt.IsZero())
	assert.False(t, apiKey.UpdatedAt.IsZero())
}

func TestAPIKey_JsonUnmarshal_WithExpiry(t *testing.T) {
	jsonData := `{
		"id": 42,
		"name": "temp-key",
		"scopes": ["index_rubygems"],
		"expires_at": "2024-12-31T23:59:59Z",
		"created_at": "2023-01-15T10:30:00Z",
		"updated_at": "2023-06-20T14:00:00Z"
	}`

	var apiKey APIKey
	err := json.Unmarshal([]byte(jsonData), &apiKey)
	assert.NoError(t, err)

	assert.NotNil(t, apiKey.ExpiresAt)
	assert.Equal(t, "temp-key", apiKey.Name)
}

func TestAPIKey_JsonUnmarshal_NoExpiry(t *testing.T) {
	jsonData := `{
		"id": 42,
		"name": "permanent-key",
		"scopes": ["index_rubygems"],
		"created_at": "2023-01-15T10:30:00Z",
		"updated_at": "2023-06-20T14:00:00Z"
	}`

	var apiKey APIKey
	err := json.Unmarshal([]byte(jsonData), &apiKey)
	assert.NoError(t, err)

	assert.Nil(t, apiKey.ExpiresAt)
}

func TestCreateAPIKeyRequest_JsonMarshal(t *testing.T) {
	req := CreateAPIKeyRequest{
		Name:   "my-new-key",
		Scopes: []string{"index_rubygems", "push_rubygem"},
		MFA:    "enabled",
	}

	jsonData, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaled CreateAPIKeyRequest
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, "my-new-key", unmarshaled.Name)
	assert.Equal(t, []string{"index_rubygems", "push_rubygem"}, unmarshaled.Scopes)
	assert.Equal(t, "enabled", unmarshaled.MFA)
}

func TestUpdateAPIKeyRequest_JsonMarshal(t *testing.T) {
	req := UpdateAPIKeyRequest{
		APIKey: "old-key-value",
		Scopes: []string{"index_rubygems"},
		MFA:    "disabled",
	}

	jsonData, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaled UpdateAPIKeyRequest
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, "old-key-value", unmarshaled.APIKey)
	assert.Equal(t, []string{"index_rubygems"}, unmarshaled.Scopes)
	assert.Equal(t, "disabled", unmarshaled.MFA)
}

func TestMFAStatus_JsonUnmarshal(t *testing.T) {
	jsonData := `{
		"enabled": true,
		"level": "required",
		"methods": ["totp", "webauthn"]
	}`

	var status MFAStatus
	err := json.Unmarshal([]byte(jsonData), &status)
	assert.NoError(t, err)

	assert.True(t, status.Enabled)
	assert.Equal(t, "required", status.Level)
	assert.Equal(t, []string{"totp", "webauthn"}, status.Methods)
}

func TestMFAStatus_JsonUnmarshal_Disabled(t *testing.T) {
	jsonData := `{
		"enabled": false,
		"level": "disabled"
	}`

	var status MFAStatus
	err := json.Unmarshal([]byte(jsonData), &status)
	assert.NoError(t, err)

	assert.False(t, status.Enabled)
	assert.Equal(t, "disabled", status.Level)
	assert.Empty(t, status.Methods)
}
