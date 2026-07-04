package models

// DependencyInfo is used for the /api/v1/dependencies endpoint
// reference: https://guides.rubygems.org/rubygems-org-api-v2/#dependencies
type DependencyInfo struct {
	// package name
	Name string `json:"name"`

	// dependent package name
	DependentName string `json:"dependent_name"`

	// version requirement, e.g.: ">= 1.0.0"
	Requirements string `json:"requirements"`

	// dependency type, common values: "runtime", "development"
	DependentType string `json:"dependent_type"`
}
