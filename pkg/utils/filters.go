package utils

import (
	"strings"

	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/models"
)

// FilterResources filters resources based on the provided options
func FilterResources(resources []*models.KubernetesResource, options *models.Options) []*models.KubernetesResource {
	if options.Namespace == "" && options.ResourceType == "" && 
	   !options.FastMode && len(options.ExcludeTypes) == 0 && 
	   options.LabelSelector == "" {
		return resources
	}
	
	var filtered []*models.KubernetesResource
	
	for _, resource := range resources {
		// Skip if excluded by type
		if options.IsTypeExcluded(resource.Kind) {
			continue
		}
		
		// Skip if not relevant in fast mode
		if !options.IsFastModeRelevant(resource.Kind) {
			continue
		}
		
		// Filter by namespace
		if options.Namespace != "" && resource.KubernetesResourceMeta != nil && 
		   resource.KubernetesResourceMeta.Namespace != options.Namespace {
			continue
		}
		
		// Filter by resource type (exact or plural form)
		if options.ResourceType != "" {
			resourceType := strings.ToLower(options.ResourceType)
			kind := strings.ToLower(resource.Kind)
			
			if kind != resourceType && 
			   kind != resourceType+"s" && 
			   (len(resourceType) > 1 && resourceType != kind+"s") {
				continue
			}
		}
		
		// Filter by label selector
		if options.LabelSelector != "" && resource.KubernetesResourceMeta != nil {
			if !matchesLabelSelector(resource.KubernetesResourceMeta.Labels, options.LabelSelector) {
				continue
			}
		}
		
		filtered = append(filtered, resource)
	}
	
	return filtered
}

// matchesLabelSelector checks if resource labels match a simple label selector
func matchesLabelSelector(labels []*models.KubernetesKeyValue, selector string) bool {
	if selector == "" || len(labels) == 0 {
		return false
	}
	
	// Parse simple selector in the form: "key=value"
	parts := strings.Split(selector, "=")
	if len(parts) != 2 {
		return false
	}
	
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	
	for _, label := range labels {
		if label.Key == key && label.Value == value {
			return true
		}
	}
	
	return false
}