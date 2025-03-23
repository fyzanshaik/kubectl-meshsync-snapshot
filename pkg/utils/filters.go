package utils

import (
	"strings"

	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/models"
)

func FilterResources(resources []*models.KubernetesResource, options *models.Options) []*models.KubernetesResource {
	if options.Namespace == "" && options.ResourceType == "" && 
	   !options.FastMode && len(options.ExcludeTypes) == 0 && 
	   options.LabelSelector == "" {
		return resources
	}

	var filtered []*models.KubernetesResource

	for _, resource := range resources {

		if options.IsTypeExcluded(resource.Kind) {
			continue
		}

		if !options.IsFastModeRelevant(resource.Kind) {
			continue
		}

		if options.Namespace != "" && resource.KubernetesResourceMeta != nil && 
		   resource.KubernetesResourceMeta.Namespace != options.Namespace {
			continue
		}

		if options.ResourceType != "" {
			resourceType := strings.ToLower(options.ResourceType)
			kind := strings.ToLower(resource.Kind)

			if kind != resourceType && 
			   kind != resourceType+"s" && 
			   (len(resourceType) > 1 && resourceType != kind+"s") {
				continue
			}
		}

		if options.LabelSelector != "" && resource.KubernetesResourceMeta != nil {
			if !matchesLabelSelector(resource.KubernetesResourceMeta.Labels, options.LabelSelector) {
				continue
			}
		}

		filtered = append(filtered, resource)
	}

	return filtered
}

func matchesLabelSelector(labels []*models.KubernetesKeyValue, selector string) bool {
	if selector == "" || len(labels) == 0 {
		return false
	}

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