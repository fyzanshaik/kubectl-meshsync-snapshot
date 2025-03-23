package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/models"
)

func SaveToFile(resources []*models.KubernetesResource, filePath string, options *models.Options) error {
	if options.VerboseMode {
		fmt.Printf("Saving %d resources to %s\n", len(resources), filePath)
	}

	snapshot := map[string]interface{}{
		"version": "v1",
		"timestamp": time.Now().Format(time.RFC3339),
		"resources": resources,
		"cluster_id": getClusterID(resources),
		"plugin_info": getPluginInfo(),
		"filter_options": getFilterOptions(options),
	}

	var data []byte
	var err error

	if options.OutputFormat == "yaml" {
		//TODO: Implement YAML output format
		return fmt.Errorf("YAML output format not yet implemented")
	} else {

		data, err = json.MarshalIndent(snapshot, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal snapshot to JSON: %w", err)
		}
	}

	if options.VerboseMode {
		fmt.Printf("JSON size: %d bytes\n", len(data))
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		if options.VerboseMode {
			fmt.Printf("Warning: Could not get absolute path: %v\n", err)
		}
		absPath = filePath
	}

	if options.VerboseMode {
		fmt.Printf("Writing to absolute path: %s\n", absPath)
	}

	err = os.WriteFile(absPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write snapshot to file: %w", err)
	}

	if _, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("failed to verify file was created: %w", err)
	}

	if options.VerboseMode {
		fmt.Printf("File successfully written: %s (%d bytes)\n", absPath, len(data))
	}
	return nil
}

func getClusterID(resources []*models.KubernetesResource) string {
	if len(resources) > 0 && resources[0] != nil {
		return resources[0].ClusterID
	}
	return "unknown"
}

func getPluginInfo() map[string]string {
	return map[string]string{
		"name": "kubectl-meshsync_snapshot",
		"version": "0.1.0",
		"description": "A kubectl plugin for capturing Kubernetes cluster state using MeshSync",
		"created_at": time.Now().Format(time.RFC3339),
	}
}

func getFilterOptions(options *models.Options) map[string]interface{} {
	result := map[string]interface{}{
		"namespaces": options.Namespace,
		"resource_type": options.ResourceType,
		"fast_mode": options.FastMode,
		"collection_time": options.CollectionTime.String(),
	}

	if options.LabelSelector != "" {
		result["label_selector"] = options.LabelSelector
	}

	if len(options.ExcludeTypes) > 0 {
		result["excluded_types"] = options.ExcludeTypes
	}

	return result
}