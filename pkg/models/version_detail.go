package models

import "time"

// VersionDetail is the detailed version info returned by the API v2 endpoint
// GET /api/v2/rubygems/{gem}/versions/{version}.json
// contains more fields than the V1 Version struct (spec_sha, yanked, full dependency info, etc.)
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

// VersionContent represents the file checksum/manifest content of a gem version
// GET /api/v2/rubygems/{gem}/versions/{version}/contents.json
type VersionContent struct {
	// mapping from file path to SHA256 checksum
	Files map[string]string `json:"files"`
}

// Attestation represents the sigstore attestation of a gem version
// GET /api/v1/attestations/{gem}-{version}.json
type Attestation struct {
	// attestation body content
	Body string `json:"body"`
}
