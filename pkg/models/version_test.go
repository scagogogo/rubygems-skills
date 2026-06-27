package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestVersion_MarshalUnmarshal(t *testing.T) {
	// Create a sample Version
	builtAt, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
	createdAt, _ := time.Parse(time.RFC3339, "2023-01-01T01:00:00Z")

	version := Version{
		Authors:        "Test Author",
		BuiltAt:        builtAt,
		CreatedAt:      createdAt,
		Description:    "Test version description",
		DownloadsCount: 1000,
		Metadata: &Metadata{
			DocumentationURI: "https://example.com/docs",
			BugTrackerURI:    "https://example.com/bugs",
		},
		Number:          "1.0.0",
		Summary:         "Test summary",
		Platform:        "ruby",
		RubygemsVersion: "3.3.26",
		RubyVersion:     "3.2.2",
		Prerelease:      false,
		Licenses:        []string{"MIT"},
		Requirements:    []string{},
		Sha:             "abcdef1234567890",
	}

	// Convert to JSON
	jsonData, err := json.Marshal(version)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Convert back from JSON
	var unmarshaledVersion Version
	err = json.Unmarshal(jsonData, &unmarshaledVersion)
	assert.NoError(t, err)

	// Check if fields match
	assert.Equal(t, version.Authors, unmarshaledVersion.Authors)
	assert.Equal(t, version.BuiltAt.Format(time.RFC3339), unmarshaledVersion.BuiltAt.Format(time.RFC3339))
	assert.Equal(t, version.CreatedAt.Format(time.RFC3339), unmarshaledVersion.CreatedAt.Format(time.RFC3339))
	assert.Equal(t, version.Description, unmarshaledVersion.Description)
	assert.Equal(t, version.DownloadsCount, unmarshaledVersion.DownloadsCount)
	assert.Equal(t, version.Number, unmarshaledVersion.Number)
	assert.Equal(t, version.Summary, unmarshaledVersion.Summary)
	assert.Equal(t, version.Platform, unmarshaledVersion.Platform)
	assert.Equal(t, version.RubygemsVersion, unmarshaledVersion.RubygemsVersion)
	assert.Equal(t, version.RubyVersion, unmarshaledVersion.RubyVersion)
	assert.Equal(t, version.Prerelease, unmarshaledVersion.Prerelease)
	assert.Equal(t, version.Licenses, unmarshaledVersion.Licenses)
	assert.Equal(t, version.Sha, unmarshaledVersion.Sha)
	assert.Equal(t, version.Metadata.DocumentationURI, unmarshaledVersion.Metadata.DocumentationURI)
	assert.Equal(t, version.Metadata.BugTrackerURI, unmarshaledVersion.Metadata.BugTrackerURI)
}

func TestVersion_JsonUnmarshal(t *testing.T) {
	// Sample JSON data
	jsonData := `{
		"authors": "David Heinemeier Hansson",
		"built_at": "2023-05-24T19:21:28.229Z",
		"created_at": "2023-05-24T19:21:28.229Z",
		"description": "Ruby on Rails is a full-stack web framework",
		"downloads_count": 54428,
		"metadata": {
			"documentation_uri": "https://api.rubyonrails.org/v7.0.5/",
			"bug_tracker_uri": "https://github.com/rails/rails/issues"
		},
		"number": "7.0.5",
		"summary": "Full-stack web application framework",
		"platform": "ruby",
		"rubygems_version": "3.3.26",
		"ruby_version": "3.2.2",
		"prerelease": false,
		"licenses": ["MIT"],
		"requirements": [],
		"sha": "57ef2baa4a1f5f954bc6e5a019b1fac8486ece36f79c1cf366e6de33210637fe"
	}`

	var version Version
	err := json.Unmarshal([]byte(jsonData), &version)
	assert.NoError(t, err)

	// Verify parsed data
	assert.Equal(t, "David Heinemeier Hansson", version.Authors)
	assert.Equal(t, "7.0.5", version.Number)
	assert.Equal(t, "ruby", version.Platform)
	assert.Equal(t, "3.3.26", version.RubygemsVersion)
	assert.Equal(t, 54428, version.DownloadsCount)
	assert.Equal(t, "Full-stack web application framework", version.Summary)
	assert.Equal(t, "Ruby on Rails is a full-stack web framework", version.Description)
	assert.Equal(t, []string{"MIT"}, version.Licenses)
	assert.Equal(t, "https://api.rubyonrails.org/v7.0.5/", version.Metadata.DocumentationURI)
	assert.Equal(t, "https://github.com/rails/rails/issues", version.Metadata.BugTrackerURI)
}

func TestLatestVersion_MarshalUnmarshal(t *testing.T) {
	// Create a sample LatestVersion
	latestVersion := LatestVersion{
		Version: "2.0.0",
	}

	// Convert to JSON
	jsonData, err := json.Marshal(latestVersion)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Convert back from JSON
	var unmarshaledLatestVersion LatestVersion
	err = json.Unmarshal(jsonData, &unmarshaledLatestVersion)
	assert.NoError(t, err)

	// Check if fields match
	assert.Equal(t, latestVersion.Version, unmarshaledLatestVersion.Version)
}

func TestLatestVersion_JsonUnmarshal(t *testing.T) {
	// Sample JSON data
	jsonData := `{
		"version": "7.0.5"
	}`

	var latestVersion LatestVersion
	err := json.Unmarshal([]byte(jsonData), &latestVersion)
	assert.NoError(t, err)

	// Verify parsed data
	assert.Equal(t, "7.0.5", latestVersion.Version)
}
