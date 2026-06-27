package models

import "time"

type Version struct {
	Authors         string    `json:"authors"`
	BuiltAt         time.Time `json:"built_at"`
	CreatedAt       time.Time `json:"created_at"`
	Description     string    `json:"description"`
	DownloadsCount  int       `json:"downloads_count"`
	Metadata        *Metadata `json:"metadata,omitempty"`
	Number          string    `json:"number"`
	Summary         string    `json:"summary"`
	Platform        string    `json:"platform"`
	RubygemsVersion string    `json:"rubygems_version"`
	RubyVersion     string    `json:"ruby_version"`
	Prerelease      bool      `json:"prerelease"`
	Licenses        []string  `json:"licenses"`

	// Requirements 版本要求，通常是一个字符串数组
	// 例如: [">= 2.5.0", "< 3.0"]
	Requirements []string `json:"requirements"`

	Sha string `json:"sha"`
}

type LatestVersion struct {
	Version string `json:"version"`
}
