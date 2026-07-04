package models

// Webhook represents the Webhook configuration on RubyGems.org
// GET /api/v1/web_hooks.json
type Webhook struct {
	// Webhook callback URL
	URL string `json:"url"`

	// failure count
	FailureCount int `json:"failure_count"`
}

// TopDownloadedGem represents a gem package in the top 50 by downloads
// GET /api/v1/downloads/all.json
type TopDownloadedGem struct {
	// gem package name
	Name string `json:"name"`

	// total downloads
	Downloads int `json:"downloads"`
}
