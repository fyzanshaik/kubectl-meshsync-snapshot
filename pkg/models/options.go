package models

import (
	"time"
)

// Options represents global options used across packages
type Options struct {
	// Output options
	OutputFile      string
	AutoName        bool
	OutputFormat    string
	
	// Filter options
	Namespace       string
	ResourceType    string
	LabelSelector   string
	ExcludeTypes    []string
	
	// Execution options
	FastMode        bool
	CollectionTime  time.Duration
	
	// Display options
	QuietMode       bool
	VerboseMode     bool
	PreviewMode     bool
}

// NewDefaultOptions creates a new Options with default values
func NewDefaultOptions() *Options {
	return &Options{
		OutputFile:     "meshsync-snapshot.json",
		OutputFormat:   "json",
		CollectionTime: 30 * time.Second,
		QuietMode:      false,
		VerboseMode:    false,
		PreviewMode:    false,
		FastMode:       false,
		ExcludeTypes:   []string{},
	}
}

// IsTypeExcluded checks if a resource type is in the exclusion list
func (o *Options) IsTypeExcluded(resourceType string) bool {
	for _, t := range o.ExcludeTypes {
		if t == resourceType {
			return true
		}
	}
	return false
}

// IsFastModeRelevant checks if a resource type should be included in fast mode
func (o *Options) IsFastModeRelevant(resourceType string) bool {
	if !o.FastMode {
		return true
	}
	
	essentialTypes := map[string]bool{
		"Namespace":  true,
		"Pod":        true,
		"Service":    true,
		"Deployment": true,
		"Node":       true,
	}
	
	return essentialTypes[resourceType]
}