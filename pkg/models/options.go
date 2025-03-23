package models

import (
	"time"
)

type Options struct {

	OutputFile      string
	AutoName        bool
	OutputFormat    string

	Namespace       string
	ResourceType    string
	LabelSelector   string
	ExcludeTypes    []string

	FastMode        bool
	CollectionTime  time.Duration

	QuietMode       bool
	VerboseMode     bool
	PreviewMode     bool
}

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

func (o *Options) IsTypeExcluded(resourceType string) bool {
	for _, t := range o.ExcludeTypes {
		if t == resourceType {
			return true
		}
	}
	return false
}

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